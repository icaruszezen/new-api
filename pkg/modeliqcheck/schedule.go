package modeliqcheck

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/model_iq_check_setting"
)

// IsDue 判断检查项此刻是否应当运行：管理员请求过立即运行、从未运行过，
// 或距离最近一次开始已超过检查间隔。以开始时间计间隔，长耗时检查不会把节奏越拖越慢。
func IsDue(check Check, state model.ModelIQCheckState, now int64) bool {
	if !check.Enabled {
		return false
	}
	if state.RunRequestedAt > state.LastStartedAt {
		return true
	}
	if state.LastStartedAt == 0 {
		return true
	}
	return now-state.LastStartedAt >= int64(intervalMinutesOf(check))*60
}

// DueChecks 返回此刻到期的启用检查项；功能关闭时返回空。
func DueChecks(now int64) ([]Check, error) {
	if !model_iq_check_setting.IsEnabled() {
		return nil, nil
	}
	checks := EnabledChecks()
	if len(checks) == 0 {
		return nil, nil
	}

	checkIds := make([]string, 0, len(checks))
	for _, check := range checks {
		checkIds = append(checkIds, check.Id)
	}
	states, err := model.GetModelIQCheckStates(checkIds)
	if err != nil {
		return nil, err
	}
	stateById := make(map[string]model.ModelIQCheckState, len(states))
	for _, state := range states {
		stateById[state.CheckId] = state
	}

	due := make([]Check, 0, len(checks))
	for _, check := range checks {
		if IsDue(check, stateById[check.Id], now) {
			due = append(due, check)
		}
	}
	return due, nil
}

// HasDueChecks 供调度器判断是否需要创建任务行，数据库错误时按「无到期」处理并记录日志。
func HasDueChecks(now int64) bool {
	due, err := DueChecks(now)
	if err != nil {
		common.SysError("failed to load model iq check states: " + err.Error())
		return false
	}
	return len(due) > 0
}
