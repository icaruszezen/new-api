package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fakeUpstreamUserId = 77
	fakeUpstreamToken  = "upstream-access-token"
)

// fakeUpstreamNewAPI 模拟上游 new-api 的 /api/log/self 和 /api/user/self，按上游请求 ID 返回预设的日志，
// 并记下每个请求 ID 被查了几次。
type fakeUpstreamNewAPI struct {
	mu   sync.Mutex
	hits map[string]int
	logs map[string][]map[string]any
}

func newFakeUpstreamNewAPI(t *testing.T) (*fakeUpstreamNewAPI, *httptest.Server) {
	t.Helper()
	upstream := &fakeUpstreamNewAPI{
		hits: map[string]int{},
		logs: map[string][]map[string]any{
			"up-charged": {{"type": model.LogTypeConsume, "quota": 900, "request_id": "up-charged"}},
			"up-free": {
				{"type": model.LogTypeError, "quota": 0, "request_id": "up-free"},
				{"type": model.LogTypeConsume, "quota": 0, "request_id": "up-free"},
			},
			"up-ignored": {{"type": model.LogTypeConsume, "quota": 100, "request_id": "someone-else"}},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer "+fakeUpstreamToken || r.Header.Get("New-Api-User") != "77" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"success":false,"message":"access token is invalid"}`))
			return
		}
		switch r.URL.Path {
		case "/api/user/self":
			_, _ = w.Write([]byte(`{"success":true,"message":"","data":{"id":77,"username":"relay-bot"}}`))
		case "/api/log/self":
			requestId := r.URL.Query().Get("request_id")
			upstream.mu.Lock()
			upstream.hits[requestId]++
			items := upstream.logs[requestId]
			upstream.mu.Unlock()
			if items == nil {
				items = []map[string]any{}
			}
			body, _ := common.Marshal(map[string]any{
				"success": true,
				"message": "",
				"data":    map[string]any{"page": 1, "page_size": 100, "total": len(items), "items": items},
			})
			_, _ = w.Write(body)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return upstream, server
}

func (u *fakeUpstreamNewAPI) hitsFor(requestId string) int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.hits[requestId]
}

// seedUpstreamCheckedChannel 建一个渠道和它的规则，上游核对指向 baseURL。
func seedUpstreamCheckedChannel(t *testing.T, baseURL string, autoRefund bool, checkSince int64) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.Channel{Id: correctedChannelId, Name: "upstream-new-api", BaseURL: &baseURL, Status: common.ChannelStatusEnabled}).Error)
	seedBillingCorrectionRules(t, &model.BillingCorrectionRule{
		ChannelId:             correctedChannelId,
		PromptTokensThreshold: billingCorrectionTrigger,
		Enabled:               true,
		UpstreamCheckEnabled:  true,
		UpstreamUserId:        fakeUpstreamUserId,
		UpstreamAccessToken:   fakeUpstreamToken,
		UpstreamCheckSince:    checkSince,
		AutoRefund:            autoRefund,
	})
}

func upstreamStatuses(page *BillingCorrectionRecordPage) map[string]string {
	statuses := make(map[string]string, len(page.Items))
	for _, record := range page.Items {
		statuses[record.RequestId] = record.UpstreamStatus
	}
	return statuses
}

// 手动复查只记录上游结果、不退费；复查全部时只查还没有确定结果的记录。
func TestCheckBillingCorrectionRecordsUpstream(t *testing.T) {
	upstream, server := newFakeUpstreamNewAPI(t)
	seedUpstreamCheckedChannel(t, server.URL, true, recordsDayStart)
	seedUser(t, 1, 1_000_000)
	seedCorrectionLogs(t, []seededCorrectionLog{
		{requestId: "charged", upstreamRequestId: "up-charged", channelId: correctedChannelId, createdAt: recordsDayStart + 600, promptTokens: 20_000, quota: 300_000, other: estimatedOther("/v1/messages")},
		{requestId: "free", upstreamRequestId: "up-free", channelId: correctedChannelId, createdAt: recordsDayStart + 500, promptTokens: 20_000, quota: 200_000, other: estimatedOther("/v1/messages")},
		{requestId: "missing", upstreamRequestId: "up-missing", channelId: correctedChannelId, createdAt: recordsDayStart + 400, promptTokens: 20_000, quota: 100_000, other: estimatedOther("/v1/messages")},
		{requestId: "ignored", upstreamRequestId: "up-ignored", channelId: correctedChannelId, createdAt: recordsDayStart + 300, promptTokens: 20_000, quota: 100_000, other: estimatedOther("/v1/messages")},
		{requestId: "no-upstream-id", channelId: correctedChannelId, createdAt: recordsDayStart + 200, promptTokens: 20_000, quota: 100_000, other: estimatedOther("/v1/messages")},
		{requestId: "corrected", upstreamRequestId: "up-corrected", channelId: correctedChannelId, createdAt: recordsDayStart + 100, promptTokens: 3_200_000, quota: 0, other: map[string]interface{}{
			"billing_corrected": true,
		}},
	})
	all := BillingCorrectionRecordFilter{BillingCorrectionLogFilter: model.BillingCorrectionLogFilter{StartTimestamp: recordsDayStart}}

	page, err := QueryBillingCorrectionRecords(all, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"charged":        BillingCorrectionUpstreamUnchecked,
		"free":           BillingCorrectionUpstreamUnchecked,
		"missing":        BillingCorrectionUpstreamUnchecked,
		"ignored":        BillingCorrectionUpstreamUnchecked,
		"no-upstream-id": BillingCorrectionUpstreamUnavailable,
		"corrected":      "",
	}, upstreamStatuses(page))

	result, err := CheckBillingCorrectionRecordsUpstream(context.Background(), all)
	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionUpstreamCheckResult{Checked: 4, Charged: 1, NotCharged: 1, NotFound: 1, Unsupported: 1, Skipped: 1}, result)
	assert.Zero(t, upstream.hitsFor("up-corrected"), "corrected records were never charged and are not checked")

	page, err = QueryBillingCorrectionRecords(all, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"charged":        BillingCorrectionUpstreamCharged,
		"free":           BillingCorrectionUpstreamNotCharged,
		"missing":        BillingCorrectionUpstreamNotFound,
		"ignored":        BillingCorrectionUpstreamUnsupported,
		"no-upstream-id": BillingCorrectionUpstreamUnavailable,
		"corrected":      "",
	}, upstreamStatuses(page))
	assert.Equal(t, 900, page.Items[0].UpstreamQuota)
	assert.Equal(t, 1, page.Stats.UpstreamNotChargedCount)
	assert.Equal(t, int64(200_000), page.Stats.UpstreamNotChargedQuota)
	assert.Equal(t, 1_000_000, getUserQuota(t, 1), "a manual check never refunds, even with auto refund on")

	notCharged := all
	notCharged.Upstream = BillingCorrectionUpstreamNotCharged
	page, err = QueryBillingCorrectionRecords(notCharged, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, []string{"free"}, recordRequestIds(page.Items))

	// 再复查全部：只有上游没找到的那条会重查；选中的记录不论结果都会重查。
	result, err = CheckBillingCorrectionRecordsUpstream(context.Background(), all)
	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionUpstreamCheckResult{Checked: 1, NotFound: 1, Skipped: 1}, result)
	assert.Equal(t, 2, upstream.hitsFor("up-missing"))
	assert.Equal(t, 1, upstream.hitsFor("up-charged"))

	result, err = CheckBillingCorrectionRecordsUpstream(context.Background(), BillingCorrectionRecordFilter{
		BillingCorrectionLogFilter: model.BillingCorrectionLogFilter{RequestIds: []string{"charged"}},
	})
	require.NoError(t, err)
	assert.Equal(t, BillingCorrectionUpstreamCheckResult{Checked: 1, Charged: 1}, result)
	assert.Equal(t, 2, upstream.hitsFor("up-charged"))
}

// 后台自动核对只看开启核对之后、结束超过 2 分钟的待退费记录；开启自动返还时退回上游没扣费的记录，
// 而且同一条记录不会被重复核对或重复退费。
func TestRunBillingCorrectionAutoCheck(t *testing.T) {
	tests := []struct {
		name         string
		autoRefund   bool
		wantRefunded int
	}{
		{name: "auto refund returns the quota the upstream did not charge", autoRefund: true, wantRefunded: 1},
		{name: "without auto refund the result is only recorded", autoRefund: false, wantRefunded: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream, server := newFakeUpstreamNewAPI(t)
			now := common.GetTimestamp()
			seedUpstreamCheckedChannel(t, server.URL, tt.autoRefund, now-3600)
			seedUser(t, 1, 1_000_000)
			seedToken(t, 5, 1, "sk-auto", 0)
			seedCorrectionLogs(t, []seededCorrectionLog{
				{requestId: "auto-free", upstreamRequestId: "up-free", channelId: correctedChannelId, tokenId: 5, createdAt: now - 600, promptTokens: 20_000, quota: 250_000, other: estimatedOther("/v1/chat/completions")},
				{requestId: "auto-charged", upstreamRequestId: "up-charged", channelId: correctedChannelId, tokenId: 5, createdAt: now - 500, promptTokens: 20_000, quota: 150_000, other: estimatedOther("/v1/chat/completions")},
				{requestId: "too-fresh", upstreamRequestId: "up-fresh", channelId: correctedChannelId, createdAt: now - 30, promptTokens: 20_000, quota: 150_000, other: estimatedOther("/v1/chat/completions")},
				{requestId: "before-enabled", upstreamRequestId: "up-before", channelId: correctedChannelId, createdAt: now - 7200, promptTokens: 20_000, quota: 150_000, other: estimatedOther("/v1/chat/completions")},
			})

			summary, err := RunBillingCorrectionAutoCheck(context.Background())

			require.NoError(t, err)
			assert.Equal(t, BillingCorrectionUpstreamCheckResult{Checked: 2, Charged: 1, NotCharged: 1}, summary.BillingCorrectionUpstreamCheckResult)
			assert.Equal(t, tt.wantRefunded, summary.Refunded)
			assert.Zero(t, upstream.hitsFor("up-fresh"))
			assert.Zero(t, upstream.hitsFor("up-before"))
			assert.Equal(t, 1_000_000+tt.wantRefunded*250_000, getUserQuota(t, 1))
			assert.Equal(t, tt.wantRefunded*250_000, getTokenRemainQuota(t, 5))

			refunds, err := model.GetBillingCorrectionRefundsByRequestIds([]string{"auto-free", "auto-charged"})
			require.NoError(t, err)
			assert.Len(t, refunds, tt.wantRefunded)
			if tt.wantRefunded > 0 {
				assert.Equal(t, model.BillingCorrectionRefundKindAuto, refunds["auto-free"].Kind)
			}

			summary, err = RunBillingCorrectionAutoCheck(context.Background())
			require.NoError(t, err)
			assert.Equal(t, BillingCorrectionAutoCheckSummary{}, summary)
			assert.Equal(t, 1, upstream.hitsFor("up-free"))
			assert.Equal(t, 1_000_000+tt.wantRefunded*250_000, getUserQuota(t, 1))
		})
	}
}

// 测试连接成功时返回上游用户名，鉴权失败时把上游的提示原样带回，方便管理员排查。
func TestTestBillingCorrectionUpstream(t *testing.T) {
	_, server := newFakeUpstreamNewAPI(t)
	channel := &model.Channel{BaseURL: &server.URL}

	username, err := TestBillingCorrectionUpstream(context.Background(), &model.BillingCorrectionRule{
		UpstreamUserId: fakeUpstreamUserId, UpstreamAccessToken: fakeUpstreamToken,
	}, channel)
	require.NoError(t, err)
	assert.Equal(t, "relay-bot", username)

	_, err = TestBillingCorrectionUpstream(context.Background(), &model.BillingCorrectionRule{
		UpstreamUserId: fakeUpstreamUserId, UpstreamAccessToken: "wrong-token",
	}, channel)
	require.ErrorContains(t, err, "access token is invalid")
}
