package controller

import (
	"errors"
	"fmt"
	"strconv"

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

// maxBillingCorrectionRefundSelection 是按记录标记或撤销时一次最多处理的请求数。
const maxBillingCorrectionRefundSelection = 1000

// billingCorrectionRefundRequest 是标记 / 撤销退费的请求。按选中的记录操作时只填 request_ids；
// 标记当前筛选下的全部记录时设 all=true，并带上与查询接口相同的筛选条件。
type billingCorrectionRefundRequest struct {
	RequestIds     []string `json:"request_ids"`
	All            bool     `json:"all"`
	StartTimestamp int64    `json:"start_timestamp"`
	EndTimestamp   int64    `json:"end_timestamp"`
	Username       string   `json:"username"`
	TokenName      string   `json:"token_name"`
	ModelName      string   `json:"model_name"`
	Channel        int      `json:"channel"`
}

// billingCorrectionRuleRequest 只接收管理员可编辑的字段，时间戳与 ID 由服务端维护。
type billingCorrectionRuleRequest struct {
	ChannelId             int  `json:"channel_id"`
	PromptTokensThreshold int  `json:"prompt_tokens_threshold"`
	Enabled               bool `json:"enabled"`
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

	rule.ChannelId = request.ChannelId
	rule.PromptTokensThreshold = request.PromptTokensThreshold
	rule.Enabled = request.Enabled
	return true
}

// GetBillingCorrectionRecords 在启用了规则的渠道上查询符合错误计费条件的消费记录（包括规则启用前的），
// 并统计其中已经扣费的额度。
func GetBillingCorrectionRecords(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	channel, _ := strconv.Atoi(c.Query("channel"))
	filter := model.BillingCorrectionLogFilter{
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		Username:       c.Query("username"),
		TokenName:      c.Query("token_name"),
		ModelName:      c.Query("model_name"),
		ChannelId:      channel,
	}
	page, err := service.QueryBillingCorrectionRecords(filter, c.Query("status"), pageInfo.GetPage(), pageInfo.GetPageSize())
	if err != nil {
		respondBillingCorrectionRecordsError(c, err)
		return
	}
	common.ApiSuccess(c, page)
}

// MarkBillingCorrectionRecordsRefunded 把错误计费记录标记为已退费。只做标记，不退还任何额度。
func MarkBillingCorrectionRecordsRefunded(c *gin.Context) {
	var request billingCorrectionRefundRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserInputInvalid, map[string]any{"Error": err.Error()})
		return
	}
	filter := model.BillingCorrectionLogFilter{RequestIds: request.RequestIds}
	if len(request.RequestIds) == 0 {
		if !request.All {
			common.ApiErrorI18n(c, i18n.MsgBillingCorrectionRecordsNoSelection)
			return
		}
		filter = model.BillingCorrectionLogFilter{
			StartTimestamp: request.StartTimestamp,
			EndTimestamp:   request.EndTimestamp,
			Username:       request.Username,
			TokenName:      request.TokenName,
			ModelName:      request.ModelName,
			ChannelId:      request.Channel,
		}
	} else if len(request.RequestIds) > maxBillingCorrectionRefundSelection {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionRecordsSelectionTooLarge, map[string]any{"Limit": maxBillingCorrectionRefundSelection})
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

// UnmarkBillingCorrectionRecordsRefunded 撤销选中记录的退费标记。
func UnmarkBillingCorrectionRecordsRefunded(c *gin.Context) {
	var request billingCorrectionRefundRequest
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

func respondBillingCorrectionRecordsError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrBillingCorrectionRecordsTooMany) {
		common.ApiErrorI18n(c, i18n.MsgBillingCorrectionRecordsTooMany, map[string]any{"Limit": service.BillingCorrectionRecordLimit})
		return
	}
	common.ApiError(c, err)
}
