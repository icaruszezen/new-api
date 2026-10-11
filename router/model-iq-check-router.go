package router

import (
	"net/http"

	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
)

// registerModelIQCheckRoutes 挂载智商检查入口的管理端配置接口。
// 与渠道监控状态页的开关一样，由拥有渠道权限的管理员维护。
func registerModelIQCheckRoutes(apiRouter *gin.RouterGroup) {
	iqCheckRoute := apiRouter.Group("/iq-check")
	iqCheckRoute.Use(middleware.AdminAuth())

	for _, route := range modelIQCheckPermissionRoutes {
		iqCheckRoute.Handle(route.method, route.path,
			middleware.RequirePermission(route.permission),
			route.handler,
		)
	}
}

var modelIQCheckPermissionRoutes = []permissionRoute{
	{method: http.MethodGet, path: "/config", permission: authz.ChannelRead, handler: controller.GetModelIQCheckConfig},
	{method: http.MethodPut, path: "/config", permission: authz.ChannelWrite, handler: controller.UpdateModelIQCheckConfig},
}
