package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// maxErrorMessageOverrideMatchLength �?match_substring 的列宽保持一致�?
const maxErrorMessageOverrideMatchLength = 512

// errorMessageOverrideRequest 只接收管理员可编辑的字段�?
// 时间戳与 ID 由服务端维护�?
type errorMessageOverrideRequest struct {
	MatchSubstring     string `json:"match_substring"`
	ReplacementMessage string `json:"replacement_message"`
	ChannelId          int    `json:"channel_id"`
	Priority           int    `json:"priority"`
	Enabled            bool   `json:"enabled"`
}

func GetErrorMessageOverrides(c *gin.Context) {
	overrides, err := model.GetAllErrorMessageOverrides()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.FillErrorMessageOverrideChannelNames(overrides); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, overrides)
}

func CreateErrorMessageOverride(c *gin.Context) {
	var request errorMessageOverrideRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserInputInvalid, map[string]any{"Error": err.Error()})
		return
	}
	override := &model.ErrorMessageOverride{}
	if !applyErrorMessageOverrideRequest(c, override, request) {
		return
	}
	if err := override.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.ReloadErrorMessageOverrides(); err != nil {
		common.SysError("failed to reload error message overrides: " + err.Error())
	}
	common.ApiSuccess(c, override)
}

func UpdateErrorMessageOverride(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgErrorMessageOverrideIdFormatError)
		return
	}
	var request errorMessageOverrideRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserInputInvalid, map[string]any{"Error": err.Error()})
		return
	}
	override, err := model.GetErrorMessageOverrideById(id)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgErrorMessageOverrideNotExists)
		return
	}
	if !applyErrorMessageOverrideRequest(c, override, request) {
		return
	}
	if err := override.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.ReloadErrorMessageOverrides(); err != nil {
		common.SysError("failed to reload error message overrides: " + err.Error())
	}
	common.ApiSuccess(c, override)
}

func DeleteErrorMessageOverride(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgErrorMessageOverrideIdFormatError)
		return
	}
	if err := model.DeleteErrorMessageOverrideById(id); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.ReloadErrorMessageOverrides(); err != nil {
		common.SysError("failed to reload error message overrides: " + err.Error())
	}
	common.ApiSuccess(c, nil)
}

// applyErrorMessageOverrideRequest 校验请求并写入目标规则，
// 校验失败时已经写好响应，返回 false 让调用方直接结束�?
func applyErrorMessageOverrideRequest(c *gin.Context, override *model.ErrorMessageOverride, request errorMessageOverrideRequest) bool {
	match := strings.TrimSpace(request.MatchSubstring)
	if match == "" {
		common.ApiErrorI18n(c, i18n.MsgErrorMessageOverrideMatchRequired)
		return false
	}
	if len([]rune(match)) > maxErrorMessageOverrideMatchLength {
		common.ApiErrorI18n(c, i18n.MsgErrorMessageOverrideMatchTooLong, map[string]any{"Max": maxErrorMessageOverrideMatchLength})
		return false
	}
	replacement := strings.TrimSpace(request.ReplacementMessage)
	if replacement == "" {
		common.ApiErrorI18n(c, i18n.MsgErrorMessageOverrideReplaceRequired)
		return false
	}
	if request.ChannelId < 0 {
		common.ApiErrorI18n(c, i18n.MsgErrorMessageOverrideChannelIdInvalid)
		return false
	}
	if request.ChannelId > 0 {
		if _, err := model.GetChannelById(request.ChannelId, false); err != nil {
			common.ApiErrorI18n(c, i18n.MsgChannelNotExists)
			return false
		}
	}

	override.MatchSubstring = match
	override.ReplacementMessage = replacement
	override.ChannelId = request.ChannelId
	override.Priority = request.Priority
	override.Enabled = request.Enabled
	return true
}
