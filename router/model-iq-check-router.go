package router

import (
	"net/http"

	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
)

// registerModelIQCheckRoutes 挂载模型智商检查的管理端接口。
// 检查项决定用哪个分组的渠道发起真实上游调用，因此与渠道监控一样复用渠道权限。
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
	{method: http.MethodPost, path: "/:id/run", permission: authz.ChannelOperate, handler: controller.RunModelIQCheck},
}
