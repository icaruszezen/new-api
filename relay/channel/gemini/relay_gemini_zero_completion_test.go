package gemini

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A stream cut before the final chunk leaves prompt-only usageMetadata. The
// estimated completion must be filled into the upstream metadata snapshot that
// settlement prefers, without dropping its cache and modality details.
func TestGeminiStreamHandlerZeroCompletionKeepsUpstreamCacheDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 300
	t.Cleanup(func() {
		constant.StreamingTimeout = oldStreamingTimeout
	})

	info := &relaycommon.RelayInfo{
		OriginModelName: "gemini-2.5-flash",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-2.5-flash",
		},
	}
	info.SetEstimatePromptTokens(20)

	chunk := dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{
			{
				Content: dto.GeminiChatContent{
					Role:  "model",
					Parts: []dto.GeminiPart{{Text: "partial answer"}},
				},
			},
		},
		UsageMetadata: dto.GeminiUsageMetadata{
			PromptTokenCount:        1000,
			CachedContentTokenCount: 800,
			TotalTokenCount:         1000,
			PromptTokensDetails: []dto.GeminiPromptTokensDetails{
				{Modality: "TEXT", TokenCount: 700},
				{Modality: "AUDIO", TokenCount: 300},
			},
		},
	}
	chunkData, err := common.Marshal(chunk)
	require.NoError(t, err)

	resp := &http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte("data: " + string(chunkData) + "\n" + "data: [DONE]\n"))),
	}

	usage, newAPIError := geminiStreamHandler(c, info, resp, func(_ string, _ *dto.GeminiChatResponse) bool {
		return true
	})
	require.Nil(t, newAPIError)
	require.NotNil(t, usage)
	require.Greater(t, usage.CompletionTokens, 0)

	require.NotNil(t, usage.BillingUsage)
	assert.True(t, usage.BillingUsage.Estimated)
	metadata := usage.BillingUsage.GeminiUsageMetadata
	require.NotNil(t, metadata)
	assert.Equal(t, 1000, metadata.PromptTokenCount)
	assert.Equal(t, 800, metadata.CachedContentTokenCount)
	assert.Equal(t, usage.CompletionTokens, metadata.CandidatesTokenCount)
	assert.Equal(t, 1000+usage.CompletionTokens, metadata.TotalTokenCount)
	assert.Equal(t, []dto.GeminiPromptTokensDetails{
		{Modality: "TEXT", TokenCount: 700},
		{Modality: "AUDIO", TokenCount: 300},
	}, metadata.PromptTokensDetails)
}
