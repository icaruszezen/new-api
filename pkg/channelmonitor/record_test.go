package channelmonitor

import (
	"testing"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, model.ChannelMonitorStatusUp, sample.Status)
	assert.Equal(t, 900, sample.TtftMs)

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
	assert.Equal(t, 120, restored.TtftMs)
}

func TestBucketStartAlignsToSampleWindow(t *testing.T) {
	assert.Equal(t, int64(1_700_000_000), bucketStart(1_700_000_007, 20))
	assert.Equal(t, int64(1_700_000_000), bucketStart(1_700_000_019, 20))
	assert.Equal(t, int64(1_700_000_020), bucketStart(1_700_000_020, 20))
	// A misconfigured window must not divide by zero or collapse every sample
	// into a single bucket.
	assert.Equal(t, int64(1_700_000_000), bucketStart(1_700_000_007, 0))
}
