package service

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// BillingCorrectionRecordLimit 是一次查询最多处理的匹配记录数。符合错误计费条件的记录本身很少，
// 超过这个数通常是时间范围太大，此时让管理员缩小范围，避免一次把大量日志读进内存。
const BillingCorrectionRecordLimit = 5000

// 错误计费记录的状态。
const (
	// BillingCorrectionRecordStatusPending 已扣费，还没有标记退费。
	BillingCorrectionRecordStatusPending = "pending"
	// BillingCorrectionRecordStatusRefunded 已扣费，管理员已标记为另行退费。
	BillingCorrectionRecordStatusRefunded = "refunded"
	// BillingCorrectionRecordStatusCorrected 规则在结算时已经免扣费，不需要退费。
	BillingCorrectionRecordStatusCorrected = "corrected"
)

var ErrBillingCorrectionRecordsTooMany = errors.New("too many billing correction records")

// BillingCorrectionRecord 是一条符合超大上下文错误计费条件的消费记录。
type BillingCorrectionRecord struct {
	RequestId             string `json:"request_id"`
	CreatedAt             int64  `json:"created_at"`
	UserId                int    `json:"user_id"`
	Username              string `json:"username"`
	TokenName             string `json:"token_name"`
	ModelName             string `json:"model_name"`
	ChannelId             int    `json:"channel_id"`
	ChannelName           string `json:"channel_name"`
	PromptTokens          int    `json:"prompt_tokens"`
	PromptTokensThreshold int    `json:"prompt_tokens_threshold"`
	Quota                 int    `json:"quota"`
	// OriginalQuota 是已修正记录按原规则本应扣的额度，只有已修正的记录才有。
	OriginalQuota    int    `json:"original_quota,omitempty"`
	UseTime          int    `json:"use_time"`
	IsStream         bool   `json:"is_stream"`
	Status           string `json:"status"`
	RefundedAt       int64  `json:"refunded_at,omitempty"`
	RefundOperatorId int    `json:"refund_operator_id,omitempty"`
}

// BillingCorrectionRecordStats 汇总筛选范围内的错误计费记录。已修正的记录没有扣费，只计条数。
type BillingCorrectionRecordStats struct {
	ChargedCount   int   `json:"charged_count"`
	ChargedQuota   int64 `json:"charged_quota"`
	PendingCount   int   `json:"pending_count"`
	PendingQuota   int64 `json:"pending_quota"`
	RefundedCount  int   `json:"refunded_count"`
	RefundedQuota  int64 `json:"refunded_quota"`
	CorrectedCount int   `json:"corrected_count"`
}

type BillingCorrectionRecordPage struct {
	Items    []*BillingCorrectionRecord   `json:"items"`
	Total    int                          `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
	Stats    BillingCorrectionRecordStats `json:"stats"`
}

// BillingCorrectionRefundResult 是一次标记退费的结果。Skipped 是缺少请求 ID、无法标记的记录数。
type BillingCorrectionRefundResult struct {
	Marked  int   `json:"marked"`
	Quota   int64 `json:"quota"`
	Skipped int   `json:"skipped"`
}

// QueryBillingCorrectionRecords 按筛选条件查询错误计费记录并分页。统计覆盖筛选范围内的全部记录，
// 不受 status 影响；status 为空时列出全部状态。
func QueryBillingCorrectionRecords(filter model.BillingCorrectionLogFilter, status string, page int, pageSize int) (*BillingCorrectionRecordPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = common.ItemsPerPage
	}
	records, err := findBillingCorrectionRecords(filter)
	if err != nil {
		return nil, err
	}

	result := &BillingCorrectionRecordPage{Page: page, PageSize: pageSize, Items: []*BillingCorrectionRecord{}}
	listed := make([]*BillingCorrectionRecord, 0, len(records))
	for _, record := range records {
		switch record.Status {
		case BillingCorrectionRecordStatusPending:
			result.Stats.PendingCount++
			result.Stats.PendingQuota += int64(record.Quota)
		case BillingCorrectionRecordStatusRefunded:
			result.Stats.RefundedCount++
			result.Stats.RefundedQuota += int64(record.Quota)
		case BillingCorrectionRecordStatusCorrected:
			result.Stats.CorrectedCount++
		}
		if status == "" || record.Status == status {
			listed = append(listed, record)
		}
	}
	result.Stats.ChargedCount = result.Stats.PendingCount + result.Stats.RefundedCount
	result.Stats.ChargedQuota = result.Stats.PendingQuota + result.Stats.RefundedQuota

	result.Total = len(listed)
	start := (page - 1) * pageSize
	if start < len(listed) {
		result.Items = listed[start:min(start+pageSize, len(listed))]
	}
	return result, nil
}

// MarkBillingCorrectionRecordsRefunded 把筛选范围内已扣费、未标记的错误计费记录标记为已退费。
// 只写标记，不改动任何余额，退费由管理员另行操作。按记录标记时把 filter.RequestIds 设为选中的请求。
func MarkBillingCorrectionRecordsRefunded(filter model.BillingCorrectionLogFilter, operatorId int) (BillingCorrectionRefundResult, error) {
	var result BillingCorrectionRefundResult
	records, err := findBillingCorrectionRecords(filter)
	if err != nil {
		return result, err
	}
	refunds := make([]*model.BillingCorrectionRefund, 0, len(records))
	for _, record := range records {
		if record.Status != BillingCorrectionRecordStatusPending {
			continue
		}
		if record.RequestId == "" {
			result.Skipped++
			continue
		}
		refunds = append(refunds, &model.BillingCorrectionRefund{
			RequestId:    record.RequestId,
			UserId:       record.UserId,
			ChannelId:    record.ChannelId,
			Quota:        record.Quota,
			LogCreatedAt: record.CreatedAt,
			OperatorId:   operatorId,
		})
		result.Marked++
		result.Quota += int64(record.Quota)
	}
	if err := model.CreateBillingCorrectionRefunds(refunds); err != nil {
		return BillingCorrectionRefundResult{}, err
	}
	return result, nil
}

// findBillingCorrectionRecords 用当前已启用的规则在消费日志里找出错误计费记录，并标上退费状态。
// 规则的阈值同样用于规则启用前的历史记录。
func findBillingCorrectionRecords(filter model.BillingCorrectionLogFilter) ([]*BillingCorrectionRecord, error) {
	rules, err := model.GetAllBillingCorrectionRules()
	if err != nil {
		return nil, err
	}
	if err := model.FillBillingCorrectionRuleChannelNames(rules); err != nil {
		return nil, err
	}
	thresholds := make(map[int]int, len(rules))
	channelNames := make(map[int]string, len(rules))
	for _, rule := range rules {
		if !rule.Enabled || rule.ChannelId <= 0 || rule.PromptTokensThreshold <= 0 {
			continue
		}
		thresholds[rule.ChannelId] = rule.PromptTokensThreshold
		channelNames[rule.ChannelId] = rule.ChannelName
	}

	logs, err := model.FindBillingCorrectionCandidateLogs(thresholds, filter, BillingCorrectionRecordLimit+1)
	if err != nil {
		return nil, err
	}
	if len(logs) > BillingCorrectionRecordLimit {
		return nil, ErrBillingCorrectionRecordsTooMany
	}

	records := make([]*BillingCorrectionRecord, 0, len(logs))
	pendingRequestIds := make([]string, 0, len(logs))
	for _, log := range logs {
		record := billingCorrectionRecordFromLog(log)
		if record == nil {
			continue
		}
		record.ChannelName = channelNames[record.ChannelId]
		record.PromptTokensThreshold = thresholds[record.ChannelId]
		records = append(records, record)
		if record.Status == BillingCorrectionRecordStatusPending && record.RequestId != "" {
			pendingRequestIds = append(pendingRequestIds, record.RequestId)
		}
	}

	refunds, err := model.GetBillingCorrectionRefundsByRequestIds(pendingRequestIds)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		refund, ok := refunds[record.RequestId]
		if !ok || record.Status != BillingCorrectionRecordStatusPending {
			continue
		}
		record.Status = BillingCorrectionRecordStatusRefunded
		record.RefundedAt = refund.CreatedAt
		record.RefundOperatorId = refund.OperatorId
	}
	return records, nil
}

// billingCorrectionLogOther 是判定错误计费要用到的消费日志 other 字段。
type billingCorrectionLogOther struct {
	BillingCorrected bool   `json:"billing_corrected"`
	RequestPath      string `json:"request_path"`
	AdminInfo        struct {
		LocalCountTokens  bool   `json:"local_count_tokens"`
		UsageBillingPath  string `json:"usage_billing_path"`
		BillingCorrection *struct {
			OriginalQuota int `json:"original_quota"`
		} `json:"billing_correction"`
	} `json:"admin_info"`
}

// billingCorrectionRecordFromLog 判定一条预筛出的消费日志是不是错误计费记录，不是时返回 nil。
//   - 被规则修正过的记录（billing_corrected）算已修正；
//   - 否则必须是文本生成请求、按本地估算计费（上游没有返回计费信息），并且确实扣了费。
//
// 渠道、阈值和 0 输出已由查询条件保证。
func billingCorrectionRecordFromLog(log *model.Log) *BillingCorrectionRecord {
	var other billingCorrectionLogOther
	if err := common.UnmarshalJsonStr(log.Other, &other); err != nil {
		return nil
	}

	record := &BillingCorrectionRecord{
		RequestId:    log.RequestId,
		CreatedAt:    log.CreatedAt,
		UserId:       log.UserId,
		Username:     log.Username,
		TokenName:    log.TokenName,
		ModelName:    log.ModelName,
		ChannelId:    log.ChannelId,
		PromptTokens: log.PromptTokens,
		Quota:        log.Quota,
		UseTime:      log.UseTime,
		IsStream:     log.IsStream,
	}

	if other.BillingCorrected {
		record.Status = BillingCorrectionRecordStatusCorrected
		if other.AdminInfo.BillingCorrection != nil {
			record.OriginalQuota = other.AdminInfo.BillingCorrection.OriginalQuota
		}
		return record
	}

	if log.Quota <= 0 || !billedOnLocalEstimate(other.AdminInfo.UsageBillingPath, other.AdminInfo.LocalCountTokens) {
		return nil
	}
	// 较早的日志没有 request_path，无法区分请求类型，按文本请求处理。
	if other.RequestPath != "" && !isTextGenerationRequestPath(other.RequestPath) {
		return nil
	}
	record.Status = BillingCorrectionRecordStatusPending
	return record
}

// billedOnLocalEstimate 判断一次扣费是不是按本地估算的 token 算的，即上游没有返回计费信息。
// usage_billing_path 记录了结算实际用的 usage 来源：local 是本地估算，*-estimated 是从本地估算
// 补出来的计费 usage，其余都是上游给的。Claude 在 message_start 给出真实输入后断流时，
// 也会因为补估输出带上 local_count_tokens，但计费路径是上游的，不算错误计费，与实时修正一致。
// 较早的日志没有 usage_billing_path，只能看 local_count_tokens。
func billedOnLocalEstimate(usageBillingPath string, localCountTokens bool) bool {
	switch usageBillingPath {
	case "":
		return localCountTokens
	case usageBillingPathLocal, usageBillingPathOpenAIEstimated, usageBillingPathAnthropicEstimated, usageBillingPathGeminiEstimated:
		return true
	}
	return false
}

// isTextGenerationRequestPath 判断请求路径是否是会产生输出的文本生成接口，与实时修正的范围一致：
// Chat/Completions、Claude Messages、Responses、Gemini 对话。embedding、moderation 等天然没有输出，不算。
func isTextGenerationRequestPath(path string) bool {
	return strings.HasSuffix(path, "/completions") ||
		strings.HasSuffix(path, "/messages") ||
		strings.Contains(path, "/responses") ||
		strings.Contains(path, "generateContent")
}
