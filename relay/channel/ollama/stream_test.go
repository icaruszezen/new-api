package ollama

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
	ollamaTestModel        = "llama3.1"
	ollamaTestPromptTokens = 9
)

func ollamaDelta(content string) string {
	return `{"model":"` + ollamaTestModel + `","created_at":"2026-05-27T12:00:00Z","message":{"role":"assistant","content":"` + content + `"},"done":false}` + "\n"
}

func ollamaDoneFrame(promptTokens, completionTokens int) string {
	return fmt.Sprintf(`{"model":"%s","created_at":"2026-05-27T12:00:01Z","message":{"role":"assistant","content":""},"done":true,"done_reason":"stop","prompt_eval_count":%d,"eval_count":%d}`+"\n",
		ollamaTestModel, promptTokens, completionTokens)
}

func newOllamaStreamTestInfo(drain bool) *relaycommon.RelayInfo {
	info := &relaycommon.RelayInfo{
		IsStream:                        true,
		DisablePing:                     true,
		DrainUpstreamOnClientDisconnect: drain,
		ChannelMeta:                     &relaycommon.ChannelMeta{UpstreamModelName: ollamaTestModel},
	}
	info.SetEstimatePromptTokens(ollamaTestPromptTokens)
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
	var streamID string
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
		if streamID == "" {
			streamID = chunk.Id
		}
		assert.Equal(t, streamID, chunk.Id, "every chunk carries the same id")
		assert.Equal(t, ollamaTestModel, chunk.Model)
		if chunk.Usage != nil {
			events = append(events, fmt.Sprintf("usage:%d+%d", chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens))
			continue
		}
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

// NDJSON 流：done 帧给出上游用量并结束流（上游连接不断也不再读）；没有 done 帧时按已转发的文本估算；
// 坏行让请求失败（退款），与迁移前一致。
func TestOllamaStreamHandler(t *testing.T) {
	useStreamingTimeout(t)
	estimated := service.EstimateTokenByModel(ollamaTestModel, "Hello world")
	require.Positive(t, estimated)

	tests := []struct {
		name           string
		body           string
		closeUpstream  bool
		wantEvents     []string
		wantPrompt     int
		wantCompletion int
		wantErrCode    types.ErrorCode
		wantEndReason  relaycommon.StreamEndReason
	}{
		{
			name:           "done frame bills the upstream counts and ends the stream",
			body:           ollamaDelta("Hello") + "\n" + ollamaDelta(" world") + ollamaDoneFrame(11, 42) + ollamaDelta(" after done"),
			wantEvents:     []string{"delta:", "delta:Hello", "delta: world", "delta:,finish:stop", "usage:11+42", "[DONE]"},
			wantPrompt:     11,
			wantCompletion: 42,
			wantEndReason:  relaycommon.StreamEndReasonDone,
		},
		{
			name:           "upstream cut before the done frame is estimated",
			body:           ollamaDelta("Hello") + ollamaDelta(" world"),
			closeUpstream:  true,
			wantEvents:     []string{"delta:", "delta:Hello", "delta: world"},
			wantPrompt:     ollamaTestPromptTokens,
			wantCompletion: estimated,
			wantEndReason:  relaycommon.StreamEndReasonEOF,
		},
		{
			name:          "malformed line fails the request",
			body:          ollamaDelta("Hello") + "not-json\n" + ollamaDoneFrame(11, 42),
			wantEvents:    []string{"delta:", "delta:Hello"},
			wantErrCode:   types.ErrorCodeBadResponseBody,
			wantEndReason: relaycommon.StreamEndReasonHandlerStop,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			info := newOllamaStreamTestInfo(false)
			// Unless the case needs EOF the upstream stays open, so only the
			// handler can end the stream.
			upstream, upstreamWriter := io.Pipe()
			t.Cleanup(func() { _ = upstreamWriter.Close() })
			go func() {
				_, _ = io.WriteString(upstreamWriter, tt.body)
				if tt.closeUpstream {
					_ = upstreamWriter.Close()
				}
			}()

			usage, apiErr := ollamaStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: upstream})

			assert.Equal(t, tt.wantEvents, clientEvents(t, recorder.Body.String()))
			require.NotNil(t, info.StreamStatus)
			assert.Equal(t, tt.wantEndReason, info.StreamStatus.EndReason)
			if tt.wantErrCode != "" {
				require.NotNil(t, apiErr)
				assert.Equal(t, tt.wantErrCode, apiErr.GetErrorCode())
				return
			}
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, tt.wantPrompt, usage.PromptTokens)
			assert.Equal(t, tt.wantCompletion, usage.CompletionTokens)
			assert.Equal(t, tt.wantPrompt+tt.wantCompletion, usage.TotalTokens)
		})
	}
}

// 客户端断开：开排水时读到断开后才到的 done 帧，按上游用量计费，流记为 drained；
// 不开时立即关闭上游，没有 done 帧，按断开前转发的文本估算。
func TestOllamaStreamHandlerClientDisconnect(t *testing.T) {
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
			name:            "drain bills the done frame sent after the disconnect",
			drain:           true,
			wantPrompt:      11,
			wantCompletion:  42,
			wantEndReason:   relaycommon.StreamEndReasonDrained,
			wantDrainedFrom: relaycommon.StreamEndReasonDone,
		},
		{
			name:           "without drain the upstream is closed and the text received is estimated",
			wantPrompt:     ollamaTestPromptTokens,
			wantCompletion: service.EstimateTokenByModel(ollamaTestModel, "hello"),
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
			info := newOllamaStreamTestInfo(tt.drain)

			var usage *dto.Usage
			var apiErr *types.NewAPIError
			done := make(chan struct{})
			go func() {
				defer close(done)
				usage, apiErr = ollamaStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: upstream})
			}()
			t.Cleanup(func() {
				cancel()
				_ = upstreamWriter.Close()
				<-done
			})

			_, err := io.WriteString(upstreamWriter, ollamaDelta("hello"))
			require.NoError(t, err)
			select {
			case <-ctx.Done():
			case <-time.After(2 * time.Second):
				t.Fatal("client did not disconnect")
			}
			if tt.drain {
				_, err = io.WriteString(upstreamWriter, ollamaDelta(" and the rest of the answer")+ollamaDoneFrame(11, 42))
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

			_, err = io.WriteString(upstreamWriter, ollamaDelta("late"))
			assert.ErrorIs(t, err, io.ErrClosedPipe, "the upstream body must be closed")
		})
	}
}

func TestOllamaChatHandlerNonStreamToolCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		raw    string
		wantID string
	}{
		{
			name:   "compact json per-line parse path",
			raw:    `{"model":"llama3.1","created_at":"2026-05-27T12:00:00Z","message":{"role":"assistant","content":"","tool_calls":[{"id":"call_upstream","function":{"name":"get_weather","arguments":{"city":"Paris","days":0}}}]},"done":true,"done_reason":"stop","prompt_eval_count":5,"eval_count":7}`,
			wantID: "call_upstream",
		},
		{
			name: "pretty json fallback parse path",
			raw: `{
  "model": "llama3.1",
  "created_at": "2026-05-27T12:00:00Z",
  "message": {
    "role": "assistant",
    "content": "",
    "tool_calls": [
      {
        "function": {
          "name": "get_weather",
          "arguments": {
            "city": "Paris",
            "days": 0
          }
        }
      }
    ]
  },
  "done": true,
  "done_reason": "stop",
  "prompt_eval_count": 5,
  "eval_count": 7
}`,
			wantID: "call_0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(tt.raw)),
			}

			usage, apiErr := ollamaChatHandler(c, &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "fallback-model"},
			}, resp)
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, 12, usage.TotalTokens)

			var out dto.OpenAITextResponse
			require.NoError(t, common.Unmarshal(w.Body.Bytes(), &out))
			require.Len(t, out.Choices, 1)
			assert.Equal(t, constant.FinishReasonToolCalls, out.Choices[0].FinishReason)

			var toolCalls []dto.ToolCallResponse
			require.NoError(t, common.Unmarshal(out.Choices[0].Message.ToolCalls, &toolCalls))
			require.Len(t, toolCalls, 1)
			assert.Equal(t, tt.wantID, toolCalls[0].ID)
			assert.Equal(t, "function", toolCalls[0].Type)
			assert.Equal(t, "get_weather", toolCalls[0].Function.Name)
			assert.Nil(t, toolCalls[0].Index)

			var args map[string]any
			require.NoError(t, common.Unmarshal([]byte(toolCalls[0].Function.Arguments), &args))
			assert.Equal(t, "Paris", args["city"])
			assert.Equal(t, float64(0), args["days"])
		})
	}
}

func TestBuildOllamaStreamDelta(t *testing.T) {
	tests := []struct {
		name          string
		raw           string
		startIndex    int
		wantContent   string
		wantReasoning string
		wantToolCalls []struct {
			id        string
			name      string
			index     int
			arguments string
		}
		wantNextIndex int
		wantPayload   bool
	}{
		{
			name:        "chat content",
			raw:         `{"message":{"content":"hello"}}`,
			wantContent: "hello",
			wantPayload: true,
		},
		{
			name:        "generate content",
			raw:         `{"response":"hello"}`,
			wantContent: "hello",
			wantPayload: true,
		},
		{
			name:          "thinking json string",
			raw:           `{"message":{"thinking":"consider this"}}`,
			wantReasoning: "consider this",
			wantPayload:   true,
		},
		{
			name:          "thinking raw fallback",
			raw:           `{"message":{"thinking":{"step":"consider this"}}}`,
			wantReasoning: `{"step":"consider this"}`,
			wantPayload:   true,
		},
		{
			name:       "multiple tool calls",
			raw:        `{"message":{"tool_calls":[{"id":"call_upstream","function":{"name":"get_weather","arguments":{"city":"Paris"}}},{"function":{"name":"get_time","arguments":{"timezone":"UTC"}}}]}}`,
			startIndex: 2,
			wantToolCalls: []struct {
				id        string
				name      string
				index     int
				arguments string
			}{
				{id: "call_upstream", name: "get_weather", index: 2, arguments: `{"city":"Paris"}`},
				{id: "call_3", name: "get_time", index: 3, arguments: `{"timezone":"UTC"}`},
			},
			wantNextIndex: 4,
			wantPayload:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var chunk ollamaChatStreamChunk
			require.NoError(t, common.Unmarshal([]byte(tt.raw), &chunk))

			toolCallIndex := tt.startIndex
			delta, hasPayload := buildOllamaStreamDelta(&chunk, "response-id", 123, "model", &toolCallIndex)
			assert.Equal(t, tt.wantPayload, hasPayload)
			assert.Equal(t, tt.wantContent, delta.Choices[0].Delta.GetContentString())
			assert.Equal(t, tt.wantReasoning, delta.Choices[0].Delta.GetReasoningContent())
			assert.Equal(t, tt.wantNextIndex, toolCallIndex)

			if tt.wantToolCalls == nil {
				assert.Empty(t, delta.Choices[0].Delta.ToolCalls)
				return
			}
			require.Len(t, delta.Choices[0].Delta.ToolCalls, len(tt.wantToolCalls))
			for i, want := range tt.wantToolCalls {
				got := delta.Choices[0].Delta.ToolCalls[i]
				assert.Equal(t, want.id, got.ID)
				assert.Equal(t, "function", got.Type)
				assert.Equal(t, want.name, got.Function.Name)
				assert.Equal(t, want.arguments, got.Function.Arguments)
				require.NotNil(t, got.Index)
				assert.Equal(t, want.index, *got.Index)
			}
		})
	}
}

func TestOllamaStreamHandlerDoneToolCalls(t *testing.T) {
	useStreamingTimeout(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"model":"qwen3-coder","created_at":"2026-05-27T12:00:00Z","message":{"role":"assistant","content":"","tool_calls":[{"function":{"name":"get_weather","arguments":{"city":"北京"}}},{"function":{"name":"get_time","arguments":{"timezone":"Asia/Shanghai"}}}]},"done":true,"done_reason":"stop","prompt_eval_count":5,"eval_count":7}`)),
	}

	usage, apiErr := ollamaStreamHandler(c, newOllamaStreamTestInfo(false), resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 12, usage.TotalTokens)

	var chunks []dto.ChatCompletionsStreamResponse
	for _, line := range strings.Split(w.Body.String(), "\n") {
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok || data == "[DONE]" {
			continue
		}
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.Unmarshal([]byte(data), &chunk))
		chunks = append(chunks, chunk)
	}

	var toolChunkIndex, stopChunkIndex int = -1, -1
	for i := range chunks {
		if len(chunks[i].Choices) == 0 {
			continue
		}
		if len(chunks[i].Choices[0].Delta.ToolCalls) > 0 {
			toolChunkIndex = i
		}
		if chunks[i].Choices[0].FinishReason != nil {
			stopChunkIndex = i
		}
	}
	require.NotEqual(t, -1, toolChunkIndex)
	require.NotEqual(t, -1, stopChunkIndex)
	assert.Less(t, toolChunkIndex, stopChunkIndex)

	toolCalls := chunks[toolChunkIndex].Choices[0].Delta.ToolCalls
	require.Len(t, toolCalls, 2)
	assert.Equal(t, "call_0", toolCalls[0].ID)
	assert.Equal(t, "call_1", toolCalls[1].ID)
	require.NotNil(t, toolCalls[0].Index)
	require.NotNil(t, toolCalls[1].Index)
	assert.Equal(t, 0, *toolCalls[0].Index)
	assert.Equal(t, 1, *toolCalls[1].Index)
	assert.Equal(t, constant.FinishReasonToolCalls, *chunks[stopChunkIndex].Choices[0].FinishReason)
	assert.Contains(t, w.Body.String(), "data: [DONE]")
}
