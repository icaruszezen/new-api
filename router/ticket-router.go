package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

// registerTicketRoutes 挂载工单接口：/self 供登录用户管理自己的工单，/admin 供管理员处理全部工单。
// 用户侧写操作（新建、回复、关闭）共用按用户计数的 TicketWriteRateLimit。
func registerTicketRoutes(apiRouter *gin.RouterGroup) {
	selfRoute := apiRouter.Group("/ticket/self")
	selfRoute.Use(middleware.UserAuth(), middleware.DisableCache())
	{
		selfRoute.GET("", controller.GetSelfTickets)
		selfRoute.GET("/unread", controller.GetSelfTicketUnreadCount)
		selfRoute.POST("", middleware.TicketWriteRateLimit(), controller.CreateSelfTicket)
		selfRoute.GET("/:id", controller.GetSelfTicket)
		selfRoute.POST("/:id/reply", middleware.TicketWriteRateLimit(), controller.ReplySelfTicket)
		selfRoute.POST("/:id/close", middleware.TicketWriteRateLimit(), controller.CloseSelfTicket)
		selfRoute.GET("/:id/attachments/:attachment_id", controller.GetSelfTicketAttachment)
	}

	adminRoute := apiRouter.Group("/ticket/admin")
	adminRoute.Use(middleware.AdminAuth(), middleware.DisableCache())
	{
		adminRoute.GET("", controller.AdminGetTickets)
		adminRoute.GET("/stats", controller.AdminGetTicketStats)
		adminRoute.GET("/:id", controller.AdminGetTicket)
		adminRoute.POST("/:id/reply", controller.AdminReplyTicket)
		adminRoute.PUT("/:id/status", controller.AdminUpdateTicketStatus)
		adminRoute.GET("/:id/attachments/:attachment_id", controller.AdminGetTicketAttachment)
	}
}
