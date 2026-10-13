package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ticketRoutes = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/api/ticket/self"},
	{http.MethodGet, "/api/ticket/self/unread"},
	{http.MethodPost, "/api/ticket/self"},
	{http.MethodGet, "/api/ticket/self/:id"},
	{http.MethodPost, "/api/ticket/self/:id/reply"},
	{http.MethodPost, "/api/ticket/self/:id/close"},
	{http.MethodGet, "/api/ticket/self/:id/attachments/:attachment_id"},
	{http.MethodGet, "/api/ticket/admin"},
	{http.MethodGet, "/api/ticket/admin/stats"},
	{http.MethodGet, "/api/ticket/admin/:id"},
	{http.MethodPost, "/api/ticket/admin/:id/reply"},
	{http.MethodPut, "/api/ticket/admin/:id/status"},
	{http.MethodGet, "/api/ticket/admin/:id/attachments/:attachment_id"},
}

func TestTicketRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	require.NotPanics(t, func() { SetApiRouter(engine) })

	routes := make(map[string]struct{}, len(engine.Routes()))
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, route := range ticketRoutes {
		assert.Contains(t, routes, route.method+" "+route.path)
	}
}

// 所有工单接口（含附件下载）都必须先经过登录校验，未带凭证时返回 401 而不是落到 404。
func TestTicketRoutesRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)

	for _, route := range ticketRoutes {
		path := strings.NewReplacer(":id", "1", ":attachment_id", "1").Replace(route.path)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(route.method, path, nil))
		assert.Equal(t, http.StatusUnauthorized, recorder.Code, "%s %s", route.method, path)
	}
}
