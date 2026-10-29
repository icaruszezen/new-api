package coze

import (
	"context"
	"errors"
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
	cozeTestModel        = "coze-bot"
	cozeTestPromptTokens = 37
	cozeDoneEvent        = "event:done\ndata:\"[DONE]\"\n\n"
)

func cozeEvent(event, data string) string {
	return "event:" + event + "\ndata:" + data + "\n\n"
}

func cozeDelta(content string) string {
	return cozeEvent("conversation.message.delta", `{"id":"msg_1","role":"assistant","type":"answer","content":"`+content+`","content_type":"text"}`)
}

func cozeChatCompleted(inputTokens, outputTokens int) string {
	return cozeEvent("conversation.chat.completed", fmt.Sprintf(`{"id":"chat_1","status":"completed","usage":{"token_count":%d,"output_count":%d,"input_count":%d}}`,
		inputTokens+outputTokens, outputTokens, inputTokens))
}

func newCozeStreamTestInfo(drain bool) *relaycommon.RelayInfo {
	info := &relaycommon.RelayInfo{
		IsStream:                        true,
		DisablePing:                     true,
		DrainUpstreamOnClientDisconnect: drain,
		ChannelMeta:                     &relaycommon.ChannelMeta{UpstreamModelName: cozeTestModel},
	}
	info.SetEstimatePromptTokens(cozeTestPromptTokens)
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
		assert.Equal(t, "chatcmpl-coze-req", chunk.Id)
		assert.Equal(t, cozeTestModel, chunk.Model)
		require.Len(t, chunk.Choices, 1)
		event := "delta:" + chunk.Choices[0].Delta.GetContentString()
		if finishReason := chunk.Choices[0].FinishReason; finishReason != nil {
			event += ",finish:" + *finishReason
		}
		events = append(events, event)
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

// event:/data: 流：delta 转发，chat.completed 给出用量并发 stop，done 事件结束流
// （上游连接不断也不再读）；上游读错误让请求失败，与迁移前一致。
func TestCozeChatStreamHandler(t *testing.T) {
	useStreamingTimeout(t)

	tests := []struct {
		name          string
		body          string
		upstreamErr   error
		wantEvents    []string
		wantUsage     *dto.Usage
		wantErrCode   types.ErrorCode
		wantEndReason relaycommon.StreamEndReason
	}{
		{
			name: "chat.completed usage is billed and done ends the stream",
			body: cozeEvent("conversation.chat.created", `{"id":"chat_1","status":"created"}`) +
				cozeDelta("Hello") + cozeDelta(" world") +
				cozeEvent("conversation.message.completed", `{"id":"msg_1","type":"answer","content":"Hello world"}`) +
				cozeChatCompleted(11, 42) + cozeDoneEvent + cozeDelta(" after done"),
			wantEvents:    []string{"delta:Hello", "delta: world", "delta:,finish:stop", "[DONE]"},
			wantUsage:     &dto.Usage{PromptTokens: 11, CompletionTokens: 42, TotalTokens: 53},
			wantEndReason: relaycommon.StreamEndReasonDone,
		},
		{
			name:          "upstream read error fails the request",
			body:          cozeDelta("Hello"),
			upstreamErr:   errors.New("connection reset by peer"),
			wantEvents:    []string{"delta:Hello"},
			wantErrCode:   types.ErrorCodeBadResponseBody,
			wantEndReason: relaycommon.StreamEndReasonScannerErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			c.Set(common.RequestIdKey, "coze-req")
			info := newCozeStreamTestInfo(false)
			// The upstream stays open unless the case fails it, so only the
			// handler can end the stream.
			upstream, upstreamWriter := io.Pipe()
			t.Cleanup(func() { _ = upstreamWriter.Close() })
			go func() {
				_, _ = io.WriteString(upstreamWriter, tt.body)
				if tt.upstreamErr != nil {
					_ = upstreamWriter.CloseWithError(tt.upstreamErr)
				}
			}()

			usage, apiErr := cozeChatStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: upstream})

			assert.Equal(t, tt.wantEvents, clientEvents(t, recorder.Body.String()))
			require.NotNil(t, info.StreamStatus)
			assert.Equal(t, tt.wantEndReason, info.StreamStatus.EndReason)
			if tt.wantErrCode != "" {
				require.NotNil(t, apiErr)
				assert.Equal(t, tt.wantErrCode, apiErr.GetErrorCode())
				assert.Nil(t, usage)
				return
			}
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, tt.wantUsage.PromptTokens, usage.PromptTokens)
			assert.Equal(t, tt.wantUsage.CompletionTokens, usage.CompletionTokens)
			assert.Equal(t, tt.wantUsage.TotalTokens, usage.TotalTokens)
		})
	}
}

// 客户端断开：开排水时读到断开后才到的 chat.completed，按上游用量计费，流记为 drained；
// 不开时立即关闭上游，没有 chat.completed，按请求预估的 prompt 和断开前转发的文本估算。
func TestCozeChatStreamHandlerClientDisconnect(t *testing.T) {
	useStreamingTimeout(t)

	tests := []struct {
		name            string
		drain           bool
		wantPrompt      int
		wantCompletion  int
		wantEndReason   relaycommon.StreamEndReason
		wantDrainedFrom relaycommon.StreamEndReason
	}{
		{
			name:            "drain bills the chat.completed usage sent after the disconnect",
			drain:           true,
			wantPrompt:      11,
			wantCompletion:  42,
			wantEndReason:   relaycommon.StreamEndReasonDrained,
			wantDrainedFrom: relaycommon.StreamEndReasonDone,
		},
		{
			name:           "without drain the upstream is closed and the text received is estimated",
			wantPrompt:     cozeTestPromptTokens,
			wantCompletion: service.EstimateTokenByModel(cozeTestModel, "hello"),
			wantEndReason:  relaycommon.StreamEndReasonClientGone,
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
			info := newCozeStreamTestInfo(tt.drain)

			var usage *dto.Usage
			var apiErr *types.NewAPIError
			done := make(chan struct{})
			go func() {
				defer close(done)
				usage, apiErr = cozeChatStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: upstream})
			}()
			t.Cleanup(func() {
				cancel()
				_ = upstreamWriter.Close()
				<-done
			})

			_, err := io.WriteString(upstreamWriter, cozeEvent("conversation.chat.created", `{"id":"chat_1"}`)+cozeDelta("hello"))
			require.NoError(t, err)
			select {
			case <-ctx.Done():
			case <-time.After(2 * time.Second):
				t.Fatal("client did not disconnect")
			}
			if tt.drain {
				_, err = io.WriteString(upstreamWriter, cozeDelta(" and the rest of the answer")+cozeChatCompleted(11, 42)+cozeDoneEvent)
				require.NoError(t, err)
			}
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("handler did not return")
			}

			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, tt.wantPrompt, usage.PromptTokens)
			assert.Equal(t, tt.wantCompletion, usage.CompletionTokens)
			require.NotNil(t, info.StreamStatus)
			assert.Equal(t, tt.wantEndReason, info.StreamStatus.EndReason)
			assert.Equal(t, tt.wantDrainedFrom, info.StreamStatus.DrainedFrom)
			assert.True(t, info.StreamStatus.ClientGone)
			assert.False(t, info.StreamStatus.HasErrors())
			assert.NotContains(t, recorder.Body.String(), "the rest", "nothing is written after the disconnect")

			_, err = io.WriteString(upstreamWriter, cozeDelta("late"))
			assert.ErrorIs(t, err, io.ErrClosedPipe, "the upstream body must be closed")
		})
	}
}
