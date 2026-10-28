package cohere

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

const (
	cohereTextHel  = `{"is_finished":false,"event_type":"text-generation","text":"Hel"}`
	cohereTextLo   = `{"is_finished":false,"event_type":"text-generation","text":"lo"}`
	cohereFinished = `{"is_finished":true,"event_type":"stream-end","finish_reason":"COMPLETE","response":{"response_id":"r1","text":"Hello","meta":{"billed_units":{"input_tokens":7,"output_tokens":2}}}}`
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
		ChannelMeta:                     &relaycommon.ChannelMeta{UpstreamModelName: "command-r"},
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

// chunkSummary renders each chat chunk as "content|finish_reason".
func chunkSummary(t *testing.T, payloads []string) []string {
	t.Helper()
	summary := make([]string, 0, len(payloads))
	for _, payload := range payloads {
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.UnmarshalJsonStr(payload, &chunk))
		require.Len(t, chunk.Choices, 1)
		assert.Equal(t, "command-r", chunk.Model)
		assert.Equal(t, "chat.completion.chunk", chunk.Object)
		summary = append(summary, chunk.Choices[0].Delta.GetContentString()+"|"+lo.FromPtr(chunk.Choices[0].FinishReason))
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

func TestCohereStreamHandlerRelaysChunksAndBilledUnits(t *testing.T) {
	c, recorder, _, info := newClientContext(t, false)
	resp, upstream := startFakeUpstream(t, []string{cohereTextHel, "", cohereTextLo, cohereFinished}, nil)

	var usage *dto.Usage
	done := goDone(func() { usage, _ = cohereStreamHandler(c, info, resp) })

	// The upstream keeps the connection open: the stream must end on is_finished.
	waitClosed(t, done, "handler did not stop at the is_finished event")
	waitClosed(t, upstream.upstreamClosed, "upstream connection was not closed after the is_finished event")
	assert.Equal(t, &dto.Usage{PromptTokens: 7, CompletionTokens: 2, TotalTokens: 9}, usage)
	payloads := sseData(recorder.Body.String())
	require.NotEmpty(t, payloads)
	assert.Equal(t, "[DONE]", payloads[len(payloads)-1])
	assert.Equal(t, []string{"Hel|", "lo|", "|stop"}, chunkSummary(t, payloads[:len(payloads)-1]))
	assert.Equal(t, relaycommon.StreamEndReasonDone, info.StreamStatus.EndReason)
}

func TestCohereStreamHandlerDrainBillsFinishedEventAfterDisconnect(t *testing.T) {
	c, recorder, cancel, info := newClientContext(t, true)
	resp, upstream := startFakeUpstream(t, []string{cohereTextHel}, []string{cohereTextLo, cohereFinished})

	var usage *dto.Usage
	done := goDone(func() { usage, _ = cohereStreamHandler(c, info, resp) })
	waitClosed(t, recorder.firstWrite, "first chunk never reached the client")
	cancel()
	select {
	case <-done:
		t.Fatal("handler returned before the upstream finished although drain is on")
	case <-time.After(100 * time.Millisecond):
	}
	close(upstream.release)
	waitClosed(t, done, "handler did not return after the upstream finished")
	waitClosed(t, upstream.upstreamClosed, "upstream connection was not closed after the drain")

	assert.Equal(t, &dto.Usage{PromptTokens: 7, CompletionTokens: 2, TotalTokens: 9}, usage)
	assert.Equal(t, relaycommon.StreamEndReasonDrained, info.StreamStatus.EndReason)
	assert.Equal(t, relaycommon.StreamEndReasonDone, info.StreamStatus.DrainedFrom)
	assert.False(t, info.StreamStatus.HasErrors())
	// Nothing after the disconnect is written, [DONE] included.
	assert.Equal(t, []string{"Hel|"}, chunkSummary(t, sseData(recorder.Body.String())))
}

func TestCohereStreamHandlerWithoutDrainClosesUpstreamOnDisconnect(t *testing.T) {
	c, recorder, cancel, info := newClientContext(t, false)
	resp, upstream := startFakeUpstream(t, []string{cohereTextHel}, []string{cohereFinished})

	var usage *dto.Usage
	done := goDone(func() { usage, _ = cohereStreamHandler(c, info, resp) })
	waitClosed(t, recorder.firstWrite, "first chunk never reached the client")
	cancel()
	waitClosed(t, done, "handler did not return after the client disconnected")
	waitClosed(t, upstream.upstreamClosed, "upstream connection was not closed after the client disconnected")

	assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
	// Without billed_units the partial text is estimated.
	require.NotNil(t, usage)
	assert.Equal(t, 5, usage.PromptTokens)
	assert.Equal(t, service.EstimateTokenByModel("command-r", "Hel"), usage.CompletionTokens)
}
