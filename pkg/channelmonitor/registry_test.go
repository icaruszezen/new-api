package channelmonitor

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/setting/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func allowAll(string) bool              { return true }
func allowAllPairs(string, string) bool { return true }
func denyGroup(string) bool             { return false }
func denyModel(string, string) bool     { return false }

func TestParseMonitorsSortsByConfiguredOrder(t *testing.T) {
	monitors, err := ParseMonitors(`[
		{"id":"b","name":"Second","group":"vip","model":"gpt-5","enabled":true,"sort":2},
		{"id":"a","name":"First","group":"default","model":"gpt-5","enabled":true,"sort":1}
	]`)
	require.NoError(t, err)
	require.Len(t, monitors, 2)
	assert.Equal(t, "a", monitors[0].Id)
	assert.Equal(t, "b", monitors[1].Id)
}

func TestParseMonitorsTreatsBlankConfigAsEmpty(t *testing.T) {
	for _, raw := range []string{"", "  ", "[]", "null"} {
		monitors, err := ParseMonitors(raw)
		require.NoError(t, err, "raw=%q", raw)
		assert.Empty(t, monitors, "raw=%q", raw)
	}
}

func TestParseMonitorsRejectsMalformedJSON(t *testing.T) {
	_, err := ParseMonitors(`[{"name":`)
	assert.Error(t, err)
}

func TestValidateMonitorsNormalizesAndAssignsIdentity(t *testing.T) {
	normalized, err := ValidateMonitors([]Monitor{
		{Name: "  Pro  ", Group: " vip ", Model: " gpt-5 ", Enabled: true, Sort: 99},
		{Id: "keep-me", Name: "Plus", Group: "default", Model: "gpt-5", Enabled: false},
	}, allowAll, allowAllPairs)
	require.NoError(t, err)
	require.Len(t, normalized, 2)

	assert.Equal(t, "Pro", normalized[0].Name)
	assert.Equal(t, "vip", normalized[0].Group)
	assert.Equal(t, "gpt-5", normalized[0].Model)
	assert.NotEmpty(t, normalized[0].Id, "a missing id must be generated so beats stay attributable")
	// Sort is rewritten from the submitted order so the admin list order wins.
	assert.Equal(t, 0, normalized[0].Sort)

	assert.Equal(t, "keep-me", normalized[1].Id, "an existing id must survive so historical beats stay linked")
	assert.Equal(t, 1, normalized[1].Sort)
	assert.Equal(t, UptimeScopeRecent, normalized[0].UptimeScope)
	assert.Equal(t, UptimeScopeRecent, normalized[1].UptimeScope)
}

func TestParseMonitorsReadsUptimeScopeAndDefaultsMissingToEmpty(t *testing.T) {
	monitors, err := ParseMonitors(`[
		{"id":"a","name":"Recent","group":"default","model":"gpt-5","enabled":true,"sort":1},
		{"id":"b","name":"All","group":"vip","model":"gpt-5","enabled":true,"sort":2,"uptime_scope":"all"}
	]`)
	require.NoError(t, err)
	require.Len(t, monitors, 2)
	assert.Empty(t, monitors[0].UptimeScope)
	assert.Equal(t, UptimeScopeRecent, uptimeScopeOf(monitors[0]))
	assert.Equal(t, UptimeScopeAll, monitors[1].UptimeScope)
}

func TestValidateMonitorsNormalizesUptimeScope(t *testing.T) {
	normalized, err := ValidateMonitors([]Monitor{
		{Name: "Recent", Group: "default", Model: "gpt-5", UptimeScope: "  "},
		{Name: "All", Group: "vip", Model: "gpt-5", UptimeScope: UptimeScopeAll},
	}, allowAll, allowAllPairs)
	require.NoError(t, err)
	assert.Equal(t, UptimeScopeRecent, normalized[0].UptimeScope)
	assert.Equal(t, UptimeScopeAll, normalized[1].UptimeScope)
}

func TestValidateMonitorsRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name         string
		monitors     []Monitor
		groupExists  func(string) bool
		modelInGroup func(string, string) bool
		wantMessage  string
	}{
		{
			name:         "empty name",
			monitors:     []Monitor{{Group: "default", Model: "gpt-5"}},
			groupExists:  allowAll,
			modelInGroup: allowAllPairs,
			wantMessage:  "display name is required",
		},
		{
			name:         "name too long",
			monitors:     []Monitor{{Name: strings.Repeat("x", MaxMonitorNameLength+1), Group: "default", Model: "gpt-5"}},
			groupExists:  allowAll,
			modelInGroup: allowAllPairs,
			wantMessage:  "display name must be at most",
		},
		{
			name:         "missing group",
			monitors:     []Monitor{{Name: "Pro", Model: "gpt-5"}},
			groupExists:  allowAll,
			modelInGroup: allowAllPairs,
			wantMessage:  "group is required",
		},
		{
			name:         "missing model",
			monitors:     []Monitor{{Name: "Pro", Group: "default"}},
			groupExists:  allowAll,
			modelInGroup: allowAllPairs,
			wantMessage:  "model is required",
		},
		{
			name:         "unknown group",
			monitors:     []Monitor{{Name: "Pro", Group: "ghost", Model: "gpt-5"}},
			groupExists:  denyGroup,
			modelInGroup: allowAllPairs,
			wantMessage:  "does not exist",
		},
		{
			name:         "model not routable in group",
			monitors:     []Monitor{{Name: "Pro", Group: "default", Model: "gpt-5"}},
			groupExists:  allowAll,
			modelInGroup: denyModel,
			wantMessage:  "is not available in group",
		},
		{
			name: "duplicated group and model pair",
			monitors: []Monitor{
				{Name: "Pro", Group: "default", Model: "gpt-5"},
				{Name: "Pro copy", Group: "default", Model: "gpt-5"},
			},
			groupExists:  allowAll,
			modelInGroup: allowAllPairs,
			wantMessage:  "duplicated monitor for group",
		},
		{
			name: "duplicated id",
			monitors: []Monitor{
				{Id: "same", Name: "Pro", Group: "default", Model: "gpt-5"},
				{Id: "same", Name: "Plus", Group: "vip", Model: "gpt-5"},
			},
			groupExists:  allowAll,
			modelInGroup: allowAllPairs,
			wantMessage:  "duplicated monitor id",
		},
		{
			name:         "invalid uptime scope",
			monitors:     []Monitor{{Name: "Pro", Group: "default", Model: "gpt-5", UptimeScope: "week"}},
			groupExists:  allowAll,
			modelInGroup: allowAllPairs,
			wantMessage:  "uptime scope must be",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ValidateMonitors(testCase.monitors, testCase.groupExists, testCase.modelInGroup)
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.wantMessage)
		})
	}
}

func TestValidateMonitorsEnforcesMaxCount(t *testing.T) {
	monitors := make([]Monitor, 0, MaxMonitors+1)
	for index := 0; index <= MaxMonitors; index++ {
		monitors = append(monitors, Monitor{
			Name:  "Monitor",
			Group: "default",
			Model: "model-" + strings.Repeat("x", index+1),
		})
	}

	_, err := ValidateMonitors(monitors, allowAll, allowAllPairs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most")
}

func TestRegistryKeepsLastGoodSnapshotWhenParseFails(t *testing.T) {
	resetMonitorRegistry()
	t.Cleanup(resetMonitorRegistry)

	saved := config.GlobalConfig.ExportAllConfigs()
	t.Cleanup(func() {
		_ = config.GlobalConfig.LoadFromDB(saved)
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"channel_monitoring_setting.monitors": `[{"id":"keep","name":"Pro","group":"default","model":"gpt-5","enabled":true}]`,
	}))
	got := Monitors()
	require.Len(t, got, 1)
	assert.Equal(t, "keep", got[0].Id)

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"channel_monitoring_setting.monitors": `[{"name":`,
	}))
	got = Monitors()
	require.Len(t, got, 1)
	assert.Equal(t, "keep", got[0].Id, "a malformed option must not wipe the last valid monitor list")
}

func resetMonitorRegistry() {
	monitorRegistry.mu.Lock()
	monitorRegistry.raw = ""
	monitorRegistry.monitors = nil
	monitorRegistry.byPair = nil
	monitorRegistry.mu.Unlock()
}

func TestValidateMonitorsAllowsSameModelAcrossGroups(t *testing.T) {
	normalized, err := ValidateMonitors([]Monitor{
		{Name: "Default", Group: "default", Model: "gpt-5"},
		{Name: "VIP", Group: "vip", Model: "gpt-5"},
	}, allowAll, allowAllPairs)
	require.NoError(t, err)
	assert.Len(t, normalized, 2)
}
