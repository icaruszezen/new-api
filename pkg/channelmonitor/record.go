package channelmonitor

import (
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/channel_monitoring_setting"
)

// hotBeats 保存尚未落库的采样桶，key 为 beatKey。
var hotBeats sync.Map

type beatKey struct {
	monitorId string
	bucketTs  int64
}

// hotBeat 是某个采样窗口内的当选样本。
//
// 降采样规则：一个窗口只显示一格，成功样本优先于失败样本，多个成功样本取首字最快的一次。
// 这意味着窗口内只要有一次成功请求，该格就是绿色；只有整个窗口都失败才会变红。
type hotBeat struct {
	mu     sync.Mutex
	filled bool
	sample Sample
}

// RecordRelaySample 在请求结束时采集监控样本。
// 只统计流式请求；系统探测走 RecordProbeResult，这里显式排除以避免重复计数。
func RecordRelaySample(info *relaycommon.RelayInfo, success bool) {
	if info == nil || !info.IsStream || info.IsChannelTest {
		return
	}
	if !channel_monitoring_setting.IsEnabled() {
		return
	}
	monitor, ok := Lookup(info.UsingGroup, info.OriginModelName)
	if !ok {
		return
	}

	hasTtft := info.HasSendResponse()
	ttftMs := 0
	if hasTtft {
		ttftMs = int(info.FirstResponseTime.Sub(info.StartTime).Milliseconds())
		if ttftMs < 0 {
			ttftMs = 0
		}
	}

	Record(Sample{
		MonitorId: monitor.Id,
		Status:    resolveStatus(success, hasTtft, ttftMs),
		TtftMs:    ttftMs,
		HasTtft:   hasTtft,
		ChannelId: info.GetChannelID(),
		Source:    model.ChannelMonitorSourceUser,
	})
}

// RecordProbeResult 记录一次系统主动探测的结果。
func RecordProbeResult(monitorId string, success bool, ttftMs int, channelId int) {
	if monitorId == "" {
		return
	}
	hasTtft := success && ttftMs > 0
	if ttftMs < 0 {
		ttftMs = 0
	}
	Record(Sample{
		MonitorId: monitorId,
		Status:    resolveStatus(success, hasTtft, ttftMs),
		TtftMs:    ttftMs,
		HasTtft:   hasTtft,
		ChannelId: channelId,
		Source:    model.ChannelMonitorSourceProbe,
	})
}

// Record 把样本写入当前采样桶，桶内按降采样规则择优保留。
func Record(sample Sample) {
	if sample.MonitorId == "" {
		return
	}
	setting := channel_monitoring_setting.GetSetting()
	if !setting.Enabled {
		return
	}

	key := beatKey{
		monitorId: sample.MonitorId,
		bucketTs:  bucketStart(time.Now().Unix(), int64(setting.SampleWindowSeconds)),
	}
	actual, _ := hotBeats.LoadOrStore(key, &hotBeat{})
	actual.(*hotBeat).offer(sample)
}

func (b *hotBeat) offer(sample Sample) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.filled || preferSample(sample, b.sample) {
		b.sample = sample
		b.filled = true
	}
}

// take 取出并清空当选样本，供 flush 使用。
func (b *hotBeat) take() (Sample, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.filled {
		return Sample{}, false
	}
	sample := b.sample
	b.filled = false
	return sample, true
}

// restore 在落库失败时把样本放回桶里，等下一轮重试。
func (b *hotBeat) restore(sample Sample) {
	b.offer(sample)
}

// preferSample 判定 candidate 是否应该取代 current：成功优先，其次首字更快者优先。
func preferSample(candidate Sample, current Sample) bool {
	candidateFailed := candidate.Status == model.ChannelMonitorStatusDown
	currentFailed := current.Status == model.ChannelMonitorStatusDown
	if candidateFailed != currentFailed {
		return currentFailed
	}
	if candidate.HasTtft != current.HasTtft {
		return candidate.HasTtft
	}
	if !candidate.HasTtft {
		return false
	}
	return candidate.TtftMs < current.TtftMs
}

// resolveStatus 把请求结果映射到状态条配色：失败为红，首字超过慢阈值为黄，其余为绿。
func resolveStatus(success bool, hasTtft bool, ttftMs int) int {
	if !success {
		return model.ChannelMonitorStatusDown
	}
	if hasTtft && ttftMs > channel_monitoring_setting.GetSetting().SlowThresholdMs {
		return model.ChannelMonitorStatusSlow
	}
	return model.ChannelMonitorStatusUp
}

func bucketStart(ts int64, windowSeconds int64) int64 {
	if windowSeconds <= 0 {
		windowSeconds = channel_monitoring_setting.DefaultSampleWindowSeconds
	}
	return ts - (ts % windowSeconds)
}
