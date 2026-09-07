package channelmonitor

// Monitor 是管理员配置的一条监控项：对一个分组下的一个模型做监控。
// Group 与 ChannelId 属于内部信息，公开接口不会返回。
type Monitor struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Group   string `json:"group"`
	Model   string `json:"model"`
	Icon    string `json:"icon,omitempty"`
	Enabled bool   `json:"enabled"`
	Sort    int    `json:"sort"`
}

// Sample 是一次被纳入监控的请求结果。TtftMs 只在流式首字已产生时有效。
type Sample struct {
	MonitorId string
	Status    int
	TtftMs    int
	HasTtft   bool
	ChannelId int
	Source    int
}

// BeatView 是状态条上的一格，ts 为采样窗口起始秒。
type BeatView struct {
	Ts     int64 `json:"ts"`
	Status int   `json:"status"`
	TtftMs int   `json:"ttft_ms"`
}

// MonitorView 是公开状态页的单卡片数据。
// 刻意不含 group 与 channel 信息，避免向未登录访客暴露内部路由拓扑。
type MonitorView struct {
	Id        string     `json:"id"`
	Name      string     `json:"name"`
	Model     string     `json:"model"`
	Icon      string     `json:"icon"`
	Status    string     `json:"status"`
	AvgTtftMs int        `json:"avg_ttft_ms"`
	PingMs    int        `json:"ping_ms"`
	Uptime    float64    `json:"uptime"`
	Beats     []BeatView `json:"beats"`
}

// StatusView 是 GET /api/channel-monitoring/status 的响应体。
// BeatLimit 同时决定状态条格数与「可用性」的统计样本数。
type StatusView struct {
	Enabled             bool          `json:"enabled"`
	SampleWindowSeconds int           `json:"sample_window_seconds"`
	BeatLimit           int           `json:"beat_limit"`
	Monitors            []MonitorView `json:"monitors"`
}

// 卡片状态，取最近一格 beat 的状态。
const (
	MonitorStatusUnknown  = "unknown"
	MonitorStatusUp       = "up"
	MonitorStatusDegraded = "degraded"
	MonitorStatusDown     = "down"
)
