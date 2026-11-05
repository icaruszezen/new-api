package channel

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authTaskAdaptor sets its own credentials, as every built-in task adaptor does.
type authTaskAdaptor struct {
	stubTaskAdaptor
}

func (a *authTaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer adaptor")
	req.Header.Set("Content-Type", "application/json")
	return nil
}

// Task submission honours the channel Header Override like chat requests do:
// overrides are applied after the adaptor's headers, with placeholders resolved.
func TestDoTaskApiRequestAppliesChannelHeaderOverride(t *testing.T) {
	service.InitHttpClient()
	received := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ApiKey: "channel-key",
		HeadersOverride: map[string]any{
			"Authorization":    "Bearer override",
			"X-Upstream-Key":   "{api_key}",
			"X-Upstream-Trace": "{client_header:X-Trace-Id}",
		},
	}}

	payload := []byte(`{"model":"test-model","prompt":"hello"}`)
	resp, err := DoTaskApiRequest(&authTaskAdaptor{stubTaskAdaptor{baseURL: server.URL}}, ctx, info, bytes.NewReader(payload))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	headers := <-received
	assert.Equal(t, "Bearer override", headers.Get("Authorization"))
	assert.Equal(t, "channel-key", headers.Get("X-Upstream-Key"))
	assert.Equal(t, "trace-123", headers.Get("X-Upstream-Trace"))
	assert.Equal(t, "application/json", headers.Get("Content-Type"), "headers the override does not name stay as the adaptor set them")
}
