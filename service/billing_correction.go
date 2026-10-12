package service

import (
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/gin-gonic/gin"
)

// billingCorrectionLogContent 写进消费日志 content，用户在使用记录里能直接看到。
const billingCorrectionLogContent = "超大上下文错误计费修正：上游未返回计费信息且无输出，本次不计费"

// enabledBillingCorrectionRule 是缓存里一条已启用的渠道规则。
type enabledBillingCorrectionRule struct {
	id                    int
	promptTokensThreshold int
	upstreamCheck         bool
}

// billingCorrectionCache 以渠道 ID 为键，只收录已启用的规则，结算热路径无需查库。
var billingCorrectionCache atomic.Pointer[map[int]enabledBillingCorrectionRule]

// ReloadBillingCorrectionRules 重建内存规则缓存。
func ReloadBillingCorrectionRules() error {
	rules, err := model.GetAllBillingCorrectionRules()
	if err != nil {
		return err
	}
	byChannel := make(map[int]enabledBillingCorrectionRule, len(rules))
	for _, rule := range rules {
		if !rule.Enabled || rule.ChannelId <= 0 || rule.PromptTokensThreshold <= 0 {
			continue
		}
		byChannel[rule.ChannelId] = enabledBillingCorrectionRule{
			id:                    rule.Id,
			promptTokensThreshold: rule.PromptTokensThreshold,
			upstreamCheck:         rule.HasUpstreamCheck(),
		}
	}
	billingCorrectionCache.Store(&byChannel)
	return nil
}

// HasBillingCorrectionUpstreamChecks 判断是否有已启用的规则配置了上游核对，决定后台自动核对要不要运行。
func HasBillingCorrectionUpstreamChecks() bool {
	rules := billingCorrectionCache.Load()
	if rules == nil {
		return false
	}
	for _, rule := range *rules {
		if rule.upstreamCheck {
			return true
		}
	}
	return false
}

// SyncBillingCorrectionRules 周期性重载规则，让非 master 节点也能拿到最新配置。
func SyncBillingCorrectionRules(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		if err := ReloadBillingCorrectionRules(); err != nil {
			common.SysError("failed to sync billing correction rules: " + err.Error())
		}
	}
}

// billingCorrection 记录一次命中的修正，原样写入消费日志的 admin_info.billing_correction。
type billingCorrection struct {
	RuleId                int `json:"rule_id"`
	ChannelId             int `json:"channel_id"`
	PromptTokensThreshold int `json:"prompt_tokens_threshold"`
	PromptTokens          int `json:"prompt_tokens"`
	OriginalQuota         int `json:"original_quota"`
}

// matchBillingCorrection 判断一次文本请求是否属于超大上下文错误计费，命中时返回修正记录。
// 典型场景是上游只发心跳、没有任何输出也没有返回计费信息就断流，new-api 只能按本地
// 估算的输入 token 计费。以下条件全部满足才算命中：
//   - 最终成功的渠道配置了已启用的规则；
//   - 请求是文本生成（Chat/Completions、Claude Messages、Gemini 对话、Responses），
//     embedding、moderation 等天然没有输出的请求不参与；
//   - 上游没有返回计费信息：输入 token 来自本地估算，而不是上游的 usage；
//   - 输入 token 严格大于阈值，且输出 token 为 0；
//   - 按原规则本次确实要扣费。
func matchBillingCorrection(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, originUsage *dto.Usage, summary textQuotaSummary) *billingCorrection {
	rules := billingCorrectionCache.Load()
	if rules == nil || len(*rules) == 0 {
		return nil
	}
	channelId := relayInfo.GetChannelID()
	rule, ok := (*rules)[channelId]
	if !ok {
		return nil
	}
	if summary.Quota <= 0 || summary.CompletionTokens != 0 || summary.PromptTokens <= rule.promptTokensThreshold {
		return nil
	}

	switch relayInfo.RelayFormat {
	case types.RelayFormatClaude, types.RelayFormatOpenAIResponses, types.RelayFormatOpenAIResponsesCompaction:
	case types.RelayFormatOpenAI:
		if relayInfo.RelayMode != relayconstant.RelayModeChatCompletions && relayInfo.RelayMode != relayconstant.RelayModeCompletions {
			return nil
		}
	case types.RelayFormatGemini:
		if _, isChat := relayInfo.Request.(*dto.GeminiChatRequest); !isChat {
			return nil
		}
	default:
		return nil
	}

	// 本地估算标记会在重试间残留，Claude 的 message_start 已给出真实输入时也会因为补估输出而置位，
	// 所以还要求计费用的输入 token 恰好等于请求前的本地估算值，确认它不是上游给的数。
	locallyEstimated := originUsage == nil ||
		common.GetContextKeyBool(ctx, constant.ContextKeyLocalCountTokens) ||
		(originUsage.BillingUsage != nil && originUsage.BillingUsage.Estimated)
	if !locallyEstimated || summary.PromptTokens != relayInfo.GetEstimatePromptTokens() {
		return nil
	}

	return &billingCorrection{
		RuleId:                rule.id,
		ChannelId:             channelId,
		PromptTokensThreshold: rule.promptTokensThreshold,
		PromptTokens:          summary.PromptTokens,
		OriginalQuota:         summary.Quota,
	}
}
