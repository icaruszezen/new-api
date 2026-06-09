package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestDetectModelReasoningEffort(t *testing.T) {
	tests := []struct {
		name   string
		model  string
		effort string
		ok     bool
	}{
		{name: "xhigh wins over high", model: "gpt-5-xhigh", effort: "xhigh", ok: true},
		{name: "case insensitive", model: "Claude-HIGH", effort: "high", ok: true},
		{name: "medium", model: "o3-medium-preview", effort: "medium", ok: true},
		{name: "low", model: "reasoning-low-model", effort: "low", ok: true},
		{name: "no match", model: "gpt-4o", effort: "", ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			effort, ok := detectModelReasoningEffort(tc.model)
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.effort, effort)
		})
	}
}

func TestApplyOpenAIReasoningEffortFromModel(t *testing.T) {
	reasoning := mustMarshalForReasoningEffortTest(t, map[string]any{"summary": "auto", "effort": "low"})
	request := &dto.GeneralOpenAIRequest{Model: "mapped-model", Reasoning: reasoning}

	changed := applyOpenAIReasoningEffortFromModel(request, "downstream-xhigh", true)

	require.True(t, changed)
	require.Equal(t, "xhigh", request.ReasoningEffort)
	var reasoningObj map[string]any
	require.NoError(t, common.Unmarshal(request.Reasoning, &reasoningObj))
	require.Equal(t, "xhigh", reasoningObj["effort"])
	require.Equal(t, "auto", reasoningObj["summary"])
}

func TestApplyOpenAIReasoningEffortFromModelDisabled(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{}

	changed := applyOpenAIReasoningEffortFromModel(request, "downstream-high", false)

	require.False(t, changed)
	require.Empty(t, request.ReasoningEffort)
}

func TestApplyResponsesReasoningEffortFromModel(t *testing.T) {
	request := &dto.OpenAIResponsesRequest{Model: "mapped-model", Reasoning: &dto.Reasoning{Summary: "auto", Effort: "low"}}

	changed := applyResponsesReasoningEffortFromModel(request, "downstream-medium", true)

	require.True(t, changed)
	require.NotNil(t, request.Reasoning)
	require.Equal(t, "medium", request.Reasoning.Effort)
	require.Equal(t, "auto", request.Reasoning.Summary)
}

func TestApplyClaudeReasoningEffortFromModel(t *testing.T) {
	request := &dto.ClaudeRequest{
		Model:        "mapped-model",
		Thinking:     &dto.Thinking{Type: "enabled", Display: "summarized"},
		OutputConfig: mustMarshalForReasoningEffortTest(t, map[string]any{"foo": "bar", "effort": "low"}),
	}

	changed := applyClaudeReasoningEffortFromModel(request, "downstream-high", true)

	require.True(t, changed)
	require.NotNil(t, request.Thinking)
	require.Equal(t, "enabled", request.Thinking.Type)
	require.Equal(t, "summarized", request.Thinking.Display)
	var outputConfig map[string]any
	require.NoError(t, common.Unmarshal(request.OutputConfig, &outputConfig))
	require.Equal(t, "high", outputConfig["effort"])
	require.Equal(t, "bar", outputConfig["foo"])
}

func TestApplyClaudeReasoningEffortCreatesThinking(t *testing.T) {
	request := &dto.ClaudeRequest{Model: "mapped-model"}

	changed := applyClaudeReasoningEffortFromModel(request, "downstream-low", true)

	require.True(t, changed)
	require.NotNil(t, request.Thinking)
	require.Equal(t, "adaptive", request.Thinking.Type)
	var outputConfig map[string]any
	require.NoError(t, common.Unmarshal(request.OutputConfig, &outputConfig))
	require.Equal(t, "low", outputConfig["effort"])
}

func TestApplyPassThroughReasoningEffortFromModel(t *testing.T) {
	body := mustMarshalForReasoningEffortTest(t, map[string]any{
		"model":     "client-xhigh",
		"reasoning": map[string]any{"summary": "auto", "effort": "low"},
	})

	updated, changed, err := applyResponsesPassThroughReasoningEffortFromModel(body, true)

	require.NoError(t, err)
	require.True(t, changed)
	var result map[string]any
	require.NoError(t, common.Unmarshal(updated, &result))
	reasoningObj, ok := result["reasoning"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "xhigh", reasoningObj["effort"])
	require.Equal(t, "auto", reasoningObj["summary"])
}

func TestApplyPassThroughReasoningEffortFromModelNoMatch(t *testing.T) {
	body := mustMarshalForReasoningEffortTest(t, map[string]any{"model": "gpt-4o"})

	updated, changed, err := applyOpenAIPassThroughReasoningEffortFromModel(body, true)

	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, body, updated)
}

func mustMarshalForReasoningEffortTest(t *testing.T, value any) []byte {
	t.Helper()
	data, err := common.Marshal(value)
	require.NoError(t, err)
	return data
}
