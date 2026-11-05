package dto

import (
	"testing"

	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// The Responses schema requires summary on every reasoning item; other item
// types have no summary field.
func TestResponsesOutputMarshalsReasoningSummary(t *testing.T) {
	tests := []struct {
		name        string
		output      ResponsesOutput
		wantSummary string
	}{
		{
			name:        "reasoning item announced without summary",
			output:      ResponsesOutput{Type: "reasoning", ID: "rs_1", Status: "in_progress"},
			wantSummary: `[]`,
		},
		{
			name: "reasoning item with summary text",
			output: ResponsesOutput{Type: "reasoning", ID: "rs_1", Summary: []ResponsesReasoningSummaryPart{
				{Type: "summary_text", Text: "think"},
			}},
			wantSummary: `[{"type":"summary_text","text":"think"}]`,
		},
		{
			name:   "message item",
			output: ResponsesOutput{Type: "message", ID: "msg_1", Role: "assistant"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := kitutil.Marshal(tt.output)
			require.NoError(t, err)

			summary := gjson.GetBytes(body, "summary")
			if tt.wantSummary == "" {
				assert.False(t, summary.Exists())
			} else {
				assert.Equal(t, tt.wantSummary, summary.Raw)
			}
			assert.Equal(t, tt.output.Type, gjson.GetBytes(body, "type").String())
			assert.Equal(t, tt.output.ID, gjson.GetBytes(body, "id").String())
		})
	}
}
