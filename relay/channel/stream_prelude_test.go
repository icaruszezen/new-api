package channel

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPreludeTestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req, err := http.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	require.NoError(t, err)
	c.Request = req
	return c, recorder
}

func newPreludeTestInfo(format types.RelayFormat, model string) *common.RelayInfo {
	return &common.RelayInfo{
		RelayFormat:     format,
		OriginModelName: model,
	}
}

// TestDeliverStreamPrelude_SendsPerProtocol 验证三种受支持协议下 prelude 内容符合下游客户端期望：
// Chat 发空首字 chunk；Claude / Responses 仅发 SSE 注释行，不发伪造业务事件。
func TestDeliverStreamPrelude_SendsPerProtocol(t *testing.T) {
	t.Run("openai chat sends empty start chunk", func(t *testing.T) {
		c, recorder := newPreludeTestContext(t)
		info := newPreludeTestInfo(types.RelayFormatOpenAI, "gpt-4o-mini")

		sent := deliverStreamPrelude(c, info)

		assert.True(t, sent)
		body := recorder.Body.String()
		assert.Contains(t, body, "chat.completion.chunk")
		assert.Contains(t, body, "\"role\":\"assistant\"")
		assert.Contains(t, body, "\"content\":\"\"")
		assert.Contains(t, body, "\"model\":\"gpt-4o-mini\"")
	})

	t.Run("claude sends comment line only", func(t *testing.T) {
		c, recorder := newPreludeTestContext(t)
		info := newPreludeTestInfo(types.RelayFormatClaude, "claude-3-5-sonnet")

		sent := deliverStreamPrelude(c, info)

		assert.True(t, sent)
		body := recorder.Body.String()
		assert.Equal(t, ": \n\n", body)
		assert.NotContains(t, body, "message_start")
		assert.NotContains(t, body, "data:")
	})

	t.Run("responses sends comment line only", func(t *testing.T) {
		c, recorder := newPreludeTestContext(t)
		info := newPreludeTestInfo(types.RelayFormatOpenAIResponses, "gpt-4o")

		sent := deliverStreamPrelude(c, info)

		assert.True(t, sent)
		body := recorder.Body.String()
		assert.Equal(t, ": \n\n", body)
		assert.NotContains(t, body, "response.created")
		assert.NotContains(t, body, "data:")
	})
}

// TestDeliverStreamPrelude_SkipWhenUpstreamStarted 验证上游已返回业务内容时不发 prelude（上游优先）。
func TestDeliverStreamPrelude_SkipWhenUpstreamStarted(t *testing.T) {
	c, recorder := newPreludeTestContext(t)
	info := newPreludeTestInfo(types.RelayFormatOpenAI, "gpt-4o-mini")
	info.MarkStreamUpstreamStarted()

	sent := deliverStreamPrelude(c, info)

	assert.False(t, sent)
	assert.Empty(t, recorder.Body.String())
}

// TestDeliverStreamPrelude_OnlyOnce 验证每请求最多发送一次 prelude。
func TestDeliverStreamPrelude_OnlyOnce(t *testing.T) {
	c, recorder := newPreludeTestContext(t)
	info := newPreludeTestInfo(types.RelayFormatClaude, "claude-3-5-sonnet")

	require.True(t, deliverStreamPrelude(c, info))
	first := recorder.Body.String()

	assert.False(t, deliverStreamPrelude(c, info))
	assert.Equal(t, first, recorder.Body.String(), "second call must not write again")
}

// TestDeliverStreamPrelude_UnsupportedFormat 验证非受支持协议不发 prelude。
func TestDeliverStreamPrelude_UnsupportedFormat(t *testing.T) {
	c, recorder := newPreludeTestContext(t)
	info := newPreludeTestInfo(types.RelayFormatGemini, "gemini-1.5-pro")

	sent := deliverStreamPrelude(c, info)

	assert.False(t, sent)
	assert.Empty(t, recorder.Body.String())
}

// TestStreamPreludeDelay_Bounds 验证随机延迟落在 [Min, Max] 秒区间内，且对非法配置做钳制。
func TestStreamPreludeDelay_Bounds(t *testing.T) {
	cases := []struct {
		name        string
		minSeconds  int
		maxSeconds  int
		expectMinMs int64
		expectMaxMs int64
	}{
		{name: "fixed zero", minSeconds: 0, maxSeconds: 0, expectMinMs: 0, expectMaxMs: 0},
		{name: "fixed value", minSeconds: 3, maxSeconds: 3, expectMinMs: 3000, expectMaxMs: 3000},
		{name: "range", minSeconds: 2, maxSeconds: 5, expectMinMs: 2000, expectMaxMs: 5000},
		{name: "max below min clamped", minSeconds: 4, maxSeconds: 1, expectMinMs: 4000, expectMaxMs: 4000},
		{name: "negative min clamped", minSeconds: -3, maxSeconds: 2, expectMinMs: 0, expectMaxMs: 2000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := &common.RelayInfo{
				ChannelMeta: &common.ChannelMeta{
					ChannelSetting: dto.ChannelSettings{
						StreamPreludeEnabled:         true,
						StreamPreludeDelayMinSeconds: tc.minSeconds,
						StreamPreludeDelayMaxSeconds: tc.maxSeconds,
					},
				},
			}
			for i := 0; i < 50; i++ {
				ms := streamPreludeDelay(info).Milliseconds()
				assert.GreaterOrEqual(t, ms, tc.expectMinMs)
				assert.LessOrEqual(t, ms, tc.expectMaxMs)
			}
		})
	}
}

// TestPreludeEnabledForFormat 验证渠道开关与协议共同决定 prelude 是否启用。
func TestPreludeEnabledForFormat(t *testing.T) {
	makeInfo := func(enabled bool, format types.RelayFormat) *common.RelayInfo {
		return &common.RelayInfo{
			RelayFormat: format,
			ChannelMeta: &common.ChannelMeta{
				ChannelSetting: dto.ChannelSettings{StreamPreludeEnabled: enabled},
			},
		}
	}

	assert.True(t, preludeEnabledForFormat(makeInfo(true, types.RelayFormatOpenAI)))
	assert.True(t, preludeEnabledForFormat(makeInfo(true, types.RelayFormatClaude)))
	assert.True(t, preludeEnabledForFormat(makeInfo(true, types.RelayFormatOpenAIResponses)))
	assert.False(t, preludeEnabledForFormat(makeInfo(false, types.RelayFormatOpenAI)))
	assert.False(t, preludeEnabledForFormat(makeInfo(true, types.RelayFormatGemini)))
}
