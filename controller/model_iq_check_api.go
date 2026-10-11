package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/modeliqcheck"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_iq_check_setting"

	"github.com/gin-gonic/gin"
)

// maxModelIQCheckResultIds 限制单次批量读取的结果数，与检查项上限一致：
// 前端每次只取每张卡片的最新结果加上用户选中的历史结果。
const maxModelIQCheckResultIds = 50

// GetModelIQCheckStatus 是对外公开的智商检查页面数据接口，无需登录。
// 响应不包含分组名与渠道信息，历史只带元数据。
func GetModelIQCheckStatus(c *gin.Context) {
	status, err := modeliqcheck.Status()
	if err != nil {
		common.ApiErrorMsg(c, "failed to load model iq check status")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    status,
	})
}

// GetModelIQCheckResults 按 id 批量返回完整检查结果（ids=1,2,3），无需登录。
// 只返回当前启用检查项的结果，错误信息已脱敏。
func GetModelIQCheckResults(c *gin.Context) {
	ids := make([]int64, 0)
	seen := make(map[int64]struct{})
	for _, part := range strings.Split(c.Query("ids"), ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			return
		}
		if _, duplicated := seen[id]; duplicated {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
		if len(ids) > maxModelIQCheckResultIds {
			common.ApiErrorI18n(c, i18n.MsgBatchTooMany, map[string]any{"Max": maxModelIQCheckResultIds})
			return
		}
	}

	results, err := modeliqcheck.ResultDetails(ids)
	if err != nil {
		common.ApiErrorMsg(c, "failed to load model iq check results")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    results,
	})
}

// adminModelIQCheckView 是管理端的检查项视图，附带解析后的图标与最近一次运行情况。
// LastError 为原始错误信息，仅管理端可见。
type adminModelIQCheckView struct {
	modeliqcheck.Check
	ResolvedIcon   string `json:"resolved_icon"`
	LastStartedAt  int64  `json:"last_started_at"`
	LastFinishedAt int64  `json:"last_finished_at"`
	RunRequestedAt int64  `json:"run_requested_at"`
	LastStatus     string `json:"last_status"`
	LastCheckedAt  int64  `json:"last_checked_at"`
	LastError      string `json:"last_error"`
}

// GetModelIQCheckConfig 返回管理端智商检查配置。
func GetModelIQCheckConfig(c *gin.Context) {
	checks := modeliqcheck.Checks()
	checkIds := make([]string, 0, len(checks))
	for _, check := range checks {
		checkIds = append(checkIds, check.Id)
	}
	states, err := model.GetModelIQCheckStates(checkIds)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	history, err := model.GetModelIQCheckHistory(checkIds)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	stateById := make(map[string]model.ModelIQCheckState, len(states))
	for _, state := range states {
		stateById[state.CheckId] = state
	}
	// 历史按新到旧排列，每个检查项第一次出现的就是最近一次结果。
	latestById := make(map[string]model.ModelIQCheckResult, len(checks))
	for _, row := range history {
		if _, exists := latestById[row.CheckId]; !exists {
			latestById[row.CheckId] = row
		}
	}

	views := make([]adminModelIQCheckView, 0, len(checks))
	for _, check := range checks {
		state := stateById[check.Id]
		latest := latestById[check.Id]
		views = append(views, adminModelIQCheckView{
			Check:          check,
			ResolvedIcon:   modeliqcheck.ResolveIcon(check),
			LastStartedAt:  state.LastStartedAt,
			LastFinishedAt: state.LastFinishedAt,
			RunRequestedAt: state.RunRequestedAt,
			LastStatus:     modeliqcheck.ResultStatusName(latest.Status),
			LastCheckedAt:  latest.CheckedAt,
			LastError:      latest.ErrorMessage,
		})
	}

	setting := model_iq_check_setting.GetSetting()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"enabled":                 setting.Enabled,
			"history_limit":           setting.HistoryLimit,
			"request_timeout_seconds": setting.RequestTimeoutSeconds,
			"checks":                  views,
		},
	})
}

// updateModelIQCheckConfigRequest 用指针区分「未提交」与「提交了零值」，让各项独立更新。
type updateModelIQCheckConfigRequest struct {
	Enabled               *bool                 `json:"enabled"`
	Checks                *[]modeliqcheck.Check `json:"checks"`
	HistoryLimit          *int                  `json:"history_limit"`
	RequestTimeoutSeconds *int                  `json:"request_timeout_seconds"`
}

// UpdateModelIQCheckConfig 校验并保存管理端提交的智商检查配置。
// 走独立的管理端接口而不是 root-only 的 /api/option/，拥有渠道写权限的管理员即可维护。
// 新增、重新启用，或分组、模型、提示词、输出上限有变化的启用检查项会被立即排队重跑，
// 让页面尽快展示与新配置对应的回答。
func UpdateModelIQCheckConfig(c *gin.Context) {
	var request updateModelIQCheckConfigRequest
	if err := common.UnmarshalBodyReusable(c, &request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	updates := map[string]string{}
	auditParams := map[string]interface{}{}

	var submitted []modeliqcheck.Check
	if request.Checks != nil {
		groupExists, modelInGroup := groupModelValidators()
		normalized, err := modeliqcheck.ValidateChecks(*request.Checks, groupExists, modelInGroup)
		if err != nil {
			var validationErr *modeliqcheck.ValidationError
			if errors.As(err, &validationErr) {
				common.ApiErrorI18n(c, validationErr.Key, validationErr.Params)
				return
			}
			common.ApiError(c, err)
			return
		}
		// 图标缺省时按模型自动解析，保证页面总能画出供应商图标。
		for index := range normalized {
			if normalized[index].Icon == "" {
				normalized[index].Icon = model.ResolveModelIconKey(normalized[index].Model)
			}
		}
		payload, err := common.Marshal(normalized)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		updates[model_iq_check_setting.ChecksOptionKey] = string(payload)
		auditParams["check_count"] = len(normalized)
		submitted = normalized
	}

	if request.HistoryLimit != nil {
		historyLimit := *request.HistoryLimit
		if historyLimit < model_iq_check_setting.MinHistoryLimit || historyLimit > model_iq_check_setting.MaxHistoryLimit {
			common.ApiErrorI18n(c, i18n.MsgModelIQCheckHistoryLimitInvalid, map[string]any{
				"Min": model_iq_check_setting.MinHistoryLimit,
				"Max": model_iq_check_setting.MaxHistoryLimit,
			})
			return
		}
		updates[model_iq_check_setting.HistoryLimitOptionKey] = strconv.Itoa(historyLimit)
		auditParams["history_limit"] = historyLimit
	}

	if request.RequestTimeoutSeconds != nil {
		timeoutSeconds := *request.RequestTimeoutSeconds
		if timeoutSeconds < model_iq_check_setting.MinRequestTimeoutSeconds || timeoutSeconds > model_iq_check_setting.MaxRequestTimeoutSeconds {
			common.ApiErrorI18n(c, i18n.MsgModelIQCheckRequestTimeoutInvalid, map[string]any{
				"Min": model_iq_check_setting.MinRequestTimeoutSeconds,
				"Max": model_iq_check_setting.MaxRequestTimeoutSeconds,
			})
			return
		}
		updates[model_iq_check_setting.RequestTimeoutOptionKey] = strconv.Itoa(timeoutSeconds)
		auditParams["request_timeout_seconds"] = timeoutSeconds
	}

	if request.Enabled != nil {
		updates[model_iq_check_setting.EnabledOptionKey] = common.Interface2String(*request.Enabled)
		auditParams["enabled"] = *request.Enabled
	}

	previousById := make(map[string]modeliqcheck.Check)
	for _, check := range modeliqcheck.Checks() {
		previousById[check.Id] = check
	}

	// 一次事务写入，避免开关已生效但检查项未保存的中间状态。
	if err := model.UpdateOptionsBulk(updates); err != nil {
		common.ApiError(c, err)
		return
	}

	if request.Checks != nil {
		keptIds := make([]string, 0, len(submitted))
		for _, check := range submitted {
			keptIds = append(keptIds, check.Id)
		}
		if err := model.DeleteModelIQCheckDataExcept(keptIds); err != nil {
			common.SysError("failed to clean up removed model iq checks: " + err.Error())
		}

		// 停用的检查项不排队（不会运行，也不应显示为排队中）；重新启用时视同变更，
		// 以免停用期间修改过的提示词要等满一个间隔才有新回答。
		now := time.Now().Unix()
		requested := 0
		for _, check := range submitted {
			if !check.Enabled {
				continue
			}
			previous, existed := previousById[check.Id]
			if existed && previous.Enabled && previous.Group == check.Group && previous.Model == check.Model &&
				previous.Prompt == check.Prompt && previous.MaxTokens == check.MaxTokens {
				continue
			}
			if err := model.RequestModelIQCheckRun(check.Id, now); err != nil {
				common.SysError("failed to request model iq check run: " + err.Error())
				continue
			}
			requested++
		}
		if requested > 0 && model_iq_check_setting.IsEnabled() {
			if _, _, err := service.EnqueueSystemTask(model.SystemTaskTypeModelIQCheck, nil); err != nil {
				common.SysError("failed to enqueue model iq check task: " + err.Error())
			}
		}
	}

	modeliqcheck.InvalidateStatusCache()
	recordManageAudit(c, "model_iq_check.update", auditParams)

	GetModelIQCheckConfig(c)
}

// RunModelIQCheck 让指定检查项尽快重新运行。请求先落到调度状态，
// 已有活跃任务时由调度器在下一轮处理，因此无需等待当前任务结束。
func RunModelIQCheck(c *gin.Context) {
	check, found := modeliqcheck.FindCheck(c.Param("id"))
	if !found {
		common.ApiErrorI18n(c, i18n.MsgModelIQCheckNotFound)
		return
	}
	if !model_iq_check_setting.IsEnabled() {
		common.ApiErrorI18n(c, i18n.MsgModelIQCheckFeatureDisabled)
		return
	}
	if !check.Enabled {
		common.ApiErrorI18n(c, i18n.MsgModelIQCheckCheckDisabled)
		return
	}

	if err := model.RequestModelIQCheckRun(check.Id, time.Now().Unix()); err != nil {
		common.ApiError(c, err)
		return
	}
	// 入队失败不影响本次请求：调度器会发现这条到期请求并自行建任务。
	if _, _, err := service.EnqueueSystemTask(model.SystemTaskTypeModelIQCheck, nil); err != nil {
		common.SysError("failed to enqueue model iq check task: " + err.Error())
	}
	modeliqcheck.InvalidateStatusCache()
	recordManageAudit(c, "model_iq_check.run", map[string]interface{}{
		"check_id": check.Id,
	})
	common.ApiSuccessI18n(c, i18n.MsgModelIQCheckRunQueued, nil)
}
