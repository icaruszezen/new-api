package modeliqcheck

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/tidwall/gjson"
)

// ExtractOutput 从中转写给合成客户端的响应中提取模型输出。
// 检查走渠道测试链路，客户端格式为 OpenAI Chat Completions 或 Responses：
// 流式响应是 SSE（只解析 data: 行，忽略 event: 行、": PING" 保活注释与 [DONE]），
// 非流式响应是单个 JSON 对象。正文与思考内容按字节上限在 UTF-8 边界截断。
func ExtractOutput(body []byte) Output {
	collector := outputCollector{}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '{' && gjson.ValidBytes(trimmed) {
		collector.addCompletion(gjson.ParseBytes(trimmed))
	} else {
		for _, line := range bytes.Split(trimmed, []byte{'\n'}) {
			line = bytes.TrimSpace(line)
			if !bytes.HasPrefix(line, []byte("data:")) {
				continue
			}
			payload := bytes.TrimSpace(line[len("data:"):])
			if len(payload) == 0 || payload[0] != '{' || !gjson.ValidBytes(payload) {
				continue
			}
			collector.addStreamPayload(gjson.ParseBytes(payload))
		}
	}

	output := Output{
		FinishReason: collector.finishReason,
		ErrorMessage: collector.errorMessage,
	}
	content, reasoning := separateInlineThinking(collector.content.String(), collector.reasoning.String())
	output.Content, output.ContentTruncated = truncateUTF8(storableText(content), MaxResultContentBytes)
	output.Reasoning, output.ReasoningTruncated = truncateUTF8(storableText(reasoning), MaxResultReasoningBytes)
	return output
}

// storableText 去掉 NUL 并替换非法 UTF-8。PostgreSQL 的 text 列拒绝这两类内容，
// 而模型输出里 JSON 的 \u0000 会被解码成 NUL，一旦出现整条结果都会写入失败。
func storableText(value string) string {
	return strings.ReplaceAll(strings.ToValidUTF8(value, "\uFFFD"), "\x00", "")
}

// separateInlineThinking 把正文开头的 <think>…</think> 思考块移出正文。
// 部分推理模型（及其兼容中转）不用 reasoning_content 字段，而是把思考过程写在正文里；
// 不移出会让页面把思考当成回答展示，思考中出现的 HTML 片段也会被当成网页渲染。
// 已有独立思考内容时只去掉正文中的重复部分；思考块未闭合说明模型在思考中途被截断，正文为空。
func separateInlineThinking(content string, reasoning string) (string, string) {
	const openTag, closeTag = "<think>", "</think>"
	body := strings.TrimLeft(content, " \t\r\n")
	if !strings.HasPrefix(body, openTag) {
		return content, reasoning
	}
	thinking, answer, closed := strings.Cut(body[len(openTag):], closeTag)
	if !closed {
		answer = ""
	}
	if reasoning == "" {
		reasoning = strings.TrimSpace(thinking)
	}
	return strings.TrimLeft(answer, " \t\r\n"), reasoning
}

// TruncateErrorMessage 把错误信息裁剪到 MaxErrorMessageLength 个字符，
// 避免上游返回的整页错误 HTML 撑大结果行。
func TruncateErrorMessage(message string) string {
	message = strings.TrimSpace(storableText(message))
	if utf8.RuneCountInString(message) <= MaxErrorMessageLength {
		return message
	}
	return string([]rune(message)[:MaxErrorMessageLength])
}

// outputCollector 累积 Chat Completions 增量与 Responses 事件中的文本。
type outputCollector struct {
	content      strings.Builder
	reasoning    strings.Builder
	finishReason string
	errorMessage string
}

func (o *outputCollector) addStreamPayload(payload gjson.Result) {
	if message := upstreamErrorMessage(payload); message != "" {
		o.errorMessage = message
		return
	}
	if payload.Get("choices").Exists() {
		delta := payload.Get("choices.0.delta")
		o.content.WriteString(delta.Get("content").String())
		o.appendReasoning(delta)
		if reason := payload.Get("choices.0.finish_reason").String(); reason != "" {
			o.finishReason = reason
		}
		return
	}

	eventType := payload.Get("type").String()
	switch eventType {
	case "response.output_text.delta":
		o.content.WriteString(payload.Get("delta").String())
	case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
		o.reasoning.WriteString(payload.Get("delta").String())
	case "response.completed", "response.incomplete":
		o.finishResponse(payload.Get("response"), strings.TrimPrefix(eventType, "response."))
	case "response.failed":
		if message := payload.Get("response.error.message").String(); message != "" {
			o.errorMessage = message
		}
	case "error":
		if message := payload.Get("message").String(); message != "" {
			o.errorMessage = message
		}
	}
}

// addCompletion 处理非流式响应：Chat Completions 的 message，或 Responses 的最终 output。
func (o *outputCollector) addCompletion(body gjson.Result) {
	if message := upstreamErrorMessage(body); message != "" {
		o.errorMessage = message
		return
	}
	if body.Get("choices").Exists() {
		message := body.Get("choices.0.message")
		content := message.Get("content")
		if content.IsArray() {
			for _, part := range content.Array() {
				o.content.WriteString(part.Get("text").String())
			}
		} else {
			o.content.WriteString(content.String())
		}
		o.appendReasoning(message)
		o.finishReason = body.Get("choices.0.finish_reason").String()
		return
	}
	if body.Get("output").Exists() {
		o.finishResponse(body, body.Get("status").String())
	}
}

// appendReasoning 读取思考内容。上游可能用 reasoning_content 或 reasoning 字段，
// 部分中转两者同时携带同一份内容，因此只取其一。
func (o *outputCollector) appendReasoning(source gjson.Result) {
	if text := source.Get("reasoning_content").String(); text != "" {
		o.reasoning.WriteString(text)
		return
	}
	if reasoning := source.Get("reasoning"); reasoning.Type == gjson.String {
		o.reasoning.WriteString(reasoning.String())
	}
}

// finishResponse 处理 Responses 终态：没有收到增量文本时回退到最终 output，
// 并把结束原因换算成 Chat Completions 口径（max_output_tokens → length）。
func (o *outputCollector) finishResponse(response gjson.Result, status string) {
	contentEmpty := o.content.Len() == 0
	reasoningEmpty := o.reasoning.Len() == 0
	for _, item := range response.Get("output").Array() {
		switch item.Get("type").String() {
		case "message":
			if !contentEmpty {
				continue
			}
			for _, part := range item.Get("content").Array() {
				if part.Get("type").String() == "output_text" {
					o.content.WriteString(part.Get("text").String())
				}
			}
		case "reasoning":
			if !reasoningEmpty {
				continue
			}
			for _, summary := range item.Get("summary").Array() {
				o.reasoning.WriteString(summary.Get("text").String())
			}
		}
	}

	switch status {
	case "completed":
		o.finishReason = "stop"
	case "incomplete":
		reason := response.Get("incomplete_details.reason").String()
		switch reason {
		case "max_output_tokens":
			o.finishReason = "length"
		case "":
			o.finishReason = "incomplete"
		default:
			o.finishReason = reason
		}
	}
}

// upstreamErrorMessage 读取载荷顶层的 error 对象，兼容字符串形式。
func upstreamErrorMessage(payload gjson.Result) string {
	errVal := payload.Get("error")
	if !errVal.Exists() || errVal.Type == gjson.Null || errVal.Type == gjson.False {
		return ""
	}
	if errVal.Type == gjson.String {
		return strings.TrimSpace(errVal.String())
	}
	if message := strings.TrimSpace(errVal.Get("message").String()); message != "" {
		return message
	}
	return strings.TrimSpace(errVal.Raw)
}

// truncateUTF8 按字节上限截断，并回退到完整字符边界，返回是否发生截断。
func truncateUTF8(value string, maxBytes int) (string, bool) {
	if len(value) <= maxBytes {
		return value, false
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut], true
}
