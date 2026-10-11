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
	requestId    string
	username     string
	channelId    int
	createdAt    int64
	promptTokens int
	completion   int
	quota        int
	other        map[string]interface{}
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
			Type:             model.LogTypeConsume,
			RequestId:        seeded.requestId,
			UserId:           1,
			Username:         seeded.username,
			ChannelId:        seeded.channelId,
			CreatedAt:        seeded.createdAt,
			ModelName:        "claude-sonnet-4",
			PromptTokens:     seeded.promptTokens,
			CompletionTokens: seeded.completion,
			Quota:            seeded.quota,
			Other:            common.MapToJsonStr(seeded.other),
		}
		require.NoError(t, model.LOG_DB.Create(log).Error)
	}
	t.Cleanup(func() {
		model.LOG_DB.Exec("DELETE FROM logs")
		model.DB.Exec("DELETE FROM billing_correction_refunds")
		model.DB.Exec("DELETE FROM channels")
	})
}

// 一组覆盖各种情况的消费日志：只有渠道有已启用规则、输入超过阈值、输出为 0、输入是本地估算的文本请求才算错误计费。
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
		ChargedCount:   3,
		ChargedQuota:   3_100_000,
		PendingCount:   2,
		PendingQuota:   2_100_000,
		RefundedCount:  1,
		RefundedQuota:  1_000_000,
		CorrectedCount: 1,
	}

	tests := []struct {
		name      string
		filter    model.BillingCorrectionLogFilter
		status    string
		wantIds   []string
		wantStats BillingCorrectionRecordStats
	}{
		{
			name:      "lists every matching record newest first",
			filter:    today,
			wantIds:   []string{"heartbeat", "corrected", "refunded", "bob-legacy"},
			wantStats: wantStats,
		},
		{
			name:      "status filter narrows the list but not the stats",
			filter:    today,
			status:    BillingCorrectionRecordStatusPending,
			wantIds:   []string{"heartbeat", "bob-legacy"},
			wantStats: wantStats,
		},
		{
			name:    "username filter",
			filter:  model.BillingCorrectionLogFilter{StartTimestamp: today.StartTimestamp, EndTimestamp: today.EndTimestamp, Username: "bob"},
			wantIds: []string{"bob-legacy"},
			wantStats: BillingCorrectionRecordStats{
				ChargedCount: 1, ChargedQuota: 500_000, PendingCount: 1, PendingQuota: 500_000,
			},
		},
		{
			name:      "channel whose rule is disabled has no records",
			filter:    model.BillingCorrectionLogFilter{ChannelId: disabledRuleChannelId},
			wantIds:   []string{},
			wantStats: BillingCorrectionRecordStats{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := QueryBillingCorrectionRecords(tt.filter, tt.status, 1, 20)

			require.NoError(t, err)
			assert.Equal(t, tt.wantIds, recordRequestIds(page.Items))
			assert.Equal(t, len(tt.wantIds), page.Total)
			assert.Equal(t, tt.wantStats, page.Stats)
		})
	}
}

func TestQueryBillingCorrectionRecordsDescribesEachStatus(t *testing.T) {
	seedMixedCorrectionLogs(t)

	page, err := QueryBillingCorrectionRecords(model.BillingCorrectionLogFilter{StartTimestamp: recordsDayStart}, "", 1, 20)

	require.NoError(t, err)
	byId := make(map[string]*BillingCorrectionRecord, len(page.Items))
	for _, record := range page.Items {
		byId[record.RequestId] = record
	}
	require.Contains(t, byId, "heartbeat")
	assert.Equal(t, BillingCorrectionRecordStatusPending, byId["heartbeat"].Status)
	assert.Equal(t, "claude-relay", byId["heartbeat"].ChannelName)
	assert.Equal(t, billingCorrectionTrigger, byId["heartbeat"].PromptTokensThreshold)
	require.Contains(t, byId, "corrected")
	assert.Equal(t, BillingCorrectionRecordStatusCorrected, byId["corrected"].Status)
	assert.Equal(t, 1_600_000, byId["corrected"].OriginalQuota)
	require.Contains(t, byId, "refunded")
	assert.Equal(t, BillingCorrectionRecordStatusRefunded, byId["refunded"].Status)
	assert.Equal(t, recordsAdminId, byId["refunded"].RefundOperatorId)
	assert.NotZero(t, byId["refunded"].RefundedAt)
}

// 结算用的 usage 来源决定是不是错误计费：本地估算、从本地估算补出的计费 usage 都算；
// 上游给了 usage 的不算，即使日志因为补估输出带了 local_count_tokens（Claude message_start 后断流）。
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

	page, err := QueryBillingCorrectionRecords(model.BillingCorrectionLogFilter{StartTimestamp: recordsDayStart}, "", 1, 20)

	require.NoError(t, err)
	assert.Equal(t, []string{"local", "openai-estimated", "legacy"}, recordRequestIds(page.Items))
	assert.Equal(t, int64(4_800_000), page.Stats.PendingQuota)
}

// 标记退费只写标记、不退额度，而且只能标记已扣费、未标记的错误计费记录。
func TestMarkBillingCorrectionRecordsRefunded(t *testing.T) {
	seedMixedCorrectionLogs(t)
	const operatorId = 42
	today := model.BillingCorrectionLogFilter{StartTimestamp: recordsDayStart, EndTimestamp: recordsDayStart + 86_400}

	// 按选中的记录标记：已修正、已退费和不符合条件的记录都被忽略。
	result, err := MarkBillingCorrectionRecordsRefunded(model.BillingCorrectionLogFilter{
		RequestIds: []string{"heartbeat", "corrected", "refunded", "upstream-usage", "embedding"},
	}, operatorId)
	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionRefundResult{Marked: 1, Quota: 1_600_000}, result)

	refunds, err := model.GetBillingCorrectionRefundsByRequestIds([]string{"heartbeat", "corrected", "upstream-usage", "embedding"})
	require.NoError(t, err)
	require.Contains(t, refunds, "heartbeat")
	assert.Len(t, refunds, 1)
	assert.Equal(t, operatorId, refunds["heartbeat"].OperatorId)
	assert.Equal(t, 1_600_000, refunds["heartbeat"].Quota)

	// 标记筛选范围内的全部：只剩 bob 那条待退费，再标一次就没有可标的了。
	result, err = MarkBillingCorrectionRecordsRefunded(today, operatorId)
	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionRefundResult{Marked: 1, Quota: 500_000}, result)
	result, err = MarkBillingCorrectionRecordsRefunded(today, operatorId)
	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionRefundResult{}, result)

	page, err := QueryBillingCorrectionRecords(today, "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionRecordStats{
		ChargedCount: 3, ChargedQuota: 3_100_000, RefundedCount: 3, RefundedQuota: 3_100_000, CorrectedCount: 1,
	}, page.Stats)

	// 撤销标记后回到待退费。
	unmarked, err := model.DeleteBillingCorrectionRefunds([]string{"heartbeat"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, unmarked)
	page, err = QueryBillingCorrectionRecords(today, BillingCorrectionRecordStatusPending, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, []string{"heartbeat"}, recordRequestIds(page.Items))
}
