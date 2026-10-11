package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	correctedChannelId       = 7
	disabledRuleChannelId    = 8
	billingCorrectionTrigger = 1_000_000
)

func seedBillingCorrectionRules(t *testing.T, rules ...*model.BillingCorrectionRule) {
	t.Helper()
	require.NoError(t, model.DB.Where("1 = 1").Delete(&model.BillingCorrectionRule{}).Error)
	for _, rule := range rules {
		require.NoError(t, rule.Insert())
	}
	require.NoError(t, ReloadBillingCorrectionRules())
	t.Cleanup(func() {
		model.DB.Where("1 = 1").Delete(&model.BillingCorrectionRule{})
		_ = ReloadBillingCorrectionRules()
	})
}

func newBillingCorrectionRelayInfo(channelId int, format types.RelayFormat, relayMode int, estimatedPromptTokens int) *relaycommon.RelayInfo {
	info := &relaycommon.RelayInfo{
		RelayFormat: format,
		RelayMode:   relayMode,
		ChannelMeta: &relaycommon.ChannelMeta{ChannelId: channelId},
	}
	info.SetEstimatePromptTokens(estimatedPromptTokens)
	return info
}

func TestMatchBillingCorrection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	seedBillingCorrectionRules(t,
		&model.BillingCorrectionRule{ChannelId: correctedChannelId, PromptTokensThreshold: billingCorrectionTrigger, Enabled: true},
		&model.BillingCorrectionRule{ChannelId: disabledRuleChannelId, PromptTokensThreshold: billingCorrectionTrigger, Enabled: false},
	)
	const estimate = 3_200_000

	tests := []struct {
		name             string
		info             *relaycommon.RelayInfo
		localCountTokens bool
		originUsage      *dto.Usage
		summary          textQuotaSummary
		wantCorrected    bool
	}{
		{
			name:             "chat stream estimated input above threshold with no output is corrected",
			info:             newBillingCorrectionRelayInfo(correctedChannelId, types.RelayFormatOpenAI, relayconstant.RelayModeChatCompletions, estimate),
			localCountTokens: true,
			originUsage:      &dto.Usage{PromptTokens: estimate},
			summary:          textQuotaSummary{PromptTokens: estimate, Quota: 1_600_000},
			wantCorrected:    true,
		},
		{
			name:          "missing usage on claude messages is corrected",
			info:          newBillingCorrectionRelayInfo(correctedChannelId, types.RelayFormatClaude, relayconstant.RelayModeUnknown, estimate),
			originUsage:   nil,
			summary:       textQuotaSummary{PromptTokens: estimate, Quota: 1_600_000},
			wantCorrected: true,
		},
		{
			name: "gemini estimated billing usage is corrected without the local count flag",
			info: func() *relaycommon.RelayInfo {
				info := newBillingCorrectionRelayInfo(correctedChannelId, types.RelayFormatGemini, relayconstant.RelayModeGemini, estimate)
				info.Request = &dto.GeminiChatRequest{}
				return info
			}(),
			originUsage:   &dto.Usage{PromptTokens: estimate, BillingUsage: &dto.BillingUsage{Estimated: true}},
			summary:       textQuotaSummary{PromptTokens: estimate, Quota: 1_600_000},
			wantCorrected: true,
		},
		{
			name:             "responses request is corrected",
			info:             newBillingCorrectionRelayInfo(correctedChannelId, types.RelayFormatOpenAIResponses, relayconstant.RelayModeResponses, estimate),
			localCountTokens: true,
			originUsage:      &dto.Usage{PromptTokens: estimate},
			summary:          textQuotaSummary{PromptTokens: estimate, Quota: 1_600_000},
			wantCorrected:    true,
		},
		{
			name:             "input exactly at the threshold is still charged",
			info:             newBillingCorrectionRelayInfo(correctedChannelId, types.RelayFormatOpenAI, relayconstant.RelayModeChatCompletions, billingCorrectionTrigger),
			localCountTokens: true,
			originUsage:      &dto.Usage{PromptTokens: billingCorrectionTrigger},
			summary:          textQuotaSummary{PromptTokens: billingCorrectionTrigger, Quota: 500_000},
		},
		{
			name:             "any output keeps the charge",
			info:             newBillingCorrectionRelayInfo(correctedChannelId, types.RelayFormatOpenAI, relayconstant.RelayModeChatCompletions, estimate),
			localCountTokens: true,
			originUsage:      &dto.Usage{PromptTokens: estimate, CompletionTokens: 1},
			summary:          textQuotaSummary{PromptTokens: estimate, CompletionTokens: 1, Quota: 1_600_001},
		},
		{
			name:          "usage reported by the upstream keeps the charge",
			info:          newBillingCorrectionRelayInfo(correctedChannelId, types.RelayFormatOpenAI, relayconstant.RelayModeChatCompletions, estimate),
			originUsage:   &dto.Usage{PromptTokens: estimate},
			summary:       textQuotaSummary{PromptTokens: estimate, Quota: 1_600_000},
			wantCorrected: false,
		},
		{
			name:             "claude message_start input with only the output estimated keeps the charge",
			info:             newBillingCorrectionRelayInfo(correctedChannelId, types.RelayFormatClaude, relayconstant.RelayModeUnknown, estimate),
			localCountTokens: true,
			originUsage:      &dto.Usage{PromptTokens: 2_900_000},
			summary:          textQuotaSummary{PromptTokens: 2_900_000, Quota: 1_450_000},
		},
		{
			name:             "disabled rule keeps the charge",
			info:             newBillingCorrectionRelayInfo(disabledRuleChannelId, types.RelayFormatOpenAI, relayconstant.RelayModeChatCompletions, estimate),
			localCountTokens: true,
			originUsage:      &dto.Usage{PromptTokens: estimate},
			summary:          textQuotaSummary{PromptTokens: estimate, Quota: 1_600_000},
		},
		{
			name:             "channel without a rule keeps the charge",
			info:             newBillingCorrectionRelayInfo(99, types.RelayFormatOpenAI, relayconstant.RelayModeChatCompletions, estimate),
			localCountTokens: true,
			originUsage:      &dto.Usage{PromptTokens: estimate},
			summary:          textQuotaSummary{PromptTokens: estimate, Quota: 1_600_000},
		},
		{
			name:             "embedding requests are never corrected",
			info:             newBillingCorrectionRelayInfo(correctedChannelId, types.RelayFormatEmbedding, relayconstant.RelayModeEmbeddings, estimate),
			localCountTokens: true,
			originUsage:      &dto.Usage{PromptTokens: estimate},
			summary:          textQuotaSummary{PromptTokens: estimate, Quota: 1_600_000},
		},
		{
			name:             "moderation requests are never corrected",
			info:             newBillingCorrectionRelayInfo(correctedChannelId, types.RelayFormatOpenAI, relayconstant.RelayModeModerations, estimate),
			localCountTokens: true,
			originUsage:      &dto.Usage{PromptTokens: estimate},
			summary:          textQuotaSummary{PromptTokens: estimate, Quota: 1_600_000},
		},
		{
			name: "gemini embedding requests are never corrected",
			info: func() *relaycommon.RelayInfo {
				info := newBillingCorrectionRelayInfo(correctedChannelId, types.RelayFormatGemini, relayconstant.RelayModeGemini, estimate)
				info.Request = &dto.GeminiEmbeddingRequest{}
				return info
			}(),
			localCountTokens: true,
			originUsage:      &dto.Usage{PromptTokens: estimate},
			summary:          textQuotaSummary{PromptTokens: estimate, Quota: 1_600_000},
		},
		{
			name:             "free requests have nothing to correct",
			info:             newBillingCorrectionRelayInfo(correctedChannelId, types.RelayFormatOpenAI, relayconstant.RelayModeChatCompletions, estimate),
			localCountTokens: true,
			originUsage:      &dto.Usage{PromptTokens: estimate},
			summary:          textQuotaSummary{PromptTokens: estimate, Quota: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			if tt.localCountTokens {
				common.SetContextKey(c, constant.ContextKeyLocalCountTokens, true)
			}

			correction := matchBillingCorrection(c, tt.info, tt.originUsage, tt.summary)

			if !tt.wantCorrected {
				assert.Nil(t, correction)
				return
			}
			require.NotNil(t, correction)
			assert.Equal(t, correctedChannelId, correction.ChannelId)
			assert.Equal(t, billingCorrectionTrigger, correction.PromptTokensThreshold)
			assert.Equal(t, tt.summary.PromptTokens, correction.PromptTokens)
			assert.Equal(t, tt.summary.Quota, correction.OriginalQuota)
			assert.NotZero(t, correction.RuleId)
		})
	}
}

// TestPostTextConsumeQuotaBillingCorrection 走完整的预扣费→结算→日志链路：
// 规则命中时预扣额度全额退回、不累计用量，消费日志扣费为 0 并带修正标记；
// 未配置规则时同样的请求照常按估算输入扣费，证明是修正规则把扣费改成了 0。
func TestPostTextConsumeQuotaBillingCorrection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const (
		userId        = 501
		tokenId       = 601
		initialQuota  = 10_000_000
		promptTokens  = 2_000_000
		expectedQuota = promptTokens // model ratio 1, group ratio 1
	)

	tests := []struct {
		name          string
		withRule      bool
		wantCharged   int
		wantCorrected bool
	}{
		{name: "enabled rule waives the estimated charge", withRule: true, wantCharged: 0, wantCorrected: true},
		{name: "without a rule the estimated input is charged", withRule: false, wantCharged: expectedQuota},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			truncate(t)
			seedUser(t, userId, initialQuota)
			seedToken(t, tokenId, userId, "billing-correction-key", initialQuota)
			if tt.withRule {
				seedBillingCorrectionRules(t, &model.BillingCorrectionRule{ChannelId: correctedChannelId, PromptTokensThreshold: billingCorrectionTrigger, Enabled: true})
			} else {
				seedBillingCorrectionRules(t)
			}

			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			info := newBillingCorrectionRelayInfo(correctedChannelId, types.RelayFormatOpenAI, relayconstant.RelayModeChatCompletions, promptTokens)
			info.UserId = userId
			info.TokenId = tokenId
			info.TokenKey = "billing-correction-key"
			info.UsingGroup = "default"
			info.OriginModelName = "gpt-4o"
			info.IsStream = true
			info.ForcePreConsume = true
			info.StartTime = time.Now()
			info.FirstResponseTime = info.StartTime
			info.UserSetting = dto.UserSetting{BillingPreference: "wallet_only"}
			info.PriceData.ModelRatio = 1
			info.PriceData.CompletionRatio = 1
			info.PriceData.GroupRatioInfo.GroupRatio = 1

			require.Nil(t, PreConsumeBilling(c, expectedQuota, info))
			// 上游只发心跳就断流：流式 handler 拿不到 usage，按本地估算的输入兜底，输出为 0
			usage := ResponseText2Usage(c, "", info.OriginModelName, info.GetEstimatePromptTokens())

			PostTextConsumeQuota(c, info, usage, nil)

			userQuota, err := model.GetUserQuota(userId, true)
			require.NoError(t, err)
			assert.Equal(t, initialQuota-tt.wantCharged, userQuota)
			user, err := model.GetUserById(userId, true)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCharged, user.UsedQuota)
			token, err := model.GetTokenById(tokenId)
			require.NoError(t, err)
			assert.Equal(t, initialQuota-tt.wantCharged, token.RemainQuota)
			assert.Equal(t, tt.wantCharged, token.UsedQuota)

			var logs []model.Log
			require.NoError(t, model.LOG_DB.Where("user_id = ? AND type = ?", userId, model.LogTypeConsume).Find(&logs).Error)
			require.Len(t, logs, 1)
			assert.Equal(t, tt.wantCharged, logs[0].Quota)
			assert.Equal(t, promptTokens, logs[0].PromptTokens)
			assert.Equal(t, 0, logs[0].CompletionTokens)

			other := map[string]interface{}{}
			require.NoError(t, common.UnmarshalJsonStr(logs[0].Other, &other))
			adminInfo, ok := other["admin_info"].(map[string]interface{})
			require.True(t, ok)
			if !tt.wantCorrected {
				assert.NotContains(t, other, "billing_corrected")
				assert.NotContains(t, adminInfo, "billing_correction")
				assert.NotContains(t, logs[0].Content, billingCorrectionLogContent)
				assert.Equal(t, 1, user.RequestCount)
				return
			}
			assert.Equal(t, true, other["billing_corrected"])
			assert.Contains(t, logs[0].Content, billingCorrectionLogContent)
			assert.Equal(t, 0, user.RequestCount)
			correction, ok := adminInfo["billing_correction"].(map[string]interface{})
			require.True(t, ok)
			assert.EqualValues(t, expectedQuota, correction["original_quota"])
			assert.EqualValues(t, promptTokens, correction["prompt_tokens"])
			assert.EqualValues(t, billingCorrectionTrigger, correction["prompt_tokens_threshold"])
			assert.EqualValues(t, correctedChannelId, correction["channel_id"])
		})
	}
}

// 渠道 ID 可能被新建渠道复用，删除渠道的每条路径都必须连带删掉它的修正规则，
// 否则新渠道会平白继承一条免扣费规则。
func TestChannelDeletionRemovesBillingCorrectionRules(t *testing.T) {
	tests := []struct {
		name          string
		deleteChannel func(t *testing.T)
	}{
		{
			name: "single channel delete",
			deleteChannel: func(t *testing.T) {
				require.NoError(t, (&model.Channel{Id: correctedChannelId}).Delete())
			},
		},
		{
			name: "batch channel delete",
			deleteChannel: func(t *testing.T) {
				deleted, err := model.BatchDeleteChannels([]int{correctedChannelId})
				require.NoError(t, err)
				require.EqualValues(t, 1, deleted)
			},
		},
		{
			name: "delete disabled channels",
			deleteChannel: func(t *testing.T) {
				require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", correctedChannelId).
					Update("status", common.ChannelStatusManuallyDisabled).Error)
				deleted, err := model.DeleteDisabledChannel()
				require.NoError(t, err)
				require.EqualValues(t, 1, deleted)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			truncate(t)
			seedChannel(t, correctedChannelId)
			seedChannel(t, disabledRuleChannelId)
			seedBillingCorrectionRules(t,
				&model.BillingCorrectionRule{ChannelId: correctedChannelId, PromptTokensThreshold: billingCorrectionTrigger, Enabled: true},
				&model.BillingCorrectionRule{ChannelId: disabledRuleChannelId, PromptTokensThreshold: billingCorrectionTrigger, Enabled: true},
			)

			tt.deleteChannel(t)

			rules, err := model.GetAllBillingCorrectionRules()
			require.NoError(t, err)
			require.Len(t, rules, 1)
			assert.Equal(t, disabledRuleChannelId, rules[0].ChannelId)
		})
	}
}
