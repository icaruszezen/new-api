package channelmonitor

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/channel_monitoring_setting"
)

const (
	flushInterval   = 10 * time.Second
	cleanupInterval = time.Hour
	hourSeconds     = 3600
)

// Init 启动后台刷盘循环。
func Init() {
	go flushLoop()
}

func flushLoop() {
	lastCleanup := time.Now()
	for {
		time.Sleep(flushInterval)
		setting := channel_monitoring_setting.GetSetting()
		if !setting.Enabled {
			continue
		}
		flushCompletedBuckets(setting.SampleWindowSeconds)
		if time.Since(lastCleanup) >= cleanupInterval {
			cleanupExpiredData(setting.BeatRetentionHours, setting.StatRetentionDays)
			lastCleanup = time.Now()
		}
	}
}

// flushCompletedBuckets 把已结束的采样桶写入 beat 表并累加小时汇总。
// 仍在进行中的桶保留在内存里，等窗口结束后再落库。
func flushCompletedBuckets(sampleWindowSeconds int) {
	currentBucket := bucketStart(time.Now().Unix(), int64(sampleWindowSeconds))
	staleBefore := time.Now().Add(-time.Hour).Unix()

	hotBeats.Range(func(key, value any) bool {
		k := key.(beatKey)
		beat := value.(*hotBeat)

		if k.bucketTs >= currentBucket {
			return true
		}

		sample, ok := beat.take()
		if !ok {
			if k.bucketTs < staleBefore {
				hotBeats.Delete(key)
			}
			return true
		}

		if err := persistBeat(k, sample); err != nil {
			beat.restore(sample)
			common.SysError(fmt.Sprintf("failed to flush channel monitor beat monitor=%s bucket=%d: %s", k.monitorId, k.bucketTs, err.Error()))
			return true
		}

		hotBeats.Delete(key)
		return true
	})
}

func persistBeat(key beatKey, sample Sample) error {
	err := model.InsertChannelMonitorBeats([]model.ChannelMonitorBeat{{
		MonitorId: key.monitorId,
		BucketTs:  key.bucketTs,
		Status:    sample.Status,
		TtftMs:    sample.TtftMs,
		ChannelId: sample.ChannelId,
		Source:    sample.Source,
	}})
	if err != nil {
		return err
	}

	stat := &model.ChannelMonitorStat{
		MonitorId: key.monitorId,
		HourTs:    key.bucketTs - (key.bucketTs % hourSeconds),
		Total:     1,
	}
	switch sample.Status {
	case model.ChannelMonitorStatusUp:
		stat.UpCount = 1
	case model.ChannelMonitorStatusSlow:
		stat.SlowCount = 1
	default:
		stat.DownCount = 1
	}
	if sample.HasTtft {
		stat.TtftSumMs = int64(sample.TtftMs)
		stat.TtftCount = 1
	}
	if err := model.UpsertChannelMonitorStat(stat); err != nil {
		return err
	}

	return model.TouchChannelMonitorBeatTs(key.monitorId, key.bucketTs, sample.Source)
}

func cleanupExpiredData(beatRetentionHours int, statRetentionDays int) {
	beatCutoff := time.Now().Add(-time.Duration(beatRetentionHours) * time.Hour).Unix()
	if err := model.DeleteChannelMonitorBeatsBefore(beatCutoff); err != nil {
		common.SysError("failed to cleanup expired channel monitor beats: " + err.Error())
	}

	statCutoff := time.Now().Add(-time.Duration(statRetentionDays) * 24 * time.Hour).Unix()
	if err := model.DeleteChannelMonitorStatsBefore(statCutoff); err != nil {
		common.SysError("failed to cleanup expired channel monitor stats: " + err.Error())
	}

	monitorIds := make([]string, 0, len(Monitors()))
	for _, monitor := range Monitors() {
		monitorIds = append(monitorIds, monitor.Id)
	}
	if err := model.DeleteChannelMonitorStatesExcept(monitorIds); err != nil {
		common.SysError("failed to cleanup stale channel monitor states: " + err.Error())
	}
}
