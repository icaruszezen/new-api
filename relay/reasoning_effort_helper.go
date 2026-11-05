package relay

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/tidwall/gjson"
)

const (
	reasoningEffortXHigh  = "xhigh"
	reasoningEffortHigh   = "high"
	reasoningEffortMedium = "medium"
	reasoningEffortLow    = "low"
)

var modelReasoningEffortPriority = []string{
	reasoningEffortXHigh,
	reasoningEffortHigh,
	reasoningEffortMedium,
	reasoningEffortLow,
}

func detectModelReasoningEffort(model string) (string, bool) {
	model = strings.ToLower(model)
	for _, effort := range modelReasoningEffortPriority {
		if strings.Contains(model, effort) {
			return effort, true
		}
	}
	return "", false
}

func applyOpenAIReasoningEffortFromModel(request *dto.GeneralOpenAIRequest, downstreamModel string, enabled bool) bool {
	if !enabled || request == nil {
		return false
	}
	effort, ok := detectModelReasoningEffort(downstreamModel)
	if !ok {
		return false
	}
	applyOpenAIReasoningEffort(request, effort)
	return true
}

func applyOpenAIReasoningEffort(request *dto.GeneralOpenAIRequest, effort string) {
	request.ReasoningEffort = effort
	if len(request.Reasoning) == 0 || common.GetJsonType(request.Reasoning) != "object" {
		return
	}
	updated, err := setJSONRawObjectString(request.Reasoning, "effort", effort)
	if err == nil {
		request.Reasoning = updated
	}
}

func applyResponsesReasoningEffortFromModel(request *dto.OpenAIResponsesRequest, downstreamModel string, enabled bool) bool {
	if !enabled || request == nil {
		return false
	}
	effort, ok := detectModelReasoningEffort(downstreamModel)
	if !ok {
		return false
	}
	applyResponsesReasoningEffort(request, effort)
	return true
}

func applyResponsesReasoningEffort(request *dto.OpenAIResponsesRequest, effort string) {
	if request.Reasoning == nil {
		request.Reasoning = &dto.Reasoning{}
	}
	request.Reasoning.Effort = effort
}

func applyClaudeReasoningEffortFromModel(request *dto.ClaudeRequest, downstreamModel string, enabled bool) bool {
	if !enabled || request == nil {
		return false
	}
	effort, ok := detectModelReasoningEffort(downstreamModel)
	if !ok {
		return false
	}
	applyClaudeReasoningEffort(request, effort)
	return true
}

func applyClaudeReasoningEffort(request *dto.ClaudeRequest, effort string) {
	if request.Thinking == nil {
		request.Thinking = &dto.Thinking{}
	}
	if request.Thinking.Type == "" {
		request.Thinking.Type = "adaptive"
	}
	updated, err := setJSONRawObjectString(request.OutputConfig, "effort", effort)
	if err == nil {
		request.OutputConfig = updated
	}
}

func applyOpenAIPassThroughReasoningEffortFromModel(data []byte, enabled bool) ([]byte, bool, error) {
	return applyPassThroughReasoningEffortFromModel(data, enabled, func(body []byte, effort string) ([]byte, error) {
		body, err := relaycommon.SetJSONMember(body, "", "reasoning_effort", effort)
		if err != nil || !gjson.GetBytes(body, "reasoning").IsObject() {
			return body, err
		}
		return relaycommon.SetJSONMember(body, "reasoning", "effort", effort)
	})
}

func applyResponsesPassThroughReasoningEffortFromModel(data []byte, enabled bool) ([]byte, bool, error) {
	return applyPassThroughReasoningEffortFromModel(data, enabled, func(body []byte, effort string) ([]byte, error) {
		if !gjson.GetBytes(body, "reasoning").IsObject() {
			return relaycommon.SetJSONMember(body, "", "reasoning", map[string]string{"effort": effort})
		}
		return relaycommon.SetJSONMember(body, "reasoning", "effort", effort)
	})
}

func applyClaudePassThroughReasoningEffortFromModel(data []byte, enabled bool) ([]byte, bool, error) {
	return applyPassThroughReasoningEffortFromModel(data, enabled, func(body []byte, effort string) ([]byte, error) {
		var err error
		thinking := gjson.GetBytes(body, "thinking")
		if !thinking.IsObject() {
			body, err = relaycommon.SetJSONMember(body, "", "thinking", map[string]string{"type": "adaptive"})
		} else if thinking.Get("type").Type != gjson.String {
			body, err = relaycommon.SetJSONMember(body, "thinking", "type", "adaptive")
		}
		if err != nil {
			return nil, err
		}
		if !gjson.GetBytes(body, "output_config").IsObject() {
			return relaycommon.SetJSONMember(body, "", "output_config", map[string]string{"effort": effort})
		}
		return relaycommon.SetJSONMember(body, "output_config", "effort", effort)
	})
}

// applyPassThroughReasoningEffortFromModel edits only the members apply sets,
// so the rest of a pass-through body (tool and output schemas, numbers,
// characters) reaches the upstream as the client sent it. data is the shared
// request body and is never changed.
func applyPassThroughReasoningEffortFromModel(data []byte, enabled bool, apply func([]byte, string) ([]byte, error)) ([]byte, bool, error) {
	if !enabled {
		return data, false, nil
	}
	// sjson does not report an error on invalid JSON.
	if !gjson.ValidBytes(data) || common.GetJsonType(data) != "object" {
		return nil, false, errors.New("failed to parse pass-through request body for reasoning effort: request body must be a JSON object")
	}
	model := gjson.GetBytes(data, "model")
	if model.Type != gjson.String {
		return data, false, nil
	}
	effort, ok := detectModelReasoningEffort(model.Str)
	if !ok {
		return data, false, nil
	}
	updated, err := apply(data, effort)
	if err != nil {
		return nil, false, fmt.Errorf("failed to set reasoning effort in pass-through request body: %w", err)
	}
	return updated, true, nil
}

// setJSONRawObjectString sets key in the raw JSON object and keeps the other
// members as they are, so a structured-output schema keeps its property
// order. Empty or null raw starts a new object.
func setJSONRawObjectString(raw []byte, key string, value string) ([]byte, error) {
	if trimmed := bytes.TrimSpace(raw); len(trimmed) == 0 || string(trimmed) == "null" {
		raw = []byte("{}")
	}
	if !gjson.ValidBytes(raw) || common.GetJsonType(raw) != "object" {
		return nil, errors.New("not a JSON object")
	}
	return relaycommon.SetJSONMember(raw, "", key, value)
}
