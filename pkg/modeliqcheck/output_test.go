package modeliqcheck

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractOutputFromChatCompletionStream(t *testing.T) {
	body := strings.Join([]string{
		`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":""}}]}`,
		``,
		`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"reasoning_content":"Let me think. "}}]}`,
		``,
		`: PING`,
		``,
		`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"reasoning_content":"Done."}}]}`,
		``,
		`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Here is the page:\n"}}]}`,
		``,
		`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"<html></html>"},"finish_reason":null}]}`,
		``,
		`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		``,
		`data: {"id":"1","object":"chat.completion.chunk","choices":[],"usage":{"prompt_tokens":12,"completion_tokens":34,"total_tokens":46}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	output := ExtractOutput([]byte(body))

	assert.Equal(t, "Here is the page:\n<html></html>", output.Content)
	assert.Equal(t, "Let me think. Done.", output.Reasoning)
	assert.Equal(t, "stop", output.FinishReason, "the trailing usage chunk must not erase the finish reason")
	assert.Empty(t, output.ErrorMessage)
	assert.False(t, output.ContentTruncated)
	assert.False(t, output.ReasoningTruncated)
}

func TestExtractOutputReadsReasoningFieldWithoutDuplicatingReasoningContent(t *testing.T) {
	body := strings.Join([]string{
		`data: {"choices":[{"delta":{"reasoning":"thinking via reasoning"}}]}`,
		`data: {"choices":[{"delta":{"reasoning_content":" and both","reasoning":" and both"}}]}`,
		`data: {"choices":[{"delta":{"content":"42"},"finish_reason":"length"}]}`,
		`data: [DONE]`,
	}, "\n")

	output := ExtractOutput([]byte(body))

	assert.Equal(t, "42", output.Content)
	assert.Equal(t, "thinking via reasoning and both", output.Reasoning)
	assert.Equal(t, "length", output.FinishReason)
}

func TestExtractOutputFromResponsesStream(t *testing.T) {
	body := strings.Join([]string{
		`event: response.created`,
		`data: {"type":"response.created","response":{"id":"resp_1","status":"in_progress","output":[]}}`,
		``,
		`event: response.reasoning_summary_text.delta`,
		`data: {"type":"response.reasoning_summary_text.delta","delta":"Plan the answer."}`,
		``,
		`event: response.output_text.delta`,
		`data: {"type":"response.output_text.delta","delta":"1 + 1 "}`,
		``,
		`event: response.output_text.delta`,
		`data: {"type":"response.output_text.delta","delta":"= 2"}`,
		``,
		`event: response.completed`,
		`data: {"type":"response.completed","response":{"id":"resp_1","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"1 + 1 = 2"}]}]}}`,
		``,
	}, "\n")

	output := ExtractOutput([]byte(body))

	assert.Equal(t, "1 + 1 = 2", output.Content, "deltas must not be duplicated by the final response output")
	assert.Equal(t, "Plan the answer.", output.Reasoning)
	assert.Equal(t, "stop", output.FinishReason)
	assert.Empty(t, output.ErrorMessage)
}

func TestExtractOutputFromIncompleteResponsesStreamFallsBackToFinalOutput(t *testing.T) {
	body := strings.Join([]string{
		`event: response.incomplete`,
		`data: {"type":"response.incomplete","response":{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[{"type":"reasoning","summary":[{"type":"summary_text","text":"Partial plan"}]},{"type":"message","content":[{"type":"output_text","text":"<!DOCTYPE html><html><body>"}]}]}}`,
		``,
	}, "\n")

	output := ExtractOutput([]byte(body))

	assert.Equal(t, "<!DOCTYPE html><html><body>", output.Content)
	assert.Equal(t, "Partial plan", output.Reasoning)
	assert.Equal(t, "length", output.FinishReason)
}

func TestExtractOutputFromNonStreamBodies(t *testing.T) {
	cases := []struct {
		name          string
		body          string
		wantContent   string
		wantReasoning string
		wantFinish    string
	}{
		{
			name:          "chat completion",
			body:          `{"id":"1","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"Paris","reasoning_content":"Capital of France."},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":1}}`,
			wantContent:   "Paris",
			wantReasoning: "Capital of France.",
			wantFinish:    "stop",
		},
		{
			name:        "chat completion with content parts",
			body:        `{"choices":[{"message":{"role":"assistant","content":[{"type":"text","text":"Hello, "},{"type":"text","text":"world"}]},"finish_reason":"stop"}]}`,
			wantContent: "Hello, world",
			wantFinish:  "stop",
		},
		{
			name:        "responses",
			body:        `{"id":"resp_1","object":"response","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"Hi "},{"type":"output_text","text":"there"}]}]}`,
			wantContent: "Hi there",
			wantFinish:  "stop",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			output := ExtractOutput([]byte(testCase.body))
			assert.Equal(t, testCase.wantContent, output.Content)
			assert.Equal(t, testCase.wantReasoning, output.Reasoning)
			assert.Equal(t, testCase.wantFinish, output.FinishReason)
			assert.Empty(t, output.ErrorMessage)
		})
	}
}

func TestExtractOutputReportsUpstreamErrors(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		wantContent string
		wantError   string
	}{
		{
			name: "error chunk after partial stream",
			body: strings.Join([]string{
				`data: {"choices":[{"delta":{"content":"partial"}}]}`,
				`data: {"error":{"message":"upstream overloaded","type":"server_error"}}`,
				`data: [DONE]`,
			}, "\n"),
			wantContent: "partial",
			wantError:   "upstream overloaded",
		},
		{
			name:      "non-stream error object",
			body:      `{"error":{"message":"model not found","code":"model_not_found"}}`,
			wantError: "model not found",
		},
		{
			name:      "string error",
			body:      `{"error":"quota exceeded"}`,
			wantError: "quota exceeded",
		},
		{
			name: "responses failed event",
			body: strings.Join([]string{
				`event: response.failed`,
				`data: {"type":"response.failed","response":{"status":"failed","error":{"code":"server_error","message":"generation failed"}}}`,
			}, "\n"),
			wantError: "generation failed",
		},
		{
			name: "null error is not a failure",
			body: strings.Join([]string{
				`data: {"choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}],"error":null}`,
			}, "\n"),
			wantContent: "ok",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			output := ExtractOutput([]byte(testCase.body))
			assert.Equal(t, testCase.wantContent, output.Content)
			assert.Equal(t, testCase.wantError, output.ErrorMessage)
		})
	}
}

func TestExtractOutputIgnoresEmptyAndNonJSONBodies(t *testing.T) {
	for _, body := range []string{"", "   ", ": PING\n\n", "data: [DONE]\n", "data: not-json\n", "<html>502 Bad Gateway</html>"} {
		output := ExtractOutput([]byte(body))
		assert.Equal(t, Output{}, output, "body=%q", body)
	}
}

func TestExtractOutputTruncatesOnUTF8Boundary(t *testing.T) {
	// "界" is three bytes, so the byte cap lands inside a character and the cut
	// must back off to the previous character boundary.
	longContent := strings.Repeat("a", MaxResultContentBytes-1) + "界界"
	longReasoning := strings.Repeat("b", MaxResultReasoningBytes-2) + "界"

	body := `data: {"choices":[{"delta":{"content":"` + longContent + `","reasoning_content":"` + longReasoning + `"}}]}`
	output := ExtractOutput([]byte(body))

	require.True(t, output.ContentTruncated)
	assert.Equal(t, strings.Repeat("a", MaxResultContentBytes-1), output.Content)
	assert.True(t, utf8.ValidString(output.Content))

	require.True(t, output.ReasoningTruncated)
	assert.Equal(t, strings.Repeat("b", MaxResultReasoningBytes-2), output.Reasoning)
	assert.True(t, utf8.ValidString(output.Reasoning))

	exact := ExtractOutput([]byte(`{"choices":[{"message":{"content":"` + strings.Repeat("c", MaxResultContentBytes) + `"}}]}`))
	assert.False(t, exact.ContentTruncated, "content exactly at the cap is kept whole")
	assert.Len(t, exact.Content, MaxResultContentBytes)
}

func TestTruncateErrorMessageCountsCharacters(t *testing.T) {
	assert.Equal(t, "boom", TruncateErrorMessage("  boom \n"))

	long := strings.Repeat("错", MaxErrorMessageLength+10)
	truncated := TruncateErrorMessage(long)
	assert.Equal(t, MaxErrorMessageLength, utf8.RuneCountInString(truncated))
	assert.True(t, utf8.ValidString(truncated))
}
