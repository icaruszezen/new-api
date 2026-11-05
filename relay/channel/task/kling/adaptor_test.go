package kling

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
