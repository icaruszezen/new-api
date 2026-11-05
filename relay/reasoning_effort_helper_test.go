package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
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

// On pass-through channels the auto effort edits only the members it sets:
// tool and structured-output schemas keep the client's member order, which
// upstreams follow when they fill properties, and the rest of the body keeps
// its bytes (number spelling, unescaped characters).
func TestApplyPassThroughReasoningEffortFromModelKeepsClientBytes(t *testing.T) {
	tests := []struct {
		name  string
		apply func([]byte, bool) ([]byte, bool, error)
		input string
		want  string
	}{
		{
			name:  "claude adds thinking and sets output_config effort in place",
			apply: applyClaudePassThroughReasoningEffortFromModel,
			input: `{"model":"claude-x-high","output_config":{"format":{"type":"json_schema","schema":{"type":"object","properties":{"reasoning":{"type":"string"},"answer":{"type":"string"}}}}},"tools":[{"name":"a","input_schema":{"type":"object","properties":{"reasoning":{},"answer":{}}}}],"seed":12345678901234567890,"system":"<b>&","temperature":1.0}`,
			want:  `{"model":"claude-x-high","output_config":{"format":{"type":"json_schema","schema":{"type":"object","properties":{"reasoning":{"type":"string"},"answer":{"type":"string"}}}},"effort":"high"},"tools":[{"name":"a","input_schema":{"type":"object","properties":{"reasoning":{},"answer":{}}}}],"seed":12345678901234567890,"system":"<b>&","temperature":1.0,"thinking":{"type":"adaptive"}}`,
		},
		{
			name:  "claude keeps a thinking type and adds output_config",
			apply: applyClaudePassThroughReasoningEffortFromModel,
			input: `{"model":"claude-low","thinking":{"type":"enabled","budget_tokens":1024},"max_tokens":1e3}`,
			want:  `{"model":"claude-low","thinking":{"type":"enabled","budget_tokens":1024},"max_tokens":1e3,"output_config":{"effort":"low"}}`,
		},
		{
			name:  "claude sets a missing thinking type and replaces a non-object output_config",
			apply: applyClaudePassThroughReasoningEffortFromModel,
			input: `{"model":"m-medium","thinking":{"budget_tokens":2048},"output_config":null}`,
			want:  `{"model":"m-medium","thinking":{"budget_tokens":2048,"type":"adaptive"},"output_config":{"effort":"medium"}}`,
		},
		{
			name:  "openai sets reasoning_effort and reasoning.effort",
			apply: applyOpenAIPassThroughReasoningEffortFromModel,
			input: `{"model":"gpt-5-xhigh","reasoning":{"summary":"auto","effort":"low"},"response_format":{"type":"json_schema","json_schema":{"schema":{"properties":{"b":{},"a":{}}}}},"seed":12345678901234567890}`,
			want:  `{"model":"gpt-5-xhigh","reasoning":{"summary":"auto","effort":"xhigh"},"response_format":{"type":"json_schema","json_schema":{"schema":{"properties":{"b":{},"a":{}}}}},"seed":12345678901234567890,"reasoning_effort":"xhigh"}`,
		},
		{
			name:  "openai leaves a non-object reasoning",
			apply: applyOpenAIPassThroughReasoningEffortFromModel,
			input: `{"model":"o3-high","reasoning_effort":"low","reasoning":"x","n":1.0}`,
			want:  `{"model":"o3-high","reasoning_effort":"high","reasoning":"x","n":1.0}`,
		},
		{
			name:  "responses adds reasoning",
			apply: applyResponsesPassThroughReasoningEffortFromModel,
			input: `{"model":"gpt-5-medium","input":"<b>&","text":{"format":{"type":"json_schema","schema":{"properties":{"b":{},"a":{}}}}}}`,
			want:  `{"model":"gpt-5-medium","input":"<b>&","text":{"format":{"type":"json_schema","schema":{"properties":{"b":{},"a":{}}}}},"reasoning":{"effort":"medium"}}`,
		},
		{
			name:  "responses sets effort in reasoning",
			apply: applyResponsesPassThroughReasoningEffortFromModel,
			input: `{"model":"gpt-5-low","reasoning":{"summary":"auto"},"max_output_tokens":1e3}`,
			want:  `{"model":"gpt-5-low","reasoning":{"summary":"auto","effort":"low"},"max_output_tokens":1e3}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, changed, err := tt.apply([]byte(tt.input), true)
			require.NoError(t, err)
			assert.True(t, changed)
			assert.Equal(t, tt.want, string(updated))
		})
	}
}

// The DTO paths keep the raw reasoning and output_config objects as the
// client wrote them apart from effort, so a structured-output schema keeps
// its property order.
func TestApplyReasoningEffortFromModelKeepsRawObjects(t *testing.T) {
	claudeRequest := &dto.ClaudeRequest{
		OutputConfig: []byte(`{"format":{"type":"json_schema","schema":{"type":"object","properties":{"reasoning":{"type":"string"},"answer":{"type":"string"}}}},"effort":"low"}`),
	}
	openAIRequest := &dto.GeneralOpenAIRequest{Reasoning: []byte(`{"summary":"auto","effort":"low","max_tokens":1e3}`)}
	nullOutputConfigRequest := &dto.ClaudeRequest{OutputConfig: []byte(`null`)}

	require.True(t, applyClaudeReasoningEffortFromModel(claudeRequest, "claude-x-high", true))
	require.True(t, applyOpenAIReasoningEffortFromModel(openAIRequest, "gpt-5-xhigh", true))
	require.True(t, applyClaudeReasoningEffortFromModel(nullOutputConfigRequest, "claude-x-medium", true))

	assert.Equal(t, `{"format":{"type":"json_schema","schema":{"type":"object","properties":{"reasoning":{"type":"string"},"answer":{"type":"string"}}}},"effort":"high"}`, string(claudeRequest.OutputConfig))
	assert.Equal(t, `{"summary":"auto","effort":"xhigh","max_tokens":1e3}`, string(openAIRequest.Reasoning))
	assert.Equal(t, `{"effort":"medium"}`, string(nullOutputConfigRequest.OutputConfig))
}

func mustMarshalForReasoningEffortTest(t *testing.T, value any) []byte {
	t.Helper()
	data, err := common.Marshal(value)
	require.NoError(t, err)
	return data
}
