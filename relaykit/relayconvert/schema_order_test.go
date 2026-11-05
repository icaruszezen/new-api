package relayconvert

import (
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Structured outputs fill a schema's properties in the order the schema lists
// them, so converting between Chat Completions and Responses must carry the
// client's schema bytes rather than a re-encoded, key-sorted copy.
const orderedSchema = `{"type":"object","properties":{"reasoning":{"type":"string"},"answer":{"type":"string"}},"required":["reasoning","answer"]}`

func TestChatToResponsesKeepsSchemaPropertyOrder(t *testing.T) {
	req, err := ChatCompletionsRequestToResponsesRequest(&dto.GeneralOpenAIRequest{
		Model:    "m",
		Messages: []dto.Message{{Role: "user", Content: "hi"}},
		ResponseFormat: &dto.ResponseFormat{
			Type:       "json_schema",
			JsonSchema: []byte(`{"name":"answer","strict":true,"schema":` + orderedSchema + `}`),
		},
	})

	require.NoError(t, err)
	assert.Equal(t, orderedSchema, gjson.GetBytes(req.Text, "format.schema").Raw)
	assert.Equal(t, "answer", gjson.GetBytes(req.Text, "format.name").String())
}

func TestResponsesToChatKeepsSchemaPropertyOrder(t *testing.T) {
	req, err := ResponsesRequestToChatCompletionsRequest(&dto.OpenAIResponsesRequest{
		Model: "m",
		Input: []byte(`"hi"`),
		Text:  []byte(`{"format":{"type":"json_schema","name":"answer","schema":` + orderedSchema + `}}`),
		Tools: []byte(`[{"type":"function","name":"lookup","parameters":` + orderedSchema + `}]`),
	})

	require.NoError(t, err)
	require.NotNil(t, req.ResponseFormat)
	assert.Equal(t, orderedSchema, gjson.GetBytes(req.ResponseFormat.JsonSchema, "schema").Raw)
	require.Len(t, req.Tools, 1)
	assert.Equal(t, orderedSchema, string(req.Tools[0].Function.Parameters))
}
