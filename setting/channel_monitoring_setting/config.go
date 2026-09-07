package channel_monitoring_setting

import "github.com/QuantumNous/new-api/setting/config"

// ChannelMonitoringSetting 控制对外渠道监控状态页的采集与展示行为。
// Monitors 为管理员配置的监控项 JSON 数组，其结构与校验在 pkg/channelmonitor 中定义。
type ChannelMonitoringSetting struct {
	Enabled              bool   `json:"enabled"`
	Monitors             string `json:"monitors"`
	ProbeIntervalSeconds int    `json:"probe_interval_seconds"`
	SampleWindowSeconds  int    `json:"sample_window_seconds"`
	SlowThresholdMs      int    `json:"slow_threshold_ms"`
	BeatLimit            int    `json:"beat_limit"`
	BeatRetentionHours   int    `json:"beat_retention_hours"`
	StatRetentionDays    int    `json:"stat_retention_days"`
}

const (
	EnabledOptionKey  = "channel_monitoring_setting.enabled"
	MonitorsOptionKey = "channel_monitoring_setting.monitors"

	DefaultProbeIntervalSeconds = 180
	MinProbeIntervalSeconds     = 30
	MaxProbeIntervalSeconds     = 3600

	DefaultSampleWindowSeconds = 20
	MinSampleWindowSeconds     = 5
	MaxSampleWindowSeconds     = 300

	DefaultSlowThresholdMs = 60000
	MinSlowThresholdMs     = 1000

	DefaultBeatLimit = 60
	MinBeatLimit     = 10
	MaxBeatLimit     = 200

	DefaultBeatRetentionHours = 24
	MinBeatRetentionHours     = 1

	DefaultStatRetentionDays = 7
	MinStatRetentionDays     = 1
)

var channelMonitoringSetting = ChannelMonitoringSetting{
	Enabled:              false,
	Monitors:             "[]",
	ProbeIntervalSeconds: DefaultProbeIntervalSeconds,
	SampleWindowSeconds:  DefaultSampleWindowSeconds,
	SlowThresholdMs:      DefaultSlowThresholdMs,
	BeatLimit:            DefaultBeatLimit,
	BeatRetentionHours:   DefaultBeatRetentionHours,
	StatRetentionDays:    DefaultStatRetentionDays,
}

func init() {
	config.GlobalConfig.Register("channel_monitoring_setting", &channelMonitoringSetting)
}

// GetSetting 返回一份已归一化的配置副本，调用方无需再做边界检查。
func GetSetting() ChannelMonitoringSetting {
	s := channelMonitoringSetting
	s.ProbeIntervalSeconds = clamp(s.ProbeIntervalSeconds, MinProbeIntervalSeconds, MaxProbeIntervalSeconds, DefaultProbeIntervalSeconds)
	s.SampleWindowSeconds = clamp(s.SampleWindowSeconds, MinSampleWindowSeconds, MaxSampleWindowSeconds, DefaultSampleWindowSeconds)
	if s.SlowThresholdMs < MinSlowThresholdMs {
		s.SlowThresholdMs = DefaultSlowThresholdMs
	}
	s.BeatLimit = clamp(s.BeatLimit, MinBeatLimit, MaxBeatLimit, DefaultBeatLimit)
	if s.BeatRetentionHours < MinBeatRetentionHours {
		s.BeatRetentionHours = DefaultBeatRetentionHours
	}
	if s.StatRetentionDays < MinStatRetentionDays {
		s.StatRetentionDays = DefaultStatRetentionDays
	}
	if s.Monitors == "" {
		s.Monitors = "[]"
	}
	return s
}

// IsEnabled 供公开接口与前端开关读取，避免调用方为了一个布尔值归一化整份配置。
func IsEnabled() bool {
	return channelMonitoringSetting.Enabled
}

// RawMonitors 返回未解析的监控项 JSON，供注册表判断配置是否变化。
func RawMonitors() string {
	if channelMonitoringSetting.Monitors == "" {
		return "[]"
	}
	return channelMonitoringSetting.Monitors
}

func clamp(value int, minValue int, maxValue int, fallback int) int {
	if value < minValue || value > maxValue {
		return fallback
	}
	return value
}
