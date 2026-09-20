package channelmonitor

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/relaykit/types"
)

// shouldIgnoreMonitorError 判定该失败是否应从渠道监控成败分母中扣除。
// 对齐 sub2api V2 默认忽略分类：用户侧/策略类错误不算成功，也不记失败。
func shouldIgnoreMonitorError(err *types.NewAPIError) bool {
	if err == nil {
		return false
	}
	if types.IsChannelError(err) {
		return false
	}

	switch err.GetErrorCode() {
	case types.ErrorCodeInsufficientUserQuota,
		types.ErrorCodePreConsumeTokenQuotaFailed,
		types.ErrorCodeAccessDenied,
		types.ErrorCodeModelNotFound,
		types.ErrorCodePromptBlocked,
		types.ErrorCodeSensitiveWordsDetected,
		types.ErrorCodeViolationFeeGrokCSAM:
		return true
	}

	switch err.StatusCode {
	case http.StatusUnauthorized, http.StatusNotFound, 499:
		return true
	}

	if errors.Is(err, context.Canceled) {
		return true
	}

	text := strings.ToLower(strings.Join([]string{
		string(err.GetErrorCode()),
		string(err.GetErrorType()),
		err.Error(),
	}, " "))
	return containsMonitorIgnoreText(text)
}

func containsMonitorIgnoreText(text string) bool {
	needles := []string{
		"context canceled",
		"context cancelled",
		"client cancelled",
		"client canceled",
		"unauthorized",
		"invalid api key",
		"invalid_api_key",
		"content policy",
		"content_policy",
		"safety policy",
		"moderation",
		"blocked keyword",
		"context window",
		"context length",
		"maximum prompt length",
		"group not allowed",
		"group_not_allowed",
		"does not support the requested model",
		"model not supported",
		"unsupported model",
		"run out of credits",
		"insufficient balance",
		"insufficient quota",
		"quota exceeded",
	}
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
