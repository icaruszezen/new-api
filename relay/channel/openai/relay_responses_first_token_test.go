package openai

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractResponsesStreamErrorMessagePriority(t *testing.T) {
	t.Parallel()

	msg := extractResponsesStreamErrorMessage(&dto.ResponsesStreamResponse{
		Type:    "response.failed",
		Message: "top-level",
		Code:    "E1",
		Error: map[string]any{
			"message": "event-error",
		},
		Response: &dto.OpenAIResponsesResponse{
			Error: map[string]any{
				"message": "response-error",
			},
		},
	})
	assert.Equal(t, "response-error", msg)

	msg = extractResponsesStreamErrorMessage(&dto.ResponsesStreamResponse{
		Type:    "error",
		Message: "top-level",
		Code:    "E1",
		Error: map[string]any{
			"message": "event-error",
		},
	})
	assert.Equal(t, "event-error", msg)

	msg = extractResponsesStreamErrorMessage(&dto.ResponsesStreamResponse{
		Type:    "error",
		Message: "boom",
		Code:    "rate_limit",
	})
	assert.Equal(t, "rate_limit boom", msg)

	msg = extractResponsesStreamErrorMessage(&dto.ResponsesStreamResponse{
		Type:    "error",
		Message: "only-top",
	})
	assert.Equal(t, "only-top", msg)
}

func TestDecideResponsesFirstTokenError(t *testing.T) {
	t.Parallel()

	failed := responsesFirstTokenScan{actualError: true, errorMessage: "upstream failed"}
	hit := decideResponsesFirstTokenError(failed, 1, relaycommon.StreamEndReasonEOF, false)
	require.True(t, hit.Hit)
	assert.Equal(t, "upstream failed", hit.ErrorMessage)

	success := responsesFirstTokenScan{sawTrueCompleted: true}
	miss := decideResponsesFirstTokenError(success, 1, relaycommon.StreamEndReasonDone, false)
	assert.False(t, miss.Hit)

	relaxed := decideResponsesFirstTokenError(success, 1, relaycommon.StreamEndReasonDone, true)
	require.True(t, relaxed.Hit)
	assert.Equal(t, firstTokenErrorOutputOneMessage, relaxed.ErrorMessage)

	eof := decideResponsesFirstTokenError(responsesFirstTokenScan{}, 1, relaycommon.StreamEndReasonEOF, false)
	require.True(t, eof.Hit)
	assert.True(t, eof.Criterion4)
	assert.Equal(t, firstTokenErrorEOFMessage, eof.ErrorMessage)

	normal := decideResponsesFirstTokenError(responsesFirstTokenScan{sawTrueCompleted: true}, 8, relaycommon.StreamEndReasonDone, true)
	assert.False(t, normal.Hit)

	zero := decideResponsesFirstTokenError(failed, 0, relaycommon.StreamEndReasonEOF, false)
	assert.False(t, zero.Hit)
}

func TestObserveResponsesFirstTokenEvent(t *testing.T) {
	t.Parallel()
	var state responsesFirstTokenScan
	observeResponsesFirstTokenEvent(&state, &dto.ResponsesStreamResponse{
		Type: "response.completed",
		Response: &dto.OpenAIResponsesResponse{
			Status: []byte(`"completed"`),
		},
	})
	assert.True(t, state.sawTrueCompleted)
	assert.False(t, state.actualError)

	observeResponsesFirstTokenEvent(&state, &dto.ResponsesStreamResponse{
		Type: "response.failed",
		Response: &dto.OpenAIResponsesResponse{
			Status: []byte(`"failed"`),
			Error:  map[string]any{"message": "cpa failed"},
		},
	})
	assert.True(t, state.actualError)
	assert.Equal(t, "cpa failed", state.errorMessage)
}
