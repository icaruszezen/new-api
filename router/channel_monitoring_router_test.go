package router

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/service/authz"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Reading the monitor config exposes group names, and writing it changes which
// upstreams get probed, so both sides must stay behind channel permissions.
func TestChannelMonitoringConfigRoutesUseChannelPermissions(t *testing.T) {
	assertChannelMonitoringRoutePermission(t, http.MethodGet, "/config", authz.ChannelRead, controller.GetChannelMonitoringConfig)
	assertChannelMonitoringRoutePermission(t, http.MethodPut, "/config", authz.ChannelWrite, controller.UpdateChannelMonitoringConfig)
}

func TestChannelMonitoringRoutesRegisterWithoutConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api")

	require.NotPanics(t, func() {
		registerChannelRoutes(api)
		registerChannelMonitoringRoutes(api)
	})
}

func TestGetApiChannelMatchesWhenMonitoringAndRelayRoutesExist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)
	SetRelayRouter(engine)
	engine.NoRoute(controller.RelayNotFound)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/channel"},
		{http.MethodGet, "/api/channel/"},
		{http.MethodPost, "/api/channel"},
		{http.MethodPut, "/api/channel/"},
		{http.MethodGet, "/api/channel-monitoring/status"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		require.NotEqual(t, http.StatusNotFound, rec.Code,
			"%s %s fell through to RelayNotFound: %s", tc.method, tc.path, rec.Body.String())
		assert.NotContains(t, rec.Body.String(), "Invalid URL")
	}
}

func assertChannelMonitoringRoutePermission(t *testing.T, method string, path string, permission authz.Permission, handler any) {
	t.Helper()
	for _, route := range channelMonitoringPermissionRoutes {
		if route.method == method && route.path == path {
			assert.Equal(t, permission, route.permission)
			assert.Equal(t, reflect.ValueOf(handler).Pointer(), reflect.ValueOf(route.handler).Pointer())
			return
		}
	}
	t.Fatalf("route %s %s not found", method, path)
}
