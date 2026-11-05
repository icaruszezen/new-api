package gemini

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
)

func TestCleanFunctionParametersEnumSanitization(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "string enum passes through",
			params: map[string]interface{}{
				"type": "string",
				"enum": []interface{}{"red", "green", "blue"},
			},
			expected: map[string]interface{}{
				"type": "STRING",
				"enum": []interface{}{"red", "green", "blue"},
			},
		},
		{
			name: "boolean enum is dropped and type is kept",
			params: map[string]interface{}{
				"type": "boolean",
				"enum": []interface{}{true},
			},
			expected: map[string]interface{}{
				"type": "BOOLEAN",
			},
		},
		{
			name: "mixed string and boolean enum is dropped",
			params: map[string]interface{}{
				"type": "string",
				"enum": []interface{}{"on", true},
			},
			expected: map[string]interface{}{
				"type": "STRING",
			},
		},
		{
			name: "number enum is dropped",
			params: map[string]interface{}{
				"type": "number",
				"enum": []interface{}{1.0, 2.5},
			},
			expected: map[string]interface{}{
				"type": "NUMBER",
			},
		},
		{
			name: "enum nested in properties is sanitized",
			params: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"flag": map[string]interface{}{
						"type": "boolean",
						"enum": []interface{}{true},
					},
					"color": map[string]interface{}{
						"type": "string",
						"enum": []interface{}{"red"},
					},
				},
			},
			expected: map[string]interface{}{
				"type": "OBJECT",
				"properties": map[string]interface{}{
					"flag": map[string]interface{}{
						"type": "BOOLEAN",
					},
					"color": map[string]interface{}{
						"type": "STRING",
						"enum": []interface{}{"red"},
					},
				},
			},
		},
		{
			name: "enum nested in array items is sanitized",
			params: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "boolean",
					"enum": []interface{}{false},
				},
			},
			expected: map[string]interface{}{
				"type":  "ARRAY",
				"items": map[string]interface{}{"type": "BOOLEAN"},
			},
		},
		{
			name: "enum nested in anyOf is sanitized",
			params: map[string]interface{}{
				"anyOf": []interface{}{
					map[string]interface{}{
						"type": "boolean",
						"enum": []interface{}{true},
					},
					map[string]interface{}{
						"type": "string",
						"enum": []interface{}{"yes", "no"},
					},
				},
			},
			expected: map[string]interface{}{
				"anyOf": []interface{}{
					map[string]interface{}{"type": "BOOLEAN"},
					map[string]interface{}{
						"type": "STRING",
						"enum": []interface{}{"yes", "no"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaned := CleanFunctionParameters(tt.params)
			assert.Equal(t, tt.expected, cleaned)
		})
	}
}

func TestCleanFunctionParametersShallowDropsBooleanEnum(t *testing.T) {
	// Schemas beyond geminiFunctionSchemaMaxDepth fall back to the shallow
	// cleaner, which must apply the same enum sanitization.
	params := map[string]interface{}{
		"type": "boolean",
		"enum": []interface{}{true},
	}
	cleaned := cleanGeminiFunctionParametersShallow(params)
	assert.Equal(t, map[string]interface{}{"type": "BOOLEAN"}, cleaned)

	stringEnum := map[string]interface{}{
		"type": "string",
		"enum": []interface{}{"a", "b"},
	}
	cleaned = cleanGeminiFunctionParametersShallow(stringEnum)
	assert.Equal(t, map[string]interface{}{
		"type": "STRING",
		"enum": []interface{}{"a", "b"},
	}, cleaned)
}

func TestFunctionParametersRoutesSchemaByTypedSchemaFit(t *testing.T) {
	refSchema := func() map[string]any {
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"item": map[string]any{"$ref": "#/$defs/item"},
			},
			"$defs": map[string]any{
				"item": map[string]any{"type": "string"},
			},
		}
	}
	booleanEnumSchema := func() map[string]any {
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"flag": map[string]any{"type": "boolean", "enum": []any{true}},
			},
		}
	}

	tests := []struct {
		name        string
		schema      any
		jsonSchema  bool
		wantField   string
		wantValue   any
		wantMisfits []string
	}{
		{
			name: "string enum stays in typed parameters",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"color": map[string]any{"type": "string", "enum": []any{"red", "green"}},
				},
				"required": []any{"color"},
			},
			jsonSchema: true,
			wantField:  "parameters",
			wantValue: map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"color": map[string]any{"type": "STRING", "enum": []any{"red", "green"}},
				},
				"required": []any{"color"},
			},
		},
		{
			name:       "boolean enum goes unchanged to parametersJsonSchema",
			schema:     booleanEnumSchema(),
			jsonSchema: true,
			wantField:  "parametersJsonSchema",
			wantValue:  booleanEnumSchema(),
		},
		{
			name:       "boolean enum is cleaned when JSON Schema is not accepted",
			schema:     booleanEnumSchema(),
			jsonSchema: false,
			wantField:  "parameters",
			wantValue: map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"flag": map[string]any{"type": "BOOLEAN"},
				},
			},
			wantMisfits: []string{"non-string enum values"},
		},
		{
			name: "additionalProperties and $schema alone keep typed parameters",
			schema: map[string]any{
				"$schema":              "http://json-schema.org/draft-07/schema#",
				"type":                 "object",
				"properties":           map[string]any{"q": map[string]any{"type": "string"}},
				"additionalProperties": false,
			},
			jsonSchema: true,
			wantField:  "parameters",
			wantValue: map[string]any{
				"type":       "OBJECT",
				"properties": map[string]any{"q": map[string]any{"type": "STRING"}},
			},
		},
		{
			name:       "empty properties leave the parameters out",
			schema:     map[string]any{"type": "object", "properties": map[string]any{}},
			jsonSchema: true,
			wantField:  "parameters",
			wantValue:  nil,
		},
		{
			name:       "$ref and $defs go unchanged to parametersJsonSchema",
			schema:     refSchema(),
			jsonSchema: true,
			wantField:  "parametersJsonSchema",
			wantValue:  refSchema(),
		},
		{
			name:       "$ref and $defs are dropped when JSON Schema is not accepted",
			schema:     refSchema(),
			jsonSchema: false,
			wantField:  "parameters",
			wantValue: map[string]any{
				"type":       "OBJECT",
				"properties": map[string]any{"item": map[string]any{}},
			},
			wantMisfits: []string{"$defs", "$ref"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, value, misfits := FunctionParameters(tt.schema, tt.jsonSchema)

			assert.Equal(t, tt.wantField, field)
			assert.Equal(t, tt.wantValue, value)
			if len(tt.wantMisfits) == 0 {
				assert.Empty(t, misfits)
			} else {
				assert.Equal(t, tt.wantMisfits, misfits)
			}
		})
	}
}

func TestApplyResponseSchemaRoutesSchemaByTypedSchemaFit(t *testing.T) {
	// The key order is not alphabetical, so a re-marshal would show.
	nullableSchema := `{"type":"object","properties":{"zeta":{"type":["string","null"]},"alpha":{"type":"string"}},"required":["zeta","alpha"]}`
	nullableFormat := json.RawMessage(`{"name":"answer","schema":` + nullableSchema + `}`)
	strictFormat := json.RawMessage(`{"name":"answer","strict":true,"schema":{"type":"object","properties":{"value":{"type":"string"}},"required":["value"],"additionalProperties":false}}`)

	tests := []struct {
		name               string
		format             json.RawMessage
		jsonSchema         bool
		wantResponseSchema any
		wantJSONSchema     string
	}{
		{
			name:           "type array goes to responseJsonSchema as the client wrote it",
			format:         nullableFormat,
			jsonSchema:     true,
			wantJSONSchema: nullableSchema,
		},
		{
			name:       "type array stays in responseSchema when JSON Schema is not accepted",
			format:     nullableFormat,
			jsonSchema: false,
			wantResponseSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"zeta":  map[string]any{"type": []any{"string", "null"}},
					"alpha": map[string]any{"type": "string"},
				},
				"required": []any{"zeta", "alpha"},
			},
		},
		{
			name:       "strict schema keeps the responseSchema cleanup",
			format:     strictFormat,
			jsonSchema: true,
			wantResponseSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"value": map[string]any{"type": "string"}},
				"required":   []any{"value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config dto.GeminiChatGenerationConfig
			ApplyResponseSchema(&config, tt.format, "response_format.json_schema.schema", tt.jsonSchema)

			assert.Equal(t, tt.wantResponseSchema, config.ResponseSchema)
			assert.Equal(t, tt.wantJSONSchema, string(config.ResponseJsonSchema))
		})
	}
}
