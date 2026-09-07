package channelmonitor

import (
	"sort"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/channel_monitoring_setting"
)

// statusCacheTTL 保护公开接口：状态页会被未登录访客高频轮询，
// 而底层数据最快也只有一个采样窗口才变化一次。
const statusCacheTTL = 5 * time.Second

var statusCache struct {
	mu        sync.Mutex
	view      StatusView
	expiresAt time.Time
}

// Status 组装公开状态页数据。
func Status() (StatusView, error) {
	setting := channel_monitoring_setting.GetSetting()
	if !setting.Enabled {
		// The shape stays identical when disabled so clients never have to
		// special-case missing rendering parameters.
		return StatusView{
			Enabled:             false,
			SampleWindowSeconds: setting.SampleWindowSeconds,
			BeatLimit:           setting.BeatLimit,
			Monitors:            []MonitorView{},
		}, nil
	}

	statusCache.mu.Lock()
	defer statusCache.mu.Unlock()
	if time.Now().Before(statusCache.expiresAt) {
		return statusCache.view, nil
	}

	view, err := buildStatusView(setting)
	if err != nil {
		return StatusView{}, err
	}
	statusCache.view = view
	statusCache.expiresAt = time.Now().Add(statusCacheTTL)
	return view, nil
}

// InvalidateStatusCache 在监控配置变更后立即让公开接口反映新配置。
func InvalidateStatusCache() {
	statusCache.mu.Lock()
	statusCache.expiresAt = time.Time{}
	statusCache.mu.Unlock()
}

func buildStatusView(setting channel_monitoring_setting.ChannelMonitoringSetting) (StatusView, error) {
	monitors := EnabledMonitors()
	view := StatusView{
		Enabled:             true,
		SampleWindowSeconds: setting.SampleWindowSeconds,
		BeatLimit:           setting.BeatLimit,
		Monitors:            make([]MonitorView, 0, len(monitors)),
	}
	if len(monitors) == 0 {
		return view, nil
	}

	monitorIds := make([]string, 0, len(monitors))
	for _, monitor := range monitors {
		monitorIds = append(monitorIds, monitor.Id)
	}

	states, err := model.GetChannelMonitorStates(monitorIds)
	if err != nil {
		return StatusView{}, err
	}
	stateById := make(map[string]model.ChannelMonitorState, len(states))
	for _, state := range states {
		stateById[state.MonitorId] = state
	}

	hot := collectHotBeats(setting.SampleWindowSeconds)

	for _, monitor := range monitors {
		beats, err := loadBeats(monitor.Id, setting.BeatLimit, hot[monitor.Id])
		if err != nil {
			return StatusView{}, err
		}
		view.Monitors = append(view.Monitors, MonitorView{
			Id:    monitor.Id,
			Name:  monitor.Name,
			Model: monitor.Model,
			// Monitors saved without an explicit icon fall back to the provider
			// derived from the model name, so the card always shows a logo even
			// when the stored config predates icon resolution.
			Icon:      resolveIcon(monitor),
			Status:    latestStatus(beats),
			AvgTtftMs: averageTtft(beats),
			PingMs:    stateById[monitor.Id].PingMs,
			Uptime:    uptimeFromBeats(beats),
			Beats:     beats,
		})
	}

	return view, nil
}

// loadBeats 合并已落库的 beat 与内存中尚未刷盘的桶，按时间升序返回最多 limit 条。
func loadBeats(monitorId string, limit int, pending []BeatView) ([]BeatView, error) {
	rows, err := model.GetChannelMonitorRecentBeats(monitorId, limit)
	if err != nil {
		return nil, err
	}

	beats := make([]BeatView, 0, len(rows)+len(pending))
	for _, row := range rows {
		beats = append(beats, BeatView{
			Ts:     row.BucketTs,
			Status: row.Status,
			TtftMs: row.TtftMs,
		})
	}
	beats = append(beats, pending...)

	sort.SliceStable(beats, func(i, j int) bool { return beats[i].Ts < beats[j].Ts })
	beats = dedupeByTs(beats)
	if len(beats) > limit {
		beats = beats[len(beats)-limit:]
	}
	return beats, nil
}

// dedupeByTs 处理落库记录与内存桶在同一时间戳重叠的情况，保留先出现的一条。
func dedupeByTs(beats []BeatView) []BeatView {
	if len(beats) < 2 {
		return beats
	}
	out := beats[:1]
	for _, beat := range beats[1:] {
		if beat.Ts == out[len(out)-1].Ts {
			continue
		}
		out = append(out, beat)
	}
	return out
}

// collectHotBeats 快照内存中的采样桶，让状态页无需等到下一次刷盘就能看到最新一格。
func collectHotBeats(sampleWindowSeconds int) map[string][]BeatView {
	currentBucket := bucketStart(time.Now().Unix(), int64(sampleWindowSeconds))
	pending := make(map[string][]BeatView)
	hotBeats.Range(func(key, value any) bool {
		k := key.(beatKey)
		beat := value.(*hotBeat)
		beat.mu.Lock()
		filled, sample := beat.filled, beat.sample
		beat.mu.Unlock()
		if !filled {
			return true
		}
		// 当前进行中的桶也纳入展示，它的当选样本随后续请求可能变化，属于预期行为。
		if k.bucketTs > currentBucket {
			return true
		}
		pending[k.monitorId] = append(pending[k.monitorId], BeatView{
			Ts:     k.bucketTs,
			Status: sample.Status,
			TtftMs: sample.TtftMs,
		})
		return true
	})
	return pending
}

func resolveIcon(monitor Monitor) string {
	if monitor.Icon != "" {
		return monitor.Icon
	}
	return model.ResolveModelIconKey(monitor.Model)
}

func latestStatus(beats []BeatView) string {
	if len(beats) == 0 {
		return MonitorStatusUnknown
	}
	switch beats[len(beats)-1].Status {
	case model.ChannelMonitorStatusUp:
		return MonitorStatusUp
	case model.ChannelMonitorStatusSlow:
		return MonitorStatusDegraded
	case model.ChannelMonitorStatusDown:
		return MonitorStatusDown
	default:
		return MonitorStatusUnknown
	}
}

// averageTtft 是「对话延迟」指标：最近这批 beat 的首字平均值。
func averageTtft(beats []BeatView) int {
	sum := 0
	count := 0
	for _, beat := range beats {
		if beat.TtftMs > 0 {
			sum += beat.TtftMs
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / count
}

// uptimeFromBeats 用状态条上的同一批样本算可用性，保证百分比与条形图口径一致。
// 慢响应仍算作可用，只有失败计入不可用。
func uptimeFromBeats(beats []BeatView) float64 {
	total := 0
	available := 0
	for _, beat := range beats {
		switch beat.Status {
		case model.ChannelMonitorStatusUp, model.ChannelMonitorStatusSlow:
			available++
			total++
		case model.ChannelMonitorStatusDown:
			total++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(available) / float64(total) * 100
}
