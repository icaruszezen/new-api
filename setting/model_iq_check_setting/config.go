package model_iq_check_setting

import "github.com/QuantumNous/new-api/setting/config"

// ModelIQCheckSetting 控制模型智商检查：按管理员配置的提示词定期调用指定分组下的模型，
// 并在公开页面展示模型的回答，用于发现上游悄悄「降智」。
// Checks 为检查项 JSON 数组，其结构与校验在 pkg/modeliqcheck 中定义。
type ModelIQCheckSetting struct {
	Enabled               bool   `json:"enabled"`
	Checks                string `json:"checks"`
	HistoryLimit          int    `json:"history_limit"`
	RequestTimeoutSeconds int    `json:"request_timeout_seconds"`
}

const (
	EnabledOptionKey        = "model_iq_check_setting.enabled"
	ChecksOptionKey         = "model_iq_check_setting.checks"
	HistoryLimitOptionKey   = "model_iq_check_setting.history_limit"
	RequestTimeoutOptionKey = "model_iq_check_setting.request_timeout_seconds"

	// HistoryLimit 是每个检查项保留的结果条数。
	DefaultHistoryLimit = 10
	MinHistoryLimit     = 1
	MaxHistoryLimit     = 50

	// RequestTimeoutSeconds 是单次检查（含完整流式输出）的硬上限。推理模型可能先静默思考
	// 数分钟再输出，生成长网页还要数分钟，因此至少 10 分钟；低于下限的旧配置按默认值处理。
	DefaultRequestTimeoutSeconds = 600
	MinRequestTimeoutSeconds     = 600
	MaxRequestTimeoutSeconds     = 3600
)

var modelIQCheckSetting = ModelIQCheckSetting{
	Enabled:               false,
	Checks:                "[]",
	HistoryLimit:          DefaultHistoryLimit,
	RequestTimeoutSeconds: DefaultRequestTimeoutSeconds,
}

func init() {
	config.GlobalConfig.Register("model_iq_check_setting", &modelIQCheckSetting)
}

// GetSetting 返回一份已归一化的配置副本，调用方无需再做边界检查。
func GetSetting() ModelIQCheckSetting {
	s := modelIQCheckSetting
	s.HistoryLimit = clamp(s.HistoryLimit, MinHistoryLimit, MaxHistoryLimit, DefaultHistoryLimit)
	s.RequestTimeoutSeconds = clamp(s.RequestTimeoutSeconds, MinRequestTimeoutSeconds, MaxRequestTimeoutSeconds, DefaultRequestTimeoutSeconds)
	if s.Checks == "" {
		s.Checks = "[]"
	}
	return s
}

// IsEnabled 供公开接口与前端开关读取，避免调用方为了一个布尔值归一化整份配置。
func IsEnabled() bool {
	return modelIQCheckSetting.Enabled
}

// RawChecks 返回未解析的检查项 JSON，供注册表判断配置是否变化。
func RawChecks() string {
	if modelIQCheckSetting.Checks == "" {
		return "[]"
	}
	return modelIQCheckSetting.Checks
}

func clamp(value int, minValue int, maxValue int, fallback int) int {
	if value < minValue || value > maxValue {
		return fallback
	}
	return value
}
