package kling

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Task submission authenticates with the channel key: "accessKey|secretKey" is
// signed into a JWT, while a single key (or a new-api "sk-" key) is sent as is.
func TestBuildRequestHeaderAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name      string
		key       string
		wantToken string
		wantJWT   bool
		wantErr   bool
	}{
		{name: "single key", key: "plain-key", wantToken: "plain-key"},
		{name: "single key with spaces", key: "  plain-key\n", wantToken: "plain-key"},
		{name: "new-api relay key", key: "sk-abc", wantToken: "sk-abc"},
		{name: "access and secret key", key: "ak|sk", wantJWT: true},
		{name: "too many parts", key: "a|b|c", wantErr: true},
		{name: "empty", key: "", wantErr: true},
		{name: "blank", key: "   ", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adaptor := &TaskAdaptor{}
			adaptor.Init(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ApiKey: tc.key}})
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)
			req := httptest.NewRequest(http.MethodPost, "https://kling.example/v1/videos/text2video", nil)

			err := adaptor.BuildRequestHeader(ctx, req, &relaycommon.RelayInfo{})
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			token, found := strings.CutPrefix(req.Header.Get("Authorization"), "Bearer ")
			require.True(t, found)
			if !tc.wantJWT {
				assert.Equal(t, tc.wantToken, token)
				return
			}
			parsed, err := jwt.Parse(token, func(*jwt.Token) (any, error) { return []byte("sk"), nil },
				jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
			require.NoError(t, err)
			issuer, err := parsed.Claims.GetIssuer()
			require.NoError(t, err)
			assert.Equal(t, "ak", issuer)
		})
	}
}

// OpenAI-style video requests carry a WIDTHxHEIGHT size; Kling gets the nearest
// aspect ratio it supports, and a metadata aspect_ratio still overrides it.
func TestBuildRequestBodyMapsSizeToNearestAspectRatio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		size     string
		metadata map[string]any
		want     string
	}{
		{size: "1280x720", want: "16:9"},
		{size: "1920x1080", want: "16:9"},
		{size: "1792x1024", want: "16:9"},
		{size: "1280X720", want: "16:9"},
		{size: "720x1280", want: "9:16"},
		{size: "1080x1920", want: "9:16"},
		{size: "1024x1792", want: "9:16"},
		{size: "1024x1024", want: "1:1"},
		{size: "512x512", want: "1:1"},
		{size: "720x720", want: "1:1"},
		{size: "", want: "1:1"},
		{size: "abc", want: "1:1"},
		{size: "1280x720x2", want: "1:1"},
		{size: "0x720", want: "1:1"},
		{size: "1024x1792", metadata: map[string]any{"aspect_ratio": "16:9"}, want: "16:9"},
	} {
		t.Run(tc.size, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Set("task_request", relaycommon.TaskSubmitReq{Prompt: "a cat", Size: tc.size, Metadata: tc.metadata})
			adaptor := &TaskAdaptor{}

			body, err := adaptor.BuildRequestBody(ctx, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "kling-v1"}})
			require.NoError(t, err)
			data, err := io.ReadAll(body)
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, common.Unmarshal(data, &payload))
			assert.Equal(t, tc.want, payload["aspect_ratio"])
		})
	}
}
