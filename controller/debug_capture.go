package controller

import (
	"net/http"

	debugcapture "github.com/QuantumNous/new-api/pkg/debug_capture"

	"github.com/gin-gonic/gin"
)

// GetDebugCaptureErrors 返回错误捕获功能（功能1）记录的所有错误 / 计费异常请求。
// 记录含下游请求体与上游响应体等敏感数据，仅 Root/Super Admin 可访问。
func GetDebugCaptureErrors(c *gin.Context) {
	records := debugcapture.ErrorRecords()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"records": records,
			"total":   len(records),
		},
	})
}

// ClearDebugCaptureErrors 清空错误捕获功能（功能1）的全部记录。
func ClearDebugCaptureErrors(c *gin.Context) {
	debugcapture.ClearErrorRecords()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// GetDebugCaptureRecent 返回最近请求功能（功能2）保留的最近请求（最新在前）。
func GetDebugCaptureRecent(c *gin.Context) {
	records := debugcapture.RecentRecords()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"records": records,
			"total":   len(records),
		},
	})
}
