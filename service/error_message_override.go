package service

import (
	"encoding/json"
	"strings"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/gin-gonic/gin"
)

// errorMessageOverrideRule 是一条已编译的规则：匹配串预先转成小写，
// 让热路径上的大小写不敏感比较不必反复分配。
type errorMessageOverrideRule struct {
	loweredMatch string
	replacement  string
}

// errorMessageOverrideRules 已经按 priority、id 排好序，匹配时取第一条命中的规则。
type errorMessageOverrideRules struct {
	byChannel map[int][]errorMessageOverrideRule
	global    []errorMessageOverrideRule
}

func (r *errorMessageOverrideRules) isEmpty() bool {
	return r == nil || (len(r.byChannel) == 0 && len(r.global) == 0)
}

var errorMessageOverrideCache atomic.Pointer[errorMessageOverrideRules]

// ReloadErrorMessageOverrides 重建内存规则缓存，使 relay 热路径无需查库。
func ReloadErrorMessageOverrides() error {
	overrides, err := model.GetAllErrorMessageOverrides()
	if err != nil {
		return err
	}
	rules := &errorMessageOverrideRules{byChannel: make(map[int][]errorMessageOverrideRule)}
	for _, override := range overrides {
		match := strings.TrimSpace(override.MatchSubstring)
		if !override.Enabled || match == "" {
			continue
		}
		rule := errorMessageOverrideRule{
			loweredMatch: strings.ToLower(match),
			replacement:  override.ReplacementMessage,
		}
		if override.ChannelId > 0 {
			rules.byChannel[override.ChannelId] = append(rules.byChannel[override.ChannelId], rule)
			continue
		}
		rules.global = append(rules.global, rule)
	}
	errorMessageOverrideCache.Store(rules)
	return nil
}

// SyncErrorMessageOverrides 周期性重载规则，让非 master 节点也能拿到最新配置。
func SyncErrorMessageOverrides(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		if err := ReloadErrorMessageOverrides(); err != nil {
			common.SysError("failed to sync error message overrides: " + err.Error())
		}
	}
}

// MatchErrorMessageOverride 返回命中规则的替换文案。渠道专属规则优先于全局规则，
// 同级按配置顺序取第一条。
func MatchErrorMessageOverride(c *gin.Context, message string) (string, bool) {
	rules := errorMessageOverrideCache.Load()
	if rules.isEmpty() || message == "" {
		return "", false
	}
	loweredMessage := strings.ToLower(message)
	channelId := common.GetContextKeyInt(c, constant.ContextKeyChannelId)
	if channelId > 0 {
		if replacement, ok := firstMatchingOverride(rules.byChannel[channelId], loweredMessage); ok {
			return replacement, true
		}
	}
	return firstMatchingOverride(rules.global, loweredMessage)
}

func firstMatchingOverride(rules []errorMessageOverrideRule, loweredMessage string) (string, bool) {
	for _, rule := range rules {
		if strings.Contains(loweredMessage, rule.loweredMatch) {
			return rule.replacement, true
		}
	}
	return "", false
}

// ApplyErrorMessageOverride 在错误写给下游之前替换文案。只作用于上游错误，
// 额度不足、请求校验等本地错误保持原样。调用点必须排在错误日志与自动禁用之后，
// 那些诊断路径需要看到真实的上游报错。
func ApplyErrorMessageOverride(c *gin.Context, apiErr *types.NewAPIError) {
	if apiErr == nil || !isUpstreamRelayError(apiErr) {
		return
	}
	if replacement, ok := MatchErrorMessageOverride(c, apiErr.Error()); ok {
		apiErr.OverrideMessage(replacement)
	}
}

func isUpstreamRelayError(apiErr *types.NewAPIError) bool {
	switch apiErr.GetErrorType() {
	case types.ErrorTypeOpenAIError, types.ErrorTypeClaudeError, types.ErrorTypeGeminiError,
		types.ErrorTypeMidjourneyError, types.ErrorTypeRerankError, types.ErrorTypeUpstreamError:
		return true
	}
	switch apiErr.GetErrorCode() {
	case types.ErrorCodeBadResponseStatusCode, types.ErrorCodeBadResponse, types.ErrorCodeBadResponseBody,
		types.ErrorCodeEmptyResponse, types.ErrorCodeDoRequestFailed, types.ErrorCodeAwsInvokeError:
		return true
	}
	return false
}

// RewriteStreamErrorPayload 替换流式分片里的上游报错文案。只有错误结构会被改写：
// 带 "error" 对象的分片，或 type 为 error/upstream_error 且携带顶层 message 的分片。
// 正常的增量内容原样透传。
func RewriteStreamErrorPayload(c *gin.Context, data string) string {
	if !strings.Contains(data, "error") || errorMessageOverrideCache.Load().isEmpty() {
		return data
	}
	var payload map[string]json.RawMessage
	if err := common.UnmarshalJsonStr(data, &payload); err != nil {
		return data
	}

	if rawError, ok := payload["error"]; ok && common.GetJsonType(rawError) == "object" {
		var errorFields map[string]json.RawMessage
		if err := common.Unmarshal(rawError, &errorFields); err != nil {
			return data
		}
		if !overrideMessageField(c, errorFields) {
			return data
		}
		rewrittenError, err := common.Marshal(errorFields)
		if err != nil {
			return data
		}
		payload["error"] = rewrittenError
	} else if isStreamErrorEventType(payload["type"]) {
		if !overrideMessageField(c, payload) {
			return data
		}
	} else {
		return data
	}

	rewritten, err := common.Marshal(payload)
	if err != nil {
		return data
	}
	return string(rewritten)
}

func isStreamErrorEventType(rawType json.RawMessage) bool {
	eventType := strings.ToLower(strings.TrimSpace(common.JsonRawMessageToString(rawType)))
	return eventType == "error" || eventType == "upstream_error"
}

func overrideMessageField(c *gin.Context, fields map[string]json.RawMessage) bool {
	rawMessage, ok := fields["message"]
	if !ok || common.GetJsonType(rawMessage) != "string" {
		return false
	}
	var message string
	if err := common.Unmarshal(rawMessage, &message); err != nil {
		return false
	}
	replacement, ok := MatchErrorMessageOverride(c, message)
	if !ok {
		return false
	}
	encoded, err := common.Marshal(replacement)
	if err != nil {
		return false
	}
	fields["message"] = encoded
	return true
}
