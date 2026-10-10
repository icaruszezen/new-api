package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runOaiStreamHandler(t *testing.T, model string, body string) *dto.Usage {
	t.Helper()
	gin.SetMode(gin.TestMode)
	previousTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = previousTimeout })

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{
		RelayFormat:     types.RelayFormatOpenAI,
		RelayMode:       relayconstant.RelayModeChatCompletions,
		OriginModelName: model,
		IsStream:        true,
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: model},
	}
	info.SetEstimatePromptTokens(7)
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}

	usage, apiErr := OaiStreamHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	return usage
}

// Some OpenAI-compatible gateways attach the full cumulative usage to the
// penultimate event and then send an empty final event. That reported usage
// must be billed instead of falling back to a local text estimate.
func TestOaiStreamHandlerUsesPenultimateUsageWhenFinalEventHasNone(t *testing.T) {
	body := "data: " + `{"id":"c1","model":"gpt-4o-mini","choices":[{"index":0,"delta":{"role":"assistant","content":"hello"}}]}` + "\n\n" +
		"data: " + `{"id":"c1","model":"gpt-4o-mini","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":"stop"}],"usage":{"prompt_tokens":120,"completion_tokens":45,"total_tokens":165}}` + "\n\n" +
		"data: " + `{"id":"c1","model":"gpt-4o-mini","choices":[]}` + "\n\n" +
		"data: [DONE]\n\n"

	usage := runOaiStreamHandler(t, "gpt-4o-mini", body)
	assert.Equal(t, 120, usage.PromptTokens)
	assert.Equal(t, 45, usage.CompletionTokens)
	assert.Equal(t, 165, usage.TotalTokens)
}

// A usage object on the final event stays authoritative over the penultimate one.
func TestOaiStreamHandlerPrefersFinalEventUsage(t *testing.T) {
	body := "data: " + `{"id":"c1","model":"gpt-4o-mini","choices":[{"index":0,"delta":{"content":"hello"}}],"usage":{"prompt_tokens":120,"completion_tokens":1,"total_tokens":121}}` + "\n\n" +
		"data: " + `{"id":"c1","model":"gpt-4o-mini","choices":[],"usage":{"prompt_tokens":120,"completion_tokens":45,"total_tokens":165}}` + "\n\n" +
		"data: [DONE]\n\n"

	usage := runOaiStreamHandler(t, "gpt-4o-mini", body)
	assert.Equal(t, 120, usage.PromptTokens)
	assert.Equal(t, 45, usage.CompletionTokens)
	assert.Equal(t, 165, usage.TotalTokens)
}

// A prompt-only placeholder usage on the penultimate event does not report the
// output, so the local text estimate must still bill the completion.
func TestOaiStreamHandlerIgnoresPromptOnlyPenultimateUsage(t *testing.T) {
	body := "data: " + `{"id":"c1","model":"gpt-4o-mini","choices":[{"index":0,"delta":{"role":"assistant","content":"hello"}}]}` + "\n\n" +
		"data: " + `{"id":"c1","model":"gpt-4o-mini","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":"stop"}],"usage":{"prompt_tokens":120,"completion_tokens":0,"total_tokens":120}}` + "\n\n" +
		"data: " + `{"id":"c1","model":"gpt-4o-mini","choices":[]}` + "\n\n" +
		"data: [DONE]\n\n"

	usage := runOaiStreamHandler(t, "gpt-4o-mini", body)
	assert.Equal(t, 7, usage.PromptTokens)
	assert.Positive(t, usage.CompletionTokens)
}
