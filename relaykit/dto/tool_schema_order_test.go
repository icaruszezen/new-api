package dto

import (
	"testing"

	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Strict tools fill a schema's properties in the order the schema lists them,
// so a request decoded and encoded again must keep the client's schema bytes.
func TestRequestRoundTripKeepsToolSchemaOrder(t *testing.T) {
	const schema = `{"type":"object","properties":{"reasoning":{"type":"string"},"answer":{"type":"string"}},"required":["reasoning","answer"]}`
	tests := []struct {
		name       string
		raw        string
		target     any
		schemaPath string
	}{
		{
			name:       "openai chat parameters",
			raw:        `{"model":"m","tools":[{"type":"function","function":{"name":"answer","parameters":` + schema + `}}]}`,
			target:     &GeneralOpenAIRequest{},
			schemaPath: "tools.0.function.parameters",
		},
		{
			name:       "claude input_schema",
			raw:        `{"model":"m","max_tokens":8,"tools":[{"name":"answer","input_schema":` + schema + `}]}`,
			target:     &ClaudeRequest{},
			schemaPath: "tools.0.input_schema",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, kitutil.Unmarshal([]byte(tt.raw), tt.target))

			encoded, err := kitutil.Marshal(tt.target)
			require.NoError(t, err)
			assert.Equal(t, schema, gjson.GetBytes(encoded, tt.schemaPath).Raw)
		})
	}
}

// A null parameters or tools value stays absent, as the decoded value did.
func TestRequestNullToolSchemaStaysAbsent(t *testing.T) {
	var chat GeneralOpenAIRequest
	require.NoError(t, kitutil.Unmarshal([]byte(`{"model":"m","tools":[{"type":"function","function":{"name":"f","parameters":null}}]}`), &chat))
	require.Len(t, chat.Tools, 1)
	assert.Nil(t, chat.Tools[0].Function.Parameters)
	encoded, err := kitutil.Marshal(chat)
	require.NoError(t, err)
	assert.False(t, gjson.GetBytes(encoded, "tools.0.function.parameters").Exists())

	var claude ClaudeRequest
	require.NoError(t, kitutil.Unmarshal([]byte(`{"model":"m","tools":null}`), &claude))
	assert.Nil(t, claude.Tools)
	assert.Nil(t, claude.GetTools())
}

// The tool token estimate, and with it the pre-consumed quota, reads the
// schema text, not the byte values of the raw message.
func TestGeneralOpenAIRequestTokenCountUsesSchemaText(t *testing.T) {
	var req GeneralOpenAIRequest
	require.NoError(t, kitutil.Unmarshal([]byte(`{"model":"m","tools":[{"type":"function","function":{"name":"f","parameters":{"type":"object"}}}]}`), &req))

	meta := req.GetTokenCountMeta()

	assert.Equal(t, "f\n"+`{"type":"object"}`, meta.CombineText)
}
