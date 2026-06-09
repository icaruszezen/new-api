package relay

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
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
	return applyPassThroughReasoningEffortFromModel(data, enabled, func(body map[string]any, effort string) {
		body["reasoning_effort"] = effort
		if reasoningObj, ok := body["reasoning"].(map[string]any); ok {
			reasoningObj["effort"] = effort
		}
	})
}

func applyResponsesPassThroughReasoningEffortFromModel(data []byte, enabled bool) ([]byte, bool, error) {
	return applyPassThroughReasoningEffortFromModel(data, enabled, func(body map[string]any, effort string) {
		reasoningObj, ok := body["reasoning"].(map[string]any)
		if !ok {
			reasoningObj = map[string]any{}
			body["reasoning"] = reasoningObj
		}
		reasoningObj["effort"] = effort
	})
}

func applyClaudePassThroughReasoningEffortFromModel(data []byte, enabled bool) ([]byte, bool, error) {
	return applyPassThroughReasoningEffortFromModel(data, enabled, func(body map[string]any, effort string) {
		thinkingObj, ok := body["thinking"].(map[string]any)
		if !ok {
			thinkingObj = map[string]any{}
			body["thinking"] = thinkingObj
		}
		if _, ok := thinkingObj["type"].(string); !ok {
			thinkingObj["type"] = "adaptive"
		}
		outputConfigObj, ok := body["output_config"].(map[string]any)
		if !ok {
			outputConfigObj = map[string]any{}
			body["output_config"] = outputConfigObj
		}
		outputConfigObj["effort"] = effort
	})
}

func applyPassThroughReasoningEffortFromModel(data []byte, enabled bool, apply func(map[string]any, string)) ([]byte, bool, error) {
	if !enabled {
		return data, false, nil
	}
	var body map[string]any
	if err := common.Unmarshal(data, &body); err != nil {
		return nil, false, fmt.Errorf("failed to parse pass-through request body for reasoning effort: %w", err)
	}
	model, _ := body["model"].(string)
	effort, ok := detectModelReasoningEffort(model)
	if !ok {
		return data, false, nil
	}
	apply(body, effort)
	updated, err := common.Marshal(body)
	if err != nil {
		return nil, false, fmt.Errorf("failed to marshal pass-through request body for reasoning effort: %w", err)
	}
	return updated, true, nil
}

func setJSONRawObjectString(raw []byte, key string, value string) ([]byte, error) {
	obj := map[string]any{}
	if len(raw) > 0 {
		if err := common.Unmarshal(raw, &obj); err != nil {
			return nil, err
		}
	}
	obj[key] = value
	return common.Marshal(obj)
}
