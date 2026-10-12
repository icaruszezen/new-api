package controller

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// 输入 token 阈值的取值范围，与前端表单保持一致。
const (
	minBillingCorrectionThreshold = 1
	maxBillingCorrectionThreshold = 1_000_000_000
)

// maxBillingCorrectionRefundSelection 是按记录标记、撤销或补费时一次最多处理的请求数。
const maxBillingCorrectionRefundSelection = 1000

// 上游配置字段的长度上限，与数据库列宽一致。
const (
	maxBillingCorrectionUpstreamBaseUrlLength = 512
	maxBillingCorrectionUpstreamTokenLength   = 255
)

// billingCorrectionRecordsRequest 是批量处理记录（标记、撤销、补费、复查上游）的请求。
// 按选中的记录操作时只填 request_ids；处理当前筛选下的全部记录时设 all=true，并带上与查询接口相同的筛选条件。
type billingCorrectionRecordsRequest struct {
	RequestIds     []string `json:"request_ids"`
	All            bool     `json:"all"`
	StartTimestamp int64    `json:"start_timestamp"`
	EndTimestamp   int64    `json:"end_timestamp"`
	Username       string   `json:"username"`
	TokenName      string   `json:"token_name"`
	ModelName      string   `json:"model_name"`
	Channel        int      `json:"channel"`
	Status         string   `json:"status"`
	Match          string   `json:"match"`
	Upstream       string   `json:"upstream"`
}

// billingCorrectionRuleRequest 只接收管理员可编辑的字段，时间戳与 ID 由服务端维护。
// upstream_access_token 留空表示沿用已保存的令牌。
type billingCorrectionRuleRequest struct {
	ChannelId             int    `json:"channel_id"`
	PromptTokensThreshold int    `json:"prompt_tokens_threshold"`
	Enabled               bool   `json:"enabled"`
	UpstreamCheckEnabled  bool   `json:"upstream_check_enabled"`
	UpstreamBaseUrl       string `json:"upstream_base_url"`
	UpstreamUserId        int    `json:"upstream_user_id"`
	UpstreamAccessToken   string `json:"upstream_access_token"`
	AutoRefund            bool   `json:"auto_refund"`
}

// billingCorrectionUpstreamTestRequest 用填写中的上游配置测试连接；令牌留空时用 rule_id 对应规则已保存的令牌。
type billingCorrectionUpstreamTestRequest struct {
	RuleId              int    `json:"rule_id"`
	ChannelId           int    `json:"channel_id"`
	UpstreamBaseUrl     string `json:"upstream_base_url"`
	UpstreamUserId      int    `json:"upstream_user_id"`
	UpstreamAccessToken string `json:"upstream_access_token"`
}

func GetBillingCorrectionRules(c *gin.Context) {
	rules, err := model.GetAllBillingCorrectionRules()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.FillBillingCorrectionRuleChannelNames(rules); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rules)
}

func CreateBillingCorrectionRule(c *gin.Context) {
	var request billingCorrectionRuleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserInputInvalid, map[string]any{"Error": err.Error()})
		return
	}
	rule := &model.BillingCorrectionRule{}
	if !applyBillingCorrectionRuleRequest(c, rule, request) {
		return
	}
	if err := rule.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.ReloadBillingCorrectionRules(); err != nil {
		common.SysError("failed to reload billing correction rules: " + err.Error())
	}
	rule.UpstreamAccessTokenSet = rule.UpstreamAccessToken != ""
	common.ApiSuccess(c, rule)
}

func UpdateBillingCorrectionRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionIdFormatError)
		return
	}
	var request billingCorrectionRuleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserInputInvalid, map[string]any{"Error": err.Error()})
		return
	}
	rule, err := model.GetBillingCorrectionRuleById(id)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionNotExists)
		return
	}
	if !applyBillingCorrectionRuleRequest(c, rule, request) {
		return
	}
	if err := rule.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.ReloadBillingCorrectionRules(); err != nil {
		common.SysError("failed to reload billing correction rules: " + err.Error())
	}
	rule.UpstreamAccessTokenSet = rule.UpstreamAccessToken != ""
	common.ApiSuccess(c, rule)
}

func DeleteBillingCorrectionRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionIdFormatError)
		return
	}
	if err := model.DeleteBillingCorrectionRuleById(id); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.ReloadBillingCorrectionRules(); err != nil {
		common.SysError("failed to reload billing correction rules: " + err.Error())
	}
	common.ApiSuccess(c, nil)
}

// applyBillingCorrectionRuleRequest 校验请求并写入目标规则，
// 校验失败时已经写好响应，返回 false 让调用方直接结束。
func applyBillingCorrectionRuleRequest(c *gin.Context, rule *model.BillingCorrectionRule, request billingCorrectionRuleRequest) bool {
	if request.ChannelId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionChannelIdInvalid)
		return false
	}
	if _, err := model.GetChannelById(request.ChannelId, false); err != nil {
		common.ApiErrorI18n(c, i18n.MsgChannelNotExists)
		return false
	}
	taken, err := model.IsBillingCorrectionChannelTaken(request.ChannelId, rule.Id)
	if err != nil {
		common.ApiError(c, err)
		return false
	}
	if taken {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionChannelTaken)
		return false
	}
	if request.PromptTokensThreshold < minBillingCorrectionThreshold || request.PromptTokensThreshold > maxBillingCorrectionThreshold {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionThresholdInvalid, map[string]any{
			"Min": minBillingCorrectionThreshold,
			"Max": maxBillingCorrectionThreshold,
		})
		return false
	}

	upstreamBaseUrl, ok := normalizeBillingCorrectionUpstreamBaseUrl(c, request.UpstreamBaseUrl)
	if !ok {
		return false
	}
	upstreamAccessToken := strings.TrimSpace(request.UpstreamAccessToken)
	if len(upstreamAccessToken) > maxBillingCorrectionUpstreamTokenLength {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionUpstreamTokenInvalid, map[string]any{"Max": maxBillingCorrectionUpstreamTokenLength})
		return false
	}
	if upstreamAccessToken == "" {
		upstreamAccessToken = rule.UpstreamAccessToken
	}
	if request.UpstreamCheckEnabled {
		if request.UpstreamUserId <= 0 {
			common.ApiErrorI18n(c, i18n.MsgBillingCorrectionUpstreamUserIdInvalid)
			return false
		}
		if upstreamAccessToken == "" {
			common.ApiErrorI18n(c, i18n.MsgBillingCorrectionUpstreamTokenInvalid, map[string]any{"Max": maxBillingCorrectionUpstreamTokenLength})
			return false
		}
	}

	// 换渠道或重新开启上游核对时重新计时，后台自动核对只处理这之后的请求。
	if request.UpstreamCheckEnabled && (!rule.UpstreamCheckEnabled || rule.ChannelId != request.ChannelId) {
		rule.UpstreamCheckSince = common.GetTimestamp()
	}
	rule.ChannelId = request.ChannelId
	rule.PromptTokensThreshold = request.PromptTokensThreshold
	rule.Enabled = request.Enabled
	rule.UpstreamCheckEnabled = request.UpstreamCheckEnabled
	rule.UpstreamBaseUrl = upstreamBaseUrl
	rule.UpstreamUserId = max(request.UpstreamUserId, 0)
	rule.UpstreamAccessToken = upstreamAccessToken
	rule.AutoRefund = request.UpstreamCheckEnabled && request.AutoRefund
	return true
}

// normalizeBillingCorrectionUpstreamBaseUrl 校验上游地址并去掉末尾的斜杠，留空表示用渠道的 Base URL。
// 校验失败时已经写好响应，返回 false。
func normalizeBillingCorrectionUpstreamBaseUrl(c *gin.Context, raw string) (string, bool) {
	baseUrl := strings.TrimRight(strings.TrimSpace(raw), "/")
	if baseUrl == "" {
		return "", true
	}
	parsed, err := url.Parse(baseUrl)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || len(baseUrl) > maxBillingCorrectionUpstreamBaseUrlLength {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionUpstreamBaseUrlInvalid)
		return "", false
	}
	return baseUrl, true
}

// GetBillingCorrectionRecords 在启用了规则的渠道上查询输出为 0、扣了费或已被规则修正的文本请求记录
// （包括规则启用前的），并统计其中已经扣费的额度。
func GetBillingCorrectionRecords(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	channel, _ := strconv.Atoi(c.Query("channel"))
	filter := service.BillingCorrectionRecordFilter{
		BillingCorrectionLogFilter: model.BillingCorrectionLogFilter{
			StartTimestamp: startTimestamp,
			EndTimestamp:   endTimestamp,
			Username:       c.Query("username"),
			TokenName:      c.Query("token_name"),
			ModelName:      c.Query("model_name"),
			ChannelId:      channel,
		},
		Status:   c.Query("status"),
		Match:    c.Query("match"),
		Upstream: c.Query("upstream"),
	}
	page, err := service.QueryBillingCorrectionRecords(filter, pageInfo.GetPage(), pageInfo.GetPageSize())
	if err != nil {
		respondBillingCorrectionRecordsError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

// bindBillingCorrectionRecordsRequest 解析批量处理记录的请求：选中的记录最多 limit 条，
// 或 all=true 时处理当前筛选下的全部记录。校验失败时已经写好响应，返回 false。
func bindBillingCorrectionRecordsRequest(c *gin.Context, limit int) (service.BillingCorrectionRecordFilter, bool) {
	var request billingCorrectionRecordsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserInputInvalid, map[string]any{"Error": err.Error()})
		return service.BillingCorrectionRecordFilter{}, false
	}
	if len(request.RequestIds) > limit {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionRecordsSelectionTooLarge, map[string]any{"Limit": limit})
		return service.BillingCorrectionRecordFilter{}, false
	}
	if len(request.RequestIds) > 0 {
		return service.BillingCorrectionRecordFilter{
			BillingCorrectionLogFilter: model.BillingCorrectionLogFilter{RequestIds: request.RequestIds},
		}, true
	}
	if !request.All {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionRecordsNoSelection)
		return service.BillingCorrectionRecordFilter{}, false
	}
	return service.BillingCorrectionRecordFilter{
		BillingCorrectionLogFilter: model.BillingCorrectionLogFilter{
			StartTimestamp: request.StartTimestamp,
			EndTimestamp:   request.EndTimestamp,
			Username:       request.Username,
			TokenName:      request.TokenName,
			ModelName:      request.ModelName,
			ChannelId:      request.Channel,
		},
		Status:   request.Status,
		Match:    request.Match,
		Upstream: request.Upstream,
	}, true
}

// MarkBillingCorrectionRecordsRefunded 把记录标记为已退费。只做标记，不退还任何额度。
func MarkBillingCorrectionRecordsRefunded(c *gin.Context) {
	filter, ok := bindBillingCorrectionRecordsRequest(c, maxBillingCorrectionRefundSelection)
	if !ok {
		return
	}
	operatorId := c.GetInt("id")
	result, err := service.MarkBillingCorrectionRecordsRefunded(filter, operatorId)
	if err != nil {
		respondBillingCorrectionRecordsError(c, err)
		return
	}
	common.SysLog(fmt.Sprintf("admin %d marked %d billing correction records as refunded, quota %d", operatorId, result.Marked, result.Quota))
	common.ApiSuccess(c, result)
}

// UnmarkBillingCorrectionRecordsRefunded 撤销选中记录的退费标记。已经补费的记录额度已退回用户，不能撤销。
func UnmarkBillingCorrectionRecordsRefunded(c *gin.Context) {
	var request billingCorrectionRecordsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserInputInvalid, map[string]any{"Error": err.Error()})
		return
	}
	if len(request.RequestIds) == 0 {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionRecordsNoSelection)
		return
	}
	if len(request.RequestIds) > maxBillingCorrectionRefundSelection {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionRecordsSelectionTooLarge, map[string]any{"Limit": maxBillingCorrectionRefundSelection})
		return
	}
	unmarked, err := model.DeleteBillingCorrectionRefunds(request.RequestIds)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.SysLog(fmt.Sprintf("admin %d removed the refunded mark from %d billing correction records", c.GetInt("id"), unmarked))
	common.ApiSuccess(c, gin.H{"unmarked": unmarked})
}

// CreditBillingCorrectionRecords 给记录补费：把扣掉的额度退回用户钱包，每条记录最多补一次。
func CreditBillingCorrectionRecords(c *gin.Context) {
	filter, ok := bindBillingCorrectionRecordsRequest(c, maxBillingCorrectionRefundSelection)
	if !ok {
		return
	}
	operatorId := c.GetInt("id")
	result, err := service.CreditBillingCorrectionRecords(filter, operatorId)
	if err != nil {
		respondBillingCorrectionRecordsError(c, err)
		return
	}
	common.SysLog(fmt.Sprintf("admin %d credited %d billing correction records back to users, quota %d, failed %d", operatorId, result.Credited, result.Quota, result.Failed))
	common.ApiSuccess(c, result)
}

// CheckBillingCorrectionRecordsUpstream 到上游 new-api 复查记录是否真的扣了费，只记录结果，不退费。
func CheckBillingCorrectionRecordsUpstream(c *gin.Context) {
	filter, ok := bindBillingCorrectionRecordsRequest(c, service.BillingCorrectionUpstreamCheckLimit)
	if !ok {
		return
	}
	result, err := service.CheckBillingCorrectionRecordsUpstream(c.Request.Context(), filter)
	if err != nil {
		respondBillingCorrectionRecordsError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

// TestBillingCorrectionUpstream 用填写中的上游配置读取上游当前用户，确认地址、用户 ID 和令牌可用。
func TestBillingCorrectionUpstream(c *gin.Context) {
	var request billingCorrectionUpstreamTestRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserInputInvalid, map[string]any{"Error": err.Error()})
		return
	}
	channel, err := model.GetChannelById(request.ChannelId, false)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgChannelNotExists)
		return
	}
	upstreamBaseUrl, ok := normalizeBillingCorrectionUpstreamBaseUrl(c, request.UpstreamBaseUrl)
	if !ok {
		return
	}
	if request.UpstreamUserId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionUpstreamUserIdInvalid)
		return
	}
	upstreamAccessToken := strings.TrimSpace(request.UpstreamAccessToken)
	if upstreamAccessToken == "" && request.RuleId > 0 {
		if saved, err := model.GetBillingCorrectionRuleById(request.RuleId); err == nil {
			upstreamAccessToken = saved.UpstreamAccessToken
		}
	}
	if upstreamAccessToken == "" || len(upstreamAccessToken) > maxBillingCorrectionUpstreamTokenLength {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionUpstreamTokenInvalid, map[string]any{"Max": maxBillingCorrectionUpstreamTokenLength})
		return
	}

	username, err := service.TestBillingCorrectionUpstream(c.Request.Context(), &model.BillingCorrectionRule{
		UpstreamBaseUrl:     upstreamBaseUrl,
		UpstreamUserId:      request.UpstreamUserId,
		UpstreamAccessToken: upstreamAccessToken,
	}, channel)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionUpstreamTestFailed, map[string]any{"Error": err.Error()})
		return
	}
	common.ApiSuccess(c, gin.H{"username": username})
}

func respondBillingCorrectionRecordsError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrBillingCorrectionRecordsTooMany) {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionRecordsTooMany, map[string]any{"Limit": service.BillingCorrectionRecordLimit})
		return
	}
	common.ApiError(c, err)
}
