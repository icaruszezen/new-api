package modeliqcheck

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyClaudeThinkingFollowsEachModelGeneration(t *testing.T) {
	adaptive := func(display string) *dto.Thinking { return &dto.Thinking{Type: "adaptive", Display: display} }
	budget := func(tokens int) *dto.Thinking {
		return &dto.Thinking{Type: "enabled", BudgetTokens: common.GetPointer(tokens)}
	}
	cases := []struct {
		name          string
		model         string
		effort        string
		wantThinking  *dto.Thinking
		wantEffort    string
		wantMaxTokens uint
	}{
		// budget_tokens returns 400 from Opus 4.7 on, and thinking text is omitted unless summarized.
		{name: "opus 4.8 high", model: "claude-opus-4-8", effort: "high", wantThinking: adaptive("summarized"), wantEffort: "high"},
		{name: "opus 4.8 none omits thinking", model: "claude-opus-4-8", effort: "none"},
		{name: "opus 4.7 dotted name", model: "anthropic/claude-opus-4.7", effort: "xhigh", wantThinking: adaptive("summarized"), wantEffort: "xhigh"},
		{name: "opus 4.6 has no xhigh", model: "claude-opus-4-6", effort: "xhigh", wantThinking: adaptive(""), wantEffort: "max"},
		{name: "sonnet 4.6 minimal maps to low", model: "claude-sonnet-4-6", effort: "minimal", wantThinking: adaptive(""), wantEffort: "low"},
		{name: "opus 5 disables thinking", model: "claude-opus-5", effort: "none", wantThinking: &dto.Thinking{Type: "disabled"}},
		{name: "opus 5.5 cannot disable thinking", model: "claude-opus-5-5", effort: "none", wantThinking: adaptive("summarized"), wantEffort: "low"},
		{name: "sonnet 5.5 turns thinking off between tools", model: "claude-sonnet-5-5", effort: "none", wantThinking: &dto.Thinking{Type: "between_tools"}},
		{name: "sonnet 5 disables thinking", model: "claude-sonnet-5", effort: "none", wantThinking: &dto.Thinking{Type: "disabled"}},
		{name: "haiku 5.5 disables thinking", model: "claude-haiku-5-5", effort: "none", wantThinking: &dto.Thinking{Type: "disabled"}},
		{name: "fable max", model: "claude-fable-5-1", effort: "max", wantThinking: adaptive("summarized"), wantEffort: "max"},
		{name: "vertex model id", model: "claude-opus-4-8@20260301", effort: "medium", wantThinking: adaptive("summarized"), wantEffort: "medium"},
		{name: "bedrock model id", model: "us.anthropic.claude-sonnet-5-5-20260801-v1:0", effort: "low", wantThinking: adaptive("summarized"), wantEffort: "low"},
		// Older models only take a manual budget, which must stay below max_tokens.
		{name: "sonnet 4.5 medium budget", model: "claude-sonnet-4-5-20250929", effort: "medium", wantThinking: budget(4096), wantMaxTokens: 8192},
		{name: "opus 4 date suffix is not a minor version", model: "claude-opus-4-20250514", effort: "high", wantThinking: budget(6144), wantMaxTokens: 8192},
		{name: "haiku 4.5 minimal budget", model: "claude-haiku-4-5", effort: "minimal", wantThinking: budget(1024), wantMaxTokens: 8192},
		{name: "claude 3.7 sonnet", model: "claude-3-7-sonnet-20250219", effort: "low", wantThinking: budget(2048), wantMaxTokens: 8192},
		{name: "legacy none omits thinking", model: "claude-sonnet-4-5", effort: "none"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fields, err := ApplyClaudeThinking(testCase.model, testCase.effort, true, ClaudeThinkingFields{MaxTokens: 8192})
			require.NoError(t, err)

			assert.Equal(t, testCase.wantThinking, fields.Thinking)
			assert.Equal(t, testCase.wantEffort, outputConfigEffort(t, fields.OutputConfig))
			wantMaxTokens := testCase.wantMaxTokens
			if wantMaxTokens == 0 {
				wantMaxTokens = 8192
			}
			assert.Equal(t, wantMaxTokens, fields.MaxTokens)
		})
	}
}

// A manual budget needs room for both thinking and the answer, so a small
// configured limit is raised instead of producing a request Anthropic rejects.
func TestApplyClaudeThinkingRaisesSmallMaxTokensForBudget(t *testing.T) {
	fields, err := ApplyClaudeThinking("claude-sonnet-4-5", "max", false, ClaudeThinkingFields{MaxTokens: 16})
	require.NoError(t, err)

	assert.Equal(t, uint(2048), fields.MaxTokens)
	require.NotNil(t, fields.Thinking)
	assert.Equal(t, 1024, *fields.Thinking.BudgetTokens)
}

func TestApplyClaudeThinkingKeepsOtherOutputConfig(t *testing.T) {
	fields, err := ApplyClaudeThinking("claude-opus-4-8", "high", false, ClaudeThinkingFields{
		OutputConfig: json.RawMessage(`{"format":{"type":"json_schema"},"effort":"low"}`),
		MaxTokens:    8192,
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"format":{"type":"json_schema"},"effort":"high"}`, string(fields.OutputConfig))

	fields, err = ApplyClaudeThinking("claude-opus-4-8", "none", false, fields)
	require.NoError(t, err)
	assert.JSONEq(t, `{"format":{"type":"json_schema"}}`, string(fields.OutputConfig))
}

// With the model default effort only the summary is requested, and only where
// thinking runs by default with its text omitted; thinking set from a model
// name suffix is kept.
func TestApplyClaudeThinkingDefaultEffortOnlyRequestsSummary(t *testing.T) {
	cases := []struct {
		name         string
		model        string
		summary      bool
		current      *dto.Thinking
		wantThinking *dto.Thinking
	}{
		{name: "adaptive by default", model: "claude-opus-5-5", summary: true, wantThinking: &dto.Thinking{Type: "adaptive", Display: "summarized"}},
		{name: "summary off", model: "claude-opus-5-5"},
		{name: "thinking off by default", model: "claude-opus-4-8", summary: true},
		{name: "suffix thinking kept", model: "claude-opus-4-8", summary: true, current: &dto.Thinking{Type: "adaptive"}, wantThinking: &dto.Thinking{Type: "adaptive", Display: "summarized"}},
		{name: "4.6 already returns summaries", model: "claude-opus-4-6", summary: true, current: &dto.Thinking{Type: "adaptive"}, wantThinking: &dto.Thinking{Type: "adaptive"}},
		{name: "legacy budget untouched", model: "claude-sonnet-4-5", summary: true, current: &dto.Thinking{Type: "enabled", BudgetTokens: common.GetPointer(2000)}, wantThinking: &dto.Thinking{Type: "enabled", BudgetTokens: common.GetPointer(2000)}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fields, err := ApplyClaudeThinking(testCase.model, "", testCase.summary, ClaudeThinkingFields{Thinking: testCase.current, MaxTokens: 8192})
			require.NoError(t, err)
			assert.Equal(t, testCase.wantThinking, fields.Thinking)
			assert.Nil(t, fields.OutputConfig)
		})
	}
}

func TestOpenAIReasoningEffortFollowsEachModelGeneration(t *testing.T) {
	cases := []struct {
		model  string
		effort string
		want   string
	}{
		{model: "gpt-5", effort: "none", want: "minimal"},
		{model: "gpt-5-mini", effort: "xhigh", want: "high"},
		{model: "gpt-5-pro", effort: "low", want: "high"},
		{model: "gpt-5-codex", effort: "minimal", want: "low"},
		{model: "gpt-5-codex", effort: "max", want: "high"},
		{model: "gpt-5.1", effort: "minimal", want: "low"},
		{model: "gpt-5.1", effort: "none", want: "none"},
		{model: "gpt-5.1", effort: "xhigh", want: "high"},
		{model: "gpt-5.1-codex-max", effort: "max", want: "xhigh"},
		{model: "gpt-5.2", effort: "max", want: "xhigh"},
		{model: "openai/gpt-5.4", effort: "minimal", want: "low"},
		{model: "gpt-5.5", effort: "xhigh", want: "xhigh"},
		{model: "gpt-5.2-pro", effort: "none", want: "medium"},
		{model: "gpt-5.2-pro", effort: "max", want: "xhigh"},
		{model: "gpt-5.6", effort: "max", want: "max"},
		{model: "gpt-5.6", effort: "minimal", want: "low"},
		{model: "gpt-6-astra", effort: "none", want: "low"},
		{model: "gpt-6.1-sol", effort: "max", want: "max"},
		{model: "gpt-6-luna", effort: "none", want: "none"},
		{model: "ft:o4-mini-2025-04-16:org::abc", effort: "max", want: "high"},
		{model: "o3", effort: "none", want: "low"},
		{model: "o4-mini", effort: "max", want: "high"},
		{model: "o3-pro", effort: "medium", want: "medium"},
		// Non-reasoning OpenAI models reject the parameter.
		{model: "gpt-4o", effort: "high", want: ""},
		{model: "gpt-4.1-mini", effort: "low", want: ""},
		{model: "ft:gpt-4.1-mini-2025-04-14:org::abc", effort: "high", want: ""},
		// Other providers decide which values they accept.
		{model: "deepseek-v4-pro", effort: "max", want: "max"},
		{model: "grok-4", effort: "none", want: "none"},
		{model: "google/gemini-2.5-flash", effort: "none", want: "none"},
		// Models that cannot turn thinking off reject none behind OpenAI-compatible proxies.
		{model: "google/gemini-2.5-pro", effort: "none", want: "low"},
		{model: "google/gemini-3-flash-preview", effort: "none", want: "low"},
		{model: "anthropic/claude-opus-5.5", effort: "none", want: "low"},
		{model: "anthropic/claude-sonnet-4.5", effort: "none", want: "none"},
		{model: "gpt-5", effort: "", want: ""},
	}
	for _, testCase := range cases {
		t.Run(testCase.model+"/"+testCase.effort, func(t *testing.T) {
			assert.Equal(t, testCase.want, OpenAIReasoningEffort(testCase.model, testCase.effort))
		})
	}
}

func TestResponsesReasoningRequestsSummaryForReasoningModels(t *testing.T) {
	cases := []struct {
		name    string
		model   string
		effort  string
		summary bool
		current *dto.Reasoning
		want    *dto.Reasoning
	}{
		{name: "effort and summary", model: "gpt-5.2", effort: "max", summary: true, want: &dto.Reasoning{Effort: "xhigh", Summary: "auto"}},
		{name: "summary with model default effort", model: "gpt-5.1-codex", summary: true, want: &dto.Reasoning{Summary: "auto"}},
		{name: "no summary when reasoning is off", model: "gpt-5.2", effort: "none", summary: true, want: &dto.Reasoning{Effort: "none"}},
		{name: "keeps existing fields", model: "gpt-5", effort: "low", current: &dto.Reasoning{Summary: "detailed"}, want: &dto.Reasoning{Effort: "low", Summary: "detailed"}},
		{name: "nothing to change", model: "gpt-5"},
		{name: "non-reasoning model untouched", model: "gpt-4o", effort: "high", summary: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, ResponsesReasoning(testCase.model, testCase.effort, testCase.summary, testCase.current))
		})
	}
}

func TestGeminiThinkingConfigFollowsEachModelGeneration(t *testing.T) {
	budget := func(tokens int, include bool) *dto.GeminiThinkingConfig {
		return &dto.GeminiThinkingConfig{ThinkingBudget: common.GetPointer(tokens), IncludeThoughts: include}
	}
	level := func(value string, include bool) *dto.GeminiThinkingConfig {
		return &dto.GeminiThinkingConfig{ThinkingLevel: value, IncludeThoughts: include}
	}
	cases := []struct {
		name    string
		model   string
		effort  string
		summary bool
		want    *dto.GeminiThinkingConfig
	}{
		{name: "2.5 flash off", model: "gemini-2.5-flash", effort: "none", summary: true, want: budget(0, false)},
		{name: "2.5 pro cannot turn thinking off", model: "gemini-2.5-pro", effort: "none", want: budget(128, false)},
		{name: "2.5 pro max", model: "gemini-2.5-pro", effort: "max", summary: true, want: budget(32768, true)},
		{name: "2.5 flash max", model: "gemini-2.5-flash", effort: "max", want: budget(24576, false)},
		{name: "2.5 flash-lite minimal", model: "gemini-2.5-flash-lite", effort: "minimal", want: budget(512, false)},
		{name: "2.5 flash medium", model: "gemini-2.5-flash-preview-09-2025", effort: "medium", summary: true, want: budget(8192, true)},
		{name: "3 pro minimal is low", model: "gemini-3-pro-preview", effort: "minimal", want: level("low", false)},
		{name: "3 pro has no medium", model: "gemini-3-pro-preview", effort: "medium", want: level("high", false)},
		{name: "3.1 pro medium", model: "gemini-3.1-pro-preview", effort: "medium", summary: true, want: level("medium", true)},
		{name: "3 flash off", model: "gemini-3-flash-preview", effort: "none", want: level("minimal", false)},
		{name: "3 flash xhigh", model: "gemini-3-flash-preview", effort: "xhigh", summary: true, want: level("high", true)},
		{name: "3.5 flash-lite off", model: "gemini-3.5-flash-lite", effort: "none", want: level("minimal", false)},
		{name: "3.8 flash has no minimal", model: "gemini-3.8-flash", effort: "minimal", want: level("low", false)},
		{name: "latest alias follows the newest generation", model: "gemini-pro-latest", effort: "none", want: level("low", false)},
		{name: "latest flash alias", model: "gemini-flash-latest", effort: "medium", summary: true, want: level("medium", true)},
		{name: "summary with model default", model: "gemini-3-pro-preview", summary: true, want: &dto.GeminiThinkingConfig{IncludeThoughts: true}},
		{name: "2.0 has no thinking", model: "gemini-2.0-flash", effort: "high", summary: true},
		{name: "unrecognized model", model: "gemma-3-27b-it", effort: "high", summary: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, GeminiThinkingConfig(testCase.model, testCase.effort, testCase.summary, nil))
		})
	}
}

// The summary request keeps a budget the channel already set.
func TestGeminiThinkingConfigSummaryKeepsExistingBudget(t *testing.T) {
	current := &dto.GeminiThinkingConfig{ThinkingBudget: common.GetPointer(2048)}

	got := GeminiThinkingConfig("gemini-2.5-flash", "", true, current)

	assert.Equal(t, &dto.GeminiThinkingConfig{ThinkingBudget: common.GetPointer(2048), IncludeThoughts: true}, got)
	assert.False(t, current.IncludeThoughts, "the caller's config must not be modified in place")
}

func outputConfigEffort(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	if len(raw) == 0 {
		return ""
	}
	var config struct {
		Effort string `json:"effort"`
	}
	require.NoError(t, common.Unmarshal(raw, &config))
	return config.Effort
}
