package common

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayInfoGetFinalRequestRelayFormatPrefersExplicitFinal(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		RequestConversionChain:  []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
		FinalRequestRelayFormat: types.RelayFormatOpenAIResponses,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToConversionChain(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:            types.RelayFormatOpenAI,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatClaude), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToRelayFormat(t *testing.T) {
	info := &RelayInfo{
		RelayFormat: types.RelayFormatGemini,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatGemini), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatNilReceiver(t *testing.T) {
	var info *RelayInfo
	require.Equal(t, types.RelayFormat(""), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoMetaTypedNilReceiver(t *testing.T) {
	var info *RelayInfo
	var meta convmeta.Meta = info

	assert.Empty(t, meta.GetOriginModelName())
	assert.Empty(t, meta.GetUpstreamModelName())
	assert.False(t, meta.HasChannelMeta())
	assert.Zero(t, meta.GetChannelID())
	assert.Zero(t, meta.GetChannelType())
	assert.False(t, meta.GetIsStream())
	assert.Empty(t, meta.GetReasoningEffort())
	assert.Zero(t, meta.GetEstimatePromptTokens())
	assert.Zero(t, meta.GetSendResponseCount())

	assert.NotPanics(t, func() {
		meta.SetReasoningEffort("high")
		meta.IncrSendResponseCount()
		meta.AppendRequestConversion(types.RelayFormatClaude)
	})

	firstState := meta.EnsureClaudeConvertInfo()
	secondState := meta.EnsureClaudeConvertInfo()
	require.NotNil(t, firstState)
	require.NotNil(t, secondState)
	assert.Equal(t, convmeta.LastMessageTypeNone, firstState.LastMessagesType)
	assert.NotSame(t, firstState, secondState)

	firstOptions := meta.ConvOptions()
	secondOptions := meta.ConvOptions()
	require.NotNil(t, firstOptions)
	require.NotNil(t, secondOptions)
	assert.NotSame(t, firstOptions, secondOptions)
	assert.NotNil(t, firstOptions.Claude.DefaultMaxTokens)
	assert.NotNil(t, firstOptions.Gemini.SupportsImagine)
	assert.NotNil(t, firstOptions.Gemini.SafetySetting)
	assert.NotNil(t, firstOptions.PreserveThinkingSuffix)
}

func TestGenRelayInfoCapturesRequestReasoningEffort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name        string
		path        string
		relayFormat types.RelayFormat
		request     dto.Request
		expected    string
	}{
		{
			name:        "OpenAI chat top-level effort",
			path:        "/v1/chat/completions",
			relayFormat: types.RelayFormatOpenAI,
			request:     &dto.GeneralOpenAIRequest{Model: "gpt-5.6-sol", ReasoningEffort: " high "},
			expected:    "high",
		},
		{
			name:        "OpenRouter nested chat effort",
			path:        "/v1/chat/completions",
			relayFormat: types.RelayFormatOpenAI,
			request:     &dto.GeneralOpenAIRequest{Model: "anthropic/claude", Reasoning: json.RawMessage(`{"effort":"xhigh"}`)},
			expected:    "xhigh",
		},
		{
			name:        "OpenAI Responses effort",
			path:        "/v1/responses",
			relayFormat: types.RelayFormatOpenAIResponses,
			request:     &dto.OpenAIResponsesRequest{Model: "gpt-5.6-sol", Reasoning: &dto.Reasoning{Effort: "max"}},
			expected:    "max",
		},
		{
			name:        "explicit none is preserved",
			path:        "/v1/responses",
			relayFormat: types.RelayFormatOpenAIResponses,
			request:     &dto.OpenAIResponsesRequest{Model: "gpt-5.6-sol", Reasoning: &dto.Reasoning{Effort: "none"}},
			expected:    "none",
		},
		{
			name:        "non-string nested effort is ignored",
			path:        "/v1/chat/completions",
			relayFormat: types.RelayFormatOpenAI,
			request:     &dto.GeneralOpenAIRequest{Model: "anthropic/claude", Reasoning: json.RawMessage(`{"effort":42}`)},
			expected:    "",
		},
		{
			name:        "Claude output config effort",
			path:        "/v1/messages",
			relayFormat: types.RelayFormatClaude,
			request:     &dto.ClaudeRequest{Model: "claude-opus-4-7", OutputConfig: json.RawMessage(`{"effort":"medium"}`)},
			expected:    "medium",
		},
		{
			name:        "Gemini thinking level",
			path:        "/v1beta/models/gemini-3-pro:generateContent",
			relayFormat: types.RelayFormatGemini,
			request: &dto.GeminiChatRequest{GenerationConfig: dto.GeminiChatGenerationConfig{
				ThinkingConfig: &dto.GeminiThinkingConfig{ThinkingLevel: "low"},
			}},
			expected: "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("POST", tt.path, nil)

			info, err := GenRelayInfo(ctx, tt.relayFormat, tt.request, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, info.ReasoningEffort)
		})
	}
}

func TestInitChannelMetaRestoresRequestReasoningEffortForRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	request := &dto.OpenAIResponsesRequest{
		Model:     "gpt-5.6-sol",
		Reasoning: &dto.Reasoning{Effort: "max"},
	}
	info, err := GenRelayInfo(ctx, types.RelayFormatOpenAIResponses, request, nil)
	require.NoError(t, err)

	info.SetReasoningEffort("high")
	info.InitChannelMeta(ctx)
	assert.Equal(t, "max", info.ReasoningEffort)

	info.SetReasoningEffort("low")
	info.InitChannelMeta(ctx)
	assert.Equal(t, "max", info.ReasoningEffort)
}

func TestInitChannelMetaClearsResolvedCacheReadBillingRatio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	info := &RelayInfo{}
	info.SetResolvedCacheReadBillingRatio(0.97)

	info.InitChannelMeta(ctx)

	_, ok := info.ResolvedCacheReadBillingRatio()
	assert.False(t, ok)
}

func TestInitChannelMetaSetsDrainFromChannelSetting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name    string
		setting dto.ChannelSettings
		want    bool
	}{
		{name: "off", setting: dto.ChannelSettings{}, want: false},
		{name: "new key", setting: dto.ChannelSettings{ClientDisconnectDrainEnabled: true}, want: true},
		{name: "legacy responses-only key", setting: dto.ChannelSettings{ResponsesClientDisconnectDrainEnabled: true}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The flag must not depend on the endpoint: chat, messages and
			// responses all drain when the channel enables it.
			for _, path := range []string{"/v1/chat/completions", "/v1/messages", "/v1/responses"} {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
				ctx.Request = httptest.NewRequest("POST", path, nil)
				common.SetContextKey(ctx, constant.ContextKeyChannelSetting, tc.setting)
				info := &RelayInfo{DrainUpstreamOnClientDisconnect: !tc.want}

				info.InitChannelMeta(ctx)

				assert.Equal(t, tc.want, info.DrainUpstreamOnClientDisconnect, "path=%s", path)
			}
		})
	}
}

func TestInitChannelMetaClearsDrainUpstreamOnClientDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	info := &RelayInfo{DrainUpstreamOnClientDisconnect: true}

	info.InitChannelMeta(ctx)

	assert.False(t, info.DrainUpstreamOnClientDisconnect)
}

func TestGenRelayInfoJoinsRepeatedRequestHeaderLines(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	c.Request.Header.Add("anthropic-beta", "thinking-binding-controls-2026-08-01")
	c.Request.Header.Add("anthropic-beta", "fast-mode-2026-02-01")
	c.Request.Header.Set("anthropic-version", "2023-06-01")

	info := GenRelayInfoClaude(c, &dto.ClaudeRequest{Model: "claude-opus-5-5"})

	// Billing expressions read header("anthropic-beta"); it must see every
	// line the Claude adaptor forwards upstream, not only the first one.
	assert.Equal(t, "thinking-binding-controls-2026-08-01,fast-mode-2026-02-01", info.RequestHeaders["Anthropic-Beta"])
	assert.Equal(t, "2023-06-01", info.RequestHeaders["Anthropic-Version"])
}

// A client may repeat a key the gateway removes or replaces (the AWS
// pass-through drops "stream", alpha search rewrites "model"). Every copy
// goes in one pass over the parent object: removing one copy at a time copies
// the whole body per copy. The bound on allocated bytes fails such a
// quadratic edit without timing it.
func TestJSONMemberEditsRemoveManyRepeatedKeysLinearly(t *testing.T) {
	const copies = 20000
	body := []byte(`{"model":"m","stream":false,"messages":[{"role":"user","content":"hi"}]` +
		strings.Repeat(`,"stream":false`, copies) + `,"max_tokens":1e3}`)

	var deleted, set []byte
	var deleteErr, setErr error
	deleteAllocated := allocatedBytes(func() {
		deleted, deleteErr = DeleteJSONMember(body, "", "stream")
	})
	setAllocated := allocatedBytes(func() {
		set, setErr = SetJSONMember(body, "", "stream", true)
	})

	require.NoError(t, deleteErr)
	require.NoError(t, setErr)
	assert.Equal(t, `{"model":"m","messages":[{"role":"user","content":"hi"}],"max_tokens":1e3}`, string(deleted))
	assert.Equal(t, `{"model":"m","messages":[{"role":"user","content":"hi"}],"max_tokens":1e3,"stream":true}`, string(set))
	assert.Less(t, deleteAllocated, uint64(16*len(body)))
	assert.Less(t, setAllocated, uint64(16*len(body)))
}

// Removing repeated copies leaves the other members as the client wrote them,
// including the separators between them.
func TestDeleteJSONMemberRemovesEveryCopy(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		parent string
		key    string
		want   string
	}{
		{
			name:  "single copy",
			input: `{"a": 1, "x": 2, "b": 3}`,
			key:   "x",
			want:  `{"a": 1, "b": 3}`,
		},
		{
			name:  "first, middle and last copies",
			input: "{\"x\":0, \"a\": 1.0,\n \"x\": 2,\n \"b\": \"<b>&\", \"x\":4}",
			key:   "x",
			want:  "{\"a\": 1.0,\n \"b\": \"<b>&\"}",
		},
		{
			name:   "nested parent",
			input:  `{"stream_options":{"include_obfuscation":false,"include_usage":true,"include_obfuscation":true},"n":1e2}`,
			parent: "stream_options",
			key:    "include_obfuscation",
			want:   `{"stream_options":{"include_usage":true},"n":1e2}`,
		},
		{
			name:  "key that is path syntax",
			input: `{"a.b":1,"c":2,"a.b":3}`,
			key:   "a.b",
			want:  `{"c":2}`,
		},
		{
			name:  "every member",
			input: `{"x":1,"x":2}`,
			key:   "x",
			want:  `{}`,
		},
		{
			name:   "parent is not an object",
			input:  `{"a":[1],"x":1}`,
			parent: "a",
			key:    "x",
			want:   `{"a":[1],"x":1}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := DeleteJSONMember([]byte(tt.input), tt.parent, tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(out))
		})
	}
}
