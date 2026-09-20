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

// hotBeat 同时保存窗口代表样本和该窗计入的全部请求计数。
//
// 代表样本规则：成功优先于失败，多个成功取首字最快的一次，只决定趋势条高度。
// 计数规则：每次计入的请求都累加，失败不会被同窗成功盖掉。
type hotBeat struct {
	mu        sync.Mutex
	filled    bool
	sample    Sample
	upCount   int64
	slowCount int64
	downCount int64
	ttftSumMs int64
	ttftCount int64
}

// beatSnapshot 是 flush 取出的完整窗口状态，restore 时必须整份回写以免丢计数。
type beatSnapshot struct {
	sample    Sample
	upCount   int64
	slowCount int64
	downCount int64
	ttftSumMs int64
	ttftCount int64
}

func (s beatSnapshot) total() int64 {
	return s.upCount + s.slowCount + s.downCount
}

func (s beatSnapshot) view(bucketTs int64) BeatView {
	return BeatView{
		Ts:           bucketTs,
		Status:       s.sample.Status,
		TtftMs:       s.sample.TtftMs,
		RequestTotal: s.total(),
		RequestUp:    s.upCount,
		RequestSlow:  s.slowCount,
		RequestDown:  s.downCount,
		TtftSumMs:    s.ttftSumMs,
		TtftCount:    s.ttftCount,
	}
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
	if !success && shouldIgnoreMonitorError(info.LastError) {
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

// Record 把样本写入当前采样桶：累加计数，并按规则更新代表样本。
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

// HasActivitySince 报告该监控是否还有未刷盘的热桶。
// 采样窗口可以长于探测间隔，所以只要内存里有当选样本，就视为仍有流量，
// 不能只拿已落库的 LastBeatTs 判断空闲。
func HasActivitySince(monitorId string, sinceTs int64) bool {
	if monitorId == "" {
		return false
	}
	found := false
	hotBeats.Range(func(key, value any) bool {
		k := key.(beatKey)
		if k.monitorId != monitorId {
			return true
		}
		beat := value.(*hotBeat)
		beat.mu.Lock()
		filled := beat.filled
		beat.mu.Unlock()
		if !filled {
			return true
		}
		// 窗口起点可能早于 sinceTs（采样窗口长于探测间隔），只要热桶里还有样本就算活跃。
		_ = sinceTs
		found = true
		return false
	})
	return found
}

func (b *hotBeat) offer(sample Sample) {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch sample.Status {
	case model.ChannelMonitorStatusUp:
		b.upCount++
	case model.ChannelMonitorStatusSlow:
		b.slowCount++
	case model.ChannelMonitorStatusDown:
		b.downCount++
	}
	if sample.HasTtft && sample.TtftMs > 0 {
		b.ttftSumMs += int64(sample.TtftMs)
		b.ttftCount++
	}
	if !b.filled || preferSample(sample, b.sample) {
		b.sample = sample
		b.filled = true
	}
}

func (b *hotBeat) take() (beatSnapshot, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.filled {
		return beatSnapshot{}, false
	}
	snap := beatSnapshot{
		sample:    b.sample,
		upCount:   b.upCount,
		slowCount: b.slowCount,
		downCount: b.downCount,
		ttftSumMs: b.ttftSumMs,
		ttftCount: b.ttftCount,
	}
	b.filled = false
	b.sample = Sample{}
	b.upCount = 0
	b.slowCount = 0
	b.downCount = 0
	b.ttftSumMs = 0
	b.ttftCount = 0
	return snap, true
}

// restore 在落库失败时把窗口状态放回桶里。若 take 之后已有新样本入桶，把计数合并进去。
func (b *hotBeat) restore(snap beatSnapshot) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.filled {
		b.sample = snap.sample
		b.filled = true
		b.upCount = snap.upCount
		b.slowCount = snap.slowCount
		b.downCount = snap.downCount
		b.ttftSumMs = snap.ttftSumMs
		b.ttftCount = snap.ttftCount
		return
	}
	b.upCount += snap.upCount
	b.slowCount += snap.slowCount
	b.downCount += snap.downCount
	b.ttftSumMs += snap.ttftSumMs
	b.ttftCount += snap.ttftCount
}

func (b *hotBeat) view(bucketTs int64) (BeatView, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.filled {
		return BeatView{}, false
	}
	return beatSnapshot{
		sample:    b.sample,
		upCount:   b.upCount,
		slowCount: b.slowCount,
		downCount: b.downCount,
		ttftSumMs: b.ttftSumMs,
		ttftCount: b.ttftCount,
	}.view(bucketTs), true
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

// resolveStatus 把请求结果映射到状态条高度：失败为矮红，首字超过慢阈值为黄，其余为绿。
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
