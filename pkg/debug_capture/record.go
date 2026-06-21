package debug_capture

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// 捕获原因。错误捕获功能据此区分记录类型；最近请求功能统一使用 ReasonRecent。
// 计费异常原因直接复用 relaycommon 中的常量，保证与 RelayInfo.DebugBillingIssue 写入值一致。
const (
	ReasonError             = "error"
	ReasonBillingNoUsage    = relaycommon.DebugBillingIssueNoUsage
	ReasonBillingZeroTokens = relaycommon.DebugBillingIssueZeroTokens
	ReasonRecent            = "recent"
)

// CaptureRecord 是一条完整的请求捕获记录，包含下游请求体与上游响应体。
// 该结构会直接序列化返回给前端，仅 Root/Super Admin 可见。
type CaptureRecord struct {
	ID        uint64 `json:"id"`
	Timestamp int64  `json:"timestamp"` // 毫秒
	RequestID string `json:"request_id"`
	Reason    string `json:"reason"`

	Method     string `json:"method"`
	Path       string `json:"path"`
	Model      string `json:"model"`
	ChannelID  int    `json:"channel_id"`
	UserID     int    `json:"user_id"`
	IsStream   bool   `json:"is_stream"`
	RetryIndex int    `json:"retry_index"`

	ErrorCode      string `json:"error_code,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	UpstreamStatus int    `json:"upstream_status,omitempty"`

	DownstreamRequest string `json:"downstream_request"`
	UpstreamResponse  string `json:"upstream_response"`

	// UpstreamCaptureIncomplete 为 true 表示该渠道走 SDK / 非 HTTP 协议（如 AWS SDK、
	// xunfei websocket、coze 非流式），未能捕获上游响应体。
	UpstreamCaptureIncomplete bool `json:"upstream_capture_incomplete"`
	// Truncated 表示上游响应体因超过容量上限被截断。
	Truncated bool `json:"truncated"`
}

// buildRecord 从请求上下文、RelayInfo 与 relay 错误构建共享记录（不含 Reason 与 ID，
// 由各 store 在写入时分别赋值）。relayInfo 可能为 nil（早期失败），此时仅尽力捕获下游请求体。
func buildRecord(c *gin.Context, info *relaycommon.RelayInfo, relayErr *types.NewAPIError) CaptureRecord {
	rec := CaptureRecord{
		Timestamp: time.Now().UnixMilli(),
	}

	if c != nil {
		rec.RequestID = c.GetString(common.RequestIdKey)
		if c.Request != nil {
			rec.Method = c.Request.Method
			if c.Request.URL != nil {
				rec.Path = c.Request.URL.Path
			}
		}
		rec.DownstreamRequest = readDownstreamBody(c)
	}

	if info != nil {
		rec.Model = info.OriginModelName
		rec.UserID = info.UserId
		rec.IsStream = info.IsStream
		rec.RetryIndex = info.RetryIndex
		if rec.RequestID == "" {
			rec.RequestID = info.RequestId
		}
		// ChannelId 由内嵌的 *ChannelMeta 提升而来；早期失败时 ChannelMeta 可能为 nil。
		channelID := 0
		if info.ChannelMeta != nil {
			channelID = info.ChannelId
		}
		rec.ChannelID = channelID
		if info.UpstreamCapture != nil {
			rec.UpstreamResponse = info.UpstreamCapture.String()
			rec.Truncated = info.UpstreamCapture.Truncated()
		}
		// 选中了渠道但没有 HTTP 层捕获缓冲，说明该渠道未经过 doRequest（SDK / 非 HTTP），
		// 或在抵达上游之前就失败，上游响应体不可用。
		rec.UpstreamCaptureIncomplete = channelID != 0 && info.UpstreamCapture == nil
	}

	if relayErr != nil {
		rec.ErrorCode = string(relayErr.GetErrorCode())
		rec.ErrorMessage = relayErr.MaskSensitiveError()
		rec.UpstreamStatus = relayErr.StatusCode
	}

	return rec
}

// readDownstreamBody 读取下游（客户端 → 网关）完整请求体。请求体在 BodyStorage 创建时
// 已受 MAX_REQUEST_BODY_MB 限制，因此无需再次截断。读取失败时返回空串。
func readDownstreamBody(c *gin.Context) string {
	bs, err := common.GetBodyStorage(c)
	if err != nil || bs == nil {
		return ""
	}
	data, err := bs.Bytes()
	if err != nil {
		return ""
	}
	return string(data)
}

// errorReason 返回错误捕获功能应记录的原因；若该请求既无错误也无计费异常则返回空串。
func errorReason(info *relaycommon.RelayInfo, relayErr *types.NewAPIError) string {
	if relayErr != nil {
		return ReasonError
	}
	if info != nil && info.DebugBillingIssue != "" {
		return info.DebugBillingIssue
	}
	return ""
}
