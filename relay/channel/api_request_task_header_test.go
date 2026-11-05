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

// authTaskAdaptor sets its own credentials and body type, as every built-in
// task adaptor does.
type authTaskAdaptor struct {
	stubTaskAdaptor
}

func (a *authTaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer adaptor")
	req.Header.Set("Content-Type", "application/json")
	return nil
}

// Task submission sends the headers the adaptor builds and nothing from the
// channel Header Override. The poller's FetchTask and the content proxy build
// their headers from the channel key alone, so a submission that used the
// override could create an upstream job its own poll cannot read. A
// passthrough rule must also not relabel the adaptor's JSON body with the
// client's multipart Content-Type.
func TestDoTaskApiRequestSendsOnlyAdaptorHeaders(t *testing.T) {
	service.InitHttpClient()
	received := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	ctx.Request.Header.Set("Content-Type", "multipart/form-data; boundary=abc")
	ctx.Request.Header.Set("X-Trace-Id", "trace-123")
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ApiKey: "channel-key",
		HeadersOverride: map[string]any{
			"*":              "",
			"Authorization":  "Bearer override",
			"X-Upstream-Key": "{api_key}",
		},
	}}

	payload := []byte(`{"model":"test-model","prompt":"hello"}`)
	resp, err := DoTaskApiRequest(&authTaskAdaptor{stubTaskAdaptor{baseURL: server.URL}}, ctx, info, bytes.NewReader(payload))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	headers := <-received
	assert.Equal(t, "Bearer adaptor", headers.Get("Authorization"))
	assert.Equal(t, "application/json", headers.Get("Content-Type"))
	assert.Empty(t, headers.Get("X-Upstream-Key"))
	assert.Empty(t, headers.Get("X-Trace-Id"))
}
