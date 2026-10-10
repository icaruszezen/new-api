package relayconvert

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func strictChatRequest() *dto.GeneralOpenAIRequest {
	strict := true
	return &dto.GeneralOpenAIRequest{
		Model:    "test-model",
		Messages: []dto.Message{{Role: "user", Content: "hello"}},
		Tools: []dto.ToolCallRequest{{
			Type: "function",
			Function: dto.FunctionRequest{
				Name:       "lookup",
				Parameters: map[string]any{"type": "object", "properties": map[string]any{"q": map[string]any{"type": "string"}}},
				Strict:     &strict,
			},
		}},
	}
}

func TestChatAndResponsesConversionsCarryFunctionStrict(t *testing.T) {
	responsesReq, err := ChatCompletionsRequestToResponsesRequest(strictChatRequest())
	require.NoError(t, err)
	assert.Equal(t, "true", gjson.GetBytes(responsesReq.Tools, "0.strict").Raw)

	chatReq, err := ResponsesRequestToChatCompletionsRequest(&dto.OpenAIResponsesRequest{
		Model: "test-model",
		Input: []byte(`"hello"`),
		Tools: []byte(`[{"type":"function","name":"lookup","parameters":{"type":"object"},"strict":false},{"type":"function","name":"plain","parameters":{"type":"object"}}]`),
	})
	require.NoError(t, err)
	require.Len(t, chatReq.Tools, 2)
	require.NotNil(t, chatReq.Tools[0].Function.Strict)
	assert.False(t, *chatReq.Tools[0].Function.Strict)
	assert.Nil(t, chatReq.Tools[1].Function.Strict)
}

// Gemini function declarations have no strict field and the API rejects
// unknown fields, so the OpenAI flag must not leak into a Gemini request.
func TestGeminiConversionsDropFunctionStrict(t *testing.T) {
	fromChat, err := OpenAIChatRequestToGeminiGenerateContent(context.Background(), *strictChatRequest(), nil)
	require.NoError(t, err)
	body, err := kitutil.Marshal(fromChat)
	require.NoError(t, err)
	assert.Equal(t, "lookup", gjson.GetBytes(body, "tools.0.functionDeclarations.0.name").String())
	assert.NotContains(t, string(body), `"strict"`)

	fromResponses, err := OpenAIResponsesRequestToGeminiChat(context.Background(), &dto.OpenAIResponsesRequest{
		Model: "test-model",
		Input: []byte(`"hello"`),
		Tools: []byte(`[{"type":"function","name":"lookup","parameters":{"type":"object","properties":{"q":{"type":"string"}}},"strict":true}]`),
	}, nil)
	require.NoError(t, err)
	body, err = kitutil.Marshal(fromResponses)
	require.NoError(t, err)
	assert.Equal(t, "lookup", gjson.GetBytes(body, "tools.0.functionDeclarations.0.name").String())
	assert.NotContains(t, string(body), `"strict"`)
}
