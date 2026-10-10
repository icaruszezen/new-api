package router

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/service/authz"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Reading the IQ check config exposes group names and raw upstream errors,
// writing it decides which upstreams get called, and a manual run spends
// upstream quota, so every admin route must stay behind channel permissions.
func TestModelIQCheckAdminRoutesUseChannelPermissions(t *testing.T) {
	cases := []struct {
		method     string
		path       string
		permission authz.Permission
		handler    gin.HandlerFunc
	}{
		{http.MethodGet, "/config", authz.ChannelRead, controller.GetModelIQCheckConfig},
		{http.MethodPut, "/config", authz.ChannelWrite, controller.UpdateModelIQCheckConfig},
		{http.MethodPost, "/:id/run", authz.ChannelOperate, controller.RunModelIQCheck},
	}
	require.Len(t, modelIQCheckPermissionRoutes, len(cases), "every admin route needs an explicit permission expectation")

	for _, testCase := range cases {
		route, found := findModelIQCheckRoute(testCase.method, testCase.path)
		require.True(t, found, "route %s %s not found", testCase.method, testCase.path)
		assert.Equal(t, testCase.permission, route.permission, "%s %s", testCase.method, testCase.path)
		assert.Equal(t, reflect.ValueOf(testCase.handler).Pointer(), reflect.ValueOf(route.handler).Pointer(), "%s %s", testCase.method, testCase.path)
	}
}

func TestModelIQCheckRoutesRegisterWithoutConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api")

	require.NotPanics(t, func() {
		api.GET("/iq-check/status", controller.GetModelIQCheckStatus)
		api.GET("/iq-check/results", controller.GetModelIQCheckResults)
		registerChannelRoutes(api)
		registerChannelMonitoringRoutes(api)
		registerModelIQCheckRoutes(api)
	})
}

// The public page polls /api/iq-check/status and /api/iq-check/results without
// a session; they must resolve to their handlers instead of falling through to
// the relay router, while the admin routes on the same prefix still reach
// AdminAuth.
func TestModelIQCheckRoutesMatchWhenApiAndRelayRoutesExist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Rate limiters bind to the backend chosen at registration time; use the
	// in-memory limiter so the public handlers actually run.
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
	})
	engine := gin.New()
	SetApiRouter(engine)
	SetRelayRouter(engine)
	engine.NoRoute(controller.RelayNotFound)

	for _, path := range []string{"/api/iq-check/status", "/api/iq-check/results?ids=1,2"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, "GET %s: %s", path, rec.Body.String())
		assert.Contains(t, rec.Body.String(), `"success":true`, "GET %s", path)
	}

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/iq-check/config"},
		{http.MethodPut, "/api/iq-check/config"},
		{http.MethodPost, "/api/iq-check/abc/run"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		require.NotEqual(t, http.StatusNotFound, rec.Code,
			"%s %s fell through to RelayNotFound: %s", tc.method, tc.path, rec.Body.String())
		assert.NotContains(t, rec.Body.String(), "Invalid URL")
	}
}

func findModelIQCheckRoute(method string, path string) (permissionRoute, bool) {
	for _, route := range modelIQCheckPermissionRoutes {
		if route.method == method && route.path == path {
			return route, true
		}
	}
	return permissionRoute{}, false
}
