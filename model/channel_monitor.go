package model

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 渠道监控 beat 状态，与前端状态条配色一一对应。
const (
	ChannelMonitorStatusUp   = 1
	ChannelMonitorStatusSlow = 2
	ChannelMonitorStatusDown = 3
)

// beat 来源，用于区分真实用户流量与系统主动探测。
const (
	ChannelMonitorSourceUser  = 1
	ChannelMonitorSourceProbe = 2
)

// ChannelMonitorBeat 是状态条上的一格。BucketTs 已按采样窗口对齐，
// 同一 (monitor_id, bucket_ts) 只保留一条记录。
type ChannelMonitorBeat struct {
	Id        int64  `json:"id" gorm:"primaryKey"`
	MonitorId string `json:"monitor_id" gorm:"size:64;uniqueIndex:idx_cm_beat_monitor_bucket,priority:1"`
	BucketTs  int64  `json:"bucket_ts" gorm:"uniqueIndex:idx_cm_beat_monitor_bucket,priority:2;index:idx_cm_beat_bucket_ts"`
	Status    int    `json:"status" gorm:"default:0"`
	TtftMs    int    `json:"ttft_ms" gorm:"default:0"`
	ChannelId int    `json:"channel_id" gorm:"default:0"`
	Source    int    `json:"source" gorm:"default:0"`
}

func (ChannelMonitorBeat) TableName() string {
	return "channel_monitor_beats"
}

// ChannelMonitorStat 是按小时聚合的可用性汇总，用于计算多天可用性而无需保留全部 beat。
type ChannelMonitorStat struct {
	Id        int64  `json:"id" gorm:"primaryKey"`
	MonitorId string `json:"monitor_id" gorm:"size:64;uniqueIndex:idx_cm_stat_monitor_hour,priority:1"`
	HourTs    int64  `json:"hour_ts" gorm:"uniqueIndex:idx_cm_stat_monitor_hour,priority:2;index:idx_cm_stat_hour_ts"`
	Total     int64  `json:"total" gorm:"default:0"`
	UpCount   int64  `json:"up_count" gorm:"default:0"`
	SlowCount int64  `json:"slow_count" gorm:"default:0"`
	DownCount int64  `json:"down_count" gorm:"default:0"`
	TtftSumMs int64  `json:"ttft_sum_ms" gorm:"default:0"`
	TtftCount int64  `json:"ttft_count" gorm:"default:0"`
}

func (ChannelMonitorStat) TableName() string {
	return "channel_monitor_stats"
}

// ChannelMonitorState 每个监控项一行，保存端点 ping 与最近一次 beat/探测时间。
// 这些字段跨实例共享，使非 master 节点也能展示 ping，master 节点也能判断空闲。
// ResetAt 是跨实例 tombstone：其它节点不得把该时间点之前的热桶再写回。
type ChannelMonitorState struct {
	MonitorId     string `json:"monitor_id" gorm:"primaryKey;size:64"`
	PingMs        int    `json:"ping_ms" gorm:"default:0"`
	PingUpdatedAt int64  `json:"ping_updated_at" gorm:"bigint;default:0"`
	LastBeatTs    int64  `json:"last_beat_ts" gorm:"bigint;default:0"`
	LastProbeTs   int64  `json:"last_probe_ts" gorm:"bigint;default:0"`
	ChannelId     int    `json:"channel_id" gorm:"default:0"`
	ResetAt       int64  `json:"reset_at" gorm:"bigint;default:0"`
}

func (ChannelMonitorState) TableName() string {
	return "channel_monitor_state"
}

// InsertChannelMonitorBeats 批量写入 beat。多实例可能同时刷同一个采样桶，
// 先写入者获胜，冲突直接忽略。
func InsertChannelMonitorBeats(beats []ChannelMonitorBeat) error {
	if len(beats) == 0 {
		return nil
	}
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "monitor_id"},
			{Name: "bucket_ts"},
		},
		DoNothing: true,
	}).Create(&beats).Error
}

// GetChannelMonitorRecentBeats 取某个监控最近的 limit 条 beat，按时间升序返回。
// 逐监控查询而不是用窗口函数，以兼容 MySQL 5.7。
func GetChannelMonitorRecentBeats(monitorId string, limit int) ([]ChannelMonitorBeat, error) {
	if monitorId == "" || limit <= 0 {
		return nil, nil
	}
	var beats []ChannelMonitorBeat
	err := DB.Model(&ChannelMonitorBeat{}).
		Where("monitor_id = ?", monitorId).
		Order("bucket_ts DESC").
		Limit(limit).
		Find(&beats).Error
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(beats)-1; i < j; i, j = i+1, j-1 {
		beats[i], beats[j] = beats[j], beats[i]
	}
	return beats, nil
}

// UpsertChannelMonitorStat 累加小时汇总，冲突时在数据库侧做加法，避免读改写竞争。
func UpsertChannelMonitorStat(stat *ChannelMonitorStat) error {
	return upsertChannelMonitorStatTx(DB, stat)
}

func upsertChannelMonitorStatTx(tx *gorm.DB, stat *ChannelMonitorStat) error {
	if tx == nil || stat == nil || stat.Total == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "monitor_id"},
			{Name: "hour_ts"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"total":       gorm.Expr("channel_monitor_stats.total + ?", stat.Total),
			"up_count":    gorm.Expr("channel_monitor_stats.up_count + ?", stat.UpCount),
			"slow_count":  gorm.Expr("channel_monitor_stats.slow_count + ?", stat.SlowCount),
			"down_count":  gorm.Expr("channel_monitor_stats.down_count + ?", stat.DownCount),
			"ttft_sum_ms": gorm.Expr("channel_monitor_stats.ttft_sum_ms + ?", stat.TtftSumMs),
			"ttft_count":  gorm.Expr("channel_monitor_stats.ttft_count + ?", stat.TtftCount),
		}),
	}).Create(stat).Error
}

// PersistChannelMonitorSample 把一个采样桶写入 beat 表，并只在该桶首次插入时累加小时汇总。
// 不信任 OnConflict 的 RowsAffected：先锁 state 行，再 Count 判重。
// 若 bucketTs <= ResetAt，样本被丢弃，避免其它实例把 reset 前的热桶写回。
func PersistChannelMonitorSample(beat ChannelMonitorBeat, stat *ChannelMonitorStat) error {
	if beat.MonitorId == "" {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureChannelMonitorStateTx(tx, beat.MonitorId); err != nil {
			return err
		}
		var state ChannelMonitorState
		if err := lockForUpdate(tx).Where("monitor_id = ?", beat.MonitorId).First(&state).Error; err != nil {
			return err
		}
		if state.ResetAt > 0 && beat.BucketTs <= state.ResetAt {
			return nil
		}

		var existing int64
		if err := tx.Model(&ChannelMonitorBeat{}).
			Where("monitor_id = ? AND bucket_ts = ?", beat.MonitorId, beat.BucketTs).
			Count(&existing).Error; err != nil {
			return err
		}
		if existing == 0 {
			if err := tx.Create(&beat).Error; err != nil {
				return err
			}
			if err := upsertChannelMonitorStatTx(tx, stat); err != nil {
				return err
			}
		}
		return touchChannelMonitorBeatTsTx(tx, beat.MonitorId, beat.BucketTs, beat.Source)
	})
}

func ensureChannelMonitorStateTx(tx *gorm.DB, monitorId string) error {
	if tx == nil || monitorId == "" {
		return nil
	}
	state := ChannelMonitorState{MonitorId: monitorId}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "monitor_id"}},
		DoNothing: true,
	}).Create(&state).Error
}

// ChannelMonitorUptime 是某个监控在统计窗口内的可用性汇总结果。
type ChannelMonitorUptime struct {
	MonitorId string `json:"monitor_id"`
	Total     int64  `json:"total"`
	UpCount   int64  `json:"up_count"`
	SlowCount int64  `json:"slow_count"`
	DownCount int64  `json:"down_count"`
}

// GetChannelMonitorUptimes 汇总 sinceHourTs 之后的可用性数据。
func GetChannelMonitorUptimes(monitorIds []string, sinceHourTs int64) ([]ChannelMonitorUptime, error) {
	if len(monitorIds) == 0 {
		return nil, nil
	}
	var uptimes []ChannelMonitorUptime
	err := DB.Model(&ChannelMonitorStat{}).
		Select("monitor_id, SUM(total) as total, SUM(up_count) as up_count, SUM(slow_count) as slow_count, SUM(down_count) as down_count").
		Where("monitor_id IN ? AND hour_ts >= ?", monitorIds, sinceHourTs).
		Group("monitor_id").
		Find(&uptimes).Error
	return uptimes, err
}

// GetChannelMonitorStates 读取指定监控的状态行。
func GetChannelMonitorStates(monitorIds []string) ([]ChannelMonitorState, error) {
	if len(monitorIds) == 0 {
		return nil, nil
	}
	var states []ChannelMonitorState
	err := DB.Model(&ChannelMonitorState{}).Where("monitor_id IN ?", monitorIds).Find(&states).Error
	return states, err
}

// UpsertChannelMonitorPing 写入最近一次端点 ping 结果。
func UpsertChannelMonitorPing(monitorId string, pingMs int, channelId int, now int64) error {
	if monitorId == "" {
		return nil
	}
	state := ChannelMonitorState{
		MonitorId:     monitorId,
		PingMs:        pingMs,
		PingUpdatedAt: now,
		ChannelId:     channelId,
	}
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "monitor_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"ping_ms",
			"ping_updated_at",
			"channel_id",
		}),
	}).Create(&state).Error
}

// TouchChannelMonitorBeatTs 记录该监控最近一次产生 beat 的时间，供空闲判定使用。
func TouchChannelMonitorBeatTs(monitorId string, beatTs int64, source int) error {
	return touchChannelMonitorBeatTsTx(DB, monitorId, beatTs, source)
}

func touchChannelMonitorBeatTsTx(tx *gorm.DB, monitorId string, beatTs int64, source int) error {
	if tx == nil || monitorId == "" {
		return nil
	}
	state := ChannelMonitorState{
		MonitorId:  monitorId,
		LastBeatTs: beatTs,
	}
	updates := map[string]interface{}{"last_beat_ts": beatTs}
	if source == ChannelMonitorSourceProbe {
		state.LastProbeTs = beatTs
		updates["last_probe_ts"] = beatTs
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "monitor_id"}},
		DoUpdates: clause.Assignments(updates),
	}).Create(&state).Error
}

// DeleteChannelMonitorStatesExcept 清理已被管理员删除的监控项残留状态。
// 空列表视为解析失败或未提供白名单，禁止全表删除。
func DeleteChannelMonitorStatesExcept(monitorIds []string) error {
	if len(monitorIds) == 0 {
		return nil
	}
	return DB.Where("monitor_id NOT IN ?", monitorIds).Delete(&ChannelMonitorState{}).Error
}

// DeleteAllChannelMonitorStates 仅在监控配置被成功解析为空时清理残留 state。
func DeleteAllChannelMonitorStates() error {
	return DB.Where("1 = 1").Delete(&ChannelMonitorState{}).Error
}

// DeleteChannelMonitorBeatsBefore 按保留期清理原始 beat。
func DeleteChannelMonitorBeatsBefore(cutoffTs int64) error {
	if cutoffTs <= 0 {
		return nil
	}
	return DB.Where("bucket_ts < ?", cutoffTs).Delete(&ChannelMonitorBeat{}).Error
}

// DeleteChannelMonitorStatsBefore 按保留期清理小时汇总。
func DeleteChannelMonitorStatsBefore(cutoffTs int64) error {
	if cutoffTs <= 0 {
		return nil
	}
	return DB.Where("hour_ts < ?", cutoffTs).Delete(&ChannelMonitorStat{}).Error
}

// DeleteChannelMonitorData 清空单个监控项的历史采样和小时汇总，并留下 ResetAt tombstone。
// 配置本身不受影响；空 id 视为无操作，避免误清整表。
func DeleteChannelMonitorData(monitorId string) error {
	if monitorId == "" {
		return nil
	}
	now := time.Now().Unix()
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureChannelMonitorStateTx(tx, monitorId); err != nil {
			return err
		}
		var state ChannelMonitorState
		if err := lockForUpdate(tx).Where("monitor_id = ?", monitorId).First(&state).Error; err != nil {
			return err
		}
		if err := tx.Where("monitor_id = ?", monitorId).Delete(&ChannelMonitorBeat{}).Error; err != nil {
			return err
		}
		if err := tx.Where("monitor_id = ?", monitorId).Delete(&ChannelMonitorStat{}).Error; err != nil {
			return err
		}
		return tx.Model(&ChannelMonitorState{}).Where("monitor_id = ?", monitorId).Updates(map[string]interface{}{
			"reset_at":        now,
			"ping_ms":         0,
			"ping_updated_at": 0,
			"last_beat_ts":    0,
			"last_probe_ts":   0,
			"channel_id":      0,
		}).Error
	})
}
