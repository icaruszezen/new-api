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

// Responses converted from another protocol carry the reasoning text in the
// item content instead of the summary.
func TestExtractOutputFromConvertedResponsesFallsBackToReasoningContent(t *testing.T) {
	body := strings.Join([]string{
		`event: response.completed`,
		`data: {"type":"response.completed","response":{"status":"completed","output":[{"type":"reasoning","content":[{"type":"summary_text","text":"Converted plan"}]},{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}}`,
		``,
	}, "\n")

	output := ExtractOutput([]byte(body))

	assert.Equal(t, "ok", output.Content)
	assert.Equal(t, "Converted plan", output.Reasoning)
	assert.Equal(t, "stop", output.FinishReason)
}

func TestExtractOutputFromClaudeMessagesStream(t *testing.T) {
	body := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"usage":{"input_tokens":9,"output_tokens":1}}}`,
		``,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"Count the r's. "}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"Three."}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"sig"}}`,
		``,
		`event: content_block_stop`,
		`data: {"type":"content_block_stop","index":0}`,
		``,
		`event: ping`,
		`data: {"type":"ping"}`,
		``,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"There are "}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"3."}}`,
		``,
		`event: content_block_stop`,
		`data: {"type":"content_block_stop","index":1}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":20}}`,
		``,
		`event: message_stop`,
		`data: {"type":"message_stop"}`,
		``,
	}, "\n")

	output := ExtractOutput([]byte(body))

	assert.Equal(t, "There are 3.", output.Content)
	assert.Equal(t, "Count the r's. Three.", output.Reasoning)
	assert.Equal(t, "stop", output.FinishReason)
	assert.Empty(t, output.ErrorMessage)
}

// A Claude stream cut before message_delta has no stop reason, which the check
// treats as an incomplete answer.
func TestExtractOutputFromClaudeStreamWithoutStopReason(t *testing.T) {
	body := strings.Join([]string{
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"partial"}}`,
		``,
	}, "\n")

	output := ExtractOutput([]byte(body))

	assert.Equal(t, "partial", output.Content)
	assert.Empty(t, output.FinishReason)
}

func TestExtractOutputFromGeminiStream(t *testing.T) {
	cases := []struct {
		name       string
		lastReason string
		wantFinish string
	}{
		{name: "stop", lastReason: "STOP", wantFinish: "stop"},
		{name: "max tokens", lastReason: "MAX_TOKENS", wantFinish: "length"},
		{name: "safety", lastReason: "SAFETY", wantFinish: "content_filter"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			body := strings.Join([]string{
				`data: {"candidates":[{"content":{"role":"model","parts":[{"text":"Weighing options.","thought":true}]},"index":0}],"usageMetadata":{"promptTokenCount":5}}`,
				``,
				`data: {"candidates":[{"content":{"role":"model","parts":[{"text":"<html>"}]},"finishReason":null,"index":0}],"usageMetadata":{"promptTokenCount":5}}`,
				``,
				`data: {"candidates":[{"content":{"role":"model","parts":[{"text":"</html>"}]},"finishReason":"` + testCase.lastReason + `","index":0}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":4,"thoughtsTokenCount":3}}`,
				``,
			}, "\n")

			output := ExtractOutput([]byte(body))

			assert.Equal(t, "<html></html>", output.Content)
			assert.Equal(t, "Weighing options.", output.Reasoning)
			assert.Equal(t, testCase.wantFinish, output.FinishReason)
		})
	}
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
		{
			name:          "claude message",
			body:          `{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"thinking","thinking":"Compare sizes.","signature":"sig"},{"type":"text","text":"Jupiter"}],"stop_reason":"max_tokens"}`,
			wantContent:   "Jupiter",
			wantReasoning: "Compare sizes.",
			wantFinish:    "length",
		},
		{
			name:          "gemini generate content",
			body:          `{"candidates":[{"content":{"role":"model","parts":[{"text":"Plan.","thought":true},{"text":"Done"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":3}}`,
			wantContent:   "Done",
			wantReasoning: "Plan.",
			wantFinish:    "stop",
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
			name: "claude error event",
			body: strings.Join([]string{
				`event: content_block_delta`,
				`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"partial"}}`,
				`event: error`,
				`data: {"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`,
			}, "\n"),
			wantContent: "partial",
			wantError:   "Overloaded",
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

// Some reasoning models write their thinking inline as <think>…</think> at the
// start of the answer instead of in reasoning_content.
func TestExtractOutputMovesLeadingThinkBlockToReasoning(t *testing.T) {
	testCases := []struct {
		name          string
		body          string
		wantContent   string
		wantReasoning string
	}{
		{
			name: "streamed think block becomes the reasoning",
			body: strings.Join([]string{
				`data: {"choices":[{"delta":{"content":"<think>\nThe page needs"}}]}`,
				`data: {"choices":[{"delta":{"content":" a <html> root.\n</think>\n\n<!DOCTYPE html><html></html>"}}]}`,
				`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			}, "\n"),
			wantContent:   "<!DOCTYPE html><html></html>",
			wantReasoning: "The page needs a <html> root.",
		},
		{
			name:          "separate reasoning wins over the inline copy",
			body:          `{"choices":[{"message":{"content":"<think>copy</think>42","reasoning_content":"original"},"finish_reason":"stop"}]}`,
			wantContent:   "42",
			wantReasoning: "original",
		},
		{
			name:          "unclosed think block means the answer never started",
			body:          `{"choices":[{"message":{"content":"<think>still thinking"},"finish_reason":"length"}]}`,
			wantContent:   "",
			wantReasoning: "still thinking",
		},
		{
			name:          "think tags later in the answer are content",
			body:          `{"choices":[{"message":{"content":"Use <think>tags</think> like this."},"finish_reason":"stop"}]}`,
			wantContent:   "Use <think>tags</think> like this.",
			wantReasoning: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			output := ExtractOutput([]byte(testCase.body))
			assert.Equal(t, testCase.wantContent, output.Content)
			assert.Equal(t, testCase.wantReasoning, output.Reasoning)
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

// PostgreSQL rejects NUL bytes and invalid UTF-8 in text columns, so a single
// stray byte in a model answer would lose the whole result.
func TestExtractOutputKeepsOnlyStorableText(t *testing.T) {
	body := "data: {\"choices\":[{\"delta\":{\"content\":\"a\\u0000b\xffc\",\"reasoning_content\":\"r\\u0000\"}}]}\n"

	output := ExtractOutput([]byte(body))

	assert.Equal(t, "ab�c", output.Content)
	assert.Equal(t, "r", output.Reasoning)
	assert.Equal(t, "boom", TruncateErrorMessage("bo\x00om"))
}
