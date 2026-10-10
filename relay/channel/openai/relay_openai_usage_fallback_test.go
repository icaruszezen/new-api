package openai

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 上游非流式响应不带 usage 时，输入 token 用本地估算并打上本地计数标记；
// 只有工具调用的回复、completions 的 text 输出也必须计出输出 token，否则会被当成"无输出"的错误计费而免扣费。
func TestOpenaiHandlerEstimatesUsageWhenUpstreamOmitsIt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const estimatedPromptTokens = 3_000_000

	tests := []struct {
		name           string
		relayMode      int
		body           string
		wantCompletion bool
	}{
		{
			name:           "empty assistant message has no output",
			relayMode:      relayconstant.RelayModeChatCompletions,
			body:           `{"id":"chatcmpl-1","object":"chat.completion","model":"gpt-4o","choices":[{"index":0,"message":{"role":"assistant","content":""},"finish_reason":"stop"}]}`,
			wantCompletion: false,
		},
		{
			name:           "tool-call-only message counts as output",
			relayMode:      relayconstant.RelayModeChatCompletions,
			body:           `{"id":"chatcmpl-2","object":"chat.completion","model":"gpt-4o","choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"main.go\"}"}}]},"finish_reason":"tool_calls"}]}`,
			wantCompletion: true,
		},
		{
			name:           "legacy completions text counts as output",
			relayMode:      relayconstant.RelayModeCompletions,
			body:           `{"id":"cmpl-1","object":"text_completion","model":"gpt-3.5-turbo-instruct","choices":[{"index":0,"text":"Hello from the model","finish_reason":"stop"}]}`,
			wantCompletion: true,
		},
		{
			name:           "empty legacy completions text has no output",
			relayMode:      relayconstant.RelayModeCompletions,
			body:           `{"id":"cmpl-2","object":"text_completion","model":"gpt-3.5-turbo-instruct","choices":[{"index":0,"text":"","finish_reason":"stop"}]}`,
			wantCompletion: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			info := &relaycommon.RelayInfo{
				RelayFormat:     types.RelayFormatOpenAI,
				RelayMode:       tt.relayMode,
				OriginModelName: "gpt-4o",
				ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "gpt-4o"},
			}
			info.SetEstimatePromptTokens(estimatedPromptTokens)
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(tt.body))),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}

			usage, apiErr := OpenaiHandler(c, info, resp)

			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, estimatedPromptTokens, usage.PromptTokens)
			assert.True(t, common.GetContextKeyBool(c, constant.ContextKeyLocalCountTokens))
			if tt.wantCompletion {
				assert.Greater(t, usage.CompletionTokens, 0)
			} else {
				assert.Equal(t, 0, usage.CompletionTokens)
			}
		})
	}
}

// 流式兜底：Claude（/v1/messages）和 Gemini 格式的请求转给 OpenAI 类渠道后，上游回的同样是 chat chunk，
// 输出也要计入，否则上游不给 usage 时这些请求被估成 0 输出。
func TestProcessTokenDataCountsOutputForEachRelayMode(t *testing.T) {
	const contentChunk = `{"choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"}}]}`
	const toolChunk = `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"main.go\"}"}}]}}]}`

	tests := []struct {
		name          string
		relayMode     int
		chunk         string
		wantText      string
		wantToolCount int
	}{
		{name: "chat completions", relayMode: relayconstant.RelayModeChatCompletions, chunk: contentChunk, wantText: "Hello"},
		{name: "claude messages converted to chat", relayMode: relayconstant.RelayModeUnknown, chunk: contentChunk, wantText: "Hello"},
		{name: "claude messages tool call", relayMode: relayconstant.RelayModeUnknown, chunk: toolChunk, wantText: `read_file{"path":"main.go"}`, wantToolCount: 1},
		{name: "gemini converted to chat", relayMode: relayconstant.RelayModeGemini, chunk: contentChunk, wantText: "Hello"},
		{name: "legacy completions", relayMode: relayconstant.RelayModeCompletions, chunk: `{"choices":[{"text":"Hello","finish_reason":null}]}`, wantText: "Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var responseText strings.Builder
			toolCount := 0

			err := processTokenData(tt.relayMode, tt.chunk, &responseText, &toolCount)

			require.NoError(t, err)
			assert.Equal(t, tt.wantText, responseText.String())
			assert.Equal(t, tt.wantToolCount, toolCount)
		})
	}
}
