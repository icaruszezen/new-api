package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChannelMonitorRecentBeatsReturnsNewestWindowInAscendingOrder(t *testing.T) {
	truncateTables(t)

	beats := make([]ChannelMonitorBeat, 0, 5)
	for index := 0; index < 5; index++ {
		beats = append(beats, ChannelMonitorBeat{
			MonitorId: "m1",
			BucketTs:  int64(1_700_000_000 + index*20),
			Status:    ChannelMonitorStatusUp,
			TtftMs:    100 + index,
		})
	}
	beats = append(beats, ChannelMonitorBeat{
		MonitorId: "other",
		BucketTs:  1_700_000_100,
		Status:    ChannelMonitorStatusDown,
	})
	require.NoError(t, InsertChannelMonitorBeats(beats))

	// The status bar renders past-to-now, so the newest slice must come back
	// oldest-first even though the query selects the newest rows.
	got, err := GetChannelMonitorRecentBeats("m1", 3)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, int64(1_700_000_040), got[0].BucketTs)
	assert.Equal(t, int64(1_700_000_060), got[1].BucketTs)
	assert.Equal(t, int64(1_700_000_080), got[2].BucketTs)
}

func TestInsertChannelMonitorBeatsIgnoresDuplicateSampleWindow(t *testing.T) {
	truncateTables(t)

	require.NoError(t, InsertChannelMonitorBeats([]ChannelMonitorBeat{{
		MonitorId: "m1",
		BucketTs:  1_700_000_000,
		Status:    ChannelMonitorStatusUp,
		TtftMs:    120,
	}}))

	// Two instances can flush the same window; the first write must win instead
	// of erroring out or duplicating the slot.
	require.NoError(t, InsertChannelMonitorBeats([]ChannelMonitorBeat{{
		MonitorId: "m1",
		BucketTs:  1_700_000_000,
		Status:    ChannelMonitorStatusDown,
		TtftMs:    0,
	}}))

	got, err := GetChannelMonitorRecentBeats("m1", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, ChannelMonitorStatusUp, got[0].Status)
	assert.Equal(t, 120, got[0].TtftMs)
}

func TestUpsertChannelMonitorStatAccumulatesWithinTheSameHour(t *testing.T) {
	truncateTables(t)

	require.NoError(t, UpsertChannelMonitorStat(&ChannelMonitorStat{
		MonitorId: "m1",
		HourTs:    1_700_000_000,
		Total:     1,
		UpCount:   1,
		TtftSumMs: 200,
		TtftCount: 1,
	}))
	require.NoError(t, UpsertChannelMonitorStat(&ChannelMonitorStat{
		MonitorId: "m1",
		HourTs:    1_700_000_000,
		Total:     1,
		DownCount: 1,
	}))
	require.NoError(t, UpsertChannelMonitorStat(&ChannelMonitorStat{
		MonitorId: "m1",
		HourTs:    1_700_003_600,
		Total:     1,
		SlowCount: 1,
		TtftSumMs: 90_000,
		TtftCount: 1,
	}))

	uptimes, err := GetChannelMonitorUptimes([]string{"m1"}, 1_700_000_000)
	require.NoError(t, err)
	require.Len(t, uptimes, 1)
	assert.Equal(t, int64(3), uptimes[0].Total)
	assert.Equal(t, int64(1), uptimes[0].UpCount)
	assert.Equal(t, int64(1), uptimes[0].SlowCount)
	assert.Equal(t, int64(1), uptimes[0].DownCount)
}

func TestGetChannelMonitorUptimesExcludesHoursOutsideTheWindow(t *testing.T) {
	truncateTables(t)

	require.NoError(t, UpsertChannelMonitorStat(&ChannelMonitorStat{
		MonitorId: "m1", HourTs: 1_700_000_000, Total: 5, UpCount: 5,
	}))
	require.NoError(t, UpsertChannelMonitorStat(&ChannelMonitorStat{
		MonitorId: "m1", HourTs: 1_699_000_000, Total: 100, DownCount: 100,
	}))

	uptimes, err := GetChannelMonitorUptimes([]string{"m1"}, 1_700_000_000)
	require.NoError(t, err)
	require.Len(t, uptimes, 1)
	assert.Equal(t, int64(5), uptimes[0].Total)
	assert.Equal(t, int64(0), uptimes[0].DownCount)
}

func TestUpsertChannelMonitorPingKeepsLastBeatTimestamp(t *testing.T) {
	truncateTables(t)

	require.NoError(t, TouchChannelMonitorBeatTs("m1", 1_700_000_000, ChannelMonitorSourceUser))
	require.NoError(t, UpsertChannelMonitorPing("m1", 23, 7, 1_700_000_050))

	states, err := GetChannelMonitorStates([]string{"m1"})
	require.NoError(t, err)
	require.Len(t, states, 1)
	// Ping refresh and beat tracking write the same row from different code
	// paths; neither may clobber the other's columns.
	assert.Equal(t, 23, states[0].PingMs)
	assert.Equal(t, 7, states[0].ChannelId)
	assert.Equal(t, int64(1_700_000_000), states[0].LastBeatTs)
	assert.Equal(t, int64(0), states[0].LastProbeTs)
}

func TestTouchChannelMonitorBeatTsRecordsProbeTimeOnlyForProbes(t *testing.T) {
	truncateTables(t)

	require.NoError(t, TouchChannelMonitorBeatTs("m1", 1_700_000_000, ChannelMonitorSourceProbe))
	require.NoError(t, TouchChannelMonitorBeatTs("m1", 1_700_000_100, ChannelMonitorSourceUser))

	states, err := GetChannelMonitorStates([]string{"m1"})
	require.NoError(t, err)
	require.Len(t, states, 1)
	assert.Equal(t, int64(1_700_000_100), states[0].LastBeatTs)
	assert.Equal(t, int64(1_700_000_000), states[0].LastProbeTs)
}

func TestDeleteChannelMonitorStatesExceptDropsRemovedMonitors(t *testing.T) {
	truncateTables(t)

	require.NoError(t, TouchChannelMonitorBeatTs("keep", 1, ChannelMonitorSourceUser))
	require.NoError(t, TouchChannelMonitorBeatTs("gone", 1, ChannelMonitorSourceUser))

	require.NoError(t, DeleteChannelMonitorStatesExcept([]string{"keep"}))

	states, err := GetChannelMonitorStates([]string{"keep", "gone"})
	require.NoError(t, err)
	require.Len(t, states, 1)
	assert.Equal(t, "keep", states[0].MonitorId)
}

func TestDeleteChannelMonitorBeatsBeforeRespectsRetentionCutoff(t *testing.T) {
	truncateTables(t)

	require.NoError(t, InsertChannelMonitorBeats([]ChannelMonitorBeat{
		{MonitorId: "m1", BucketTs: 100, Status: ChannelMonitorStatusUp},
		{MonitorId: "m1", BucketTs: 200, Status: ChannelMonitorStatusUp},
	}))

	require.NoError(t, DeleteChannelMonitorBeatsBefore(200))

	got, err := GetChannelMonitorRecentBeats("m1", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, int64(200), got[0].BucketTs)

	// A non-positive cutoff must be a no-op rather than wiping the table.
	require.NoError(t, DeleteChannelMonitorBeatsBefore(0))
	got, err = GetChannelMonitorRecentBeats("m1", 10)
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestResolveModelIconKeyFollowsFallbackChain(t *testing.T) {
	truncateTables(t)

	vendor := Vendor{Name: "Test Vendor", Icon: "Vendor.Color", Status: 1}
	require.NoError(t, vendor.Insert())

	require.NoError(t, DB.Create(&Model{
		ModelName: "model-with-own-icon",
		Icon:      "Explicit.Color",
		VendorID:  vendor.Id,
		Status:    1,
	}).Error)
	require.NoError(t, DB.Create(&Model{
		ModelName: "model-with-vendor-icon",
		VendorID:  vendor.Id,
		Status:    1,
	}).Error)
	require.NoError(t, DB.Create(&Model{
		ModelName: "model-without-any-icon",
		Status:    1,
	}).Error)

	cases := []struct {
		name      string
		modelName string
		expect    string
	}{
		{name: "model icon wins", modelName: "model-with-own-icon", expect: "Explicit.Color"},
		{name: "falls back to vendor icon", modelName: "model-with-vendor-icon", expect: "Vendor.Color"},
		{name: "unknown model falls back to name rules", modelName: "gpt-5-mini", expect: "OpenAI"},
		{name: "known name rule wins for unregistered model", modelName: "claude-opus-4", expect: "Claude.Color"},
		{name: "unrecognizable model yields no icon", modelName: "model-without-any-icon", expect: ""},
		{name: "blank input yields no icon", modelName: "   ", expect: ""},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expect, ResolveModelIconKey(testCase.modelName))
		})
	}
}
