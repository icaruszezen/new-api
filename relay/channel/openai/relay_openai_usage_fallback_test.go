package openai

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 上游非流式响应不带 usage 时，输入 token 用本地估算并打上本地计数标记；
// 只有工具调用的回复也必须计出输出 token，否则会被当成"无输出"的错误计费而免扣费。
func TestOpenaiHandlerEstimatesUsageWhenUpstreamOmitsIt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const estimatedPromptTokens = 3_000_000

	tests := []struct {
		name           string
		body           string
		wantCompletion bool
	}{
		{
			name:           "empty assistant message has no output",
			body:           `{"id":"chatcmpl-1","object":"chat.completion","model":"gpt-4o","choices":[{"index":0,"message":{"role":"assistant","content":""},"finish_reason":"stop"}]}`,
			wantCompletion: false,
		},
		{
			name:           "tool-call-only message counts as output",
			body:           `{"id":"chatcmpl-2","object":"chat.completion","model":"gpt-4o","choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"main.go\"}"}}]},"finish_reason":"tool_calls"}]}`,
			wantCompletion: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			info := &relaycommon.RelayInfo{
				RelayFormat:     types.RelayFormatOpenAI,
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
