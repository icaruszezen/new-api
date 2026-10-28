package baidu

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ERNIE reports usage on its stream chunks. A stream that ends without it
// (client gone with drain off, drain cut, upstream cut) bills the text
// received; an error chunk is a rejection, not a cut-short reply.
func TestBaiduStreamHandlerUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	const estimatedPrompt = 9
	estimatedCompletion := service.EstimateTokenByModel("ernie-test", "hello world")
	require.Positive(t, estimatedCompletion)

	tests := []struct {
		name      string
		chunks    []string
		wantUsage *dto.Usage
	}{
		{
			name: "upstream usage is billed",
			chunks: []string{
				`{"id":"as-1","result":"hello","is_end":false,"usage":{"prompt_tokens":4,"total_tokens":5}}`,
				`{"id":"as-1","result":" world","is_end":true,"usage":{"prompt_tokens":4,"total_tokens":7}}`,
			},
			wantUsage: &dto.Usage{PromptTokens: 4, CompletionTokens: 3, TotalTokens: 7},
		},
		{
			name: "no usage bills the text received",
			chunks: []string{
				`{"id":"as-1","result":"hello","is_end":false}`,
				`{"id":"as-1","result":" world","is_end":false}`,
			},
			wantUsage: &dto.Usage{PromptTokens: estimatedPrompt, CompletionTokens: estimatedCompletion, TotalTokens: estimatedPrompt + estimatedCompletion},
		},
		{
			name:      "an error chunk is not billed",
			chunks:    []string{`{"error_code":18,"error_msg":"Open api qps request limit reached"}`},
			wantUsage: &dto.Usage{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			info := &relaycommon.RelayInfo{
				DisablePing: true,
				ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "ernie-test"},
			}
			info.SetEstimatePromptTokens(estimatedPrompt)
			var body strings.Builder
			for _, chunk := range tt.chunks {
				body.WriteString("data: " + chunk + "\n\n")
			}

			apiErr, usage := baiduStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body.String()))})

			require.Nil(t, apiErr)
			assert.Equal(t, tt.wantUsage, usage)
		})
	}
}
