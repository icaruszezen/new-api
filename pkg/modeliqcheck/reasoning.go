package modeliqcheck

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
)

// 检查项的思考深度。空值不改动模型默认的思考设置；其余取值按上游协议与模型代际
// 换算成该模型接受的参数，模型没有的档位取最接近的档位。
const (
	ReasoningEffortNone    = "none"
	ReasoningEffortMinimal = "minimal"
	ReasoningEffortLow     = "low"
	ReasoningEffortMedium  = "medium"
	ReasoningEffortHigh    = "high"
	ReasoningEffortXHigh   = "xhigh"
	ReasoningEffortMax     = "max"
)

// ReasoningEfforts 是允许配置的思考深度，空值表示模型默认。
var ReasoningEfforts = []string{
	"",
	ReasoningEffortNone,
	ReasoningEffortMinimal,
	ReasoningEffortLow,
	ReasoningEffortMedium,
	ReasoningEffortHigh,
	ReasoningEffortXHigh,
	ReasoningEffortMax,
}

// EndpointTypes 是允许配置的请求格式，空值沿用渠道测试的自动检测
// （OpenAI Chat Completions；codex 模型与 Codex 渠道使用 Responses）。
var EndpointTypes = []string{
	"",
	string(constant.EndpointTypeOpenAI),
	string(constant.EndpointTypeOpenAIResponse),
	string(constant.EndpointTypeAnthropic),
	string(constant.EndpointTypeGemini),
}

// claudeThinkingOff 是关闭 Claude 思考的方式。
type claudeThinkingOff int

const (
	// claudeThinkingOffOmit：省略 thinking 即不思考（4.6 及更早的模型、Opus 4.7 / 4.8）。
	claudeThinkingOffOmit claudeThinkingOff = iota
	// claudeThinkingOffDisabled：发送 {type: disabled}（Opus 5、Sonnet 5、Haiku 5）。
	claudeThinkingOffDisabled
	// claudeThinkingOffBetweenTools：发送 {type: between_tools}（Sonnet 5.5），disabled 会返回 400。
	claudeThinkingOffBetweenTools
	// claudeThinkingOffUnsupported：无法关闭思考（Opus 5.5、Fable、Mythos 及未知的更新代际），
	// 只能把 effort 降到 low。
	claudeThinkingOffUnsupported
)

// claudeThinkingProfile 是 Claude 模型对思考参数的要求：
//   - 4.6 之前只支持手动思考 {type: enabled, budget_tokens}；
//   - Opus / Sonnet 4.6 起使用自适应思考 {type: adaptive} 加 output_config.effort，
//     Opus 4.7 起 budget_tokens 与 {type: enabled} 返回 400，思考文本默认不返回；
//   - Opus 5、Sonnet 5、Haiku 5、Fable 起省略 thinking 即开启自适应思考。
type claudeThinkingProfile struct {
	budgetTokens   bool
	onByDefault    bool
	displayOmitted bool
	// noXHigh：effort 没有 xhigh 档（Opus / Sonnet 4.6 只有 low / medium / high / max）。
	noXHigh bool
	off     claudeThinkingOff
}

// claudeModelPattern 匹配模型系列与版本号，兼容 claude-opus-4-8、claude-opus-4.8、
// anthropic.claude-sonnet-4-5-20250929-v1:0、claude-opus-4-1@20250805 等写法；
// 版本号后紧跟的 8 位日期不会被当作小版本。
var claudeModelPattern = regexp.MustCompile(`(opus|sonnet|haiku|fable|mythos)(?:[-_.]?(\d{1,2})(?:[-_.](\d{1,2}))?)?(?:[^0-9]|$)`)

func claudeThinkingProfileFor(model string) claudeThinkingProfile {
	budget := claudeThinkingProfile{budgetTokens: true}
	match := claudeModelPattern.FindStringSubmatch(strings.ToLower(model))
	if match == nil {
		// claude-3.x 与无法识别的模型（包括兼容 Anthropic 接口的第三方模型）
		// 使用最通用的手动思考预算。
		return budget
	}
	adaptive := claudeThinkingProfile{onByDefault: true, displayOmitted: true, off: claudeThinkingOffUnsupported}
	family := match[1]
	if family == "fable" || family == "mythos" {
		return adaptive
	}
	if match[2] == "" {
		return budget
	}
	major, _ := strconv.Atoi(match[2])
	minor, _ := strconv.Atoi(match[3])
	version := major*100 + minor
	adaptive46 := claudeThinkingProfile{noXHigh: true, off: claudeThinkingOffOmit}

	switch family {
	case "opus":
		switch {
		case version < 406:
			return budget
		case version < 407:
			return adaptive46
		case version < 500:
			return claudeThinkingProfile{displayOmitted: true, off: claudeThinkingOffOmit}
		case version < 505:
			adaptive.off = claudeThinkingOffDisabled
		}
	case "sonnet":
		switch {
		case version < 406:
			return budget
		case version < 500:
			return adaptive46
		case version < 505:
			adaptive.off = claudeThinkingOffDisabled
		case version < 600:
			adaptive.off = claudeThinkingOffBetweenTools
		}
	case "haiku":
		switch {
		case version < 500:
			return budget
		case version < 600:
			adaptive.off = claudeThinkingOffDisabled
		}
	}
	return adaptive
}

// claudeBudgetPercent 是手动思考模型的思考预算占 max_tokens 的比例。
var claudeBudgetPercent = map[string]uint{
	ReasoningEffortMinimal: 0,
	ReasoningEffortLow:     25,
	ReasoningEffortMedium:  50,
	ReasoningEffortHigh:    75,
	ReasoningEffortXHigh:   85,
	ReasoningEffortMax:     90,
}

// minClaudeThinkingBudget 是 Anthropic 要求的最小 budget_tokens。
const minClaudeThinkingBudget = 1024

// ClaudeThinkingFields 是 Claude Messages 请求中与思考相关的字段。
type ClaudeThinkingFields struct {
	Thinking     *dto.Thinking
	OutputConfig json.RawMessage
	MaxTokens    uint
}

// ApplyClaudeThinking 按模型代际把思考深度换算成 Claude Messages 参数：
// 自适应思考模型发送 {type: adaptive} 与 output_config.effort，
// 手动思考模型按 max_tokens 的比例换算 budget_tokens（必要时抬高 max_tokens）。
// effort 为空时只在 summary 为真时为默认开启思考的模型补上 display=summarized，
// 其余保持原样，避免覆盖渠道按模型名后缀设置的思考参数。
func ApplyClaudeThinking(model string, effort string, summary bool, fields ClaudeThinkingFields) (ClaudeThinkingFields, error) {
	profile := claudeThinkingProfileFor(model)
	if effort == "" {
		if !summary || profile.budgetTokens || !profile.displayOmitted {
			return fields, nil
		}
		if fields.Thinking == nil && profile.onByDefault {
			fields.Thinking = &dto.Thinking{Type: "adaptive"}
		}
		if fields.Thinking != nil && fields.Thinking.Type == "adaptive" && fields.Thinking.Display == "" {
			thinking := *fields.Thinking
			thinking.Display = "summarized"
			fields.Thinking = &thinking
		}
		return fields, nil
	}

	if profile.budgetTokens {
		var err error
		fields.OutputConfig, err = setOutputConfigEffort(fields.OutputConfig, "")
		if err != nil {
			return fields, err
		}
		if effort == ReasoningEffortNone {
			fields.Thinking = nil
			return fields, nil
		}
		// max_tokens 必须大于 budget_tokens：至少留出 1024 的思考预算与 1024 的回答空间。
		if fields.MaxTokens < 2*minClaudeThinkingBudget {
			fields.MaxTokens = 2 * minClaudeThinkingBudget
		}
		percent := claudeBudgetPercent[effort]
		budget := fields.MaxTokens/100*percent + fields.MaxTokens%100*percent/100
		budget = max(budget, minClaudeThinkingBudget)
		budget = min(budget, fields.MaxTokens-minClaudeThinkingBudget)
		fields.Thinking = &dto.Thinking{Type: "enabled", BudgetTokens: common.GetPointer(int(budget))}
		return fields, nil
	}

	level := effort
	switch effort {
	case ReasoningEffortNone:
		if profile.off != claudeThinkingOffUnsupported {
			var err error
			fields.OutputConfig, err = setOutputConfigEffort(fields.OutputConfig, "")
			if err != nil {
				return fields, err
			}
			switch profile.off {
			case claudeThinkingOffOmit:
				fields.Thinking = nil
			case claudeThinkingOffDisabled:
				fields.Thinking = &dto.Thinking{Type: "disabled"}
			case claudeThinkingOffBetweenTools:
				fields.Thinking = &dto.Thinking{Type: "between_tools"}
			}
			return fields, nil
		}
		level = ReasoningEffortLow
	case ReasoningEffortMinimal:
		level = ReasoningEffortLow
	case ReasoningEffortXHigh:
		if profile.noXHigh {
			level = ReasoningEffortMax
		}
	}
	thinking := &dto.Thinking{Type: "adaptive"}
	if summary && profile.displayOmitted {
		thinking.Display = "summarized"
	}
	fields.Thinking = thinking
	var err error
	fields.OutputConfig, err = setOutputConfigEffort(fields.OutputConfig, level)
	return fields, err
}

// setOutputConfigEffort 设置或删除 output_config.effort，保留 output_config 的其他字段。
func setOutputConfigEffort(raw json.RawMessage, effort string) (json.RawMessage, error) {
	config := map[string]any{}
	if len(raw) > 0 && common.GetJsonType(raw) == "object" {
		if err := common.Unmarshal(raw, &config); err != nil {
			return raw, err
		}
	}
	if effort == "" {
		delete(config, "effort")
	} else {
		config["effort"] = effort
	}
	if len(config) == 0 {
		return nil, nil
	}
	return common.Marshal(config)
}

// openAIModelPattern 匹配 OpenAI 模型名中的版本号，如 gpt-5、gpt-5.1-codex-max、gpt-4o、o3、o4-mini。
var (
	openAIGPTPattern = regexp.MustCompile(`^gpt-(\d+)(?:\.(\d+))?(?:[^0-9.]|$)`)
	openAIOPattern   = regexp.MustCompile(`^o\d+(?:-|$)`)
)

// openAIModelVersion 解析 OpenAI 模型名（忽略 openai/ 之类的前缀与微调模型的 ft: 前缀），
// 返回小写的模型名、是否为 o 系列与 gpt 版本号；都不是时 gptVersion 为 -1。
func openAIModelVersion(model string) (name string, oSeries bool, gptVersion int) {
	name = strings.ToLower(strings.TrimSpace(model))
	name = strings.TrimPrefix(name[strings.LastIndex(name, "/")+1:], "ft:")
	if openAIOPattern.MatchString(name) {
		return name, true, -1
	}
	match := openAIGPTPattern.FindStringSubmatch(name)
	if match == nil {
		return name, false, -1
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	return name, false, major*100 + minor
}

// OpenAIReasoningEffort 把思考深度换算成 OpenAI 协议（Chat Completions 的 reasoning_effort、
// Responses 的 reasoning.effort）的取值，返回空串表示不发送：
//   - o 系列只有 low / medium / high；
//   - gpt-5 是 minimal / low / medium / high，gpt-5-codex 只有 low / medium / high，
//     gpt-5-pro 只有 high；
//   - gpt-5.1 是 none / low / medium / high（gpt-5.1-codex-max 另有 xhigh）；
//   - gpt-5.2 至 gpt-5.5 是 none / low / medium / high / xhigh，其 pro 版只有 medium / high / xhigh；
//   - gpt-5.6 起另有 max；gpt-6 起只有 gpt-6-luna 能关闭思考，其余最低为 low；
//   - gpt-4.x 等非推理模型不接受该参数，不发送；
//   - 其他供应商的模型（如经 OpenRouter）原样发送，由上游换算；
//     但不能关闭思考的 Gemini 与 Claude 模型会拒绝 none，改用 low。
func OpenAIReasoningEffort(model string, effort string) string {
	if effort == "" {
		return ""
	}
	name, oSeries, version := openAIModelVersion(model)
	isPro := strings.Contains(name, "-pro")
	switch {
	case oSeries:
		switch effort {
		case ReasoningEffortNone, ReasoningEffortMinimal:
			return ReasoningEffortLow
		case ReasoningEffortXHigh, ReasoningEffortMax:
			return ReasoningEffortHigh
		}
	case version < 0:
		if effort == ReasoningEffortNone && !thinkingCanBeDisabled(name) {
			return ReasoningEffortLow
		}
	case version < 500:
		return ""
	case version < 501:
		switch {
		case isPro:
			return ReasoningEffortHigh
		case effort == ReasoningEffortXHigh, effort == ReasoningEffortMax:
			return ReasoningEffortHigh
		case strings.Contains(name, "codex") && (effort == ReasoningEffortNone || effort == ReasoningEffortMinimal):
			return ReasoningEffortLow
		case effort == ReasoningEffortNone:
			return ReasoningEffortMinimal
		}
	case version < 502:
		switch effort {
		case ReasoningEffortMinimal:
			return ReasoningEffortLow
		case ReasoningEffortXHigh, ReasoningEffortMax:
			if strings.Contains(name, "codex-max") {
				return ReasoningEffortXHigh
			}
			return ReasoningEffortHigh
		}
	case isPro && version < 506:
		switch effort {
		case ReasoningEffortNone, ReasoningEffortMinimal, ReasoningEffortLow:
			return ReasoningEffortMedium
		case ReasoningEffortMax:
			return ReasoningEffortXHigh
		}
	case version < 506:
		switch effort {
		case ReasoningEffortMinimal:
			return ReasoningEffortLow
		case ReasoningEffortMax:
			return ReasoningEffortXHigh
		}
	default:
		switch {
		case effort == ReasoningEffortMinimal:
			return ReasoningEffortLow
		case effort == ReasoningEffortNone && version >= 600 && !strings.Contains(name, "luna"):
			return ReasoningEffortLow
		}
	}
	return effort
}

// thinkingCanBeDisabled 判断经 OpenAI 兼容接口访问的其他供应商模型能否关闭思考：
// Gemini 2.5 Pro 与 Gemini 3 起的模型、以及 claudeThinkingOffUnsupported 的 Claude 模型不能关闭。
func thinkingCanBeDisabled(name string) bool {
	if version, ok := geminiModelVersion(name); ok {
		return version < 205 || (version < 300 && !strings.Contains(name, "-pro"))
	}
	if strings.Contains(name, "claude") {
		return claudeThinkingProfileFor(name).off != claudeThinkingOffUnsupported
	}
	return true
}

// ResponsesReasoning 返回 Responses 请求的 reasoning 字段：写入换算后的 effort，
// summary 为真时请求 auto 摘要（OpenAI 只有设置了 summary 才返回思考内容）。
// gpt-4.x 等非推理模型不接受 reasoning，原样返回。
func ResponsesReasoning(model string, effort string, summary bool, current *dto.Reasoning) *dto.Reasoning {
	_, oSeries, version := openAIModelVersion(model)
	if !oSeries && version >= 0 && version < 500 {
		return current
	}
	upstreamEffort := OpenAIReasoningEffort(model, effort)
	if upstreamEffort == "" && !summary {
		return current
	}
	reasoning := &dto.Reasoning{}
	if current != nil {
		copied := *current
		reasoning = &copied
	}
	if upstreamEffort != "" {
		reasoning.Effort = upstreamEffort
	}
	if summary && reasoning.Effort != ReasoningEffortNone {
		reasoning.Summary = "auto"
	}
	return reasoning
}

// geminiModelPattern 匹配 Gemini 版本号，如 gemini-2.5-pro、gemini-3-flash-preview、gemini-3.1-pro。
var (
	geminiModelPattern       = regexp.MustCompile(`gemini-(\d+)(?:\.(\d+))?(?:[^0-9.]|$)`)
	geminiLatestAliasPattern = regexp.MustCompile(`gemini-(?:pro|flash|flash-lite)-latest`)
)

// geminiLatestAliasVersion 是 gemini-*-latest 别名的版本：别名指向最新一代（Gemini 3 起），
// 按 3.x 最新模型的规则换算。
const geminiLatestAliasVersion = 399

// geminiModelVersion 解析 Gemini 模型名中的版本号（如 2.5 记为 205），无法识别时返回 false。
func geminiModelVersion(name string) (int, bool) {
	match := geminiModelPattern.FindStringSubmatch(name)
	if match == nil {
		return geminiLatestAliasVersion, geminiLatestAliasPattern.MatchString(name)
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	return major*100 + minor, true
}

// GeminiThinkingConfig 把思考深度换算成 Gemini 的 thinkingConfig，返回 nil 或 current 表示不改动：
//   - Gemini 2.5 使用 thinkingBudget：Pro 范围 128~32768 且不能关闭，Flash-Lite 最小 512，
//     其余 0~24576；
//   - Gemini 3 起使用 thinkingLevel 且都不能关闭思考：Pro 最低为 low（3.0 Pro 只有 low / high），
//     Flash 与 Flash-Lite 有 minimal / low / medium / high（3.8 Flash 起没有 minimal）；
//   - 2.5 之前的模型与无法识别的模型不支持思考参数，不改动。
//
// thinkingBudget 与 thinkingLevel 不能同时设置，换算时只保留其一。
func GeminiThinkingConfig(model string, effort string, summary bool, current *dto.GeminiThinkingConfig) *dto.GeminiThinkingConfig {
	name := strings.ToLower(model)
	version, ok := geminiModelVersion(name)
	if !ok || version < 205 {
		return current
	}
	if effort == "" {
		if !summary {
			return current
		}
		config := &dto.GeminiThinkingConfig{}
		if current != nil {
			copied := *current
			config = &copied
		}
		config.IncludeThoughts = true
		return config
	}

	isPro := strings.Contains(name, "-pro")
	isFlashLite := strings.Contains(name, "flash-lite")
	if version < 300 {
		var budget int
		switch effort {
		case ReasoningEffortNone, ReasoningEffortMinimal:
			switch {
			case isPro:
				budget = 128
			case isFlashLite && effort == ReasoningEffortMinimal:
				budget = 512
			case effort == ReasoningEffortMinimal:
				budget = 128
			}
		case ReasoningEffortLow:
			budget = 1024
		case ReasoningEffortMedium:
			budget = 8192
		case ReasoningEffortHigh:
			budget = 24576
		default:
			budget = 24576
			if isPro {
				budget = 32768
			}
		}
		return &dto.GeminiThinkingConfig{ThinkingBudget: common.GetPointer(budget), IncludeThoughts: summary && budget > 0}
	}

	var level string
	switch effort {
	case ReasoningEffortNone, ReasoningEffortMinimal:
		level = ReasoningEffortMinimal
		if isPro || (!isFlashLite && version >= 308) {
			level = ReasoningEffortLow
		}
	case ReasoningEffortLow:
		level = ReasoningEffortLow
	case ReasoningEffortMedium:
		level = ReasoningEffortMedium
		if isPro && version == 300 {
			level = ReasoningEffortHigh
		}
	default:
		level = ReasoningEffortHigh
	}
	return &dto.GeminiThinkingConfig{ThinkingLevel: level, IncludeThoughts: summary}
}
