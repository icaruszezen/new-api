package aws

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// InvokeModel takes the model and streaming from the request, so the
// pass-through body keeps no copy of either, even when the client repeats
// them. Everything else keeps the client's bytes, and the stored request body
// that a retry reads again stays as the client sent it.
func TestBuildAwsRequestBodyPassThroughKeepsClientBytes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "repeated model and stream",
			raw:  `{"model":"a","stream":true,"max_tokens":8,"stream":false,"model":"b"}`,
			want: `{"max_tokens":8}`,
		},
		{
			name: "schema order and number spelling",
			raw:  `{"model":"a","tools":[{"name":"t","input_schema":{"type":"object","properties":{"z":{"type":"string"},"a":{"type":"number","maximum":1.0}}}}],"max_tokens":1e3}`,
			want: `{"tools":[{"name":"t","input_schema":{"type":"object","properties":{"z":{"type":"string"},"a":{"type":"number","maximum":1.0}}}}],"max_tokens":1e3}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(tt.raw))
			t.Cleanup(func() { common.CleanupBodyStorage(c) })
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
			info.ChannelSetting.PassThroughBodyEnabled = true

			body, err := buildAwsRequestBody(c, info, nil)

			require.NoError(t, err)
			assert.Equal(t, tt.want, string(body))
			storage, err := common.GetBodyStorage(c)
			require.NoError(t, err)
			stored, err := storage.Bytes()
			require.NoError(t, err)
			assert.Equal(t, tt.raw, string(stored))
		})
	}
}

func TestBuildAwsRequestBodyPassThroughRejectsNonObject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(`[{"model":"a"}]`))
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
	info.ChannelSetting.PassThroughBodyEnabled = true

	_, err := buildAwsRequestBody(c, info, nil)

	require.Error(t, err)
}
