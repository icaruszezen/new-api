package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// 上游 new-api 的扣费核对结果。前五种会落库，后两种只在列表里出现。
const (
	// BillingCorrectionUpstreamCharged 上游有这次请求的消费日志，并且扣了费。
	BillingCorrectionUpstreamCharged = "charged"
	// BillingCorrectionUpstreamNotCharged 上游有这次请求的日志，但没有扣费（消费日志额度为 0，或只有错误日志）。
	BillingCorrectionUpstreamNotCharged = "not_charged"
	// BillingCorrectionUpstreamNotFound 上游查不到这次请求的日志。
	BillingCorrectionUpstreamNotFound = "not_found"
	// BillingCorrectionUpstreamFailed 请求上游失败，例如网络错误、鉴权失败。
	BillingCorrectionUpstreamFailed = "failed"
	// BillingCorrectionUpstreamUnsupported 上游不支持按请求 ID 查询使用日志。
	BillingCorrectionUpstreamUnsupported = "unsupported"
	// BillingCorrectionUpstreamUnchecked 可以核对，但还没有核对过。
	BillingCorrectionUpstreamUnchecked = "unchecked"
	// BillingCorrectionUpstreamUnavailable 渠道配置了上游核对，但这条日志没有上游请求 ID，无法核对。
	BillingCorrectionUpstreamUnavailable = "unavailable"
)

const (
	// BillingCorrectionUpstreamCheckLimit 是手动复查一次最多核对的记录数，复查在请求里同步完成。
	BillingCorrectionUpstreamCheckLimit = 100

	billingCorrectionUpstreamTimeout     = 10 * time.Second
	billingCorrectionUpstreamConcurrency = 8
	billingCorrectionUpstreamMaxBody     = 4 << 20
	billingCorrectionUpstreamMaxMessage  = 255
	// billingCorrectionUpstreamLogWindow 是按请求 ID 查上游日志时前后放宽的时间（秒），容忍两边时钟偏差，
	// 同时让上游只扫一小段日志。
	billingCorrectionUpstreamLogWindow = 24 * 3600

	// 后台自动核对：请求结束 2 分钟后再查，给上游写日志留出时间；只看最近 24 小时；
	// 上游没找到或核对失败的最多查 5 次，每次间隔按次数递增 5 分钟。
	billingCorrectionAutoCheckDelay       = 120
	billingCorrectionAutoCheckWindow      = 24 * 3600
	billingCorrectionAutoCheckMaxAttempts = 5
	billingCorrectionAutoCheckBackoff     = 300
	billingCorrectionAutoCheckBatch       = 200
)

var errBillingCorrectionUpstreamUnsupported = errors.New("upstream does not support this query")

// BillingCorrectionUpstreamCheckResult 汇总一次手动复查。Skipped 是没法核对的记录数：
// 渠道没有配置上游核对、日志没有上游请求 ID 或本地请求 ID。
type BillingCorrectionUpstreamCheckResult struct {
	Checked     int `json:"checked"`
	Charged     int `json:"charged"`
	NotCharged  int `json:"not_charged"`
	NotFound    int `json:"not_found"`
	Failed      int `json:"failed"`
	Unsupported int `json:"unsupported"`
	Skipped     int `json:"skipped"`
	Remaining   int `json:"remaining"`
}

func (r *BillingCorrectionUpstreamCheckResult) count(status string) {
	r.Checked++
	switch status {
	case BillingCorrectionUpstreamCharged:
		r.Charged++
	case BillingCorrectionUpstreamNotCharged:
		r.NotCharged++
	case BillingCorrectionUpstreamNotFound:
		r.NotFound++
	case BillingCorrectionUpstreamUnsupported:
		r.Unsupported++
	default:
		r.Failed++
	}
}

// BillingCorrectionAutoCheckSummary 是一轮后台自动核对的结果，写进系统任务的运行记录。
type BillingCorrectionAutoCheckSummary struct {
	BillingCorrectionUpstreamCheckResult
	Refunded      int   `json:"refunded"`
	RefundedQuota int64 `json:"refunded_quota"`
}

// billingCorrectionUpstream 是访问上游 new-api 的连接信息。
type billingCorrectionUpstream struct {
	baseURL     string
	userId      int
	accessToken string
	proxy       string
}

// newBillingCorrectionUpstream 用规则里的上游配置和渠道的代理组装连接信息，规则没填上游地址时用渠道的 Base URL。
func newBillingCorrectionUpstream(rule *model.BillingCorrectionRule, channel *model.Channel) *billingCorrectionUpstream {
	baseURL := rule.UpstreamBaseUrl
	if baseURL == "" {
		baseURL = channel.GetBaseURL()
	}
	return &billingCorrectionUpstream{
		baseURL:     strings.TrimRight(baseURL, "/"),
		userId:      rule.UpstreamUserId,
		accessToken: rule.UpstreamAccessToken,
		proxy:       channel.GetSetting().Proxy,
	}
}

// get 调用上游 new-api 的管理接口，返回响应里的 data。上游用系统访问令牌鉴权，
// 官方版本还要求 New-Api-User 与令牌所属用户一致。
func (u *billingCorrectionUpstream) get(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	if u.baseURL == "" {
		return nil, errors.New("upstream base URL is empty")
	}
	endpoint := u.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	ctx, cancel := context.WithTimeout(ctx, billingCorrectionUpstreamTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+u.accessToken)
	req.Header.Set("New-Api-User", strconv.Itoa(u.userId))
	req.Header.Set("Accept", "application/json")
	client, err := GetHttpClientWithProxy(u.proxy)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, billingCorrectionUpstreamMaxBody))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, errBillingCorrectionUpstreamUnsupported
	}

	var parsed struct {
		Success bool            `json:"success"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := common.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("HTTP %d: response is not a new-api JSON response", resp.StatusCode)
	}
	if !parsed.Success {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, parsed.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return parsed.Data, nil
}

// checkBilling 在上游的使用日志里按请求 ID 查这次请求有没有扣费。
// 只认 request_id 完全一致的日志：上游如果忽略了 request_id 条件、返回了别的日志，算作不支持。
func (u *billingCorrectionUpstream) checkBilling(ctx context.Context, upstreamRequestId string, createdAt int64) (status string, quota int, message string) {
	query := url.Values{}
	query.Set("request_id", upstreamRequestId)
	query.Set("p", "1")
	query.Set("page_size", "100")
	if createdAt > 0 {
		query.Set("start_timestamp", strconv.FormatInt(createdAt-billingCorrectionUpstreamLogWindow, 10))
		query.Set("end_timestamp", strconv.FormatInt(createdAt+billingCorrectionUpstreamLogWindow, 10))
	}
	data, err := u.get(ctx, "/api/log/self", query)
	if errors.Is(err, errBillingCorrectionUpstreamUnsupported) {
		return BillingCorrectionUpstreamUnsupported, 0, "upstream has no /api/log/self endpoint"
	}
	if err != nil {
		return BillingCorrectionUpstreamFailed, 0, err.Error()
	}

	type upstreamLog struct {
		Type      int    `json:"type"`
		Quota     int    `json:"quota"`
		RequestId string `json:"request_id"`
	}
	var logs []upstreamLog
	switch common.GetJsonType(data) {
	case "array":
		err = common.Unmarshal(data, &logs)
	case "object":
		var page struct {
			Items []upstreamLog `json:"items"`
		}
		err = common.Unmarshal(data, &page)
		logs = page.Items
	default:
		err = errors.New("unexpected log list format")
	}
	if err != nil {
		return BillingCorrectionUpstreamFailed, 0, err.Error()
	}

	matched := 0
	for _, log := range logs {
		if log.RequestId != upstreamRequestId {
			continue
		}
		matched++
		if log.Type == model.LogTypeConsume && log.Quota > 0 {
			quota += log.Quota
		}
	}
	switch {
	case matched == 0 && len(logs) > 0:
		return BillingCorrectionUpstreamUnsupported, 0, "upstream ignored the request_id filter"
	case matched == 0:
		return BillingCorrectionUpstreamNotFound, 0, ""
	case quota > 0:
		return BillingCorrectionUpstreamCharged, quota, ""
	}
	return BillingCorrectionUpstreamNotCharged, 0, ""
}

// checkBillingCorrectionRecordsUpstream 并发到上游核对这些记录，把结果写库并回填到记录上。
// 调用方保证每条记录都能核对（有本地和上游请求 ID，渠道配置了上游核对）。
func checkBillingCorrectionRecordsUpstream(ctx context.Context, records []*BillingCorrectionRecord, channels map[int]*billingCorrectionChannel) {
	var wg sync.WaitGroup
	slots := make(chan struct{}, billingCorrectionUpstreamConcurrency)
	for _, record := range records {
		wg.Add(1)
		slots <- struct{}{}
		go func(record *BillingCorrectionRecord) {
			defer wg.Done()
			defer func() { <-slots }()
			status, quota, message := channels[record.ChannelId].upstream.checkBilling(ctx, record.UpstreamRequestId, record.CreatedAt)
			if utf8.RuneCountInString(message) > billingCorrectionUpstreamMaxMessage {
				message = string([]rune(message)[:billingCorrectionUpstreamMaxMessage])
			}
			check := &model.BillingCorrectionUpstreamCheck{
				RequestId:         record.RequestId,
				ChannelId:         record.ChannelId,
				UpstreamRequestId: record.UpstreamRequestId,
				Status:            status,
				UpstreamQuota:     quota,
				Message:           message,
				Attempts:          record.upstreamAttempts + 1,
				CheckedAt:         common.GetTimestamp(),
			}
			if err := model.SaveBillingCorrectionUpstreamCheck(check); err != nil {
				common.SysError(fmt.Sprintf("failed to save billing correction upstream check %s: %s", record.RequestId, err.Error()))
			}
			record.UpstreamStatus = status
			record.UpstreamQuota = quota
			record.UpstreamMessage = message
			record.UpstreamCheckedAt = check.CheckedAt
			record.upstreamAttempts = check.Attempts
		}(record)
	}
	wg.Wait()
}

// RunBillingCorrectionAutoCheck 是后台自动核对：在配置了上游核对的渠道上，把开启核对之后、最近 24 小时内
// 还没退费的 0 输出扣费记录拿到上游核对。规则开启了自动返还时，上游确认没有扣费的记录直接补费。
// 已经核对过的记录不会因为后来打开自动返还而被补费，那些记录由管理员在列表里筛选后手动处理。
func RunBillingCorrectionAutoCheck(ctx context.Context) (BillingCorrectionAutoCheckSummary, error) {
	var summary BillingCorrectionAutoCheckSummary
	channels, err := loadBillingCorrectionChannels()
	if err != nil {
		return summary, err
	}
	now := common.GetTimestamp()
	for channelId, channel := range channels {
		if ctx.Err() != nil {
			return summary, ctx.Err()
		}
		if channel.upstream == nil {
			continue
		}
		since := max(now-billingCorrectionAutoCheckWindow, channel.rule.UpstreamCheckSince)
		until := now - billingCorrectionAutoCheckDelay
		if since >= until {
			continue
		}
		records, _, err := findBillingCorrectionRecords(BillingCorrectionRecordFilter{
			BillingCorrectionLogFilter: model.BillingCorrectionLogFilter{
				ChannelId:      channelId,
				StartTimestamp: since,
				EndTimestamp:   until,
			},
		})
		if err != nil {
			common.SysError(fmt.Sprintf("billing correction auto check skipped channel %d: %s", channelId, err.Error()))
			continue
		}

		candidates := make([]*BillingCorrectionRecord, 0, min(len(records), billingCorrectionAutoCheckBatch))
		for _, record := range records {
			if len(candidates) >= billingCorrectionAutoCheckBatch {
				break
			}
			if record.Status != BillingCorrectionRecordStatusPending || record.RequestId == "" || record.UpstreamRequestId == "" {
				continue
			}
			retry := (record.UpstreamStatus == BillingCorrectionUpstreamNotFound || record.UpstreamStatus == BillingCorrectionUpstreamFailed) &&
				record.upstreamAttempts < billingCorrectionAutoCheckMaxAttempts &&
				now-record.UpstreamCheckedAt >= int64(record.upstreamAttempts)*billingCorrectionAutoCheckBackoff
			if record.UpstreamStatus == BillingCorrectionUpstreamUnchecked || retry {
				candidates = append(candidates, record)
			}
		}

		checkBillingCorrectionRecordsUpstream(ctx, candidates, channels)
		for _, record := range candidates {
			summary.count(record.UpstreamStatus)
			if !channel.rule.AutoRefund || record.UpstreamStatus != BillingCorrectionUpstreamNotCharged {
				continue
			}
			credited, err := creditBillingCorrectionRecord(record, model.BillingCorrectionRefundKindAuto, 0)
			if err != nil {
				common.SysError(fmt.Sprintf("failed to auto refund billing correction record %s: %s", record.RequestId, err.Error()))
				continue
			}
			if credited {
				summary.Refunded++
				summary.RefundedQuota += int64(record.Quota)
			}
		}
	}
	return summary, nil
}

// TestBillingCorrectionUpstream 用给定的上游配置读取上游的当前用户，确认地址和令牌可用，返回上游用户名。
func TestBillingCorrectionUpstream(ctx context.Context, rule *model.BillingCorrectionRule, channel *model.Channel) (string, error) {
	data, err := newBillingCorrectionUpstream(rule, channel).get(ctx, "/api/user/self", nil)
	if err != nil {
		return "", err
	}
	var user struct {
		Id       int    `json:"id"`
		Username string `json:"username"`
	}
	if err := common.Unmarshal(data, &user); err != nil {
		return "", err
	}
	if user.Id != 0 && user.Id != rule.UpstreamUserId {
		return "", fmt.Errorf("the access token belongs to upstream user %d, not %d", user.Id, rule.UpstreamUserId)
	}
	return user.Username, nil
}
