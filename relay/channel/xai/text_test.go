package xai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXAIStreamHandlerKeepsUsageTokenDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	body := "data: " + `{"id":"x1","object":"chat.completion.chunk","model":"grok-4","choices":[{"index":0,"delta":{"content":"hi"}}]}` + "\n\n" +
		"data: " + `{"id":"x1","object":"chat.completion.chunk","model":"grok-4","choices":[],` +
		`"usage":{"prompt_tokens":1200,"completion_tokens":30,"total_tokens":1250,` +
		`"prompt_tokens_details":{"cached_tokens":1024},"completion_tokens_details":{"reasoning_tokens":20}}}` + "\n\n" +
		"data: [DONE]\n\n"
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{
		IsStream:    true,
		DisablePing: true,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "grok-4"},
	}

	usage, apiErr := xAIStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))})

	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 1200, usage.PromptTokens)
	// xAI's total includes reasoning tokens; completion is derived from it.
	assert.Equal(t, 50, usage.CompletionTokens)
	assert.Equal(t, 1250, usage.TotalTokens)
	// Cached prompt tokens must reach billing so they get the cache price.
	assert.Equal(t, 1024, usage.PromptTokensDetails.CachedTokens)
	assert.Equal(t, 20, usage.CompletionTokenDetails.ReasoningTokens)
}
