package controller

import (
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
