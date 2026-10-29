package helper

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
	if constant.StreamingTimeout == 0 {
		constant.StreamingTimeout = 30
	}
}

func setupStreamTest(t *testing.T, body io.Reader) (*gin.Context, *http.Response, *relaycommon.RelayInfo) {
	t.Helper()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	resp := &http.Response{
		Body: io.NopCloser(body),
	}

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	}

	return c, resp, info
}

func buildSSEBody(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "data: {\"id\":%d,\"choices\":[{\"delta\":{\"content\":\"token_%d\"}}]}\n", i, i)
	}
	b.WriteString("data: [DONE]\n")
	return b.String()
}

// ---------- Basic correctness ----------

func TestStreamScannerHandler_NilInputs(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}

	StreamScannerHandler(c, nil, info, func(data string, sr *StreamResult) {})
	StreamScannerHandler(c, &http.Response{Body: io.NopCloser(strings.NewReader(""))}, info, nil)
}

func TestNewStreamScanner_AllowsLargeStreamLine(t *testing.T) {
	oldBufferMB := constant.StreamScannerMaxBufferMB
	constant.StreamScannerMaxBufferMB = 1
	t.Cleanup(func() {
		constant.StreamScannerMaxBufferMB = oldBufferMB
	})

	payload := strings.Repeat("x", 128<<10)
	scanner := NewStreamScanner(strings.NewReader("data: " + payload + "\n"))
	scanner.Split(bufio.ScanLines)

	require.True(t, scanner.Scan())
	assert.Equal(t, "data: "+payload, scanner.Text())
	require.NoError(t, scanner.Err())
}

func TestStreamScannerHandler_EmptyBody(t *testing.T) {
	t.Parallel()

	c, resp, info := setupStreamTest(t, strings.NewReader(""))

	var called atomic.Bool
	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
		called.Store(true)
	})

	assert.False(t, called.Load(), "handler should not be called for empty body")
}

func TestStreamScannerHandler_1000Chunks(t *testing.T) {
	t.Parallel()

	const numChunks = 1000
	body := buildSSEBody(numChunks)
	c, resp, info := setupStreamTest(t, strings.NewReader(body))

	var count atomic.Int64
	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
		count.Add(1)
	})

	assert.Equal(t, int64(numChunks), count.Load())
	assert.Equal(t, numChunks, info.ReceivedResponseCount)
}

func TestStreamScannerHandler_OrderPreserved(t *testing.T) {
	t.Parallel()

	const numChunks = 500
	body := buildSSEBody(numChunks)
	c, resp, info := setupStreamTest(t, strings.NewReader(body))

	var mu sync.Mutex
	received := make([]string, 0, numChunks)

	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
		mu.Lock()
		received = append(received, data)
		mu.Unlock()
	})

	require.Equal(t, numChunks, len(received))
	for i := 0; i < numChunks; i++ {
		expected := fmt.Sprintf("{\"id\":%d,\"choices\":[{\"delta\":{\"content\":\"token_%d\"}}]}", i, i)
		assert.Equal(t, expected, received[i], "chunk %d out of order", i)
	}
}

func TestStreamScannerHandler_DoneStopsScanner(t *testing.T) {
	t.Parallel()

	body := buildSSEBody(50) + "data: should_not_appear\n"
	c, resp, info := setupStreamTest(t, strings.NewReader(body))

	var count atomic.Int64
	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
		count.Add(1)
	})

	assert.Equal(t, int64(50), count.Load(), "data after [DONE] must not be processed")
}

func TestStreamScannerHandler_StopStopsStream(t *testing.T) {
	t.Parallel()

	const numChunks = 200
	body := buildSSEBody(numChunks)
	c, resp, info := setupStreamTest(t, strings.NewReader(body))

	const stopAt int64 = 50
	var count atomic.Int64
	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
		n := count.Add(1)
		if n >= stopAt {
			sr.Stop(fmt.Errorf("fatal at %d", n))
		}
	})

	assert.Equal(t, stopAt, count.Load())
	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonHandlerStop, info.StreamStatus.EndReason)
}

func TestStreamScannerHandler_SkipsNonDataLines(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	b.WriteString(": comment line\n")
	b.WriteString("event: message\n")
	b.WriteString("id: 12345\n")
	b.WriteString("retry: 5000\n")
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&b, "data: payload_%d\n", i)
		b.WriteString(": interleaved comment\n")
	}
	b.WriteString("data: [DONE]\n")

	c, resp, info := setupStreamTest(t, strings.NewReader(b.String()))

	var count atomic.Int64
	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
		count.Add(1)
	})

	assert.Equal(t, int64(100), count.Load())
}

func TestStreamScannerHandler_DataWithExtraSpaces(t *testing.T) {
	t.Parallel()

	body := "data:   {\"trimmed\":true}  \ndata: [DONE]\n"
	c, resp, info := setupStreamTest(t, strings.NewReader(body))

	var got string
	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
		got = data
	})

	assert.Equal(t, "{\"trimmed\":true}", got)
}

// TestStreamScannerHandler_ClientCancelAbortsUpstreamAndReturns pins the
// disconnect contract: when the client goes away, the handler must return
// promptly (all goroutines joined, so the gin.Context can never leak into a
// pooled reuse), the upstream body must be closed to stop token generation,
// and no data received after the disconnect may be processed or written.
func TestStreamScannerHandler_ClientCancelAbortsUpstreamAndReturns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pr, pw := io.Pipe()
	t.Cleanup(func() {
		_ = pr.Close()
		_ = pw.Close()
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil).WithContext(ctx)

	resp := &http.Response{Body: pr}
	info := &relaycommon.RelayInfo{
		DisablePing: true,
		ChannelMeta: &relaycommon.ChannelMeta{},
	}

	var count atomic.Int64
	firstHandled := make(chan struct{})
	done := make(chan struct{})
	go func() {
		StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
			count.Add(1)
			_ = StringData(c, data)
			if data == "first" {
				close(firstHandled)
			}
		})
		close(done)
	}()

	_, err := fmt.Fprint(pw, "data: first\n")
	require.NoError(t, err)

	select {
	case <-firstHandled:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first chunk")
	}

	cancel()

	// The handler must return without any further upstream input: cleanup
	// closes resp.Body, which unblocks the scanner goroutine.
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not return after client disconnect")
	}

	// Upstream read side must be closed so the provider stops generating
	// (and billing) for a request nobody is listening to.
	_, err = fmt.Fprint(pw, "data: second\n")
	require.ErrorIs(t, err, io.ErrClosedPipe, "upstream body should be closed after client disconnect")

	assert.Equal(t, int64(1), count.Load(), "no chunk after disconnect should be processed")
	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)

	body := recorder.Body.String()
	assert.Contains(t, body, "first")
	assert.NotContains(t, body, "second")
}

func TestStreamScannerHandler_ClientCancelDrainsUntilTerminalWhenEnabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pr, pw := io.Pipe()
	t.Cleanup(func() {
		_ = pr.Close()
		_ = pw.Close()
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil).WithContext(ctx)

	resp := &http.Response{Body: pr}
	info := &relaycommon.RelayInfo{
		DisablePing:                     true,
		DrainUpstreamOnClientDisconnect: true,
		ChannelMeta:                     &relaycommon.ChannelMeta{},
	}

	var last atomic.Value
	firstHandled := make(chan struct{})
	done := make(chan struct{})
	go func() {
		StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
			last.Store(data)
			_ = StringData(c, data)
			if strings.Contains(data, `"type":"response.created"`) {
				close(firstHandled)
			}
		})
		close(done)
	}()

	_, err := fmt.Fprint(pw, `data: {"type":"response.created","response":{"id":"resp_1"}}`+"\n")
	require.NoError(t, err)

	select {
	case <-firstHandled:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first chunk")
	}

	cancel()

	select {
	case <-done:
		t.Fatal("handler returned immediately after client disconnect")
	case <-time.After(200 * time.Millisecond):
	}

	completed := `{"type":"response.completed","response":{"usage":{"input_tokens":3,"output_tokens":5}}}`
	_, err = fmt.Fprint(pw, "data: "+completed+"\n")
	require.NoError(t, err)
	_, err = fmt.Fprint(pw, "data: [DONE]\n")
	require.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not return after upstream terminal")
	}

	got, _ := last.Load().(string)
	assert.Contains(t, got, `"type":"response.completed"`)

	_, err = fmt.Fprint(pw, "data: after-close\n")
	require.ErrorIs(t, err, io.ErrClosedPipe, "upstream body should be closed after drain finishes")

	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonDrained, info.StreamStatus.EndReason)
	assert.True(t, info.StreamStatus.IsNormalEnd())
	assert.False(t, info.StreamStatus.HasErrors())

	body := recorder.Body.String()
	assert.Contains(t, body, "response.created")
	assert.NotContains(t, body, "response.completed")
}

func TestStreamScannerHandler_ClientCancelDrainIdleTimeoutClosesBody(t *testing.T) {
	// Not parallel: modifies global constant.StreamingTimeout
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 1
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pr, pw := io.Pipe()
	t.Cleanup(func() {
		_ = pr.Close()
		_ = pw.Close()
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil).WithContext(ctx)

	resp := &http.Response{Body: pr}
	info := &relaycommon.RelayInfo{
		DisablePing:                     true,
		DrainUpstreamOnClientDisconnect: true,
		ChannelMeta:                     &relaycommon.ChannelMeta{},
	}

	firstHandled := make(chan struct{})
	done := make(chan struct{})
	go func() {
		StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
			_ = StringData(c, data)
			close(firstHandled)
		})
		close(done)
	}()

	_, err := fmt.Fprint(pw, "data: {\"type\":\"response.created\"}\n")
	require.NoError(t, err)

	select {
	case <-firstHandled:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first chunk")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not return after drain idle timeout")
	}

	_, err = fmt.Fprint(pw, "data: late\n")
	require.ErrorIs(t, err, io.ErrClosedPipe, "upstream body should be closed after drain timeout")

	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
	assert.True(t, info.StreamStatus.HasErrors())
}

// drainTestSetup wires a gin context whose request context the test cancels to
// simulate a client disconnect, and an upstream body the test writes to.
func drainTestSetup(t *testing.T, drain bool) (*gin.Context, *httptest.ResponseRecorder, context.CancelFunc, *io.PipeWriter, *http.Response, *relaycommon.RelayInfo) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	pr, pw := io.Pipe()
	t.Cleanup(func() {
		cancel()
		_ = pr.Close()
		_ = pw.Close()
	})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil).WithContext(ctx)
	info := &relaycommon.RelayInfo{
		DisablePing:                     true,
		DrainUpstreamOnClientDisconnect: drain,
		ChannelMeta:                     &relaycommon.ChannelMeta{},
	}
	return c, recorder, cancel, pw, &http.Response{Body: pr}, info
}

func waitHandlerDone(t *testing.T, done <-chan struct{}, within time.Duration, msg string) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(within):
		t.Fatal(msg)
	}
}

func TestStreamScannerHandler_ClientCancelDrainKeepsReadingThroughComments(t *testing.T) {
	// Not parallel: modifies global constant.StreamingTimeout
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 1
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	c, _, cancel, pw, resp, info := drainTestSetup(t, true)
	var last atomic.Value
	firstHandled := make(chan struct{})
	done := make(chan struct{})
	go func() {
		StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
			last.Store(data)
			sr.Error(StringData(c, data))
			if data == "first" {
				close(firstHandled)
			}
		})
		close(done)
	}()

	_, err := fmt.Fprint(pw, "data: first\n")
	require.NoError(t, err)
	<-firstHandled
	cancel()

	// An upstream that only sends SSE comments for longer than STREAMING_TIMEOUT
	// (long reasoning) is alive, exactly as the connected path treats it.
	for i := 0; i < 8; i++ {
		time.Sleep(200 * time.Millisecond)
		_, err = fmt.Fprint(pw, ": keepalive\n")
		require.NoError(t, err, "drain must not close an upstream that keeps sending lines")
	}
	_, err = fmt.Fprint(pw, "data: usage\ndata: [DONE]\n")
	require.NoError(t, err)
	waitHandlerDone(t, done, 2*time.Second, "handler did not return after upstream terminal")

	assert.Equal(t, "usage", last.Load())
	assert.Equal(t, relaycommon.StreamEndReasonDrained, info.StreamStatus.EndReason)
	assert.Equal(t, relaycommon.StreamEndReasonDone, info.StreamStatus.UpstreamEndReason())
	assert.True(t, info.StreamStatus.ClientGone)
	assert.Positive(t, info.StreamStatus.DrainDuration)
	// Writes skipped because the client left are not stream errors.
	assert.False(t, info.StreamStatus.HasErrors())
}

func TestStreamScannerHandler_ClientCancelDrainStopsAtMaxDuration(t *testing.T) {
	// Not parallel: modifies global constants
	oldTimeout, oldMax := constant.StreamingTimeout, constant.StreamDrainMaxSeconds
	constant.StreamingTimeout = 30
	constant.StreamDrainMaxSeconds = 1
	t.Cleanup(func() { constant.StreamingTimeout, constant.StreamDrainMaxSeconds = oldTimeout, oldMax })

	c, _, cancel, pw, resp, info := drainTestSetup(t, true)
	done := make(chan struct{})
	go func() {
		StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {})
		close(done)
	}()
	cancel()

	// Keepalives alone must not hold a drain open forever.
	deadline := time.After(3 * time.Second)
	keepalive := time.NewTicker(100 * time.Millisecond)
	defer keepalive.Stop()
	for {
		select {
		case <-done:
			_, err := fmt.Fprint(pw, "data: late\n")
			require.ErrorIs(t, err, io.ErrClosedPipe, "upstream body should be closed after the drain cap")
			assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
			require.True(t, info.StreamStatus.HasErrors())
			assert.Equal(t, "drain max duration reached", info.StreamStatus.Errors[0].Message)
			return
		case <-deadline:
			t.Fatal("drain was not capped by STREAM_DRAIN_MAX_SECONDS")
		case <-keepalive.C:
			_, _ = fmt.Fprint(pw, ": keepalive\n")
		}
	}
}

func TestStreamScannerHandler_ClientCancelBeforeDataKeepsOriginalIdleWindow(t *testing.T) {
	// Not parallel: modifies global constant.StreamingTimeout
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 1
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	c, _, cancel, _, resp, info := drainTestSetup(t, true)
	start := time.Now()
	done := make(chan struct{})
	go func() {
		StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {})
		close(done)
	}()
	time.Sleep(800 * time.Millisecond)
	cancel()
	waitHandlerDone(t, done, 3*time.Second, "handler did not return after drain idle timeout")

	// The silent upstream is cut when the idle window that started with the
	// stream runs out, not a fresh STREAMING_TIMEOUT after the disconnect.
	assert.Less(t, time.Since(start), 1500*time.Millisecond)
	assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
	require.True(t, info.StreamStatus.HasErrors())
	assert.Equal(t, "drain idle timeout", info.StreamStatus.Errors[0].Message)
}

func TestStreamScannerHandler_ClientCancelDrainEOFKeepsUpstreamEnd(t *testing.T) {
	c, _, cancel, pw, resp, info := drainTestSetup(t, true)
	firstHandled := make(chan struct{})
	done := make(chan struct{})
	go func() {
		StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
			if data == "first" {
				close(firstHandled)
			}
		})
		close(done)
	}()
	_, err := fmt.Fprint(pw, "data: first\n")
	require.NoError(t, err)
	<-firstHandled
	cancel()
	_, err = fmt.Fprint(pw, "data: last\n")
	require.NoError(t, err)
	require.NoError(t, pw.Close())
	waitHandlerDone(t, done, 2*time.Second, "handler did not return after upstream EOF")

	assert.Equal(t, relaycommon.StreamEndReasonDrained, info.StreamStatus.EndReason)
	// Callers that must tell a truncated upstream from a completed one still can.
	assert.Equal(t, relaycommon.StreamEndReasonEOF, info.StreamStatus.UpstreamEndReason())
}

func TestStreamScannerHandler_ClientCancelDrainKeepsHandlerStop(t *testing.T) {
	c, _, cancel, pw, resp, info := drainTestSetup(t, true)
	firstHandled := make(chan struct{})
	done := make(chan struct{})
	go func() {
		StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
			switch data {
			case "first":
				close(firstHandled)
			case "upstream-error":
				sr.Stop(fmt.Errorf("upstream error event"))
			}
		})
		close(done)
	}()
	_, err := fmt.Fprint(pw, "data: first\n")
	require.NoError(t, err)
	<-firstHandled
	cancel()
	_, err = fmt.Fprint(pw, "data: upstream-error\n")
	require.NoError(t, err)
	waitHandlerDone(t, done, 2*time.Second, "handler did not return after handler stop")

	// A drain that ends on an upstream error is not relabelled as drained.
	assert.Equal(t, relaycommon.StreamEndReasonHandlerStop, info.StreamStatus.EndReason)
	assert.True(t, info.StreamStatus.ClientGone)
}

func TestStreamScannerHandler_DrainNotMaskedByPingFailingAfterDisconnect(t *testing.T) {
	setting := operation_setting.GetGeneralSetting()
	oldEnabled, oldSec := setting.PingIntervalEnabled, setting.PingIntervalSeconds
	setting.PingIntervalEnabled = true
	setting.PingIntervalSeconds = 1
	t.Cleanup(func() { setting.PingIntervalEnabled, setting.PingIntervalSeconds = oldEnabled, oldSec })

	c, _, cancel, pw, resp, info := drainTestSetup(t, true)
	info.DisablePing = false

	firstDone := make(chan struct{})
	var calls atomic.Int32
	done := make(chan struct{})
	go func() {
		StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
			if calls.Add(1) == 1 {
				// A write to a dead client blocks while holding the write lock
				// until net/http fails it and cancels the request context. The
				// ping tick at 1s waits on the same lock meanwhile.
				time.Sleep(1300 * time.Millisecond)
				cancel()
				close(firstDone)
			}
			sr.Error(StringData(c, data))
		})
		close(done)
	}()

	_, err := fmt.Fprint(pw, "data: first\n")
	require.NoError(t, err)
	<-firstDone
	_, err = fmt.Fprint(pw, "data: usage\ndata: [DONE]\n")
	require.NoError(t, err)
	waitHandlerDone(t, done, 3*time.Second, "handler did not return")

	assert.Equal(t, relaycommon.StreamEndReasonDrained, info.StreamStatus.EndReason)
	assert.False(t, info.StreamStatus.HasErrors())
}

func TestStreamScannerHandler_ClientCancelWithoutDrainNotMaskedByPing(t *testing.T) {
	setting := operation_setting.GetGeneralSetting()
	oldEnabled, oldSec := setting.PingIntervalEnabled, setting.PingIntervalSeconds
	setting.PingIntervalEnabled = true
	setting.PingIntervalSeconds = 1
	t.Cleanup(func() { setting.PingIntervalEnabled, setting.PingIntervalSeconds = oldEnabled, oldSec })

	c, _, cancel, pw, resp, info := drainTestSetup(t, false)
	info.DisablePing = false

	var calls atomic.Int32
	done := make(chan struct{})
	go func() {
		StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
			if calls.Add(1) == 1 {
				time.Sleep(1300 * time.Millisecond)
				cancel()
			}
		})
		close(done)
	}()

	_, err := fmt.Fprint(pw, "data: first\n")
	require.NoError(t, err)
	waitHandlerDone(t, done, 3*time.Second, "handler did not return after client disconnect")

	assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
	assert.True(t, info.StreamStatus.ClientGone)
}

func TestStreamScannerHandler_BareDoneLineEndsStream(t *testing.T) {
	t.Parallel()
	body := "data: {\"id\":1}\n[DONE]\ndata: {\"id\":2}\n"
	c, resp, info := setupStreamTest(t, strings.NewReader(body))
	info.DisablePing = true

	var got []string
	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
		got = append(got, data)
	})

	assert.Equal(t, []string{`{"id":1}`}, got)
	assert.Equal(t, relaycommon.StreamEndReasonDone, info.StreamStatus.EndReason)
}

func TestStreamLineScannerHandler_PassesRawLinesUntilDone(t *testing.T) {
	t.Parallel()
	body := "{\"done\":false}\n\n  event: message  \ndata: [DONE]\n{\"done\":true}\n{\"after\":1}\n"
	c, resp, info := setupStreamTest(t, strings.NewReader(body))
	info.DisablePing = true

	var got []string
	StreamLineScannerHandler(c, resp, info, func(line string, sr *StreamResult) {
		got = append(got, line)
		if line == `{"done":true}` {
			sr.Done()
		}
	})

	// Blank lines are skipped; other lines arrive untouched (raw-text
	// protocols keep their whitespace), and "[DONE]" is only a terminator for
	// SSE upstreams. Nothing after sr.Done() reaches the handler.
	assert.Equal(t, []string{`{"done":false}`, `  event: message  `, `data: [DONE]`, `{"done":true}`}, got)
	assert.True(t, info.StreamStatus.IsNormalEnd())
}

func TestStreamLineScannerHandler_ClientCancelDrainsToEOF(t *testing.T) {
	c, _, cancel, pw, resp, info := drainTestSetup(t, true)
	var last atomic.Value
	firstHandled := make(chan struct{})
	done := make(chan struct{})
	go func() {
		StreamLineScannerHandler(c, resp, info, func(line string, sr *StreamResult) {
			last.Store(line)
			if line == `{"n":1}` {
				close(firstHandled)
			}
		})
		close(done)
	}()
	_, err := fmt.Fprint(pw, "{\"n\":1}\n")
	require.NoError(t, err)
	<-firstHandled
	cancel()
	_, err = fmt.Fprint(pw, "{\"n\":2,\"usage\":true}\n")
	require.NoError(t, err)
	require.NoError(t, pw.Close())
	waitHandlerDone(t, done, 2*time.Second, "handler did not return after upstream EOF")

	assert.Equal(t, `{"n":2,"usage":true}`, last.Load())
	assert.Equal(t, relaycommon.StreamEndReasonDrained, info.StreamStatus.EndReason)
}

// ---------- Ping tests ----------

func TestStreamScannerHandler_PingSentDuringSlowUpstream(t *testing.T) {
	setting := operation_setting.GetGeneralSetting()
	oldEnabled := setting.PingIntervalEnabled
	oldSeconds := setting.PingIntervalSeconds
	setting.PingIntervalEnabled = true
	setting.PingIntervalSeconds = 1
	t.Cleanup(func() {
		setting.PingIntervalEnabled = oldEnabled
		setting.PingIntervalSeconds = oldSeconds
	})

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		for i := 0; i < 4; i++ {
			fmt.Fprintf(pw, "data: chunk_%d\n", i)
			time.Sleep(400 * time.Millisecond)
		}
		fmt.Fprint(pw, "data: [DONE]\n")
	}()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	resp := &http.Response{Body: pr}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}

	var count atomic.Int64
	done := make(chan struct{})
	go func() {
		StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
			count.Add(1)
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for stream to finish")
	}

	assert.Equal(t, int64(4), count.Load())

	body := recorder.Body.String()
	pingCount := strings.Count(body, ": PING")
	assert.GreaterOrEqual(t, pingCount, 1,
		"expected at least 1 ping during slow stream with 1s interval; got %d", pingCount)
}

func TestStreamScannerHandler_PingDisabledByRelayInfo(t *testing.T) {
	setting := operation_setting.GetGeneralSetting()
	oldEnabled := setting.PingIntervalEnabled
	oldSeconds := setting.PingIntervalSeconds
	setting.PingIntervalEnabled = true
	setting.PingIntervalSeconds = 1
	t.Cleanup(func() {
		setting.PingIntervalEnabled = oldEnabled
		setting.PingIntervalSeconds = oldSeconds
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	resp := &http.Response{Body: io.NopCloser(strings.NewReader(buildSSEBody(5)))}
	info := &relaycommon.RelayInfo{
		DisablePing: true,
		ChannelMeta: &relaycommon.ChannelMeta{},
	}

	var count atomic.Int64
	done := make(chan struct{})
	go func() {
		StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
			count.Add(1)
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	assert.Equal(t, int64(5), count.Load())

	body := recorder.Body.String()
	pingCount := strings.Count(body, ": PING")
	assert.Equal(t, 0, pingCount, "pings should be disabled when DisablePing=true")
}

// ---------- StreamStatus integration ----------

func TestStreamScannerHandler_StreamStatus_DoneReason(t *testing.T) {
	t.Parallel()

	body := buildSSEBody(10)
	c, resp, info := setupStreamTest(t, strings.NewReader(body))

	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {})

	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonDone, info.StreamStatus.EndReason)
	assert.Nil(t, info.StreamStatus.EndError)
	assert.True(t, info.StreamStatus.IsNormalEnd())
	assert.False(t, info.StreamStatus.HasErrors())
}

func TestStreamScannerHandler_StreamStatus_EOFWithoutDone(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&b, "data: {\"id\":%d}\n", i)
	}
	c, resp, info := setupStreamTest(t, strings.NewReader(b.String()))

	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {})

	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonEOF, info.StreamStatus.EndReason)
	assert.True(t, info.StreamStatus.IsNormalEnd())
}

func TestStreamScannerHandler_StreamStatus_HandlerStop(t *testing.T) {
	t.Parallel()

	body := buildSSEBody(100)
	c, resp, info := setupStreamTest(t, strings.NewReader(body))

	var count atomic.Int64
	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
		n := count.Add(1)
		if n >= 10 {
			sr.Stop(fmt.Errorf("stop at 10"))
		}
	})

	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonHandlerStop, info.StreamStatus.EndReason)
	assert.True(t, info.StreamStatus.HasErrors())
}

func TestStreamScannerHandler_StreamStatus_HandlerDone(t *testing.T) {
	t.Parallel()

	body := buildSSEBody(20)
	c, resp, info := setupStreamTest(t, strings.NewReader(body))

	var count atomic.Int64
	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
		n := count.Add(1)
		if n >= 5 {
			sr.Done()
		}
	})

	assert.Equal(t, int64(5), count.Load())
	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonDone, info.StreamStatus.EndReason)
	assert.False(t, info.StreamStatus.HasErrors())
}

func TestStreamScannerHandler_StreamStatus_Timeout(t *testing.T) {
	// Not parallel: modifies global constant.StreamingTimeout
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 1
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	pr, pw := io.Pipe()
	go func() {
		fmt.Fprint(pw, "data: {\"id\":1}\n")
		time.Sleep(2 * time.Second)
		pw.Close()
	}()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	resp := &http.Response{Body: pr}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}

	done := make(chan struct{})
	go func() {
		StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for stream timeout")
	}

	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonTimeout, info.StreamStatus.EndReason)
	assert.False(t, info.StreamStatus.IsNormalEnd())
}

func TestStreamScannerHandler_StreamStatus_SoftErrors(t *testing.T) {
	t.Parallel()

	body := buildSSEBody(10)
	c, resp, info := setupStreamTest(t, strings.NewReader(body))

	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
		sr.Error(fmt.Errorf("soft error for chunk"))
	})

	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonDone, info.StreamStatus.EndReason)
	assert.True(t, info.StreamStatus.HasErrors())
	assert.Equal(t, 10, info.StreamStatus.TotalErrorCount())
}

func TestStreamScannerHandler_StreamStatus_MultipleErrorsPerChunk(t *testing.T) {
	t.Parallel()

	body := buildSSEBody(5)
	c, resp, info := setupStreamTest(t, strings.NewReader(body))

	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
		sr.Error(fmt.Errorf("error A"))
		sr.Error(fmt.Errorf("error B"))
	})

	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonDone, info.StreamStatus.EndReason)
	assert.Equal(t, 10, info.StreamStatus.TotalErrorCount())
}

func TestStreamScannerHandler_StreamStatus_ErrorThenStop(t *testing.T) {
	t.Parallel()

	// Use a large body without [DONE] to avoid race between scanner's [DONE]
	// and handler's Stop on the sync.Once EndReason.
	var b strings.Builder
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&b, "data: {\"id\":%d}\n", i)
	}
	c, resp, info := setupStreamTest(t, strings.NewReader(b.String()))

	var count atomic.Int64
	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {
		count.Add(1)
		sr.Error(fmt.Errorf("soft error"))
		sr.Stop(fmt.Errorf("fatal"))
	})

	assert.Equal(t, int64(1), count.Load())
	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonHandlerStop, info.StreamStatus.EndReason)
	assert.Equal(t, 2, info.StreamStatus.TotalErrorCount())
}

func TestStreamScannerHandler_StreamStatus_InitializedIfNil(t *testing.T) {
	t.Parallel()

	body := buildSSEBody(1)
	c, resp, info := setupStreamTest(t, strings.NewReader(body))

	assert.Nil(t, info.StreamStatus)

	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {})

	assert.NotNil(t, info.StreamStatus)
}

func TestStreamScannerHandler_StreamStatus_ReplacesPreInitialized(t *testing.T) {
	t.Parallel()

	body := buildSSEBody(5)
	c, resp, info := setupStreamTest(t, strings.NewReader(body))

	info.StreamStatus = relaycommon.NewStreamStatus()
	info.StreamStatus.RecordError("pre-existing error")

	StreamScannerHandler(c, resp, info, func(data string, sr *StreamResult) {})

	assert.Equal(t, relaycommon.StreamEndReasonDone, info.StreamStatus.EndReason)
	assert.Equal(t, 0, info.StreamStatus.TotalErrorCount())
}
