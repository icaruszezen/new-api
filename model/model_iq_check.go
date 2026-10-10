package model

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 模型智商检查单次结果状态。
const (
	ModelIQCheckStatusSuccess = 1
	ModelIQCheckStatusFailed  = 2
)

// ModelIQCheckResult 是一次智商检查的完整结果。
// Prompt / ModelName 是运行时快照：管理员之后修改提示词，历史结果仍对应当时的问题。
// Prompt / Content / Reasoning / ErrorMessage 不加 type 标签，由 GORM 映射为
// MySQL longtext、PostgreSQL / SQLite text，避免 MySQL TEXT 的 64KB 上限截断网页输出。
type ModelIQCheckResult struct {
	Id                 int64  `json:"id" gorm:"primaryKey"`
	CheckId            string `json:"check_id" gorm:"size:64;index:idx_iq_result_check_id,priority:1"`
	CheckedAt          int64  `json:"checked_at" gorm:"bigint;index"`
	Status             int    `json:"status"`
	ModelName          string `json:"model_name" gorm:"size:255"`
	Prompt             string `json:"prompt"`
	Content            string `json:"content"`
	ContentTruncated   bool   `json:"content_truncated"`
	Reasoning          string `json:"reasoning"`
	ReasoningTruncated bool   `json:"reasoning_truncated"`
	FinishReason       string `json:"finish_reason" gorm:"size:64"`
	// ErrorMessage 保存原始错误，仅管理端可见：上游错误可能带组织 ID、密钥尾号或内网地址，
	// 公开视图只返回 ErrorType 分类。
	ErrorMessage     string `json:"error_message"`
	ErrorType        string `json:"error_type" gorm:"size:32"`
	DurationMs       int64  `json:"duration_ms"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	ReasoningTokens  int    `json:"reasoning_tokens"`
	// ChannelId 仅供管理端排查，公开视图不返回。
	ChannelId int `json:"channel_id"`
}

func (ModelIQCheckResult) TableName() string {
	return "model_iq_check_results"
}

// ModelIQCheckState 每个检查项一行，跨实例共享调度状态：
// 最近一次开始 / 结束时间决定是否到期与「运行中」展示，RunRequestedAt 记录管理员的立即运行请求。
type ModelIQCheckState struct {
	CheckId        string `json:"check_id" gorm:"primaryKey;size:64"`
	LastStartedAt  int64  `json:"last_started_at" gorm:"bigint;default:0"`
	LastFinishedAt int64  `json:"last_finished_at" gorm:"bigint;default:0"`
	RunRequestedAt int64  `json:"run_requested_at" gorm:"bigint;default:0"`
}

func (ModelIQCheckState) TableName() string {
	return "model_iq_check_states"
}

// modelIQCheckHistoryColumns 是历史列表需要的元数据列。
// 历史列表随公开状态页高频读取，刻意不读 prompt / content / reasoning 这些大字段。
var modelIQCheckHistoryColumns = []string{
	"id",
	"check_id",
	"checked_at",
	"status",
	"duration_ms",
	"completion_tokens",
	"error_message",
}

// SaveModelIQCheckResult 在同一事务中写入结果，并只保留该检查项最新的 historyLimit 条。
func SaveModelIQCheckResult(result *ModelIQCheckResult, historyLimit int) error {
	if result == nil || result.CheckId == "" {
		return nil
	}
	if historyLimit < 1 {
		historyLimit = 1
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(result).Error; err != nil {
			return err
		}
		// 找到第 historyLimit 新的一条，删除比它更旧的；不足 historyLimit 条时无需清理。
		var boundaryIds []int64
		if err := tx.Model(&ModelIQCheckResult{}).
			Where("check_id = ?", result.CheckId).
			Order("id DESC").
			Offset(historyLimit-1).
			Limit(1).
			Pluck("id", &boundaryIds).Error; err != nil {
			return err
		}
		if len(boundaryIds) == 0 {
			return nil
		}
		return tx.Where("check_id = ? AND id < ?", result.CheckId, boundaryIds[0]).
			Delete(&ModelIQCheckResult{}).Error
	})
}

// GetModelIQCheckHistory 返回指定检查项的历史元数据，按新到旧排序。
func GetModelIQCheckHistory(checkIds []string) ([]ModelIQCheckResult, error) {
	if len(checkIds) == 0 {
		return nil, nil
	}
	var results []ModelIQCheckResult
	err := DB.Model(&ModelIQCheckResult{}).
		Select(modelIQCheckHistoryColumns).
		Where("check_id IN ?", checkIds).
		Order("id DESC").
		Find(&results).Error
	return results, err
}

// GetModelIQCheckResultsByIds 读取完整结果，只返回属于 allowedCheckIds 的行，
// 避免已停用或已删除检查项的历史结果经公开接口泄露。
func GetModelIQCheckResultsByIds(ids []int64, allowedCheckIds []string) ([]ModelIQCheckResult, error) {
	if len(ids) == 0 || len(allowedCheckIds) == 0 {
		return nil, nil
	}
	var results []ModelIQCheckResult
	err := DB.Model(&ModelIQCheckResult{}).
		Where("id IN ? AND check_id IN ?", ids, allowedCheckIds).
		Order("id DESC").
		Find(&results).Error
	return results, err
}

// GetModelIQCheckStates 读取指定检查项的调度状态。
func GetModelIQCheckStates(checkIds []string) ([]ModelIQCheckState, error) {
	if len(checkIds) == 0 {
		return nil, nil
	}
	var states []ModelIQCheckState
	err := DB.Model(&ModelIQCheckState{}).Where("check_id IN ?", checkIds).Find(&states).Error
	return states, err
}

// MarkModelIQCheckStarted 记录一次检查开始，只更新 last_started_at。
func MarkModelIQCheckStarted(checkId string, ts int64) error {
	return upsertModelIQCheckStateColumn(ModelIQCheckState{CheckId: checkId, LastStartedAt: ts}, "last_started_at")
}

// MarkModelIQCheckFinished 记录一次检查结束，只更新 last_finished_at。
func MarkModelIQCheckFinished(checkId string, ts int64) error {
	return upsertModelIQCheckStateColumn(ModelIQCheckState{CheckId: checkId, LastFinishedAt: ts}, "last_finished_at")
}

// RequestModelIQCheckRun 记录一次立即运行请求，调度器发现它晚于最近一次开始时间即视为到期。
func RequestModelIQCheckRun(checkId string, ts int64) error {
	return upsertModelIQCheckStateColumn(ModelIQCheckState{CheckId: checkId, RunRequestedAt: ts}, "run_requested_at")
}

// upsertModelIQCheckStateColumn 插入状态行，冲突时只覆盖指定列，
// 让开始、结束与立即运行请求互不覆盖。
func upsertModelIQCheckStateColumn(state ModelIQCheckState, column string) error {
	if state.CheckId == "" {
		return nil
	}
	return DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "check_id"}},
		DoUpdates: clause.AssignmentColumns([]string{column}),
	}).Create(&state).Error
}

// DeleteModelIQCheckDataExcept 删除不在 checkIds 中的检查项的结果与状态。
// 空列表表示管理员删除了全部检查项，会清空两张表；调用方只能传入校验通过的完整配置。
func DeleteModelIQCheckDataExcept(checkIds []string) error {
	condition, args := "check_id NOT IN ?", []interface{}{checkIds}
	if len(checkIds) == 0 {
		condition, args = "1 = 1", nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(condition, args...).Delete(&ModelIQCheckResult{}).Error; err != nil {
			return err
		}
		return tx.Where(condition, args...).Delete(&ModelIQCheckState{}).Error
	})
}
