package modeliqcheck

import (
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/model_iq_check_setting"
)

// statusCacheTTL 保护公开接口：页面会被未登录访客轮询，而结果最快也要几分钟才变化一次。
const statusCacheTTL = 10 * time.Second

// runningGraceSeconds 让「运行中」在执行节点崩溃后自动失效：
// 开始后超过请求超时再加该余量仍未结束，即视为已中断。
const runningGraceSeconds = 120

var statusCache struct {
	mu        sync.Mutex
	view      StatusView
	expiresAt time.Time
}

// Status 组装公开页面数据，只包含启用中的检查项与历史元数据。
func Status() (StatusView, error) {
	setting := model_iq_check_setting.GetSetting()
	if !setting.Enabled {
		return StatusView{Enabled: false, Checks: []CheckView{}}, nil
	}

	statusCache.mu.Lock()
	defer statusCache.mu.Unlock()
	if time.Now().Before(statusCache.expiresAt) {
		return statusCache.view, nil
	}

	view := StatusView{Enabled: true, Checks: []CheckView{}}
	checks := EnabledChecks()
	if len(checks) > 0 {
		checkIds := make([]string, 0, len(checks))
		for index := range checks {
			checkIds = append(checkIds, checks[index].Id)
			// 未显式配置图标的检查项按模型名推断，卡片总能显示供应商图标。
			checks[index].Icon = ResolveIcon(checks[index])
		}
		states, err := model.GetModelIQCheckStates(checkIds)
		if err != nil {
			return StatusView{}, err
		}
		history, err := model.GetModelIQCheckHistory(checkIds)
		if err != nil {
			return StatusView{}, err
		}
		view.Checks = buildCheckViews(checks, states, history, time.Now().Unix(), setting)
	}

	statusCache.view = view
	statusCache.expiresAt = time.Now().Add(statusCacheTTL)
	return view, nil
}

// InvalidateStatusCache 在配置变更或本节点写入新结果后，让公开接口立即反映最新数据。
func InvalidateStatusCache() {
	statusCache.mu.Lock()
	statusCache.expiresAt = time.Time{}
	statusCache.mu.Unlock()
}

// buildCheckViews 把配置、调度状态与历史元数据拼成卡片视图，不访问数据库。
// checks 的图标需已解析；history 需按新到旧排列（GetModelIQCheckHistory 的顺序），
// 每个检查项最多保留 HistoryLimit 条。
func buildCheckViews(checks []Check, states []model.ModelIQCheckState, history []model.ModelIQCheckResult, now int64, setting model_iq_check_setting.ModelIQCheckSetting) []CheckView {
	stateById := make(map[string]model.ModelIQCheckState, len(states))
	for _, state := range states {
		stateById[state.CheckId] = state
	}
	historyById := make(map[string][]HistoryItemView, len(checks))
	for _, row := range history {
		items := historyById[row.CheckId]
		if len(items) >= setting.HistoryLimit {
			continue
		}
		historyById[row.CheckId] = append(items, HistoryItemView{
			Id:               row.Id,
			CheckedAt:        row.CheckedAt,
			Status:           ResultStatusName(row.Status),
			DurationMs:       row.DurationMs,
			CompletionTokens: row.CompletionTokens,
		})
	}

	runningWindow := int64(setting.RequestTimeoutSeconds + runningGraceSeconds)
	views := make([]CheckView, 0, len(checks))
	for _, check := range checks {
		state := stateById[check.Id]
		items := historyById[check.Id]
		if items == nil {
			items = []HistoryItemView{}
		}
		view := CheckView{
			Id:              check.Id,
			Name:            check.Name,
			Model:           check.Model,
			Icon:            check.Icon,
			Prompt:          check.Prompt,
			IntervalMinutes: intervalMinutesOf(check),
			Running:         state.LastStartedAt > state.LastFinishedAt && now-state.LastStartedAt <= runningWindow,
			Queued:          state.RunRequestedAt > state.LastStartedAt,
			History:         items,
		}
		if len(items) > 0 {
			view.LastCheckedAt = items[0].CheckedAt
		}
		if state.LastStartedAt > 0 {
			view.NextCheckAt = state.LastStartedAt + int64(view.IntervalMinutes)*60
		}
		views = append(views, view)
	}
	return views
}

// ResultDetails 返回指定结果的完整内容。只返回当前启用检查项的结果，
// 已停用或已删除检查项的历史不会经公开接口泄露；不存在的 id 直接忽略。
func ResultDetails(ids []int64) ([]ResultView, error) {
	views := []ResultView{}
	if len(ids) == 0 || !model_iq_check_setting.IsEnabled() {
		return views, nil
	}
	checks := EnabledChecks()
	if len(checks) == 0 {
		return views, nil
	}
	allowedCheckIds := make([]string, 0, len(checks))
	for _, check := range checks {
		allowedCheckIds = append(allowedCheckIds, check.Id)
	}

	rows, err := model.GetModelIQCheckResultsByIds(ids, allowedCheckIds)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		views = append(views, publicResultView(row))
	}
	return views, nil
}

// publicResultView 把结果行转换为公开视图：不含渠道与原始错误，失败原因只给分类。
func publicResultView(row model.ModelIQCheckResult) ResultView {
	return ResultView{
		Id:                 row.Id,
		CheckId:            row.CheckId,
		CheckedAt:          row.CheckedAt,
		Status:             ResultStatusName(row.Status),
		Model:              row.ModelName,
		Prompt:             row.Prompt,
		Content:            row.Content,
		ContentTruncated:   row.ContentTruncated,
		Reasoning:          row.Reasoning,
		ReasoningTruncated: row.ReasoningTruncated,
		FinishReason:       row.FinishReason,
		ErrorType:          row.ErrorType,
		DurationMs:         row.DurationMs,
		PromptTokens:       row.PromptTokens,
		CompletionTokens:   row.CompletionTokens,
		ReasoningTokens:    row.ReasoningTokens,
	}
}

// ResultStatusName 把落库状态映射为接口中的状态字符串，未知值返回空串。
func ResultStatusName(status int) string {
	switch status {
	case model.ModelIQCheckStatusSuccess:
		return ResultStatusSuccess
	case model.ModelIQCheckStatusFailed:
		return ResultStatusFailed
	default:
		return ""
	}
}

// ResolveIcon 返回检查项的图标；未显式配置时按模型名推断供应商图标。
func ResolveIcon(check Check) string {
	if check.Icon != "" {
		return check.Icon
	}
	return model.ResolveModelIconKey(check.Model)
}
