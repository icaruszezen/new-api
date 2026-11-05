package relayconvert

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Schemas that Gemini's typed Schema cannot hold (const, type arrays) go to
// parametersJsonSchema and responseJsonSchema when the upstream model accepts
// them, and keep the typed fields otherwise.
func TestOpenAIToGeminiSendsUnfitSchemasAsJSONSchema(t *testing.T) {
	const toolParameters = `{"type":"object","properties":{"mode":{"type":"string","const":"fast"}},"required":["mode"]}`
	const outputSchema = `{"type":"object","properties":{"value":{"type":["string","null"]}},"required":["value"]}`

	converters := []struct {
		name    string
		convert func(t *testing.T, meta convmeta.Meta) []byte
	}{
		{
			name: "chat completions",
			convert: func(t *testing.T, meta convmeta.Meta) []byte {
				t.Helper()
				var parameters any
				require.NoError(t, kitutil.Unmarshal([]byte(toolParameters), &parameters))
				got, err := OpenAIChatRequestToGeminiGenerateContent(context.Background(), dto.GeneralOpenAIRequest{
					Model:    "gemini-alias",
					Messages: []dto.Message{{Role: "user", Content: "hello"}},
					Tools: []dto.ToolCallRequest{{
						Type:     "function",
						Function: dto.FunctionRequest{Name: "pick", Parameters: parameters},
					}},
					ResponseFormat: &dto.ResponseFormat{
						Type:       "json_schema",
						JsonSchema: []byte(`{"name":"answer","schema":` + outputSchema + `}`),
					},
				}, meta)
				require.NoError(t, err)
				body, err := kitutil.Marshal(got)
				require.NoError(t, err)
				return body
			},
		},
		{
			name: "responses",
			convert: func(t *testing.T, meta convmeta.Meta) []byte {
				t.Helper()
				got, err := OpenAIResponsesRequestToGeminiChat(context.Background(), &dto.OpenAIResponsesRequest{
					Model: "gemini-alias",
					Input: []byte(`"hello"`),
					Tools: []byte(`[{"type":"function","name":"pick","parameters":` + toolParameters + `}]`),
					Text:  []byte(`{"format":{"type":"json_schema","name":"answer","schema":` + outputSchema + `}}`),
				}, meta)
				require.NoError(t, err)
				body, err := kitutil.Marshal(got)
				require.NoError(t, err)
				return body
			},
		},
	}

	options := &convmeta.Options{Gemini: convmeta.GeminiOptions{
		SupportsJSONSchema: func(modelName string) bool { return modelName == "gemini-v1beta-model" },
	}}

	for _, converter := range converters {
		t.Run(converter.name, func(t *testing.T) {
			t.Run("upstream accepts JSON Schema", func(t *testing.T) {
				body := converter.convert(t, &convmeta.Values{
					UpstreamModelName:   "gemini-v1beta-model",
					ChannelMetaAttached: true,
					Options:             options,
				})

				declaration := gjson.GetBytes(body, "tools.0.functionDeclarations.0")
				assert.Equal(t, "pick", declaration.Get("name").String())
				assert.Equal(t, "fast", declaration.Get("parametersJsonSchema.properties.mode.const").String())
				assert.False(t, declaration.Get("parameters").Exists())

				config := gjson.GetBytes(body, "generationConfig")
				assert.Equal(t, `["string","null"]`, config.Get("responseJsonSchema.properties.value.type").Raw)
				assert.False(t, config.Get("responseSchema").Exists())
			})

			t.Run("upstream takes only the typed Schema", func(t *testing.T) {
				body := converter.convert(t, &convmeta.Values{
					UpstreamModelName:   "gemini-v1-model",
					ChannelMetaAttached: true,
					Options:             options,
				})

				declaration := gjson.GetBytes(body, "tools.0.functionDeclarations.0")
				assert.Equal(t, "STRING", declaration.Get("parameters.properties.mode.type").String())
				assert.False(t, declaration.Get("parameters.properties.mode.const").Exists())
				assert.False(t, declaration.Get("parametersJsonSchema").Exists())

				config := gjson.GetBytes(body, "generationConfig")
				assert.True(t, config.Get("responseSchema").Exists())
				assert.False(t, config.Get("responseJsonSchema").Exists())
			})
		})
	}
}
