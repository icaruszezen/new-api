package channelmonitor

import (
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreferSampleKeepsFastestSuccessInWindow(t *testing.T) {
	cases := []struct {
		name      string
		current   Sample
		candidate Sample
		expect    bool
	}{
		{
			name:      "faster success replaces slower success",
			current:   Sample{Status: model.ChannelMonitorStatusUp, TtftMs: 900, HasTtft: true},
			candidate: Sample{Status: model.ChannelMonitorStatusUp, TtftMs: 300, HasTtft: true},
			expect:    true,
		},
		{
			name:      "slower success does not replace faster success",
			current:   Sample{Status: model.ChannelMonitorStatusUp, TtftMs: 300, HasTtft: true},
			candidate: Sample{Status: model.ChannelMonitorStatusUp, TtftMs: 900, HasTtft: true},
			expect:    false,
		},
		{
			name:      "success replaces failure",
			current:   Sample{Status: model.ChannelMonitorStatusDown},
			candidate: Sample{Status: model.ChannelMonitorStatusUp, TtftMs: 4000, HasTtft: true},
			expect:    true,
		},
		{
			name:      "failure never replaces success",
			current:   Sample{Status: model.ChannelMonitorStatusUp, TtftMs: 4000, HasTtft: true},
			candidate: Sample{Status: model.ChannelMonitorStatusDown},
			expect:    false,
		},
		{
			name:      "slow success still wins over failure",
			current:   Sample{Status: model.ChannelMonitorStatusDown},
			candidate: Sample{Status: model.ChannelMonitorStatusSlow, TtftMs: 61000, HasTtft: true},
			expect:    true,
		},
		{
			name:      "sample with first token wins over one without",
			current:   Sample{Status: model.ChannelMonitorStatusUp},
			candidate: Sample{Status: model.ChannelMonitorStatusUp, TtftMs: 5000, HasTtft: true},
			expect:    true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expect, preferSample(testCase.candidate, testCase.current))
		})
	}
}

func TestHotBeatOfferKeepsWinningSample(t *testing.T) {
	beat := &hotBeat{}

	beat.offer(Sample{MonitorId: "m1", Status: model.ChannelMonitorStatusDown})
	beat.offer(Sample{MonitorId: "m1", Status: model.ChannelMonitorStatusUp, TtftMs: 900, HasTtft: true})
	beat.offer(Sample{MonitorId: "m1", Status: model.ChannelMonitorStatusUp, TtftMs: 1500, HasTtft: true})

	sample, ok := beat.take()
	assert.True(t, ok)
	assert.Equal(t, model.ChannelMonitorStatusUp, sample.sample.Status)
	assert.Equal(t, 900, sample.sample.TtftMs)
	assert.Equal(t, int64(1), sample.downCount)
	assert.Equal(t, int64(2), sample.upCount)
	assert.Equal(t, int64(3), sample.total())
	assert.Equal(t, int64(2400), sample.ttftSumMs)
	assert.Equal(t, int64(2), sample.ttftCount)

	_, ok = beat.take()
	assert.False(t, ok, "a drained bucket must not emit the same beat twice")
}

func TestHasActivitySinceSeesUnflushedHotBeats(t *testing.T) {
	t.Cleanup(clearHotBeats)
	clearHotBeats()

	assert.False(t, HasActivitySince("m1", 0))

	beat := &hotBeat{}
	beat.offer(Sample{MonitorId: "m1", Status: model.ChannelMonitorStatusUp, TtftMs: 100, HasTtft: true})
	hotBeats.Store(beatKey{monitorId: "m1", bucketTs: 100}, beat)

	assert.True(t, HasActivitySince("m1", 1_000))
	assert.False(t, HasActivitySince("m2", 0))
}

func TestHotBeatRestorePutsSampleBackForRetry(t *testing.T) {
	beat := &hotBeat{}
	beat.offer(Sample{MonitorId: "m1", Status: model.ChannelMonitorStatusUp, TtftMs: 120, HasTtft: true})

	sample, ok := beat.take()
	assert.True(t, ok)

	beat.restore(sample)

	restored, ok := beat.take()
	assert.True(t, ok, "a failed flush must leave the sample available for the next pass")
	assert.Equal(t, 120, restored.sample.TtftMs)
	assert.Equal(t, int64(1), restored.upCount)
}

func TestHotBeatOfferCountsFailuresEvenWhenASuccessWins(t *testing.T) {
	beat := &hotBeat{}
	beat.offer(Sample{MonitorId: "m1", Status: model.ChannelMonitorStatusDown})
	beat.offer(Sample{MonitorId: "m1", Status: model.ChannelMonitorStatusDown})
	beat.offer(Sample{MonitorId: "m1", Status: model.ChannelMonitorStatusUp, TtftMs: 400, HasTtft: true})

	snap, ok := beat.take()
	assert.True(t, ok)
	assert.Equal(t, model.ChannelMonitorStatusUp, snap.sample.Status)
	assert.Equal(t, int64(1), snap.upCount)
	assert.Equal(t, int64(2), snap.downCount)
	assert.Equal(t, int64(3), snap.total())
}

func TestHotBeatRestoreMergesCountsWhenNewSamplesArrived(t *testing.T) {
	beat := &hotBeat{}
	beat.offer(Sample{MonitorId: "m1", Status: model.ChannelMonitorStatusDown})

	snap, ok := beat.take()
	assert.True(t, ok)

	beat.offer(Sample{MonitorId: "m1", Status: model.ChannelMonitorStatusUp, TtftMs: 80, HasTtft: true})
	beat.restore(snap)

	merged, ok := beat.take()
	assert.True(t, ok)
	assert.Equal(t, int64(1), merged.upCount)
	assert.Equal(t, int64(1), merged.downCount)
	assert.Equal(t, int64(2), merged.total())
	assert.Equal(t, int64(80), merged.ttftSumMs)
}

func TestRecordRelaySampleSkipsIgnoredUserErrors(t *testing.T) {
	t.Cleanup(clearHotBeats)
	t.Cleanup(resetMonitorRegistry)
	clearHotBeats()
	resetMonitorRegistry()
	enableMonitoredRelay(t)

	info := &relaycommon.RelayInfo{
		IsStream:        true,
		UsingGroup:      "default",
		OriginModelName: "gpt-5",
		LastError:       types.NewError(errors.New("insufficient quota"), types.ErrorCodeInsufficientUserQuota),
	}
	RecordRelaySample(info, false)
	assert.Equal(t, 0, countFilledHotBeats("m1"))
}

func TestRecordRelaySampleRecordsUpstreamFailures(t *testing.T) {
	t.Cleanup(clearHotBeats)
	t.Cleanup(resetMonitorRegistry)
	clearHotBeats()
	resetMonitorRegistry()
	enableMonitoredRelay(t)

	info := &relaycommon.RelayInfo{
		IsStream:        true,
		UsingGroup:      "default",
		OriginModelName: "gpt-5",
		LastError: types.NewErrorWithStatusCode(
			errors.New("bad gateway"),
			types.ErrorCodeBadResponseStatusCode,
			http.StatusBadGateway,
		),
	}
	RecordRelaySample(info, false)
	assert.Equal(t, 1, countFilledHotBeats("m1"))
}

func enableMonitoredRelay(t *testing.T) {
	t.Helper()
	saved := config.GlobalConfig.ExportAllConfigs()
	t.Cleanup(func() {
		_ = config.GlobalConfig.LoadFromDB(saved)
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"channel_monitoring_setting.enabled":  "true",
		"channel_monitoring_setting.monitors": `[{"id":"m1","name":"Pro","group":"default","model":"gpt-5","enabled":true}]`,
	}))
}

func TestBucketStartAlignsToSampleWindow(t *testing.T) {
	assert.Equal(t, int64(1_700_000_000), bucketStart(1_700_000_007, 20))
	assert.Equal(t, int64(1_700_000_000), bucketStart(1_700_000_019, 20))
	assert.Equal(t, int64(1_700_000_020), bucketStart(1_700_000_020, 20))
	// A misconfigured window must not divide by zero or collapse every sample
	// into a single bucket.
	assert.Equal(t, int64(1_700_000_000), bucketStart(1_700_000_007, 0))
}
