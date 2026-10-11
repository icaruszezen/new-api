package modeliqcheck

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/model_iq_check_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCheckViewsAssemblesSchedulingStateAndHistory(t *testing.T) {
	const now = int64(1_700_000_000)
	setting := model_iq_check_setting.ModelIQCheckSetting{
		Enabled:               true,
		HistoryLimit:          2,
		RequestTimeoutSeconds: 300,
	}
	checks := []Check{
		{Id: "done", Name: "Arithmetic", Group: "vip", Model: "gpt-5", Prompt: "1+1=?", IntervalMinutes: 30, Icon: "OpenAI", Enabled: true},
		{Id: "running", Name: "Page", Group: "vip", Model: "claude-sonnet-4", Prompt: "Write a page", IntervalMinutes: 60, Icon: "Claude", Enabled: true},
		{Id: "stale", Name: "Crashed", Group: "default", Model: "gpt-5", Prompt: "p", IntervalMinutes: 60, Icon: "OpenAI", Enabled: true},
		{Id: "never", Name: "New", Group: "default", Model: "gemini-2.5-pro", Prompt: "p", Icon: "Gemini", Enabled: true},
	}
	states := []model.ModelIQCheckState{
		{CheckId: "done", LastStartedAt: now - 600, LastFinishedAt: now - 540, RunRequestedAt: now - 30},
		{CheckId: "running", LastStartedAt: now - 60, LastFinishedAt: now - 3600},
		// Started longer ago than timeout + grace without finishing: the runner
		// died, so the card must not claim it is still running forever.
		{CheckId: "stale", LastStartedAt: now - 300 - runningGraceSeconds - 1, LastFinishedAt: now - 7200},
	}
	history := []model.ModelIQCheckResult{
		{Id: 9, CheckId: "done", CheckedAt: now - 540, Status: model.ModelIQCheckStatusFailed, DurationMs: 60_000, CompletionTokens: 0, ErrorMessage: "secret upstream https://internal.example.com failure"},
		{Id: 7, CheckId: "done", CheckedAt: now - 2400, Status: model.ModelIQCheckStatusSuccess, DurationMs: 12_000, CompletionTokens: 321},
		{Id: 5, CheckId: "done", CheckedAt: now - 4200, Status: model.ModelIQCheckStatusSuccess, DurationMs: 11_000, CompletionTokens: 300},
		{Id: 8, CheckId: "running", CheckedAt: now - 3600, Status: model.ModelIQCheckStatusSuccess, DurationMs: 90_000, CompletionTokens: 4096},
	}

	views := buildCheckViews(checks, states, history, now, setting)
	require.Len(t, views, 4)

	done := views[0]
	assert.Equal(t, "done", done.Id)
	assert.Equal(t, "Arithmetic", done.Name)
	assert.Equal(t, "gpt-5", done.Model)
	assert.Equal(t, "OpenAI", done.Icon)
	assert.Equal(t, "1+1=?", done.Prompt)
	assert.Equal(t, 30, done.IntervalMinutes)
	assert.False(t, done.Running)
	assert.True(t, done.Queued, "a run requested after the last start is queued")
	assert.Equal(t, now-540, done.LastCheckedAt)
	assert.Equal(t, now-600+30*60, done.NextCheckAt)
	require.Len(t, done.History, 2, "history is capped at HistoryLimit")
	assert.Equal(t, HistoryItemView{Id: 9, CheckedAt: now - 540, Status: ResultStatusFailed, DurationMs: 60_000}, done.History[0])
	assert.Equal(t, HistoryItemView{Id: 7, CheckedAt: now - 2400, Status: ResultStatusSuccess, DurationMs: 12_000, CompletionTokens: 321}, done.History[1])

	running := views[1]
	assert.True(t, running.Running)
	assert.False(t, running.Queued)
	assert.Equal(t, now-3600, running.LastCheckedAt)
	assert.Equal(t, now-60+60*60, running.NextCheckAt)

	stale := views[2]
	assert.False(t, stale.Running)
	assert.Empty(t, stale.History)

	never := views[3]
	assert.False(t, never.Running)
	assert.False(t, never.Queued)
	assert.Equal(t, DefaultIntervalMinutes, never.IntervalMinutes, "an unset interval is reported as the effective default")
	assert.Zero(t, never.LastCheckedAt)
	assert.Zero(t, never.NextCheckAt)
	require.NotNil(t, never.History, "an empty history must serialize as [] rather than null")
	assert.Empty(t, never.History)
}

// The status endpoint is public: it must never reveal routing groups, channel
// ids, or raw upstream errors to anonymous visitors.
func TestCheckViewJSONOmitsInternalRoutingAndErrors(t *testing.T) {
	views := buildCheckViews(
		[]Check{{Id: "c1", Name: "Pro", Group: "secret-vip-group", Model: "gpt-5", Prompt: "p", Icon: "OpenAI", Enabled: true}},
		[]model.ModelIQCheckState{{CheckId: "c1", LastStartedAt: 100, LastFinishedAt: 200}},
		[]model.ModelIQCheckResult{{Id: 1, CheckId: "c1", CheckedAt: 200, Status: model.ModelIQCheckStatusFailed, ChannelId: 42, ErrorMessage: "secret-upstream-error"}},
		1_000,
		model_iq_check_setting.ModelIQCheckSetting{HistoryLimit: 10, RequestTimeoutSeconds: 300},
	)

	payload, err := common.Marshal(StatusView{Enabled: true, Checks: views})
	require.NoError(t, err)
	body := string(payload)
	assert.NotContains(t, body, "secret-vip-group")
	assert.NotContains(t, body, "secret-upstream-error")
	assert.NotContains(t, body, `"group"`)
	assert.NotContains(t, body, `"channel`)
	assert.NotContains(t, body, `"error_message"`)
}

// Result details are public too: a failure is reported by category only, so
// organisation ids, key tails, or internal hosts in upstream errors stay private.
func TestPublicResultViewReportsErrorCategoryOnly(t *testing.T) {
	view := publicResultView(model.ModelIQCheckResult{
		Id:           7,
		CheckId:      "c1",
		CheckedAt:    200,
		Status:       model.ModelIQCheckStatusFailed,
		ModelName:    "gpt-5",
		Prompt:       "p",
		FinishReason: "length",
		ErrorMessage: "org-secret: invalid key sk-...abcd at http://10.0.0.5",
		ErrorType:    ErrorTypeUpstream,
		ChannelId:    42,
	})

	assert.Equal(t, ResultStatusFailed, view.Status)
	assert.Equal(t, ErrorTypeUpstream, view.ErrorType)
	assert.Equal(t, "gpt-5", view.Model)

	payload, err := common.Marshal(view)
	require.NoError(t, err)
	body := string(payload)
	assert.Contains(t, body, `"error_type":"upstream_error"`)
	assert.NotContains(t, body, "org-secret")
	assert.NotContains(t, body, `"error_message"`)
	assert.NotContains(t, body, `"channel`)
}
