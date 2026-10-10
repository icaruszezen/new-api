package modeliqcheck

const (
	// MaxChecks 同时限制调度开销与公开响应体积。
	MaxChecks = 50
	// MaxNameLength / MaxPromptLength 按字符计，与前端表单校验保持一致。
	MaxNameLength   = 64
	MaxIconLength   = 128
	MaxIdLength     = 64
	MaxPromptLength = 20000

	DefaultIntervalMinutes = 60
	MinIntervalMinutes     = 5
	MaxIntervalMinutes     = 10080

	// DefaultMaxTokens 是前端新建检查项的默认值；0 表示请求中不携带 max tokens 字段。
	DefaultMaxTokens = 8192
	MaxMaxTokens     = 200000

	// 单条结果落库前的上限：网页类输出通常几十 KB，留出足够余量的同时避免异常输出撑爆存储。
	MaxResultContentBytes   = 256 << 10
	MaxResultReasoningBytes = 128 << 10
	// MaxErrorMessageLength 按字符计。
	MaxErrorMessageLength = 2000
)

// 公开视图中的结果状态。
const (
	ResultStatusSuccess = "success"
	ResultStatusFailed  = "failed"
)

// Check 是管理员配置的一条智商检查：定期用 Prompt 调用 Group 分组下的 Model。
// 同一分组同一模型可以配置多条（不同提示词）。Group 属于内部信息，公开接口不会返回。
type Check struct {
	Id              string `json:"id"`
	Name            string `json:"name"`
	Group           string `json:"group"`
	Model           string `json:"model"`
	Prompt          string `json:"prompt"`
	IntervalMinutes int    `json:"interval_minutes"`
	// MaxTokens 为 0 时请求不携带 max tokens 字段，由上游使用默认值。
	MaxTokens int    `json:"max_tokens"`
	Icon      string `json:"icon,omitempty"`
	Enabled   bool   `json:"enabled"`
	Sort      int    `json:"sort"`
}

// Output 是从中转写给合成客户端的响应中提取出的模型输出。
type Output struct {
	Content            string
	Reasoning          string
	FinishReason       string
	ErrorMessage       string
	ContentTruncated   bool
	ReasoningTruncated bool
}

// StatusView 是 GET /api/iq-check/status 的响应体。
type StatusView struct {
	Enabled bool        `json:"enabled"`
	Checks  []CheckView `json:"checks"`
}

// CheckView 是公开页面的单卡片数据。
// 刻意不含 group 与 channel 信息，避免向未登录访客暴露内部路由拓扑；
// 历史只带元数据，完整输出通过 /api/iq-check/results 按需读取。
type CheckView struct {
	Id              string `json:"id"`
	Name            string `json:"name"`
	Model           string `json:"model"`
	Icon            string `json:"icon"`
	Prompt          string `json:"prompt"`
	IntervalMinutes int    `json:"interval_minutes"`
	Running         bool   `json:"running"`
	Queued          bool   `json:"queued"`
	// LastCheckedAt 为最新一条结果的完成时间，0 表示从未检查。
	LastCheckedAt int64 `json:"last_checked_at"`
	// NextCheckAt 为最近一次开始时间加检查间隔，0 表示从未开始。
	NextCheckAt int64             `json:"next_check_at"`
	History     []HistoryItemView `json:"history"`
}

// HistoryItemView 是历史下拉列表中的一项，按新到旧排列。
type HistoryItemView struct {
	Id               int64  `json:"id"`
	CheckedAt        int64  `json:"checked_at"`
	Status           string `json:"status"`
	DurationMs       int64  `json:"duration_ms"`
	CompletionTokens int    `json:"completion_tokens"`
}

// ResultView 是 GET /api/iq-check/results 返回的单条完整结果。
// Model / Prompt 是该次运行的快照；ErrorMessage 已脱敏。
type ResultView struct {
	Id                 int64  `json:"id"`
	CheckId            string `json:"check_id"`
	CheckedAt          int64  `json:"checked_at"`
	Status             string `json:"status"`
	Model              string `json:"model"`
	Prompt             string `json:"prompt"`
	Content            string `json:"content"`
	ContentTruncated   bool   `json:"content_truncated"`
	Reasoning          string `json:"reasoning"`
	ReasoningTruncated bool   `json:"reasoning_truncated"`
	FinishReason       string `json:"finish_reason"`
	ErrorMessage       string `json:"error_message"`
	DurationMs         int64  `json:"duration_ms"`
	PromptTokens       int    `json:"prompt_tokens"`
	CompletionTokens   int    `json:"completion_tokens"`
	ReasoningTokens    int    `json:"reasoning_tokens"`
}
