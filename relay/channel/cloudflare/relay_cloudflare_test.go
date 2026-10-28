package cloudflare

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	cfTestModel        = "@cf/meta/llama-3.1-8b-instruct"
	cfTestPromptTokens = 9
)

func cfDelta(content string) string {
	return `{"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"` + content + `"}}]}`
}

func cfSSE(payloads ...string) string {
	var b strings.Builder
	for _, payload := range payloads {
		b.WriteString("data: " + payload + "\n\n")
	}
	return b.String()
}

func newCfStreamTestInfo(drain bool) *relaycommon.RelayInfo {
	info := &relaycommon.RelayInfo{
		IsStream:                        true,
		ShouldIncludeUsage:              true,
		DisablePing:                     true,
		DrainUpstreamOnClientDisconnect: drain,
		ChannelMeta:                     &relaycommon.ChannelMeta{UpstreamModelName: cfTestModel},
	}
	info.SetEstimatePromptTokens(cfTestPromptTokens)
	return info
}

func useStreamingTimeout(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })
}

// clientEvents summarizes every SSE data payload the client received, so a
// test can compare the whole client-visible stream in one assertion.
func clientEvents(t *testing.T, body string) []string {
	t.Helper()
	var events []string
	for _, line := range strings.Split(body, "\n") {
		payload, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}
		if payload == "[DONE]" {
			events = append(events, payload)
			continue
		}
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.UnmarshalJsonStr(payload, &chunk))
		assert.Equal(t, "chatcmpl-cf-req", chunk.Id)
		assert.Equal(t, cfTestModel, chunk.Model)
		if chunk.Usage != nil {
			events = append(events, fmt.Sprintf("usage:%d+%d", chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens))
			continue
		}
		require.Len(t, chunk.Choices, 1)
		events = append(events, "delta:"+chunk.Choices[0].Delta.GetContentString())
	}
	return events
}

// disconnectAfterWriter cancels the client context once a write containing
// needle reaches the client, so the disconnect lands at a known point.
type disconnectAfterWriter struct {
	gin.ResponseWriter
	needle string
	cancel context.CancelFunc
}

func (w *disconnectAfterWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	if strings.Contains(string(p), w.needle) {
		w.cancel()
	}
	return n, err
}

func (w *disconnectAfterWriter) WriteString(s string) (int, error) {
	n, err := w.ResponseWriter.WriteString(s)
	if strings.Contains(s, w.needle) {
		w.cancel()
	}
	return n, err
}

// 正常流：逐块转发并改写 id/model，坏行跳过，[DONE] 之后的内容不再读，
// 用量按转发的文本本地估算，ShouldIncludeUsage 时补一块 usage。
func TestCfStreamHandlerForwardsChunksAndEstimatesUsage(t *testing.T) {
	useStreamingTimeout(t)
	wantCompletion := service.EstimateTokenByModel(cfTestModel, "Hello world")
	require.Positive(t, wantCompletion)
	usageEvent := fmt.Sprintf("usage:%d+%d", cfTestPromptTokens, wantCompletion)

	tests := []struct {
		name         string
		includeUsage bool
		wantEvents   []string
	}{
		{
			name:         "with stream usage",
			includeUsage: true,
			wantEvents:   []string{"delta:Hello", "delta: world", usageEvent, "[DONE]"},
		},
		{
			name:       "without stream usage",
			wantEvents: []string{"delta:Hello", "delta: world", "[DONE]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			c.Set(common.RequestIdKey, "cf-req")
			info := newCfStreamTestInfo(false)
			info.ShouldIncludeUsage = tt.includeUsage
			body := cfSSE(cfDelta("Hello"), "not-json", cfDelta(" world"), "[DONE]", cfDelta(" after done"))

			apiErr, usage := cfStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))})

			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, cfTestPromptTokens, usage.PromptTokens)
			assert.Equal(t, wantCompletion, usage.CompletionTokens)
			assert.Equal(t, cfTestPromptTokens+wantCompletion, usage.TotalTokens)
			assert.Equal(t, tt.wantEvents, clientEvents(t, recorder.Body.String()))
			require.NotNil(t, info.StreamStatus)
			assert.Equal(t, relaycommon.StreamEndReasonDone, info.StreamStatus.EndReason)
			assert.Equal(t, 1, info.StreamStatus.TotalErrorCount(), "the malformed line is a soft error")
		})
	}
}

// 客户端断开：开排水时断开后上游继续发的文本也计入用量，流记为 drained；
// 不开时立即关闭上游，只按断开前转发的文本估算。
func TestCfStreamHandlerClientDisconnect(t *testing.T) {
	useStreamingTimeout(t)

	tests := []struct {
		name            string
		drain           bool
		wantText        string
		wantEndReason   relaycommon.StreamEndReason
		wantDrainedFrom relaycommon.StreamEndReason
	}{
		{
			name:            "drain bills the text streamed after the disconnect",
			drain:           true,
			wantText:        "hello and the rest of the answer",
			wantEndReason:   relaycommon.StreamEndReasonDrained,
			wantDrainedFrom: relaycommon.StreamEndReasonDone,
		},
		{
			name:          "without drain the upstream is closed at once",
			wantText:      "hello",
			wantEndReason: relaycommon.StreamEndReasonClientGone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			upstream, upstreamWriter := io.Pipe()
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil).WithContext(ctx)
			c.Writer = &disconnectAfterWriter{ResponseWriter: c.Writer, needle: "hello", cancel: cancel}
			info := newCfStreamTestInfo(tt.drain)

			var usage *dto.Usage
			var apiErr *types.NewAPIError
			done := make(chan struct{})
			go func() {
				defer close(done)
				apiErr, usage = cfStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: upstream})
			}()
			t.Cleanup(func() {
				cancel()
				_ = upstreamWriter.Close()
				<-done
			})

			_, err := io.WriteString(upstreamWriter, cfSSE(cfDelta("hello")))
			require.NoError(t, err)
			select {
			case <-ctx.Done():
			case <-time.After(2 * time.Second):
				t.Fatal("client did not disconnect")
			}
			if tt.drain {
				_, err = io.WriteString(upstreamWriter, cfSSE(cfDelta(" and the rest of the answer"), "[DONE]"))
				require.NoError(t, err)
			}
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("handler did not return")
			}

			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, cfTestPromptTokens, usage.PromptTokens)
			assert.Equal(t, service.EstimateTokenByModel(cfTestModel, tt.wantText), usage.CompletionTokens)
			require.NotNil(t, info.StreamStatus)
			assert.Equal(t, tt.wantEndReason, info.StreamStatus.EndReason)
			assert.Equal(t, tt.wantDrainedFrom, info.StreamStatus.DrainedFrom)
			assert.True(t, info.StreamStatus.ClientGone)
			assert.False(t, info.StreamStatus.HasErrors())
			assert.NotContains(t, recorder.Body.String(), "the rest", "nothing is written after the disconnect")

			_, err = io.WriteString(upstreamWriter, cfSSE(cfDelta("late")))
			assert.ErrorIs(t, err, io.ErrClosedPipe, "the upstream body must be closed")
		})
	}
}
