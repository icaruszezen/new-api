package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	recordsDayStart = int64(1_790_000_000)
	recordsAdminId  = 1
)

type seededCorrectionLog struct {
	requestId         string
	upstreamRequestId string
	username          string
	channelId         int
	tokenId           int
	createdAt         int64
	promptTokens      int
	completion        int
	quota             int
	other             map[string]interface{}
}

// estimatedOther 是上游没返回 usage、按本地估算计费时消费日志的 other。
func estimatedOther(requestPath string) map[string]interface{} {
	other := map[string]interface{}{"admin_info": map[string]interface{}{"local_count_tokens": true, "usage_billing_path": "local"}}
	if requestPath != "" {
		other["request_path"] = requestPath
	}
	return other
}

func seedCorrectionLogs(t *testing.T, logs []seededCorrectionLog) {
	t.Helper()
	for _, seeded := range logs {
		log := &model.Log{
			Type:              model.LogTypeConsume,
			RequestId:         seeded.requestId,
			UpstreamRequestId: seeded.upstreamRequestId,
			UserId:            1,
			Username:          seeded.username,
			ChannelId:         seeded.channelId,
			TokenId:           seeded.tokenId,
			CreatedAt:         seeded.createdAt,
			ModelName:         "claude-sonnet-4",
			PromptTokens:      seeded.promptTokens,
			CompletionTokens:  seeded.completion,
			Quota:             seeded.quota,
			Other:             common.MapToJsonStr(seeded.other),
		}
		require.NoError(t, model.LOG_DB.Create(log).Error)
	}
	t.Cleanup(func() {
		model.LOG_DB.Exec("DELETE FROM logs")
		model.DB.Exec("DELETE FROM billing_correction_refunds")
		model.DB.Exec("DELETE FROM billing_correction_upstream_checks")
		model.DB.Exec("DELETE FROM channels")
		model.DB.Exec("DELETE FROM users")
		model.DB.Exec("DELETE FROM tokens")
	})
}

// 一组覆盖各种情况的消费日志：渠道有已启用规则、输出为 0 又扣了费（或已被修正）的文本请求都会列出；
// 其中输入超过阈值、输入是本地估算的符合修正规则。
func seedMixedCorrectionLogs(t *testing.T) {
	t.Helper()
	seedBillingCorrectionRules(t,
		&model.BillingCorrectionRule{ChannelId: correctedChannelId, PromptTokensThreshold: billingCorrectionTrigger, Enabled: true},
		&model.BillingCorrectionRule{ChannelId: disabledRuleChannelId, PromptTokensThreshold: billingCorrectionTrigger, Enabled: false},
	)
	require.NoError(t, model.DB.Create(&model.Channel{Id: correctedChannelId, Name: "claude-relay"}).Error)
	seedCorrectionLogs(t, []seededCorrectionLog{
		{requestId: "heartbeat", username: "alice", channelId: correctedChannelId, createdAt: recordsDayStart + 900, promptTokens: 3_200_000, quota: 1_600_000, other: estimatedOther("/v1/messages")},
		{requestId: "corrected", username: "alice", channelId: correctedChannelId, createdAt: recordsDayStart + 800, promptTokens: 3_200_000, quota: 0, other: map[string]interface{}{
			"billing_corrected": true,
			"admin_info":        map[string]interface{}{"local_count_tokens": true, "billing_correction": map[string]interface{}{"original_quota": 1_600_000}},
		}},
		{requestId: "upstream-usage", username: "alice", channelId: correctedChannelId, createdAt: recordsDayStart + 700, promptTokens: 3_200_000, quota: 1_600_000, other: map[string]interface{}{"admin_info": map[string]interface{}{}}},
		{requestId: "has-output", username: "alice", channelId: correctedChannelId, createdAt: recordsDayStart + 600, promptTokens: 3_200_000, completion: 12, quota: 1_600_000, other: estimatedOther("/v1/messages")},
		{requestId: "at-threshold", username: "alice", channelId: correctedChannelId, createdAt: recordsDayStart + 500, promptTokens: billingCorrectionTrigger, quota: 500_000, other: estimatedOther("/v1/messages")},
		{requestId: "disabled-rule", username: "alice", channelId: disabledRuleChannelId, createdAt: recordsDayStart + 400, promptTokens: 3_200_000, quota: 1_600_000, other: estimatedOther("/v1/messages")},
		{requestId: "embedding", username: "alice", channelId: correctedChannelId, createdAt: recordsDayStart + 300, promptTokens: 3_200_000, quota: 100_000, other: estimatedOther("/v1/embeddings")},
		{requestId: "refunded", username: "alice", channelId: correctedChannelId, createdAt: recordsDayStart + 200, promptTokens: 2_000_000, quota: 1_000_000, other: estimatedOther("/v1/chat/completions")},
		// 较早的日志没有 request_path 和 usage_billing_path，只有本地估算标记。
		{requestId: "bob-legacy", username: "bob", channelId: correctedChannelId, createdAt: recordsDayStart + 100, promptTokens: 1_500_000, quota: 500_000, other: map[string]interface{}{
			"admin_info": map[string]interface{}{"local_count_tokens": true},
		}},
		{requestId: "yesterday", username: "alice", channelId: correctedChannelId, createdAt: recordsDayStart - 100, promptTokens: 3_200_000, quota: 1_600_000, other: estimatedOther("/v1/messages")},
	})
	require.NoError(t, model.CreateBillingCorrectionRefunds([]*model.BillingCorrectionRefund{
		{RequestId: "refunded", UserId: 1, ChannelId: correctedChannelId, Quota: 1_000_000, OperatorId: recordsAdminId},
	}))
}

func recordRequestIds(records []*BillingCorrectionRecord) []string {
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.RequestId)
	}
	return ids
}

func TestQueryBillingCorrectionRecords(t *testing.T) {
	seedMixedCorrectionLogs(t)
	today := model.BillingCorrectionLogFilter{StartTimestamp: recordsDayStart, EndTimestamp: recordsDayStart + 86_400}
	wantStats := BillingCorrectionRecordStats{
		ChargedCount:     5,
		ChargedQuota:     5_200_000,
		RuleMatchedCount: 3,
		RuleMatchedQuota: 3_100_000,
		PendingCount:     4,
		PendingQuota:     4_200_000,
		RefundedCount:    1,
		RefundedQuota:    1_000_000,
		CorrectedCount:   1,
	}

	tests := []struct {
		name      string
		filter    BillingCorrectionRecordFilter
		wantIds   []string
		wantStats BillingCorrectionRecordStats
	}{
		{
			name:      "lists every zero-output charged or corrected text request newest first",
			filter:    BillingCorrectionRecordFilter{BillingCorrectionLogFilter: today},
			wantIds:   []string{"heartbeat", "corrected", "upstream-usage", "at-threshold", "refunded", "bob-legacy"},
			wantStats: wantStats,
		},
		{
			name:      "status filter narrows the list but not the stats",
			filter:    BillingCorrectionRecordFilter{BillingCorrectionLogFilter: today, Status: BillingCorrectionRecordStatusPending},
			wantIds:   []string{"heartbeat", "upstream-usage", "at-threshold", "bob-legacy"},
			wantStats: wantStats,
		},
		{
			name:    "rule match filter keeps records above the threshold billed on a local estimate",
			filter:  BillingCorrectionRecordFilter{BillingCorrectionLogFilter: today, Match: BillingCorrectionRecordMatchRule},
			wantIds: []string{"heartbeat", "corrected", "refunded", "bob-legacy"},
			wantStats: BillingCorrectionRecordStats{
				ChargedCount: 3, ChargedQuota: 3_100_000, RuleMatchedCount: 3, RuleMatchedQuota: 3_100_000,
				PendingCount: 2, PendingQuota: 2_100_000, RefundedCount: 1, RefundedQuota: 1_000_000, CorrectedCount: 1,
			},
		},
		{
			name:    "zero output filter keeps charged records the rule does not cover",
			filter:  BillingCorrectionRecordFilter{BillingCorrectionLogFilter: today, Match: BillingCorrectionRecordMatchZeroOutput},
			wantIds: []string{"upstream-usage", "at-threshold"},
			wantStats: BillingCorrectionRecordStats{
				ChargedCount: 2, ChargedQuota: 2_100_000, PendingCount: 2, PendingQuota: 2_100_000,
			},
		},
		{
			name: "username filter",
			filter: BillingCorrectionRecordFilter{BillingCorrectionLogFilter: model.BillingCorrectionLogFilter{
				StartTimestamp: today.StartTimestamp, EndTimestamp: today.EndTimestamp, Username: "bob",
			}},
			wantIds: []string{"bob-legacy"},
			wantStats: BillingCorrectionRecordStats{
				ChargedCount: 1, ChargedQuota: 500_000, RuleMatchedCount: 1, RuleMatchedQuota: 500_000, PendingCount: 1, PendingQuota: 500_000,
			},
		},
		{
			name:      "channel whose rule is disabled has no records",
			filter:    BillingCorrectionRecordFilter{BillingCorrectionLogFilter: model.BillingCorrectionLogFilter{ChannelId: disabledRuleChannelId}},
			wantIds:   []string{},
			wantStats: BillingCorrectionRecordStats{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := QueryBillingCorrectionRecords(tt.filter, 1, 20)

			require.NoError(t, err)
			assert.Equal(t, tt.wantIds, recordRequestIds(page.Items))
			assert.Equal(t, len(tt.wantIds), page.Total)
			assert.Equal(t, tt.wantStats, page.Stats)
		})
	}
}

func TestQueryBillingCorrectionRecordsDescribesEachStatus(t *testing.T) {
	seedMixedCorrectionLogs(t)

	page, err := QueryBillingCorrectionRecords(BillingCorrectionRecordFilter{
		BillingCorrectionLogFilter: model.BillingCorrectionLogFilter{StartTimestamp: recordsDayStart},
	}, 1, 20)

	require.NoError(t, err)
	byId := make(map[string]*BillingCorrectionRecord, len(page.Items))
	for _, record := range page.Items {
		byId[record.RequestId] = record
	}
	require.Contains(t, byId, "heartbeat")
	assert.Equal(t, BillingCorrectionRecordStatusPending, byId["heartbeat"].Status)
	assert.Equal(t, BillingCorrectionRecordMatchRule, byId["heartbeat"].Match)
	assert.Equal(t, "claude-relay", byId["heartbeat"].ChannelName)
	assert.Equal(t, billingCorrectionTrigger, byId["heartbeat"].PromptTokensThreshold)
	require.Contains(t, byId, "at-threshold")
	assert.Equal(t, BillingCorrectionRecordMatchZeroOutput, byId["at-threshold"].Match, "the rule needs input strictly above the threshold")
	require.Contains(t, byId, "corrected")
	assert.Equal(t, BillingCorrectionRecordStatusCorrected, byId["corrected"].Status)
	assert.Equal(t, 1_600_000, byId["corrected"].OriginalQuota)
	require.Contains(t, byId, "refunded")
	assert.Equal(t, BillingCorrectionRecordStatusRefunded, byId["refunded"].Status)
	assert.Equal(t, model.BillingCorrectionRefundKindMark, byId["refunded"].RefundKind)
	assert.Equal(t, recordsAdminId, byId["refunded"].RefundOperatorId)
	assert.NotZero(t, byId["refunded"].RefundedAt)
	assert.Empty(t, byId["heartbeat"].UpstreamStatus, "channels without upstream checking show no upstream status")
}

// 结算用的 usage 来源决定是否符合修正规则：本地估算、从本地估算补出的计费 usage 都算；
// 上游给了 usage 的只算 0 输出，即使日志因为补估输出带了 local_count_tokens（Claude message_start 后断流）。
func TestQueryBillingCorrectionRecordsByUsageSource(t *testing.T) {
	seedBillingCorrectionRules(t,
		&model.BillingCorrectionRule{ChannelId: correctedChannelId, PromptTokensThreshold: billingCorrectionTrigger, Enabled: true},
	)
	seedCorrectionLogs(t, []seededCorrectionLog{
		{requestId: "local", channelId: correctedChannelId, createdAt: recordsDayStart + 500, promptTokens: 3_200_000, quota: 1_600_000, other: map[string]interface{}{
			"request_path": "/v1/chat/completions",
			"admin_info":   map[string]interface{}{"local_count_tokens": true, "usage_billing_path": "local"},
		}},
		{requestId: "openai-estimated", channelId: correctedChannelId, createdAt: recordsDayStart + 400, promptTokens: 3_200_000, quota: 1_600_000, other: map[string]interface{}{
			"request_path": "/v1/responses",
			"admin_info":   map[string]interface{}{"usage_billing_path": "billing-usage-openai-estimated"},
		}},
		{requestId: "anthropic-upstream-input", channelId: correctedChannelId, createdAt: recordsDayStart + 300, promptTokens: 3_200_000, quota: 1_600_000, other: map[string]interface{}{
			"request_path": "/v1/messages",
			"admin_info":   map[string]interface{}{"local_count_tokens": true, "usage_billing_path": "billing-usage-anthropic"},
		}},
		{requestId: "upstream", channelId: correctedChannelId, createdAt: recordsDayStart + 200, promptTokens: 3_200_000, quota: 1_600_000, other: map[string]interface{}{
			"request_path": "/v1/chat/completions",
			"admin_info":   map[string]interface{}{"usage_billing_path": "upstream"},
		}},
		{requestId: "legacy", channelId: correctedChannelId, createdAt: recordsDayStart + 100, promptTokens: 3_200_000, quota: 1_600_000, other: map[string]interface{}{
			"admin_info": map[string]interface{}{"local_count_tokens": true},
		}},
	})

	page, err := QueryBillingCorrectionRecords(BillingCorrectionRecordFilter{
		BillingCorrectionLogFilter: model.BillingCorrectionLogFilter{StartTimestamp: recordsDayStart},
	}, 1, 20)

	require.NoError(t, err)
	matches := make(map[string]string, len(page.Items))
	for _, record := range page.Items {
		matches[record.RequestId] = record.Match
	}
	assert.Equal(t, map[string]string{
		"local":                    BillingCorrectionRecordMatchRule,
		"openai-estimated":         BillingCorrectionRecordMatchRule,
		"anthropic-upstream-input": BillingCorrectionRecordMatchZeroOutput,
		"upstream":                 BillingCorrectionRecordMatchZeroOutput,
		"legacy":                   BillingCorrectionRecordMatchRule,
	}, matches)
	assert.Equal(t, int64(4_800_000), page.Stats.RuleMatchedQuota)
	assert.Equal(t, int64(8_000_000), page.Stats.PendingQuota)
}

// 标记退费只写标记、不退额度，而且只能标记已扣费、还没退费的记录。
func TestMarkBillingCorrectionRecordsRefunded(t *testing.T) {
	seedMixedCorrectionLogs(t)
	const operatorId = 42
	today := BillingCorrectionRecordFilter{BillingCorrectionLogFilter: model.BillingCorrectionLogFilter{
		StartTimestamp: recordsDayStart, EndTimestamp: recordsDayStart + 86_400,
	}}

	// 按选中的记录标记：已修正、已退费和不是文本请求的记录都被忽略。
	result, err := MarkBillingCorrectionRecordsRefunded(BillingCorrectionRecordFilter{BillingCorrectionLogFilter: model.BillingCorrectionLogFilter{
		RequestIds: []string{"heartbeat", "corrected", "refunded", "upstream-usage", "embedding"},
	}}, operatorId)
	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionRefundResult{Marked: 2, Quota: 3_200_000}, result)

	refunds, err := model.GetBillingCorrectionRefundsByRequestIds([]string{"heartbeat", "corrected", "upstream-usage", "embedding"})
	require.NoError(t, err)
	require.Contains(t, refunds, "heartbeat")
	assert.Len(t, refunds, 2)
	assert.Equal(t, operatorId, refunds["heartbeat"].OperatorId)
	assert.Equal(t, 1_600_000, refunds["heartbeat"].Quota)
	assert.Equal(t, model.BillingCorrectionRefundKindMark, refunds["heartbeat"].Kind)

	// 标记筛选范围内的全部：只剩两条待退费，再标一次就没有可标的了。
	result, err = MarkBillingCorrectionRecordsRefunded(today, operatorId)
	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionRefundResult{Marked: 2, Quota: 1_000_000}, result)
	result, err = MarkBillingCorrectionRecordsRefunded(today, operatorId)
	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionRefundResult{}, result)

	page, err := QueryBillingCorrectionRecords(today, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionRecordStats{
		ChargedCount: 5, ChargedQuota: 5_200_000, RuleMatchedCount: 3, RuleMatchedQuota: 3_100_000,
		RefundedCount: 5, RefundedQuota: 5_200_000, CorrectedCount: 1,
	}, page.Stats)

	// 撤销标记后回到待退费。
	unmarked, err := model.DeleteBillingCorrectionRefunds([]string{"heartbeat"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, unmarked)
	today.Status = BillingCorrectionRecordStatusPending
	page, err = QueryBillingCorrectionRecords(today, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, []string{"heartbeat"}, recordRequestIds(page.Items))
}

// 补费把这次扣的额度退回用户钱包和令牌，回减已用额度，并给用户留一条退款日志；
// 每条记录最多补一次，补过的记录不能撤销，也不能再标记。
func TestCreditBillingCorrectionRecords(t *testing.T) {
	seedBillingCorrectionRules(t,
		&model.BillingCorrectionRule{ChannelId: correctedChannelId, PromptTokensThreshold: billingCorrectionTrigger, Enabled: true},
	)
	seedUser(t, 1, 2_000_000)
	seedToken(t, 5, 1, "sk-credit", 300_000)
	seedChannel(t, correctedChannelId)
	seedChargedAccounting(t, 1, correctedChannelId, 5, 3_000_000, 2)
	seedCorrectionLogs(t, []seededCorrectionLog{
		{requestId: "zero-output", username: "alice", channelId: correctedChannelId, tokenId: 5, createdAt: recordsDayStart + 200, promptTokens: 20_000, quota: 700_000, other: estimatedOther("/v1/chat/completions")},
		{requestId: "marked", username: "alice", channelId: correctedChannelId, tokenId: 5, createdAt: recordsDayStart + 100, promptTokens: 20_000, quota: 400_000, other: estimatedOther("/v1/chat/completions")},
	})
	require.NoError(t, model.CreateBillingCorrectionRefunds([]*model.BillingCorrectionRefund{
		{RequestId: "marked", UserId: 1, ChannelId: correctedChannelId, Quota: 400_000, OperatorId: recordsAdminId},
	}))
	const operatorId = 42
	selected := BillingCorrectionRecordFilter{BillingCorrectionLogFilter: model.BillingCorrectionLogFilter{
		RequestIds: []string{"zero-output", "marked"},
	}}
	logsBefore := countLogs(t)

	result, err := CreditBillingCorrectionRecords(selected, operatorId)

	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionCreditResult{Credited: 1, Quota: 700_000}, result, "a record already marked as refunded is not credited again")
	assert.Equal(t, 2_700_000, getUserQuota(t, 1))
	usedQuota, requestCount := getUserUsageAccounting(t, 1)
	assert.Equal(t, 2_300_000, usedQuota)
	assert.Equal(t, 2, requestCount)
	assert.EqualValues(t, 2_300_000, getChannelUsedQuota(t, correctedChannelId))
	assert.Equal(t, 1_000_000, getTokenRemainQuota(t, 5))

	assert.Equal(t, logsBefore+1, countLogs(t))
	refundLog := getLastLog(t)
	require.NotNil(t, refundLog)
	assert.Equal(t, model.LogTypeRefund, refundLog.Type)
	assert.Equal(t, 700_000, refundLog.Quota)
	assert.Equal(t, 1, refundLog.UserId)
	assert.Contains(t, refundLog.Content, "zero-output")

	refunds, err := model.GetBillingCorrectionRefundsByRequestIds([]string{"zero-output"})
	require.NoError(t, err)
	require.Contains(t, refunds, "zero-output")
	assert.Equal(t, model.BillingCorrectionRefundKindCredit, refunds["zero-output"].Kind)
	assert.Equal(t, operatorId, refunds["zero-output"].OperatorId)

	// 再补一次、再标记一次、撤销标记，都不会动这条已经补过的记录。
	result, err = CreditBillingCorrectionRecords(selected, operatorId)
	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionCreditResult{}, result)
	marked, err := MarkBillingCorrectionRecordsRefunded(selected, operatorId)
	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionRefundResult{}, marked)
	unmarked, err := model.DeleteBillingCorrectionRefunds([]string{"zero-output"})
	require.NoError(t, err)
	assert.Zero(t, unmarked)
	assert.Equal(t, 2_700_000, getUserQuota(t, 1))

	page, err := QueryBillingCorrectionRecords(BillingCorrectionRecordFilter{
		BillingCorrectionLogFilter: model.BillingCorrectionLogFilter{StartTimestamp: recordsDayStart},
	}, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionRecordStatusCredited, page.Items[0].Status)
	assert.Equal(t, int64(700_000), page.Stats.CreditedQuota)
	assert.Equal(t, int64(1_100_000), page.Stats.RefundedQuota)
}
