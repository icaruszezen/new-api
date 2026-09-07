package channelmonitor

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestResetMonitorDataClearsHotBeatsForThatMonitor(t *testing.T) {
	setupResetTestDB(t)
	enableChannelMonitoring(t)
	t.Cleanup(clearHotBeats)

	Record(Sample{MonitorId: "m1", Status: model.ChannelMonitorStatusUp, TtftMs: 120, HasTtft: true})
	Record(Sample{MonitorId: "m2", Status: model.ChannelMonitorStatusDown})
	require.Equal(t, 1, countFilledHotBeats("m1"))
	require.Equal(t, 1, countFilledHotBeats("m2"))

	require.NoError(t, ResetMonitorData("m1"))

	assert.Equal(t, 0, countFilledHotBeats("m1"))
	assert.Equal(t, 1, countFilledHotBeats("m2"), "reset must not drop another monitor's pending sample")
}

func setupResetTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.ChannelMonitorBeat{},
		&model.ChannelMonitorStat{},
		&model.ChannelMonitorState{},
	))
	prev := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = prev
	})
}

func enableChannelMonitoring(t *testing.T) {
	t.Helper()
	saved := config.GlobalConfig.ExportAllConfigs()
	t.Cleanup(func() {
		_ = config.GlobalConfig.LoadFromDB(saved)
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"channel_monitoring_setting.enabled": "true",
	}))
}

func countFilledHotBeats(monitorId string) int {
	count := 0
	hotBeats.Range(func(key, value any) bool {
		if key.(beatKey).monitorId != monitorId {
			return true
		}
		beat := value.(*hotBeat)
		beat.mu.Lock()
		if beat.filled {
			count++
		}
		beat.mu.Unlock()
		return true
	})
	return count
}

func clearHotBeats() {
	hotBeats.Range(func(key, _ any) bool {
		hotBeats.Delete(key)
		return true
	})
}
