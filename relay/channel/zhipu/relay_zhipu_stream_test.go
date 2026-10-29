package zhipu

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
	if constant.StreamingTimeout == 0 {
		constant.StreamingTimeout = 30
	}
}

const zhipuMeta = `meta: {"task_status":"SUCCESS","usage":{"prompt_tokens":4,"completion_tokens":2,"total_tokens":6},"task_id":"t-1","request_id":"req-1"}`

var (
	zhipuFirstEvent  = []string{"event: add", "id: 1", "data:你好", ""}
	zhipuSecondEvent = []string{"event: add", "id: 1", "data:世界", ""}
	zhipuFinishEvent = []string{"event: finish", "id: 1", "data:", zhipuMeta, ""}
)

// clientRecorder records what the gateway sends to the client and signals its
// first write, so a test can disconnect the client mid-stream.
type clientRecorder struct {
	*httptest.ResponseRecorder
	firstWrite chan struct{}
	once       sync.Once
}

func (r *clientRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseRecorder.Write(b)
	r.once.Do(func() { close(r.firstWrite) })
	return n, err
}

func (r *clientRecorder) WriteString(s string) (int, error) {
	n, err := r.ResponseRecorder.WriteString(s)
	r.once.Do(func() { close(r.firstWrite) })
	return n, err
}

func newClientContext(t *testing.T, drain bool) (*gin.Context, *clientRecorder, context.CancelFunc, *relaycommon.RelayInfo) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	recorder := &clientRecorder{ResponseRecorder: httptest.NewRecorder(), firstWrite: make(chan struct{})}
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil).WithContext(ctx)
	info := &relaycommon.RelayInfo{
		DisablePing:                     true,
		DrainUpstreamOnClientDisconnect: drain,
		ChannelMeta:                     &relaycommon.ChannelMeta{UpstreamModelName: "chatglm_std"},
	}
	info.SetEstimatePromptTokens(5)
	return c, recorder, cancel, info
}

// fakeUpstream streams head lines and sends tail lines once the test closes
// release. It never ends the response itself, so the handler has to stop on the
// protocol's final event; upstreamClosed is closed once the gateway closes the
// upstream connection.
type fakeUpstream struct {
	release        chan struct{}
	upstreamClosed chan struct{}
}

func startFakeUpstream(t *testing.T, head, tail []string) (*http.Response, *fakeUpstream) {
	t.Helper()
	upstream := &fakeUpstream{release: make(chan struct{}), upstreamClosed: make(chan struct{})}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(upstream.upstreamClosed)
		for _, line := range head {
			_, _ = io.WriteString(w, line+"\n")
		}
		w.(http.Flusher).Flush()
		select {
		case <-upstream.release:
			for _, line := range tail {
				_, _ = io.WriteString(w, line+"\n")
			}
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
		}
		<-r.Context().Done()
	}))
	t.Cleanup(func() {
		// Cut the gateway side first so a hung handler fails the test instead of blocking Close.
		server.CloseClientConnections()
		server.Close()
	})
	resp, err := server.Client().Get(server.URL)
	require.NoError(t, err)
	return resp, upstream
}

// sseData returns the payload of every `data: ` event the client received.
func sseData(body string) []string {
	var payloads []string
	for _, event := range strings.Split(body, "\n\n") {
		if payload, ok := strings.CutPrefix(strings.TrimSpace(event), "data: "); ok {
			payloads = append(payloads, payload)
		}
	}
	return payloads
}

// chunkSummary renders each chat chunk as "id|content|finish_reason".
func chunkSummary(t *testing.T, payloads []string) []string {
	t.Helper()
	summary := make([]string, 0, len(payloads))
	for _, payload := range payloads {
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.UnmarshalJsonStr(payload, &chunk))
		require.Len(t, chunk.Choices, 1)
		assert.Equal(t, "chatglm", chunk.Model)
		summary = append(summary, chunk.Id+"|"+chunk.Choices[0].Delta.GetContentString()+"|"+lo.FromPtr(chunk.Choices[0].FinishReason))
	}
	return summary
}

// goDone runs fn on its own goroutine and returns a channel closed when fn
// returns, so a handler that hangs fails the test instead of the whole run.
func goDone(fn func()) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	return done
}

func waitClosed(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal(msg)
	}
}

func TestZhipuStreamHandlerRelaysChunksAndMetaUsage(t *testing.T) {
	c, recorder, _, info := newClientContext(t, false)
	lines := append(append(append([]string{}, zhipuFirstEvent...), zhipuSecondEvent...), zhipuFinishEvent...)
	resp, upstream := startFakeUpstream(t, lines, nil)

	var usage *dto.Usage
	done := goDone(func() { usage, _ = zhipuStreamHandler(c, info, resp) })

	// The upstream keeps the connection open: the stream must end on the meta line.
	waitClosed(t, done, "handler did not stop at the meta line")
	waitClosed(t, upstream.upstreamClosed, "upstream connection was not closed after the meta line")
	assert.Equal(t, &dto.Usage{PromptTokens: 4, CompletionTokens: 2, TotalTokens: 6}, usage)
	payloads := sseData(recorder.Body.String())
	require.NotEmpty(t, payloads)
	assert.Equal(t, "[DONE]", payloads[len(payloads)-1])
	assert.Equal(t, []string{"|你好|", "|世界|", "||", "req-1||stop"}, chunkSummary(t, payloads[:len(payloads)-1]))
	assert.Equal(t, relaycommon.StreamEndReasonDone, info.StreamStatus.EndReason)
}

func TestZhipuStreamHandlerDrainBillsMetaAfterDisconnect(t *testing.T) {
	c, recorder, cancel, info := newClientContext(t, true)
	resp, upstream := startFakeUpstream(t, zhipuFirstEvent, append(append([]string{}, zhipuSecondEvent...), zhipuFinishEvent...))

	var usage *dto.Usage
	done := goDone(func() { usage, _ = zhipuStreamHandler(c, info, resp) })
	waitClosed(t, recorder.firstWrite, "first chunk never reached the client")
	cancel()
	select {
	case <-done:
		t.Fatal("handler returned before the upstream finished although drain is on")
	case <-time.After(100 * time.Millisecond):
	}
	close(upstream.release)
	waitClosed(t, done, "handler did not return after the meta line")
	waitClosed(t, upstream.upstreamClosed, "upstream connection was not closed after the drain")

	assert.Equal(t, &dto.Usage{PromptTokens: 4, CompletionTokens: 2, TotalTokens: 6}, usage)
	assert.Equal(t, relaycommon.StreamEndReasonDrained, info.StreamStatus.EndReason)
	assert.Equal(t, relaycommon.StreamEndReasonDone, info.StreamStatus.DrainedFrom)
	assert.False(t, info.StreamStatus.HasErrors())
	// Nothing after the disconnect is written, [DONE] included.
	assert.Equal(t, []string{"|你好|"}, chunkSummary(t, sseData(recorder.Body.String())))
}

func TestZhipuStreamHandlerWithoutDrainClosesUpstreamAndEstimatesUsage(t *testing.T) {
	c, recorder, cancel, info := newClientContext(t, false)
	resp, upstream := startFakeUpstream(t, zhipuFirstEvent, zhipuFinishEvent)

	var usage *dto.Usage
	done := goDone(func() { usage, _ = zhipuStreamHandler(c, info, resp) })
	waitClosed(t, recorder.firstWrite, "first chunk never reached the client")
	cancel()
	waitClosed(t, done, "handler did not return after the client disconnected")
	waitClosed(t, upstream.upstreamClosed, "upstream connection was not closed after the client disconnected")

	assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
	// No meta line arrived: usage is estimated from the text, never a nil *dto.Usage.
	require.NotNil(t, usage)
	assert.Equal(t, 5, usage.PromptTokens)
	assert.Equal(t, service.EstimateTokenByModel("chatglm_std", "你好"), usage.CompletionTokens)
}
