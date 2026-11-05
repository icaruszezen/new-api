package relay

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
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestClaudeHelperForwardsThinkingBlockBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name         string
		model        string
		thinking     string
		wantThinking string
	}{
		{
			name:         "native request keeps every thinking field",
			model:        "claude-opus-5-5",
			thinking:     `{"type":"adaptive","display":"summarized","block_binding":{"prefix_mismatch_behavior":"drop_block"},"future_field":{"a":1}}`,
			wantThinking: `{"type":"adaptive","display":"summarized","block_binding":{"prefix_mismatch_behavior":"drop_block"},"future_field":{"a":1}}`,
		},
		{
			name:         "effort suffix re-renders adaptive and keeps block_binding",
			model:        "claude-opus-4-7-high",
			thinking:     `{"type":"adaptive","block_binding":{"prefix_mismatch_behavior":"drop_block"}}`,
			wantThinking: `{"type":"adaptive","display":"summarized","block_binding":{"prefix_mismatch_behavior":"drop_block"}}`,
		},
		{
			name:         "request without block_binding gets none",
			model:        "claude-opus-4-7-high",
			thinking:     `{"type":"adaptive"}`,
			wantThinking: `{"type":"adaptive","display":"summarized"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			received := make(chan []byte, 1)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				received <- body
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"rejected by test upstream"}}`))
			}))
			t.Cleanup(upstream.Close)

			raw := `{"model":"` + tt.model + `","max_tokens":1024,"thinking":` + tt.thinking +
				`,"messages":[{"role":"user","content":"hi"}]}`
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(raw))
			c.Request.Header.Set("Content-Type", "application/json")
			common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeAnthropic)
			common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, upstream.URL)
			common.SetContextKey(c, constant.ContextKeyChannelKey, "sk-test")
			common.SetContextKey(c, constant.ContextKeyOriginalModel, tt.model)

			var request dto.ClaudeRequest
			require.NoError(t, common.UnmarshalJsonStr(raw, &request))
			info := relaycommon.GenRelayInfoClaude(c, &request)

			require.NotNil(t, ClaudeHelper(c, info), "the test upstream rejects every request")
			select {
			case body := <-received:
				assert.JSONEq(t, tt.wantThinking, gjson.GetBytes(body, "thinking").Raw)
			default:
				require.FailNow(t, "the upstream received no request")
			}
		})
	}
}
