package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func firstTokenErrorTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/log")
	g.Use(func(c *gin.Context) {
		switch c.GetHeader("X-Test-Role") {
		case "admin":
			c.Set("role", common.RoleAdminUser)
			c.Set("id", 1)
			c.Set("username", "admin")
		default:
			c.Set("role", common.RoleCommonUser)
			c.Set("id", 2)
			c.Set("username", "user")
		}
		if c.GetInt("role") < common.RoleAdminUser {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "forbidden"})
			return
		}
		c.Next()
	})
	g.GET("/first-token-error", GetFirstTokenErrorLogs)
	g.GET("/first-token-error/:id/body", GetFirstTokenErrorBody)
	return r
}

func TestFirstTokenErrorAPI_NonAdminForbidden(t *testing.T) {
	r := firstTokenErrorTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/log/first-token-error", nil)
	req.Header.Set("X-Test-Role", "user")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/log/first-token-error/1/body", nil)
	req.Header.Set("X-Test-Role", "user")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}
