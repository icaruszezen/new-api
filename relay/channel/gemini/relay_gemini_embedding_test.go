package gemini

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiEmbeddingHandlersBillUpstreamUsageMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const estimate = 99
	const withUsage = `"usageMetadata":{"promptTokenCount":12,"promptTokenDetails":[{"modality":"TEXT","tokenCount":4},{"modality":"IMAGE","tokenCount":8}]}`
	tests := []struct {
		name           string
		native         bool
		batch          bool
		body           string
		wantPrompt     int
		wantImage      int
		wantLocalCount bool
	}{
		{name: "native single uses usageMetadata", native: true, body: `{"embedding":{"values":[0.1]},` + withUsage + `}`, wantPrompt: 12, wantImage: 8},
		{name: "native batch uses usageMetadata", native: true, batch: true, body: `{"embeddings":[{"values":[0.1]}],` + withUsage + `}`, wantPrompt: 12, wantImage: 8},
		{name: "OpenAI format uses usageMetadata", body: `{"embeddings":[{"values":[0.1]}],` + withUsage + `}`, wantPrompt: 12, wantImage: 8},
		{name: "native without usageMetadata estimates", native: true, body: `{"embedding":{"values":[0.1]}}`, wantPrompt: estimate, wantLocalCount: true},
		{name: "OpenAI format with zero count estimates", body: `{"embeddings":[{"values":[0.1]}],"usageMetadata":{"promptTokenCount":0}}`, wantPrompt: estimate, wantLocalCount: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
			info := &relaycommon.RelayInfo{
				IsGeminiBatchEmbedding: tt.batch,
				ChannelMeta:            &relaycommon.ChannelMeta{UpstreamModelName: "gemini-embedding-001"},
			}
			info.SetEstimatePromptTokens(estimate)
			resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(tt.body))}

			var usage *dto.Usage
			var apiErr *types.NewAPIError
			if tt.native {
				usage, apiErr = NativeGeminiEmbeddingHandler(c, resp, info)
			} else {
				usage, apiErr = GeminiEmbeddingHandler(c, info, resp)
			}
			require.Nil(t, apiErr)

			require.NotNil(t, usage)
			assert.Equal(t, tt.wantPrompt, usage.PromptTokens)
			assert.Equal(t, tt.wantPrompt, usage.TotalTokens)
			assert.Zero(t, usage.CompletionTokens)
			assert.Equal(t, tt.wantImage, usage.PromptTokensDetails.ImageTokens)
			assert.Equal(t, tt.wantLocalCount, common.GetContextKeyBool(c, constant.ContextKeyLocalCountTokens))
		})
	}
}
