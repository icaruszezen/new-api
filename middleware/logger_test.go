package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// OAuth callbacks carry one-time authorization codes and state in the query
// string; the access log must drop them while the handler still sees them.
func TestSetUpLoggerRedactsOAuthCallbackQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	common.LogWriterMu.Lock()
	previousWriter := gin.DefaultWriter
	gin.DefaultWriter = &output
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultWriter = previousWriter
		common.LogWriterMu.Unlock()
	})

	engine := gin.New()
	SetUpLogger(engine)
	var handlerCode string
	engine.GET("/api/oauth/:provider", func(c *gin.Context) {
		handlerCode = c.Query("code")
		c.Status(http.StatusOK)
	})
	engine.GET("/api/status", func(c *gin.Context) { c.Status(http.StatusOK) })

	tests := []struct {
		name       string
		target     string
		wantLogged string
		forbidden  string
	}{
		{name: "oauth callback", target: "/api/oauth/github?code=secret-code&state=secret-state", wantLogged: " /api/oauth/github\n", forbidden: "secret"},
		{name: "other routes keep query", target: "/api/status?verbose=1", wantLogged: " /api/status?verbose=1\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output.Reset()
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.target, nil))

			require.Equal(t, http.StatusOK, recorder.Code)
			logged := output.String()
			assert.Contains(t, logged, tt.wantLogged)
			if tt.forbidden != "" {
				assert.NotContains(t, logged, tt.forbidden)
			}
		})
	}
	assert.Equal(t, "secret-code", handlerCode)
}
