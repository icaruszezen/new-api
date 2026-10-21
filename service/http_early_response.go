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
// on the wire, a peer reset on a later write is swallowed and the rest of that
// request is discarded, so the transport goes on to read whatever the
// upstream answered. If nothing was answered, the read side fails with the
// reset instead. The first write of a request is never swallowed: net/http
// relies on that failure to resend a request on a fresh connection when a
// pooled one turns out to be dead, and a later request on a connection that
// was reset fails its first write with the stored error for the same reason.
type earlyResponseConn struct {
	net.Conn

	mu sync.Mutex
	// multiplexed is set for HTTP/2 connections, which carry concurrent
	// requests and have no resend-on-dead-connection rule to preserve.
	multiplexed bool
	// armed covers the writes of the traced HTTP/1 request using the
	// connection, from GotConn to WroteRequest.
	armed bool
	// wrote reports that the armed request has put bytes on the wire.
	wrote bool
	// discarding stays set from a swallowed reset until the next request,
	// which also covers the transport's final buffer flush after WroteRequest.
	discarding bool
	resetErr   error
}

func (c *earlyResponseConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	if c.resetErr != nil {
		discarding, resetErr := c.discarding, c.resetErr
		c.mu.Unlock()
		if discarding {
			return len(p), nil
		}
		return 0, resetErr
	}
	c.mu.Unlock()

	n, err := c.Conn.Write(p)

	c.mu.Lock()
	defer c.mu.Unlock()
	tracked := c.armed || c.multiplexed
	if tracked && n > 0 {
		c.wrote = true
	}
	if err == nil || !tracked || !c.wrote || !isPeerResetError(err) {
		return n, err
	}
	c.resetErr = err
	c.discarding = true
	return len(p), nil
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
	c.discarding = false
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
		return &earlyResponseConn{Conn: conn}, nil
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
