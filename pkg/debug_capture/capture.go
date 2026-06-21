package debug_capture

import (
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// FinalizeCapture 是两个调试捕获功能的统一入口，在 relay 请求生命周期结束时（Relay defer）调用。
//   - 功能2（最近请求）：开启时无条件保留本次请求。
//   - 功能1（错误 / 计费异常）：开启时仅在请求发生错误，或上游未返回计费信息时保留。
//
// 两个功能互相独立，各自判断开关、各自存储。调用方需保证已排除 WebSocket Realtime 请求。
func FinalizeCapture(c *gin.Context, info *relaycommon.RelayInfo, relayErr *types.NewAPIError) {
	setting := operation_setting.GetDebugSetting()
	if !setting.RecentCaptureEnabled && !setting.ErrorCaptureEnabled {
		return
	}

	rec := buildRecord(c, info, relayErr)

	if setting.RecentCaptureEnabled {
		recent := rec
		recent.Reason = ReasonRecent
		recentStore.Push(recent)
	}

	if setting.ErrorCaptureEnabled {
		if reason := errorReason(info, relayErr); reason != "" {
			errRec := rec
			errRec.Reason = reason
			errorStore.Append(errRec)
		}
	}
}
