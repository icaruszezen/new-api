package modeliqcheck

import (
	"errors"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/setting/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func allowAllGroups(string) bool         { return true }
func allowAllModels(string, string) bool { return true }

func validCheck(name string) Check {
	return Check{Name: name, Group: "default", Model: "gpt-5", Prompt: "1+1=?", Enabled: true}
}

func TestParseChecksSortsByConfiguredOrderAndTreatsBlankAsEmpty(t *testing.T) {
	checks, err := ParseChecks(`[
		{"id":"b","name":"Second","group":"vip","model":"gpt-5","prompt":"p","enabled":true,"sort":2},
		{"id":"a","name":"First","group":"default","model":"gpt-5","prompt":"p","enabled":true,"sort":1}
	]`)
	require.NoError(t, err)
	require.Len(t, checks, 2)
	assert.Equal(t, "a", checks[0].Id)
	assert.Equal(t, "b", checks[1].Id)

	for _, raw := range []string{"", "  ", "[]", "null"} {
		checks, err := ParseChecks(raw)
		require.NoError(t, err, "raw=%q", raw)
		assert.Empty(t, checks, "raw=%q", raw)
	}

	_, err = ParseChecks(`[{"name":`)
	assert.Error(t, err)
}

func TestValidateChecksNormalizesAndAppliesDefaults(t *testing.T) {
	normalized, err := ValidateChecks([]Check{
		{Name: "  Pro  ", Group: " vip ", Model: " gpt-5 ", Prompt: "\n  Write a web page  \n", Icon: " OpenAI ", Enabled: true, Sort: 99},
		{Id: "keep-me", Name: "Plus", Group: "default", Model: "gpt-5", Prompt: "1+1=?", IntervalMinutes: 30, MaxTokens: 4096,
			EndpointType: " Anthropic ", ReasoningEffort: " XHigh ", ReasoningSummary: true},
	}, allowAllGroups, allowAllModels)
	require.NoError(t, err)
	require.Len(t, normalized, 2)

	first := normalized[0]
	assert.Equal(t, "Pro", first.Name)
	assert.Equal(t, "vip", first.Group)
	assert.Equal(t, "gpt-5", first.Model)
	assert.Equal(t, "Write a web page", first.Prompt)
	assert.Equal(t, "OpenAI", first.Icon)
	assert.Len(t, first.Id, 16, "a missing id must be generated so results stay attributable")
	assert.Equal(t, DefaultIntervalMinutes, first.IntervalMinutes, "an unset interval falls back to the default")
	assert.Equal(t, 0, first.MaxTokens, "0 means the request carries no max tokens field")
	assert.Empty(t, first.EndpointType, "an empty request format keeps auto detection")
	assert.Empty(t, first.ReasoningEffort, "an empty reasoning effort keeps the model default")
	// Sort is rewritten from the submitted order so the admin list order wins.
	assert.Equal(t, 0, first.Sort)

	second := normalized[1]
	assert.Equal(t, "keep-me", second.Id, "an existing id must survive so its history stays linked")
	assert.Equal(t, 30, second.IntervalMinutes)
	assert.Equal(t, 4096, second.MaxTokens)
	assert.Equal(t, "anthropic", second.EndpointType)
	assert.Equal(t, "xhigh", second.ReasoningEffort)
	assert.True(t, second.ReasoningSummary)
	assert.Equal(t, 1, second.Sort)
	assert.False(t, second.Enabled)
}

func TestValidateChecksAllowsSameModelWithDifferentPrompts(t *testing.T) {
	text := validCheck("Arithmetic")
	page := validCheck("Landing page")
	page.Prompt = "Write a landing page in a single HTML file"

	normalized, err := ValidateChecks([]Check{text, page}, allowAllGroups, allowAllModels)
	require.NoError(t, err)
	require.Len(t, normalized, 2)
	assert.NotEqual(t, normalized[0].Id, normalized[1].Id)
}

func TestValidateChecksAcceptsBoundaryValues(t *testing.T) {
	check := validCheck(strings.Repeat("名", MaxNameLength))
	check.Prompt = strings.Repeat("问", MaxPromptLength)
	check.IntervalMinutes = MinIntervalMinutes
	check.MaxTokens = MaxMaxTokens
	check.Icon = strings.Repeat("x", MaxIconLength)
	check.Id = strings.Repeat("i", MaxIdLength)

	other := validCheck("Weekly")
	other.IntervalMinutes = MaxIntervalMinutes

	_, err := ValidateChecks([]Check{check, other}, allowAllGroups, allowAllModels)
	require.NoError(t, err)
}

func TestValidateChecksRejectsInvalidInput(t *testing.T) {
	withField := func(mutate func(*Check)) []Check {
		check := validCheck("Pro")
		mutate(&check)
		return []Check{check}
	}

	cases := []struct {
		name         string
		checks       []Check
		groupExists  func(string) bool
		modelInGroup func(string, string) bool
		wantKey      string
		wantParams   map[string]any
	}{
		{
			name:       "empty name",
			checks:     withField(func(c *Check) { c.Name = "   " }),
			wantKey:    i18n.MsgModelIQCheckNameRequired,
			wantParams: map[string]any{"Index": 1},
		},
		{
			name:       "name too long counts characters",
			checks:     withField(func(c *Check) { c.Name = strings.Repeat("名", MaxNameLength+1) }),
			wantKey:    i18n.MsgModelIQCheckNameTooLong,
			wantParams: map[string]any{"Name": strings.Repeat("名", MaxNameLength+1), "Max": MaxNameLength},
		},
		{
			name:    "missing group",
			checks:  withField(func(c *Check) { c.Group = "" }),
			wantKey: i18n.MsgModelIQCheckGroupRequired,
		},
		{
			name:    "missing model",
			checks:  withField(func(c *Check) { c.Model = "" }),
			wantKey: i18n.MsgModelIQCheckModelRequired,
		},
		{
			name:        "unknown group",
			checks:      withField(func(c *Check) { c.Group = "ghost" }),
			groupExists: func(group string) bool { return group != "ghost" },
			wantKey:     i18n.MsgModelIQCheckGroupNotFound,
			wantParams:  map[string]any{"Name": "Pro", "Group": "ghost"},
		},
		{
			name:         "model not routable in group",
			checks:       withField(func(c *Check) {}),
			modelInGroup: func(string, string) bool { return false },
			wantKey:      i18n.MsgModelIQCheckModelNotInGroup,
			wantParams:   map[string]any{"Name": "Pro", "Model": "gpt-5", "Group": "default"},
		},
		{
			name:    "blank prompt",
			checks:  withField(func(c *Check) { c.Prompt = " \n\t " }),
			wantKey: i18n.MsgModelIQCheckPromptRequired,
		},
		{
			name:    "prompt too long",
			checks:  withField(func(c *Check) { c.Prompt = strings.Repeat("问", MaxPromptLength+1) }),
			wantKey: i18n.MsgModelIQCheckPromptTooLong,
		},
		{
			name:       "interval below minimum",
			checks:     withField(func(c *Check) { c.IntervalMinutes = MinIntervalMinutes - 1 }),
			wantKey:    i18n.MsgModelIQCheckIntervalInvalid,
			wantParams: map[string]any{"Name": "Pro", "Min": MinIntervalMinutes, "Max": MaxIntervalMinutes},
		},
		{
			name:    "negative interval",
			checks:  withField(func(c *Check) { c.IntervalMinutes = -60 }),
			wantKey: i18n.MsgModelIQCheckIntervalInvalid,
		},
		{
			name:    "interval above maximum",
			checks:  withField(func(c *Check) { c.IntervalMinutes = MaxIntervalMinutes + 1 }),
			wantKey: i18n.MsgModelIQCheckIntervalInvalid,
		},
		{
			name:    "negative max tokens",
			checks:  withField(func(c *Check) { c.MaxTokens = -1 }),
			wantKey: i18n.MsgModelIQCheckMaxTokensInvalid,
		},
		{
			name:    "max tokens above maximum",
			checks:  withField(func(c *Check) { c.MaxTokens = MaxMaxTokens + 1 }),
			wantKey: i18n.MsgModelIQCheckMaxTokensInvalid,
		},
		{
			name:       "unsupported request format",
			checks:     withField(func(c *Check) { c.EndpointType = "embeddings" }),
			wantKey:    i18n.MsgModelIQCheckEndpointInvalid,
			wantParams: map[string]any{"Name": "Pro", "Value": "embeddings"},
		},
		{
			name:       "unsupported reasoning effort",
			checks:     withField(func(c *Check) { c.ReasoningEffort = "extreme" }),
			wantKey:    i18n.MsgModelIQCheckEffortInvalid,
			wantParams: map[string]any{"Name": "Pro", "Value": "extreme"},
		},
		{
			name:    "icon too long",
			checks:  withField(func(c *Check) { c.Icon = strings.Repeat("x", MaxIconLength+1) }),
			wantKey: i18n.MsgModelIQCheckIconTooLong,
		},
		{
			name:    "id too long",
			checks:  withField(func(c *Check) { c.Id = strings.Repeat("i", MaxIdLength+1) }),
			wantKey: i18n.MsgModelIQCheckIdTooLong,
		},
		{
			name: "duplicated id",
			checks: []Check{
				{Id: "same", Name: "Pro", Group: "default", Model: "gpt-5", Prompt: "a"},
				{Id: " same ", Name: "Plus", Group: "vip", Model: "gpt-5", Prompt: "b"},
			},
			wantKey:    i18n.MsgModelIQCheckIdDuplicated,
			wantParams: map[string]any{"Id": "same"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ValidateChecks(testCase.checks, testCase.groupExists, testCase.modelInGroup)
			require.Error(t, err)
			var validationErr *ValidationError
			require.True(t, errors.As(err, &validationErr), "validation failures must carry an i18n key")
			assert.Equal(t, testCase.wantKey, validationErr.Key)
			if testCase.wantParams != nil {
				assert.Equal(t, testCase.wantParams, validationErr.Params)
			}
			assert.NotEmpty(t, err.Error(), "the English fallback message must not be empty")
		})
	}
}

func TestValidateChecksEnforcesMaxCount(t *testing.T) {
	checks := make([]Check, 0, MaxChecks+1)
	for index := 0; index <= MaxChecks; index++ {
		checks = append(checks, validCheck("Check"))
	}

	_, err := ValidateChecks(checks, allowAllGroups, allowAllModels)
	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, i18n.MsgModelIQCheckTooMany, validationErr.Key)
	assert.Equal(t, map[string]any{"Max": MaxChecks}, validationErr.Params)

	_, err = ValidateChecks(checks[:MaxChecks], allowAllGroups, allowAllModels)
	require.NoError(t, err)
}

func TestRegistryKeepsLastGoodSnapshotWhenParseFails(t *testing.T) {
	resetCheckRegistry()
	t.Cleanup(resetCheckRegistry)

	saved := config.GlobalConfig.ExportAllConfigs()
	t.Cleanup(func() {
		_ = config.GlobalConfig.LoadFromDB(saved)
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"model_iq_check_setting.checks": `[
			{"id":"off","name":"Off","group":"default","model":"gpt-5","prompt":"p","enabled":false,"sort":0},
			{"id":"keep","name":"Pro","group":"default","model":"gpt-5","prompt":"p","enabled":true,"sort":1}
		]`,
	}))
	require.Len(t, Checks(), 2)
	enabled := EnabledChecks()
	require.Len(t, enabled, 1)
	assert.Equal(t, "keep", enabled[0].Id)
	found, ok := FindCheck("off")
	require.True(t, ok, "FindCheck must also see disabled checks")
	assert.False(t, found.Enabled)
	_, ok = FindCheck("missing")
	assert.False(t, ok)

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"model_iq_check_setting.checks": `[{"name":`,
	}))
	enabled = EnabledChecks()
	require.Len(t, enabled, 1)
	assert.Equal(t, "keep", enabled[0].Id, "a malformed option must not wipe the last valid check list")
}

func resetCheckRegistry() {
	checkRegistry.mu.Lock()
	checkRegistry.raw = ""
	checkRegistry.checks = nil
	checkRegistry.mu.Unlock()
}
