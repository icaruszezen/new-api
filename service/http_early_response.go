package service

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptrace"
	"sync"
	"sync/atomic"
)

// earlyResponseConn keeps an upstream answer that arrives while the request
// body is still being sent. An upstream (or a load balancer in front of it)
// may reject a request after reading only its headers, answer for example
// 503, and close the connection with the body unread. The kernel then resets
// the connection, and net/http's body writer fails with "connection reset by
// peer" or "broken pipe". When that write error reaches the transport before
// its read loop has delivered the answer, Client.Do returns the write error
// and drops the answer, so the relay sees a 500 without the upstream status
// and the channel's retry policy cannot match it.
//
// Once a traced relay request (see TraceEarlyUpstreamResponse) has put bytes
// on the wire, a later write failing with a peer reset is held back until the
// transport closes the connection. Meanwhile the transport reads and returns
// whatever the upstream answered, or the read error if nothing was answered,
// and it never pools the connection because the write never succeeded. The
// first write of a request is not held back: net/http relies on that failure
// to resend a request on a fresh connection when a pooled one turns out to be
// dead.
//
// HTTP/2 connections carry concurrent streams, and their read loop needs the
// connection's write lock, so a held-back write would stall it. There the
// reset write is reported as successful instead and the rest of the stream is
// discarded; the read loop fails on the reset socket and retires the
// connection.
type earlyResponseConn struct {
	net.Conn

	closed    chan struct{}
	closeOnce sync.Once

	mu          sync.Mutex
	multiplexed bool
	// armed covers the writes of the traced HTTP/1 request using the
	// connection, from GotConn to WroteRequest.
	armed bool
	// wrote reports that the armed request has put bytes on the wire.
	wrote    bool
	resetErr error
}

func (c *earlyResponseConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	if c.resetErr != nil {
		multiplexed, resetErr := c.multiplexed, c.resetErr
		c.mu.Unlock()
		if multiplexed {
			return len(p), nil
		}
		return 0, resetErr
	}
	c.mu.Unlock()

	n, err := c.Conn.Write(p)

	c.mu.Lock()
	tracked := c.armed || c.multiplexed
	if tracked && n > 0 {
		c.wrote = true
	}
	if err == nil || !tracked || !c.wrote || !isPeerResetError(err) {
		c.mu.Unlock()
		return n, err
	}
	c.resetErr = err
	multiplexed := c.multiplexed
	c.mu.Unlock()
	if multiplexed {
		return len(p), nil
	}
	<-c.closed
	return n, err
}

func (c *earlyResponseConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return c.Conn.Close()
}

func (c *earlyResponseConn) beginRequest(multiplexed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if multiplexed {
		c.multiplexed = true
		return
	}
	if c.multiplexed {
		return
	}
	c.armed = true
	c.wrote = false
}

func (c *earlyResponseConn) endRequestWrite() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.armed = false
}

func dialEarlyResponseConn(dial func(ctx context.Context, network, address string) (net.Conn, error)) func(ctx context.Context, network, address string) (net.Conn, error) {
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dial(ctx, network, address)
		if err != nil {
			return nil, err
		}
		return &earlyResponseConn{Conn: conn, closed: make(chan struct{})}, nil
	}
}

// earlyResponseTrace tells the relay connection serving one request where
// that request's writes begin and end.
type earlyResponseTrace struct {
	conn atomic.Pointer[earlyResponseConn]
}

func (t *earlyResponseTrace) gotConn(info httptrace.GotConnInfo) {
	conn := info.Conn
	multiplexed := false
	if tlsConn, ok := conn.(*tls.Conn); ok {
		multiplexed = tlsConn.ConnectionState().NegotiatedProtocol == "h2"
	}
	// TLS, possibly to an HTTPS proxy and then to the upstream, sits on top
	// of the connection the dialer returned.
	for {
		inner, ok := conn.(interface{ NetConn() net.Conn })
		if !ok {
			break
		}
		conn = inner.NetConn()
	}
	earlyConn, ok := conn.(*earlyResponseConn)
	if !ok {
		t.conn.Store(nil)
		return
	}
	t.conn.Store(earlyConn)
	earlyConn.beginRequest(multiplexed)
}

func (t *earlyResponseTrace) wroteRequest(httptrace.WroteRequestInfo) {
	if conn := t.conn.Load(); conn != nil {
		conn.endRequestWrite()
	}
}

// TraceEarlyUpstreamResponse returns req traced so that an upstream answer
// arriving while its body is still being sent is returned instead of the
// body write error (see earlyResponseConn).
func TraceEarlyUpstreamResponse(req *http.Request) *http.Request {
	trace := &earlyResponseTrace{}
	return req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
		GotConn:      trace.gotConn,
		WroteRequest: trace.wroteRequest,
	}))
}
