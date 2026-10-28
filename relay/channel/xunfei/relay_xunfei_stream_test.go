package xunfei

import (
	"context"
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
	"github.com/gorilla/websocket"
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
	sparkFirstFrame  = `{"header":{"code":0,"status":0},"payload":{"choices":{"status":0,"seq":0,"text":[{"content":"你","role":"assistant","index":0}]}}}`
	sparkSecondFrame = `{"header":{"code":0,"status":1},"payload":{"choices":{"status":1,"seq":1,"text":[{"content":"好","role":"assistant","index":0}]}}}`
	sparkFinalFrame  = `{"header":{"code":0,"status":2},"payload":{"choices":{"status":2,"seq":2,"text":[{"content":"","role":"assistant","index":0}]},"usage":{"text":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}}}`
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
		DrainUpstreamOnClientDisconnect: drain,
		ChannelMeta:                     &relaycommon.ChannelMeta{},
	}
	return c, recorder, cancel, info
}

// fakeSpark is a Spark websocket upstream: it reads the chat request, sends the
// head frames, and sends the tail frames once the test closes release. It never
// closes the websocket itself; upstreamClosed is closed once the gateway does.
type fakeSpark struct {
	request        chan XunfeiChatRequest
	release        chan struct{}
	upstreamClosed chan struct{}
}

func dialFakeSpark(t *testing.T, head, tail []string) (*websocket.Conn, *fakeSpark) {
	t.Helper()
	spark := &fakeSpark{
		request:        make(chan XunfeiChatRequest, 1),
		release:        make(chan struct{}),
		upstreamClosed: make(chan struct{}),
	}
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		var request XunfeiChatRequest
		if err := conn.ReadJSON(&request); err != nil {
			return
		}
		spark.request <- request

		gatewayGone := make(chan struct{})
		go func() {
			defer close(gatewayGone)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()
		for _, frame := range head {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(frame))
		}
		select {
		case <-spark.release:
			for _, frame := range tail {
				_ = conn.WriteMessage(websocket.TextMessage, []byte(frame))
			}
		case <-gatewayGone:
		}
		<-gatewayGone
		close(spark.upstreamClosed)
	}))
	t.Cleanup(server.Close)

	conn, err := xunfeiMakeRequest(dto.GeneralOpenAIRequest{
		Model:    "SparkDesk-v3.5",
		Messages: []dto.Message{{Role: "user", Content: "hi"}},
	}, "generalv3.5", "ws"+strings.TrimPrefix(server.URL, "http"), "app-1")
	require.NoError(t, err)
	return conn, spark
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

// estimatedSparkUsage is the local estimate billed when the final frame with
// Spark's usage never arrives.
func estimatedSparkUsage(promptTokens int, text string) *dto.Usage {
	completion := service.EstimateTokenByModel("", text)
	return &dto.Usage{PromptTokens: promptTokens, CompletionTokens: completion, TotalTokens: promptTokens + completion}
}

// chunkSummary renders each chat chunk as "content|finish_reason".
func chunkSummary(t *testing.T, payloads []string) []string {
	t.Helper()
	summary := make([]string, 0, len(payloads))
	for _, payload := range payloads {
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.UnmarshalJsonStr(payload, &chunk))
		require.Len(t, chunk.Choices, 1)
		assert.Equal(t, "SparkDesk", chunk.Model)
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

func waitClosed(t *testing.T, ch <-chan struct{}, within time.Duration, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(within):
		t.Fatal(msg)
	}
}

func TestXunfeiStreamHandlerRelaysFramesAndFinalUsage(t *testing.T) {
	c, recorder, _, info := newClientContext(t, false)
	conn, spark := dialFakeSpark(t, []string{sparkFirstFrame, sparkSecondFrame, sparkFinalFrame}, nil)

	var usage *dto.Usage
	done := goDone(func() { usage = xunfeiStreamHandler(c, info, conn) })

	// The upstream keeps the websocket open: the stream must end on status==2.
	waitClosed(t, done, 2*time.Second, "handler did not stop at the final frame")
	request := <-spark.request
	assert.Equal(t, "app-1", request.Header.AppId)
	assert.Equal(t, "generalv3.5", request.Parameter.Chat.Domain)
	waitClosed(t, spark.upstreamClosed, 2*time.Second, "websocket was not closed after the final frame")
	assert.Equal(t, &dto.Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5}, usage)
	payloads := sseData(recorder.Body.String())
	require.NotEmpty(t, payloads)
	assert.Equal(t, "[DONE]", payloads[len(payloads)-1])
	assert.Equal(t, []string{"你|", "好|", "|stop"}, chunkSummary(t, payloads[:len(payloads)-1]))
	assert.Equal(t, relaycommon.StreamEndReasonDone, info.StreamStatus.EndReason)
	assert.Equal(t, 3, info.ReceivedResponseCount)
}

func TestXunfeiStreamHandlerDrainBillsFinalFrameAfterDisconnect(t *testing.T) {
	c, recorder, cancel, info := newClientContext(t, true)
	conn, spark := dialFakeSpark(t, []string{sparkFirstFrame}, []string{sparkSecondFrame, sparkFinalFrame})

	var usage *dto.Usage
	done := goDone(func() { usage = xunfeiStreamHandler(c, info, conn) })
	waitClosed(t, recorder.firstWrite, 2*time.Second, "first chunk never reached the client")
	cancel()
	select {
	case <-done:
		t.Fatal("handler returned before the final frame although drain is on")
	case <-time.After(100 * time.Millisecond):
	}
	close(spark.release)
	waitClosed(t, done, 2*time.Second, "handler did not return after the final frame")
	waitClosed(t, spark.upstreamClosed, 2*time.Second, "websocket was not closed after the drain")

	assert.Equal(t, &dto.Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5}, usage)
	assert.Equal(t, relaycommon.StreamEndReasonDrained, info.StreamStatus.EndReason)
	assert.Equal(t, relaycommon.StreamEndReasonDone, info.StreamStatus.DrainedFrom)
	assert.True(t, info.StreamStatus.ClientGone)
	assert.False(t, info.StreamStatus.HasErrors())
	// Nothing after the disconnect is written, [DONE] included.
	assert.Equal(t, []string{"你|"}, chunkSummary(t, sseData(recorder.Body.String())))
}

func TestXunfeiStreamHandlerWithoutDrainClosesWebsocketOnDisconnect(t *testing.T) {
	c, recorder, cancel, info := newClientContext(t, false)
	info.SetEstimatePromptTokens(7)
	conn, spark := dialFakeSpark(t, []string{sparkFirstFrame}, []string{sparkFinalFrame})

	var usage *dto.Usage
	done := goDone(func() { usage = xunfeiStreamHandler(c, info, conn) })
	waitClosed(t, recorder.firstWrite, 2*time.Second, "first chunk never reached the client")
	cancel()
	waitClosed(t, done, 2*time.Second, "handler did not return after the client disconnected")
	waitClosed(t, spark.upstreamClosed, 2*time.Second, "websocket was not closed after the client disconnected")

	// The final frame with the usage never came: the text received is estimated.
	assert.Equal(t, estimatedSparkUsage(7, "你"), usage)
	assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
	assert.True(t, info.StreamStatus.ClientGone)
}

// Spark rejects a request (auth, rate limit, audit) with a header.code frame
// after the handshake. That is not a cut-short reply, so nothing is estimated.
func TestXunfeiStreamHandlerErrorFrameIsNotBilled(t *testing.T) {
	c, _, _, info := newClientContext(t, false)
	info.SetEstimatePromptTokens(1500)
	const errorFrame = `{"header":{"code":11202,"message":"AppIdQpsOverFlowError","sid":"cht1","status":2}}`
	conn, spark := dialFakeSpark(t, []string{sparkFirstFrame, errorFrame}, nil)

	var usage *dto.Usage
	done := goDone(func() { usage = xunfeiStreamHandler(c, info, conn) })
	waitClosed(t, done, 2*time.Second, "handler did not stop at the error frame")
	waitClosed(t, spark.upstreamClosed, 2*time.Second, "websocket was not closed after the error frame")

	assert.Equal(t, &dto.Usage{}, usage)
	assert.Equal(t, relaycommon.StreamEndReasonHandlerStop, info.StreamStatus.EndReason)
	require.True(t, info.StreamStatus.HasErrors())
	assert.Contains(t, info.StreamStatus.Errors[0].Message, "11202")
}

func TestXunfeiStreamHandlerDrainIdleTimeoutClosesWebsocket(t *testing.T) {
	// Not parallel: modifies the global constant.StreamingTimeout.
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 1
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	c, recorder, cancel, info := newClientContext(t, true)
	info.SetEstimatePromptTokens(7)
	conn, spark := dialFakeSpark(t, []string{sparkFirstFrame}, []string{sparkFinalFrame})

	var usage *dto.Usage
	done := goDone(func() { usage = xunfeiStreamHandler(c, info, conn) })
	waitClosed(t, recorder.firstWrite, 2*time.Second, "first chunk never reached the client")
	cancel()
	waitClosed(t, done, 3*time.Second, "handler did not return after the drain idle timeout")
	waitClosed(t, spark.upstreamClosed, 2*time.Second, "websocket was not closed after the drain idle timeout")

	assert.Equal(t, estimatedSparkUsage(7, "你"), usage)
	assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
	assert.True(t, info.StreamStatus.HasErrors())
}
