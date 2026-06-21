package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// DebugSetting 控制两个互相独立的 relay 调试捕获功能：
//   - ErrorCaptureEnabled：开启后无条数上限地保留所有发生错误，
//     以及上游未返回计费信息（usage 为空或 total tokens 为 0）的请求。
//   - RecentCaptureEnabled：开启后用固定容量的内存环形缓冲保留最近的请求，
//     供前端手动刷新查看。
type DebugSetting struct {
	ErrorCaptureEnabled  bool `json:"error_capture_enabled"`
	RecentCaptureEnabled bool `json:"recent_capture_enabled"`
}

var debugSetting = DebugSetting{
	ErrorCaptureEnabled:  false,
	RecentCaptureEnabled: false,
}

func init() {
	config.GlobalConfig.Register("debug_setting", &debugSetting)
}

func GetDebugSetting() *DebugSetting {
	return &debugSetting
}

// IsAnyDebugCaptureEnabled 用于 relay 热路径快速判断是否需要捕获，
// 任一功能开启即返回 true。
func IsAnyDebugCaptureEnabled() bool {
	return debugSetting.ErrorCaptureEnabled || debugSetting.RecentCaptureEnabled
}
