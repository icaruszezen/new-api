//go:build !windows

package service

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errWriteReset = &net.OpError{Op: "write", Net: "tcp", Err: os.NewSyscallError("write", syscall.ECONNRESET)}
	errReadReset  = &net.OpError{Op: "read", Net: "tcp", Err: os.NewSyscallError("read", syscall.ECONNRESET)}
)

// earlyRejectConn is an upstream that answers after reading only the request
// headers and then resets the connection with the body unread. Its answer
// becomes readable a moment after a body write has failed, the order in which
// net/http used to report the write error and drop the answer.
type earlyRejectConn struct {
	mu        sync.Mutex
	answer    *strings.Reader
	wrote     bool
	reset     chan struct{}
	resetOnce sync.Once
	closed    chan struct{}
	closeOnce sync.Once
}

func newEarlyRejectConn(answer string) *earlyRejectConn {
	return &earlyRejectConn{answer: strings.NewReader(answer), reset: make(chan struct{}), closed: make(chan struct{})}
}

func (c *earlyRejectConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.wrote {
		c.resetOnce.Do(func() { close(c.reset) })
		return 0, errWriteReset
	}
	c.wrote = true
	return len(p), nil
}

func (c *earlyRejectConn) Read(p []byte) (int, error) {
	select {
	case <-c.reset:
	case <-c.closed:
		return 0, net.ErrClosed
	}
	select {
	case <-time.After(100 * time.Millisecond):
	case <-c.closed:
		return 0, net.ErrClosed
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.answer.Len() == 0 {
		return 0, errReadReset
	}
	return c.answer.Read(p)
}

func (c *earlyRejectConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return nil
}

func (c *earlyRejectConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 40000}
}
func (c *earlyRejectConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 80}
}
func (c *earlyRejectConn) SetDeadline(time.Time) error      { return nil }
func (c *earlyRejectConn) SetReadDeadline(time.Time) error  { return nil }
func (c *earlyRejectConn) SetWriteDeadline(time.Time) error { return nil }

func TestRelayConnKeepsUpstreamAnswerSentBeforeBodyWasRead(t *testing.T) {
	const answerBody = `{"error":{"message":"No available accounts"}}`
	rejection := "HTTP/1.1 503 Service Unavailable\r\nContent-Type: application/json\r\nContent-Length: 45\r\nConnection: close\r\n\r\n" + answerBody

	tests := []struct {
		name       string
		answer     string
		wantStatus int
	}{
		{name: "upstream answer is returned", answer: rejection, wantStatus: http.StatusServiceUnavailable},
		{name: "reset without an answer still fails", answer: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := newEarlyRejectConn(tt.answer)
			client := &http.Client{
				Timeout: 5 * time.Second,
				Transport: &http.Transport{DialContext: dialEarlyResponseConn(func(context.Context, string, string) (net.Conn, error) {
					return upstream, nil
				})},
			}
			req, err := http.NewRequest(http.MethodPost, "http://upstream.test/v1/messages", bytes.NewReader(make([]byte, 1<<20)))
			require.NoError(t, err)

			resp, err := client.Do(TraceEarlyUpstreamResponse(req))

			if tt.wantStatus == 0 {
				require.Error(t, err)
				assert.ErrorIs(t, err, syscall.ECONNRESET)
				return
			}
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.JSONEq(t, answerBody, string(body))
		})
	}
}

// resettableConn is a pooled connection whose peer reset it while it sat
// idle: writes fail, and reads keep blocking as if the reset had not been
// noticed yet.
type resettableConn struct {
	net.Conn
	reset atomic.Bool
}

func (c *resettableConn) Write(p []byte) (int, error) {
	if c.reset.Load() {
		return 0, errWriteReset
	}
	return c.Conn.Write(p)
}

func serveOKOverPipe(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			return
		}
		_, _ = io.Copy(io.Discard, req.Body)
		if _, err := io.WriteString(conn, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok"); err != nil {
			return
		}
	}
}

func TestRelayConnStillResendsOnDeadPooledConnection(t *testing.T) {
	var dials atomic.Int32
	var firstConn *resettableConn
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{DialContext: dialEarlyResponseConn(func(context.Context, string, string) (net.Conn, error) {
			clientEnd, serverEnd := net.Pipe()
			go serveOKOverPipe(serverEnd)
			conn := &resettableConn{Conn: clientEnd}
			if dials.Add(1) == 1 {
				firstConn = conn
			}
			return conn, nil
		})},
	}
	t.Cleanup(client.CloseIdleConnections)
	send := func() *http.Response {
		req, err := http.NewRequest(http.MethodPost, "http://upstream.test/v1/messages", strings.NewReader(`{"model":"m"}`))
		require.NoError(t, err)
		resp, err := client.Do(TraceEarlyUpstreamResponse(req))
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		assert.Equal(t, "ok", string(body))
		return resp
	}

	send()
	firstConn.reset.Store(true)
	resp := send()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), dials.Load())
}

func TestRelayClientReportsProxyConnectRefusal(t *testing.T) {
	withRelayHTTPTransportSettings(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			if _, err := http.ReadRequest(bufio.NewReader(conn)); err == nil {
				_, _ = io.WriteString(conn, "HTTP/1.1 503 No available proxy\r\nContent-Length: 0\r\n\r\n")
			}
			_ = conn.Close()
		}
	}()
	proxyURL, err := url.Parse("http://" + listener.Addr().String())
	require.NoError(t, err)
	client, err := newProxyHTTPClient(proxyURL)
	require.NoError(t, err)

	_, err = client.Get("https://upstream.invalid/v1/models")

	var proxyErr *ProxyConnectError
	require.ErrorAs(t, err, &proxyErr)
	assert.Equal(t, http.StatusServiceUnavailable, proxyErr.StatusCode)
	assert.Equal(t, "503 No available proxy", proxyErr.Status)
}
