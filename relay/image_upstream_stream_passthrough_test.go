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
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Upstream-stream synthesize rewrites the body to stream:true. Under global
// pass-through the raw body is forwarded instead, so synthesize must stay off:
// otherwise the upstream gets a non-stream body with a forced
// Accept: text/event-stream and the log claims an upstream-stream aggregation.
func TestImageHelperUpstreamStreamSynthesizeHonorsGlobalPassThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	globalSettings := model_setting.GetGlobalSettings()
	prevPassThrough := globalSettings.PassThroughRequestEnabled
	t.Cleanup(func() { globalSettings.PassThroughRequestEnabled = prevPassThrough })

	const raw = `{"model":"gpt-image-1","prompt":"a cat"}`
	tests := []struct {
		name              string
		globalPassThrough bool
		wantSynthesize    bool
		wantAccept        string
		wantStream        bool
	}{
		{name: "global pass-through forwards the client request as-is", globalPassThrough: true},
		{name: "without pass-through the upstream is asked to stream", wantSynthesize: true, wantAccept: "text/event-stream", wantStream: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalSettings.PassThroughRequestEnabled = tt.globalPassThrough

			type upstreamRequest struct {
				accept string
				body   []byte
			}
			received := make(chan upstreamRequest, 1)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				received <- upstreamRequest{accept: r.Header.Get("Accept"), body: body}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":{"message":"rejected by test upstream","type":"invalid_request_error"}}`))
			}))
			defer upstream.Close()

			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(raw))
			c.Request.Header.Set("Content-Type", "application/json")
			common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
			common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, upstream.URL)
			common.SetContextKey(c, constant.ContextKeyChannelKey, "sk-test")
			common.SetContextKey(c, constant.ContextKeyOriginalModel, "gpt-image-1")
			common.SetContextKey(c, constant.ContextKeyChannelSetting, dto.ChannelSettings{ImageNonStreamViaUpstreamStreamEnabled: true})

			var request dto.ImageRequest
			require.NoError(t, common.UnmarshalJsonStr(raw, &request))
			info := relaycommon.GenRelayInfoImage(c, &request)

			require.NotNil(t, ImageHelper(c, info), "the test upstream rejects every request")
			var got upstreamRequest
			select {
			case got = <-received:
			default:
				require.FailNow(t, "the upstream received no request")
			}

			assert.Equal(t, tt.wantSynthesize, info.ImageUpstreamStreamSynthesize)
			assert.Equal(t, tt.wantAccept, got.accept)
			assert.Equal(t, tt.wantStream, gjson.GetBytes(got.body, "stream").Bool())
			if tt.globalPassThrough {
				assert.Equal(t, raw, string(got.body))
			}
		})
	}
}
