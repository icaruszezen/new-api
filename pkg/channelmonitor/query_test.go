package channelmonitor

import (
	"testing"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
)

func TestLatestStatusReflectsNewestBeat(t *testing.T) {
	cases := []struct {
		name   string
		beats  []BeatView
		expect string
	}{
		{name: "no history", beats: nil, expect: MonitorStatusUnknown},
		{
			name: "newest beat wins over older failures",
			beats: []BeatView{
				{Ts: 1, Status: model.ChannelMonitorStatusDown},
				{Ts: 2, Status: model.ChannelMonitorStatusUp},
			},
			expect: MonitorStatusUp,
		},
		{
			name:   "slow beat maps to degraded",
			beats:  []BeatView{{Ts: 1, Status: model.ChannelMonitorStatusSlow}},
			expect: MonitorStatusDegraded,
		},
		{
			name:   "failed beat maps to down",
			beats:  []BeatView{{Ts: 1, Status: model.ChannelMonitorStatusDown}},
			expect: MonitorStatusDown,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expect, latestStatus(testCase.beats))
		})
	}
}

func TestAverageTtftIgnoresBeatsWithoutFirstToken(t *testing.T) {
	beats := []BeatView{
		{Ts: 1, Status: model.ChannelMonitorStatusUp, TtftMs: 200},
		{Ts: 2, Status: model.ChannelMonitorStatusDown, TtftMs: 0},
		{Ts: 3, Status: model.ChannelMonitorStatusUp, TtftMs: 400},
	}

	// Failures carry no first-token timing, so averaging them in would drag the
	// displayed latency toward zero.
	assert.Equal(t, 300, averageTtft(beats))
	assert.Equal(t, 0, averageTtft([]BeatView{{Ts: 1, TtftMs: 0}}))
	assert.Equal(t, 0, averageTtft(nil))
}

// 可用性与状态条读同一批 beat，因此百分比的分母必须是实际渲染出的样本数。
func TestUptimeFromBeatsCountsSlowSamplesAsAvailable(t *testing.T) {
	cases := []struct {
		name   string
		beats  []BeatView
		expect float64
	}{
		{name: "no samples", beats: nil, expect: 0},
		{
			name: "all successful",
			beats: []BeatView{
				{Ts: 1, Status: model.ChannelMonitorStatusUp},
				{Ts: 2, Status: model.ChannelMonitorStatusUp},
			},
			expect: 100,
		},
		{
			name: "slow counts as available",
			beats: []BeatView{
				{Ts: 1, Status: model.ChannelMonitorStatusUp},
				{Ts: 2, Status: model.ChannelMonitorStatusSlow},
			},
			expect: 100,
		},
		{
			name: "failures reduce availability",
			beats: []BeatView{
				{Ts: 1, Status: model.ChannelMonitorStatusUp},
				{Ts: 2, Status: model.ChannelMonitorStatusSlow},
				{Ts: 3, Status: model.ChannelMonitorStatusUp},
				{Ts: 4, Status: model.ChannelMonitorStatusDown},
			},
			expect: 75,
		},
		{
			// 状态条上的空档不是一次采样，不能把可用性拉低。
			name: "unknown status is not a sample",
			beats: []BeatView{
				{Ts: 1, Status: 0},
				{Ts: 2, Status: model.ChannelMonitorStatusUp},
			},
			expect: 100,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.InDelta(t, testCase.expect, uptimeFromBeats(testCase.beats), 0.0001)
		})
	}
}

func TestDedupeByTsDropsOverlappingBuckets(t *testing.T) {
	// A bucket can appear both in the beats table and in the in-memory map when
	// a flush lands while the status page is being rendered.
	beats := dedupeByTs([]BeatView{
		{Ts: 100, Status: model.ChannelMonitorStatusUp},
		{Ts: 100, Status: model.ChannelMonitorStatusDown},
		{Ts: 120, Status: model.ChannelMonitorStatusUp},
	})

	assert.Len(t, beats, 2)
	assert.Equal(t, int64(100), beats[0].Ts)
	assert.Equal(t, model.ChannelMonitorStatusUp, beats[0].Status)
	assert.Equal(t, int64(120), beats[1].Ts)
}
