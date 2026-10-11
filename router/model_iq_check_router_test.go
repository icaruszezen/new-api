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

// The IQ check entry link is shown to every visitor, so changing it must stay
// behind the same channel permissions as the channel monitoring page toggle.
func TestModelIQCheckAdminRoutesUseChannelPermissions(t *testing.T) {
	cases := []struct {
		method     string
		path       string
		permission authz.Permission
		handler    gin.HandlerFunc
	}{
		{http.MethodGet, "/config", authz.ChannelRead, controller.GetModelIQCheckConfig},
		{http.MethodPut, "/config", authz.ChannelWrite, controller.UpdateModelIQCheckConfig},
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
		registerChannelRoutes(api)
		registerChannelMonitoringRoutes(api)
		registerModelIQCheckRoutes(api)
	})
}

// The admin routes must reach AdminAuth instead of falling through to the
// relay router's not-found handler.
func TestModelIQCheckRoutesMatchWhenApiAndRelayRoutesExist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)
	SetRelayRouter(engine)
	engine.NoRoute(controller.RelayNotFound)

	for _, method := range []string{http.MethodGet, http.MethodPut} {
		req := httptest.NewRequest(method, "/api/iq-check/config", nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		require.NotEqual(t, http.StatusNotFound, rec.Code,
			"%s /api/iq-check/config fell through to RelayNotFound: %s", method, rec.Body.String())
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
