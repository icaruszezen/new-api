package modeliqcheck

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/model_iq_check_setting"
)

// runningGraceSeconds 让「运行中」在执行节点崩溃后自动失效：
// 开始后超过请求超时再加该余量仍未结束，即视为已中断。
const runningGraceSeconds = 120

// runningWindowSeconds 是一次检查从开始到被视为已中断的时长。
func runningWindowSeconds(setting model_iq_check_setting.ModelIQCheckSetting) int64 {
	return int64(setting.RequestTimeoutSeconds + runningGraceSeconds)
}

// IsDue 判断检查项此刻是否应当运行：管理员请求过立即运行、从未运行过、
// 上一次运行已中断（开始后超过 runningWindow 秒仍未结束），或距离最近一次开始已超过检查间隔。
// 以开始时间计间隔，长耗时检查不会把节奏越拖越慢。
func IsDue(check Check, state model.ModelIQCheckState, now int64, runningWindow int64) bool {
	if !check.Enabled {
		return false
	}
	if state.RunRequestedAt > state.LastStartedAt {
		return true
	}
	if state.LastStartedAt == 0 {
		return true
	}
	// 执行节点崩溃、重启或丢失任务租约时不会写入结束时间，这次运行没有结果，应当补跑。
	if state.LastStartedAt > state.LastFinishedAt && now-state.LastStartedAt > runningWindow {
		return true
	}
	return now-state.LastStartedAt >= int64(intervalMinutesOf(check))*60
}

// DueChecks 返回此刻到期的启用检查项；功能关闭时返回空。
func DueChecks(now int64) ([]Check, error) {
	setting := model_iq_check_setting.GetSetting()
	if !setting.Enabled {
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

	runningWindow := runningWindowSeconds(setting)
	due := make([]Check, 0, len(checks))
	for _, check := range checks {
		if IsDue(check, stateById[check.Id], now, runningWindow) {
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
