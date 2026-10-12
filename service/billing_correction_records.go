package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// BillingCorrectionRecordLimit 是一次查询最多处理的匹配记录数。0 输出又扣了费的记录本身很少，
// 超过这个数通常是时间范围太大，此时让管理员缩小范围，避免一次把大量日志读进内存。
const BillingCorrectionRecordLimit = 5000

// 记录的退费状态。
const (
	// BillingCorrectionRecordStatusPending 已扣费，还没有退费。
	BillingCorrectionRecordStatusPending = "pending"
	// BillingCorrectionRecordStatusRefunded 已扣费，管理员已标记为另行退费。
	BillingCorrectionRecordStatusRefunded = "refunded"
	// BillingCorrectionRecordStatusCredited 已扣费，额度已经通过补费（手动或自动返还）退回用户钱包。
	BillingCorrectionRecordStatusCredited = "credited"
	// BillingCorrectionRecordStatusCorrected 规则在结算时已经免扣费，不需要退费。
	BillingCorrectionRecordStatusCorrected = "corrected"
)

// 记录与修正规则的匹配情况。
const (
	// BillingCorrectionRecordMatchRule 符合修正规则：输入超过阈值，并且按本地估算计费。
	BillingCorrectionRecordMatchRule = "rule"
	// BillingCorrectionRecordMatchZeroOutput 只是输出为 0 又扣了费，不满足修正规则的其他条件。
	BillingCorrectionRecordMatchZeroOutput = "zero_output"
)

var ErrBillingCorrectionRecordsTooMany = errors.New("too many billing correction records")

// BillingCorrectionRecord 是监控渠道上一条输出为 0、扣了费或已被规则修正的文本请求消费记录。
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
	Match            string `json:"match"`
	Status           string `json:"status"`
	RefundKind       string `json:"refund_kind,omitempty"`
	RefundedAt       int64  `json:"refunded_at,omitempty"`
	RefundOperatorId int    `json:"refund_operator_id,omitempty"`
	// UpstreamStatus 是上游 new-api 的核对结果；渠道没有配置上游核对、也没有核对过时为空。
	UpstreamRequestId string `json:"upstream_request_id,omitempty"`
	UpstreamStatus    string `json:"upstream_status,omitempty"`
	UpstreamQuota     int    `json:"upstream_quota,omitempty"`
	UpstreamMessage   string `json:"upstream_message,omitempty"`
	UpstreamCheckedAt int64  `json:"upstream_checked_at,omitempty"`

	tokenId          int
	upstreamAttempts int
}

// BillingCorrectionRecordStats 汇总筛选范围内的记录。已修正的记录没有扣费，只计条数。
type BillingCorrectionRecordStats struct {
	ChargedCount     int   `json:"charged_count"`
	ChargedQuota     int64 `json:"charged_quota"`
	RuleMatchedCount int   `json:"rule_matched_count"`
	RuleMatchedQuota int64 `json:"rule_matched_quota"`
	PendingCount     int   `json:"pending_count"`
	PendingQuota     int64 `json:"pending_quota"`
	RefundedCount    int   `json:"refunded_count"`
	RefundedQuota    int64 `json:"refunded_quota"`
	CreditedCount    int   `json:"credited_count"`
	CreditedQuota    int64 `json:"credited_quota"`
	CorrectedCount   int   `json:"corrected_count"`
	// UpstreamNotCharged* 统计还没退费、但上游核对确认没有扣费的记录。
	UpstreamNotChargedCount int   `json:"upstream_not_charged_count"`
	UpstreamNotChargedQuota int64 `json:"upstream_not_charged_quota"`
}

type BillingCorrectionRecordPage struct {
	Items    []*BillingCorrectionRecord   `json:"items"`
	Total    int                          `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
	Stats    BillingCorrectionRecordStats `json:"stats"`
}

// BillingCorrectionRecordFilter 在日志条件之外，还能按退费状态、是否符合修正规则和上游核对结果筛选。
// 字段为空表示不限。
type BillingCorrectionRecordFilter struct {
	model.BillingCorrectionLogFilter
	Status   string
	Match    string
	Upstream string
}

// BillingCorrectionRefundResult 是一次标记退费的结果。Skipped 是缺少请求 ID、无法标记的记录数。
type BillingCorrectionRefundResult struct {
	Marked  int   `json:"marked"`
	Quota   int64 `json:"quota"`
	Skipped int   `json:"skipped"`
}

// BillingCorrectionCreditResult 是一次补费的结果。Skipped 是缺少请求 ID 或已经退过费的记录数，
// Failed 是写库失败（例如用户已被删除）的记录数。
type BillingCorrectionCreditResult struct {
	Credited int   `json:"credited"`
	Quota    int64 `json:"quota"`
	Skipped  int   `json:"skipped"`
	Failed   int   `json:"failed"`
}

// QueryBillingCorrectionRecords 按筛选条件查询记录并分页。统计覆盖筛选范围内全部状态的记录，
// 不受 filter.Status 影响。
func QueryBillingCorrectionRecords(filter BillingCorrectionRecordFilter, page int, pageSize int) (*BillingCorrectionRecordPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = common.ItemsPerPage
	}
	records, _, err := findBillingCorrectionRecords(filter)
	if err != nil {
		return nil, err
	}

	result := &BillingCorrectionRecordPage{Page: page, PageSize: pageSize, Items: []*BillingCorrectionRecord{}}
	listed := make([]*BillingCorrectionRecord, 0, len(records))
	stats := &result.Stats
	for _, record := range records {
		quota := int64(record.Quota)
		switch record.Status {
		case BillingCorrectionRecordStatusPending:
			stats.PendingCount++
			stats.PendingQuota += quota
			if record.UpstreamStatus == BillingCorrectionUpstreamNotCharged {
				stats.UpstreamNotChargedCount++
				stats.UpstreamNotChargedQuota += quota
			}
		case BillingCorrectionRecordStatusRefunded:
			stats.RefundedCount++
			stats.RefundedQuota += quota
		case BillingCorrectionRecordStatusCredited:
			stats.RefundedCount++
			stats.RefundedQuota += quota
			stats.CreditedCount++
			stats.CreditedQuota += quota
		case BillingCorrectionRecordStatusCorrected:
			stats.CorrectedCount++
		}
		if record.Status != BillingCorrectionRecordStatusCorrected && record.Match == BillingCorrectionRecordMatchRule {
			stats.RuleMatchedCount++
			stats.RuleMatchedQuota += quota
		}
		if filter.Status == "" || record.Status == filter.Status {
			listed = append(listed, record)
		}
	}
	stats.ChargedCount = stats.PendingCount + stats.RefundedCount
	stats.ChargedQuota = stats.PendingQuota + stats.RefundedQuota

	result.Total = len(listed)
	start := (page - 1) * pageSize
	if start < len(listed) {
		result.Items = listed[start:min(start+pageSize, len(listed))]
	}
	return result, nil
}

// MarkBillingCorrectionRecordsRefunded 把筛选范围内已扣费、还没退费的记录标记为已退费。
// 只写标记，不改动任何余额，退费由管理员另行操作。按记录标记时把 filter.RequestIds 设为选中的请求。
func MarkBillingCorrectionRecordsRefunded(filter BillingCorrectionRecordFilter, operatorId int) (BillingCorrectionRefundResult, error) {
	var result BillingCorrectionRefundResult
	records, _, err := findBillingCorrectionRecords(filter)
	if err != nil {
		return result, err
	}
	refunds := make([]*model.BillingCorrectionRefund, 0, len(records))
	for _, record := range records {
		if record.Status != BillingCorrectionRecordStatusPending || (filter.Status != "" && filter.Status != record.Status) {
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

// CreditBillingCorrectionRecords 给筛选范围内已扣费、还没退费的记录补费：把这次扣的额度退回用户钱包，
// 恢复令牌额度，回减用户和渠道的已用额度，并给用户留一条退款日志。每条记录最多补一次。
func CreditBillingCorrectionRecords(filter BillingCorrectionRecordFilter, operatorId int) (BillingCorrectionCreditResult, error) {
	var result BillingCorrectionCreditResult
	records, _, err := findBillingCorrectionRecords(filter)
	if err != nil {
		return result, err
	}
	for _, record := range records {
		if record.Status != BillingCorrectionRecordStatusPending || (filter.Status != "" && filter.Status != record.Status) {
			continue
		}
		if record.RequestId == "" {
			result.Skipped++
			continue
		}
		credited, err := creditBillingCorrectionRecord(record, model.BillingCorrectionRefundKindCredit, operatorId)
		if err != nil {
			common.SysError(fmt.Sprintf("failed to credit billing correction record %s: %s", record.RequestId, err.Error()))
			result.Failed++
			continue
		}
		if !credited {
			result.Skipped++
			continue
		}
		result.Credited++
		result.Quota += int64(record.Quota)
	}
	return result, nil
}

// creditBillingCorrectionRecord 退回一条记录的扣费，返回 false 表示这条请求已经退过费。
// 资金来源一律退回钱包：订阅可能已经重置或到期，退回钱包对用户最直接。
func creditBillingCorrectionRecord(record *BillingCorrectionRecord, kind string, operatorId int) (bool, error) {
	credited, err := model.CreditBillingCorrectionRefund(&model.BillingCorrectionRefund{
		RequestId:    record.RequestId,
		UserId:       record.UserId,
		ChannelId:    record.ChannelId,
		Quota:        record.Quota,
		Kind:         kind,
		LogCreatedAt: record.CreatedAt,
		OperatorId:   operatorId,
	})
	if err != nil || !credited {
		return false, err
	}

	if record.tokenId > 0 {
		if token, err := model.GetTokenById(record.tokenId); err == nil {
			if err := model.IncreaseTokenQuota(token.Id, token.Key, record.Quota); err != nil {
				common.SysError(fmt.Sprintf("failed to restore token quota for billing correction record %s: %s", record.RequestId, err.Error()))
			}
		}
	}
	model.UpdateUserUsedQuota(record.UserId, -record.Quota)
	model.UpdateChannelUsedQuota(record.ChannelId, -record.Quota)

	content := fmt.Sprintf("计费修正补费：请求 %s 输出为 0，退还本次扣费", record.RequestId)
	if kind == model.BillingCorrectionRefundKindAuto {
		content = fmt.Sprintf("计费修正自动返还：上游未对请求 %s 扣费，退还本次扣费", record.RequestId)
	}
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:    record.UserId,
		LogType:   model.LogTypeRefund,
		Content:   content,
		ChannelId: record.ChannelId,
		ModelName: record.ModelName,
		Quota:     record.Quota,
		TokenId:   record.tokenId,
		Other: map[string]interface{}{
			"billing_correction_refund": kind,
			"refunded_request_id":       record.RequestId,
			"admin_info": map[string]interface{}{
				"operator_id": operatorId,
			},
		},
	})
	return true, nil
}

// billingCorrectionChannel 是一个启用了规则的监控渠道。upstream 只在配置好上游核对时才有。
type billingCorrectionChannel struct {
	rule     *model.BillingCorrectionRule
	name     string
	upstream *billingCorrectionUpstream
}

// loadBillingCorrectionChannels 读取全部已启用的规则，以渠道 ID 为键。
func loadBillingCorrectionChannels() (map[int]*billingCorrectionChannel, error) {
	rules, err := model.GetAllBillingCorrectionRules()
	if err != nil {
		return nil, err
	}
	if err := model.FillBillingCorrectionRuleChannelNames(rules); err != nil {
		return nil, err
	}
	channels := make(map[int]*billingCorrectionChannel, len(rules))
	for _, rule := range rules {
		if !rule.Enabled || rule.ChannelId <= 0 || rule.PromptTokensThreshold <= 0 {
			continue
		}
		channel := &billingCorrectionChannel{rule: rule, name: rule.ChannelName}
		if rule.HasUpstreamCheck() {
			if relayChannel, err := model.GetChannelById(rule.ChannelId, false); err == nil {
				channel.upstream = newBillingCorrectionUpstream(rule, relayChannel)
			}
		}
		channels[rule.ChannelId] = channel
	}
	return channels, nil
}

// findBillingCorrectionRecords 在启用了规则的渠道上找出输出为 0、扣了费或已被规则修正的文本请求记录，
// 标上与规则的匹配情况、退费状态和上游核对结果，并按 filter 的匹配和上游条件过滤（退费状态由调用方处理）。
// 规则的阈值同样用于规则启用前的历史记录。
func findBillingCorrectionRecords(filter BillingCorrectionRecordFilter) ([]*BillingCorrectionRecord, map[int]*billingCorrectionChannel, error) {
	channels, err := loadBillingCorrectionChannels()
	if err != nil {
		return nil, nil, err
	}
	thresholds := make(map[int]int, len(channels))
	for channelId, channel := range channels {
		thresholds[channelId] = channel.rule.PromptTokensThreshold
	}

	logFilter := filter.BillingCorrectionLogFilter
	logFilter.RuleMatchedOnly = filter.Match == BillingCorrectionRecordMatchRule
	logs, err := model.FindBillingCorrectionCandidateLogs(thresholds, logFilter, BillingCorrectionRecordLimit+1)
	if err != nil {
		return nil, nil, err
	}
	if len(logs) > BillingCorrectionRecordLimit {
		return nil, nil, ErrBillingCorrectionRecordsTooMany
	}

	records := make([]*BillingCorrectionRecord, 0, len(logs))
	requestIds := make([]string, 0, len(logs))
	for _, log := range logs {
		channel := channels[log.ChannelId]
		if channel == nil {
			continue
		}
		record := billingCorrectionRecordFromLog(log, channel.rule.PromptTokensThreshold)
		if record == nil || (filter.Match != "" && record.Match != filter.Match) {
			continue
		}
		record.ChannelName = channel.name
		records = append(records, record)
		if record.RequestId != "" && record.Status == BillingCorrectionRecordStatusPending {
			requestIds = append(requestIds, record.RequestId)
		}
	}

	refunds, err := model.GetBillingCorrectionRefundsByRequestIds(requestIds)
	if err != nil {
		return nil, nil, err
	}
	checks, err := model.GetBillingCorrectionUpstreamChecksByRequestIds(requestIds)
	if err != nil {
		return nil, nil, err
	}
	filtered := records[:0]
	for _, record := range records {
		if refund, ok := refunds[record.RequestId]; ok && record.Status == BillingCorrectionRecordStatusPending {
			record.Status = BillingCorrectionRecordStatusRefunded
			if refund.Kind != model.BillingCorrectionRefundKindMark {
				record.Status = BillingCorrectionRecordStatusCredited
			}
			record.RefundKind = refund.Kind
			record.RefundedAt = refund.CreatedAt
			record.RefundOperatorId = refund.OperatorId
		}
		if check, ok := checks[record.RequestId]; ok && record.Status != BillingCorrectionRecordStatusCorrected {
			record.UpstreamStatus = check.Status
			record.UpstreamQuota = check.UpstreamQuota
			record.UpstreamMessage = check.Message
			record.UpstreamCheckedAt = check.CheckedAt
			record.upstreamAttempts = check.Attempts
		} else if channels[record.ChannelId].upstream != nil && record.Status != BillingCorrectionRecordStatusCorrected {
			record.UpstreamStatus = BillingCorrectionUpstreamUnchecked
			if record.UpstreamRequestId == "" || record.RequestId == "" {
				record.UpstreamStatus = BillingCorrectionUpstreamUnavailable
			}
		}
		if filter.Upstream != "" && record.UpstreamStatus != filter.Upstream {
			continue
		}
		filtered = append(filtered, record)
	}
	return filtered, channels, nil
}

// billingCorrectionLogOther 是判定记录要用到的消费日志 other 字段。
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

// billingCorrectionRecordFromLog 把一条预筛出的消费日志转成记录，不是文本生成请求时返回 nil。
//   - 被规则修正过的记录（billing_corrected）算已修正，并且符合修正规则；
//   - 其余记录都是扣了费的待退费记录：输入超过阈值、并且按本地估算计费（上游没有返回计费信息）的
//     符合修正规则，否则只是输出为 0。
//
// 渠道和 0 输出已由查询条件保证。
func billingCorrectionRecordFromLog(log *model.Log, threshold int) *BillingCorrectionRecord {
	var other billingCorrectionLogOther
	if err := common.UnmarshalJsonStr(log.Other, &other); err != nil {
		return nil
	}
	// 较早的日志没有 request_path，无法区分请求类型，按文本请求处理。
	if !other.BillingCorrected && other.RequestPath != "" && !isTextGenerationRequestPath(other.RequestPath) {
		return nil
	}

	record := &BillingCorrectionRecord{
		RequestId:             log.RequestId,
		CreatedAt:             log.CreatedAt,
		UserId:                log.UserId,
		Username:              log.Username,
		TokenName:             log.TokenName,
		ModelName:             log.ModelName,
		ChannelId:             log.ChannelId,
		PromptTokens:          log.PromptTokens,
		PromptTokensThreshold: threshold,
		Quota:                 log.Quota,
		UseTime:               log.UseTime,
		IsStream:              log.IsStream,
		UpstreamRequestId:     log.UpstreamRequestId,
		tokenId:               log.TokenId,
	}

	if other.BillingCorrected {
		record.Status = BillingCorrectionRecordStatusCorrected
		record.Match = BillingCorrectionRecordMatchRule
		if other.AdminInfo.BillingCorrection != nil {
			record.OriginalQuota = other.AdminInfo.BillingCorrection.OriginalQuota
		}
		return record
	}
	if log.Quota <= 0 {
		return nil
	}
	record.Status = BillingCorrectionRecordStatusPending
	record.Match = BillingCorrectionRecordMatchZeroOutput
	if log.PromptTokens > threshold && billedOnLocalEstimate(other.AdminInfo.UsageBillingPath, other.AdminInfo.LocalCountTokens) {
		record.Match = BillingCorrectionRecordMatchRule
	}
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

// CheckBillingCorrectionRecordsUpstream 到上游 new-api 复查记录是否真的扣了费，只记录结果，不退费。
// 按选中的记录复查时（filter.RequestIds 非空）每条能核对的都查；复查当前筛选下的全部记录时，
// 只查还没有确定结果的（未核对、上游没找到、核对失败），一次最多 BillingCorrectionUpstreamCheckLimit 条，
// Remaining 是本次没轮到的条数。
func CheckBillingCorrectionRecordsUpstream(ctx context.Context, filter BillingCorrectionRecordFilter) (BillingCorrectionUpstreamCheckResult, error) {
	var result BillingCorrectionUpstreamCheckResult
	records, channels, err := findBillingCorrectionRecords(filter)
	if err != nil {
		return result, err
	}
	selected := len(filter.RequestIds) > 0
	candidates := make([]*BillingCorrectionRecord, 0, min(len(records), BillingCorrectionUpstreamCheckLimit))
	for _, record := range records {
		if record.Status == BillingCorrectionRecordStatusCorrected || (filter.Status != "" && filter.Status != record.Status) {
			continue
		}
		checkable := record.RequestId != "" && record.UpstreamRequestId != "" && channels[record.ChannelId].upstream != nil
		if !checkable {
			result.Skipped++
			continue
		}
		if !selected && record.UpstreamStatus != BillingCorrectionUpstreamUnchecked &&
			record.UpstreamStatus != BillingCorrectionUpstreamNotFound && record.UpstreamStatus != BillingCorrectionUpstreamFailed {
			continue
		}
		if len(candidates) >= BillingCorrectionUpstreamCheckLimit {
			result.Remaining++
			continue
		}
		candidates = append(candidates, record)
	}

	checkBillingCorrectionRecordsUpstream(ctx, candidates, channels)
	for _, record := range candidates {
		result.count(record.UpstreamStatus)
	}
	return result, nil
}
