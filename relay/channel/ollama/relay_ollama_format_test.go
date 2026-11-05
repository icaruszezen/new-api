package ollama

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Ollama fills a structured-output schema in its property order, so the
// format carries the client's schema bytes. A missing or null schema leaves
// the format out of the request.
func TestToOllamaResponseFormatKeepsSchemaBytes(t *testing.T) {
	tests := []struct {
		name       string
		jsonSchema string
		want       string
	}{
		{
			name:       "schema keeps property order",
			jsonSchema: `{"name":"answer","schema":{"type":"object","properties":{"reasoning":{"type":"string"},"answer":{"type":"string"}}}}`,
			want:       `{"type":"object","properties":{"reasoning":{"type":"string"},"answer":{"type":"string"}}}`,
		},
		{
			name:       "null schema sends no format",
			jsonSchema: `{"name":"answer","schema":null}`,
			want:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := toOllamaResponseFormat(&dto.ResponseFormat{Type: "json_schema", JsonSchema: []byte(tt.jsonSchema)})
			require.NoError(t, err)

			encoded, err := common.Marshal(OllamaChatRequest{Model: "m", Format: format})
			require.NoError(t, err)
			assert.Equal(t, tt.want, gjson.GetBytes(encoded, "format").Raw)
		})
	}
}
