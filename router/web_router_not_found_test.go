package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unknown API and asset paths fall through the web router, which has already
// applied the one-week static cache header. A CDN must never keep that 404.
func TestWebRouterNotFoundIsNeverCached(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetWebRouter(engine, WebAssets{IndexPage: []byte("<html></html>")})

	for _, path := range []string{"/v1/unknown-endpoint", "/api/unknown", "/assets/index-missing.js"} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

			require.Equal(t, http.StatusNotFound, recorder.Code)
			assert.Equal(t, "no-store, no-cache, must-revalidate, private, max-age=0", recorder.Header().Get("Cache-Control"))
			assert.Equal(t, "no-cache", recorder.Header().Get("Pragma"))
			assert.Equal(t, "0", recorder.Header().Get("Expires"))
		})
	}
}
