package oairesponses

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Claude and Gemini continue a final assistant turn as a prefill, which newer
// models reject; Responses treats it as history, so the converters drop it.
func TestResponsesToClaudeAndGeminiDropTrailingAssistantText(t *testing.T) {
	user := map[string]any{"role": "user", "content": "hi"}
	assistant := map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": "hello"}}}
	call := map[string]any{"type": "function_call", "call_id": "call_1", "name": "lookup", "arguments": "{}"}
	output := map[string]any{"type": "function_call_output", "call_id": "call_1", "output": "42"}

	tests := []struct {
		name       string
		input      []map[string]any
		wantClaude []string
		wantGemini []string
	}{
		{
			name:       "assistant answer after user is dropped",
			input:      []map[string]any{user, assistant},
			wantClaude: []string{"user"},
			wantGemini: []string{"user"},
		},
		{
			name:       "assistant answer after tool result is dropped",
			input:      []map[string]any{user, call, output, assistant},
			wantClaude: []string{"user", "assistant", "user"},
			wantGemini: []string{"user", "model", "user"},
		},
		{
			name:       "pending tool call is kept",
			input:      []map[string]any{user, call},
			wantClaude: []string{"user", "assistant"},
			wantGemini: []string{"user", "model"},
		},
		{
			name:       "assistant-only input is kept",
			input:      []map[string]any{assistant},
			wantGemini: []string{"model"},
			// Claude still prepends its required leading user turn.
			wantClaude: []string{"user", "assistant"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &dto.OpenAIResponsesRequest{
				Model:           "test-model",
				Input:           mustRawMessage(t, tt.input),
				MaxOutputTokens: lo.ToPtr(uint(64)),
			}

			claudeReq, err := OpenAIResponsesRequestToClaudeMessages(context.Background(), &convmeta.Values{}, req)
			require.NoError(t, err)
			claudeRoles := make([]string, 0, len(claudeReq.Messages))
			for _, message := range claudeReq.Messages {
				claudeRoles = append(claudeRoles, message.Role)
			}
			assert.Equal(t, tt.wantClaude, claudeRoles)

			geminiReq, err := OpenAIResponsesRequestToGeminiChat(context.Background(), req, &convmeta.Values{})
			require.NoError(t, err)
			geminiRoles := make([]string, 0, len(geminiReq.Contents))
			for _, content := range geminiReq.Contents {
				geminiRoles = append(geminiRoles, content.Role)
			}
			assert.Equal(t, tt.wantGemini, geminiRoles)
		})
	}
}
