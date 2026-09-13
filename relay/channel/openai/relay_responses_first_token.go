package openai

import (
	"fmt"
	"strings"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
)

const (
	firstTokenErrorEOFMessage       = "stream closed before response.completed"
	firstTokenErrorOutputOneMessage = "output_tokens=1 treated as upstream error"
)

type responsesFirstTokenScan struct {
	actualError      bool
	sawTrueCompleted bool
	errorMessage     string
}

type responsesFirstTokenDecision struct {
	Hit          bool
	ErrorMessage string
	Criterion4   bool
}

func observeResponsesFirstTokenEvent(state *responsesFirstTokenScan, ev *dto.ResponsesStreamResponse) {
	if state == nil || ev == nil {
		return
	}
	if ev.Type == "response.completed" || ev.Type == "response.done" {
		if ev.Response != nil && relaycommon.IsCompletedResponsesStatus(ev.Response.Status) {
			state.sawTrueCompleted = true
		}
	}
	switch ev.Type {
	case "error", "response.error", "response.failed":
		state.actualError = true
	}
	if ev.Response != nil && relaycommon.IsFailedOrIncompleteResponsesStatus(ev.Response.Status) {
		state.actualError = true
	}
	if msg := extractResponsesStreamErrorMessage(ev); msg != "" {
		state.actualError = true
		if state.errorMessage == "" {
			state.errorMessage = msg
		}
	}
}

func extractResponsesStreamErrorMessage(ev *dto.ResponsesStreamResponse) string {
	if ev == nil {
		return ""
	}
	var responseErr *types.OpenAIError
	if ev.Response != nil {
		responseErr = ev.Response.GetOpenAIError()
	}
	if responseErr != nil {
		if msg := strings.TrimSpace(responseErr.Message); msg != "" {
			return msg
		}
	}
	eventErr := dto.GetOpenAIError(ev.Error)
	if eventErr != nil {
		if msg := strings.TrimSpace(eventErr.Message); msg != "" {
			return msg
		}
	}
	top := strings.TrimSpace(ev.Message)
	code := firstNonEmptyErrorCode(ev.Code, eventErr, responseErr)
	if code != "" && top != "" {
		return code + " " + top
	}
	if top != "" {
		return top
	}
	if code != "" && eventErr != nil {
		if msg := strings.TrimSpace(eventErr.Message); msg != "" {
			return code + " " + msg
		}
	}
	return ""
}

func firstNonEmptyErrorCode(eventCode any, eventErr *types.OpenAIError, responseErr *types.OpenAIError) string {
	if s := formatErrorCode(eventCode); s != "" {
		return s
	}
	if eventErr != nil {
		if s := formatErrorCode(eventErr.Code); s != "" {
			return s
		}
	}
	if responseErr != nil {
		if s := formatErrorCode(responseErr.Code); s != "" {
			return s
		}
	}
	return ""
}

func formatErrorCode(code any) string {
	if code == nil {
		return ""
	}
	switch v := code.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" {
			return ""
		}
		return s
	}
}

func decideResponsesFirstTokenError(state responsesFirstTokenScan, completionTokens int, endReason relaycommon.StreamEndReason, treatAllOutputOne bool) responsesFirstTokenDecision {
	if completionTokens != 1 {
		return responsesFirstTokenDecision{}
	}
	criterion4 := !state.sawTrueCompleted && endReason == relaycommon.StreamEndReasonEOF
	actualError := state.actualError || criterion4
	if !treatAllOutputOne && !actualError {
		return responsesFirstTokenDecision{}
	}
	msg := state.errorMessage
	if msg == "" {
		if criterion4 {
			msg = firstTokenErrorEOFMessage
		} else {
			msg = firstTokenErrorOutputOneMessage
		}
	}
	return responsesFirstTokenDecision{
		Hit:          true,
		ErrorMessage: msg,
		Criterion4:   criterion4,
	}
}

func applyResponsesStreamUsage(dst *dto.Usage, src *dto.Usage) {
	if dst == nil || src == nil {
		return
	}
	if src.InputTokens != 0 {
		dst.PromptTokens = src.InputTokens
	}
	if src.OutputTokens != 0 {
		dst.CompletionTokens = src.OutputTokens
	}
	if src.TotalTokens != 0 {
		dst.TotalTokens = src.TotalTokens
	}
	if src.InputTokensDetails != nil {
		dst.PromptTokensDetails.CachedTokens = src.InputTokensDetails.CachedTokens
		dst.PromptTokensDetails.CacheWriteTokens = src.InputTokensDetails.CacheWriteTokens
		dst.InputTokensDetails = src.InputTokensDetails
	}
}

func cloneResponsesUsage(src *dto.Usage) *dto.Usage {
	if src == nil {
		return &dto.Usage{}
	}
	copied := *src
	if src.InputTokensDetails != nil {
		details := *src.InputTokensDetails
		copied.InputTokensDetails = &details
	}
	return &copied
}

func resetResponsesUsage(usage *dto.Usage) {
	if usage == nil {
		return
	}
	*usage = dto.Usage{}
}
