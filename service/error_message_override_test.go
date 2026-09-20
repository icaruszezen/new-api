package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedErrorMessageOverrides replaces the whole rule set and rebuilds the cache,
// so each test starts from a known configuration.
func seedErrorMessageOverrides(t *testing.T, overrides ...*model.ErrorMessageOverride) {
	t.Helper()

	require.NoError(t, model.DB.Where("1 = 1").Delete(&model.ErrorMessageOverride{}).Error)
	for _, override := range overrides {
		require.NoError(t, override.Insert())
	}
	require.NoError(t, ReloadErrorMessageOverrides())
	t.Cleanup(func() {
		require.NoError(t, model.DB.Where("1 = 1").Delete(&model.ErrorMessageOverride{}).Error)
		require.NoError(t, ReloadErrorMessageOverrides())
	})
}

func relayContextForChannel(channelId int) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(c, constant.ContextKeyChannelId, channelId)
	return c
}

func TestMatchErrorMessageOverrideSelection(t *testing.T) {
	seedErrorMessageOverrides(t,
		&model.ErrorMessageOverride{MatchSubstring: "Rate Limit", ReplacementMessage: "global rate limited", Priority: 10, Enabled: true},
		&model.ErrorMessageOverride{MatchSubstring: "rate limit", ReplacementMessage: "channel rate limited", ChannelId: 7, Priority: 10, Enabled: true},
		&model.ErrorMessageOverride{MatchSubstring: "insufficient", ReplacementMessage: "low priority", Priority: 20, Enabled: true},
		&model.ErrorMessageOverride{MatchSubstring: "insufficient balance", ReplacementMessage: "high priority", Priority: 5, Enabled: true},
		&model.ErrorMessageOverride{MatchSubstring: "deprecated model", ReplacementMessage: "never applied", Enabled: false},
	)

	testCases := []struct {
		name                string
		channelId           int
		message             string
		expectedReplacement string
		expectedMatched     bool
	}{
		{
			name:                "channel rule wins over global rule",
			channelId:           7,
			message:             "upstream returned rate limit",
			expectedReplacement: "channel rate limited",
			expectedMatched:     true,
		},
		{
			name:                "other channel falls back to global rule",
			channelId:           8,
			message:             "upstream returned rate limit",
			expectedReplacement: "global rate limited",
			expectedMatched:     true,
		},
		{
			name:                "matching ignores case",
			channelId:           0,
			message:             "UPSTREAM RETURNED RATE LIMIT",
			expectedReplacement: "global rate limited",
			expectedMatched:     true,
		},
		{
			name:                "lowest priority value matches first",
			channelId:           0,
			message:             "insufficient balance for this request",
			expectedReplacement: "high priority",
			expectedMatched:     true,
		},
		{
			name:            "disabled rule never matches",
			channelId:       0,
			message:         "deprecated model requested",
			expectedMatched: false,
		},
		{
			name:            "unrelated message is left alone",
			channelId:       7,
			message:         "context length exceeded",
			expectedMatched: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			replacement, matched := MatchErrorMessageOverride(relayContextForChannel(testCase.channelId), testCase.message)

			assert.Equal(t, testCase.expectedMatched, matched)
			if testCase.expectedMatched {
				assert.Equal(t, testCase.expectedReplacement, replacement)
			}
		})
	}
}

// 上游报错要被整段替换，而额度不足这类本地错误必须原样返回，
// 否则用户会看到与实际失败原因无关的提示。
func TestApplyErrorMessageOverrideOnlyRewritesUpstreamErrors(t *testing.T) {
	seedErrorMessageOverrides(t,
		&model.ErrorMessageOverride{MatchSubstring: "quota", ReplacementMessage: "服务繁忙，请稍后再试", Enabled: true},
	)
	c := relayContextForChannel(3)

	upstreamError := types.WithOpenAIError(types.OpenAIError{
		Message: "upstream quota exhausted",
		Type:    "insufficient_quota",
		Code:    "insufficient_quota",
	}, http.StatusTooManyRequests)
	ApplyErrorMessageOverride(c, upstreamError)

	assert.Equal(t, "服务繁忙，请稍后再试", upstreamError.Error())
	assert.Equal(t, "服务繁忙，请稍后再试", upstreamError.ToOpenAIError().Message)
	assert.Equal(t, "insufficient_quota", upstreamError.ToOpenAIError().Type)
	assert.Equal(t, http.StatusTooManyRequests, upstreamError.StatusCode)

	localError := types.NewErrorWithStatusCode(
		errors.New("用户额度不足, 剩余额度: 0"),
		types.ErrorCodeInsufficientUserQuota,
		http.StatusForbidden,
	)
	ApplyErrorMessageOverride(c, localError)

	assert.Equal(t, "用户额度不足, 剩余额度: 0", localError.Error())
}

// 覆盖后的文案是管理员手写的，不能再被敏感信息脱敏改成 ***.com；
// 同时日志侧必须仍然拿到真实的上游报错。
func TestApplyErrorMessageOverrideKeepsAdminTextAndOriginalForLogs(t *testing.T) {
	seedErrorMessageOverrides(t,
		&model.ErrorMessageOverride{MatchSubstring: "connection refused", ReplacementMessage: "请访问 status.example.com 查看服务状态", Enabled: true},
	)

	apiErr := types.WithOpenAIError(types.OpenAIError{
		Message: "connection refused by https://api.upstream-vendor.com/v1/chat",
		Type:    "upstream_error",
	}, http.StatusBadGateway)
	ApplyErrorMessageOverride(relayContextForChannel(1), apiErr)

	assert.Equal(t, "请访问 status.example.com 查看服务状态", apiErr.ToOpenAIError().Message)
	assert.Equal(t, "请访问 status.example.com 查看服务状态", apiErr.ToClaudeError().Message)
	assert.Contains(t, apiErr.MaskSensitiveError(), "connection refused")
}

func TestRewriteStreamErrorPayload(t *testing.T) {
	seedErrorMessageOverrides(t,
		&model.ErrorMessageOverride{MatchSubstring: "content moderation", ReplacementMessage: "请求被安全策略拦截", Enabled: true},
	)
	c := relayContextForChannel(11)

	testCases := []struct {
		name string
		// isJSON 决定用 JSONEq 还是 Equal 比较：重写过的分片会重新序列化，键顺序可能变化，
		// 而未命中的分片必须原样返回。
		isJSON   bool
		payload  string
		expected string
	}{
		{
			name:     "nested error object is rewritten",
			isJSON:   true,
			payload:  `{"error":{"message":"content moderation failed","type":"upstream_error"}}`,
			expected: `{"error":{"message":"请求被安全策略拦截","type":"upstream_error"}}`,
		},
		{
			name:     "top level error event is rewritten",
			isJSON:   true,
			payload:  `{"message":"content moderation failed","type":"error"}`,
			expected: `{"message":"请求被安全策略拦截","type":"error"}`,
		},
		{
			name:     "delta content mentioning the rule is untouched",
			payload:  `{"choices":[{"delta":{"content":"content moderation failed"}}]}`,
			expected: `{"choices":[{"delta":{"content":"content moderation failed"}}]}`,
		},
		{
			name:     "unmatched upstream error is untouched",
			payload:  `{"error":{"message":"context length exceeded","type":"upstream_error"}}`,
			expected: `{"error":{"message":"context length exceeded","type":"upstream_error"}}`,
		},
		{
			name:     "stream terminator is untouched",
			payload:  "[DONE]",
			expected: "[DONE]",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rewritten := RewriteStreamErrorPayload(c, testCase.payload)

			if testCase.isJSON {
				assert.JSONEq(t, testCase.expected, rewritten)
				return
			}
			assert.Equal(t, testCase.expected, rewritten)
		})
	}
}
