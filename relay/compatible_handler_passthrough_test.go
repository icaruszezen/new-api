package relay

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// relayPassThroughChat runs raw through TextHelper on a pass-through channel
// whose upstream rejects every request, so the relay stops before billing.
// It returns the body the upstream received and the relay info.
func relayPassThroughChat(t *testing.T, channelType int, setting dto.ChannelSettings, raw string) ([]byte, *relaycommon.RelayInfo) {
	t.Helper()
	received := make(chan []byte, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- body
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"rejected by test upstream","type":"invalid_request_error"}}`))
	}))
	t.Cleanup(upstream.Close)

	setting.PassThroughBodyEnabled = true
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	common.SetContextKey(c, constant.ContextKeyChannelType, channelType)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, upstream.URL)
	common.SetContextKey(c, constant.ContextKeyChannelKey, "sk-test")
	common.SetContextKey(c, constant.ContextKeyOriginalModel, gjson.Get(raw, "model").String())
	common.SetContextKey(c, constant.ContextKeyChannelSetting, setting)

	var request dto.GeneralOpenAIRequest
	require.NoError(t, common.UnmarshalJsonStr(raw, &request))
	info := relaycommon.GenRelayInfoOpenAI(c, &request)

	require.NotNil(t, TextHelper(c, info), "the test upstream rejects every request")
	select {
	case body := <-received:
		return body, info
	default:
		require.FailNow(t, "the upstream received no request")
		return nil, nil
	}
}

func TestTextHelperPassThroughAsksUpstreamForStreamUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevForceStreamOption := constant.ForceStreamOption
	t.Cleanup(func() { constant.ForceStreamOption = prevForceStreamOption })

	const messages = `"messages":[{"role":"user","content":"hi"}]`
	tests := []struct {
		name              string
		channelType       int
		drain             bool
		forceStreamOption bool
		raw               string
		// want is the exact upstream body; empty means the raw body untouched.
		want                   string
		wantShouldIncludeUsage bool
	}{
		{
			name:                   "drained stream asks for usage and keeps every other byte",
			channelType:            constant.ChannelTypeOpenAI,
			drain:                  true,
			forceStreamOption:      true,
			raw:                    `{"model":"gpt-4o", "stream": true,` + messages + `}`,
			want:                   `{"model":"gpt-4o", "stream": true,` + messages + `,"stream_options":{"include_usage":true}}`,
			wantShouldIncludeUsage: true,
		},
		{
			name:              "client opt-out is overridden upstream like the non-pass-through path",
			channelType:       constant.ChannelTypeOpenAI,
			drain:             true,
			forceStreamOption: true,
			raw:               `{"model":"gpt-4o","stream":true,"stream_options":{"include_usage":false,"include_obfuscation":false},` + messages + `}`,
			want:              `{"model":"gpt-4o","stream":true,"stream_options":{"include_usage":true,"include_obfuscation":false},` + messages + `}`,
		},
		{
			name:                   "drain disabled",
			channelType:            constant.ChannelTypeOpenAI,
			forceStreamOption:      true,
			raw:                    `{"model":"gpt-4o","stream":true,` + messages + `}`,
			wantShouldIncludeUsage: true,
		},
		{
			name:                   "non-stream request",
			channelType:            constant.ChannelTypeOpenAI,
			drain:                  true,
			forceStreamOption:      true,
			raw:                    `{"model":"gpt-4o",` + messages + `}`,
			wantShouldIncludeUsage: true,
		},
		{
			name:                   "channel without stream_options support",
			channelType:            constant.ChannelTypeCustom,
			drain:                  true,
			forceStreamOption:      true,
			raw:                    `{"model":"gpt-4o","stream":true,` + messages + `}`,
			wantShouldIncludeUsage: true,
		},
		{
			name:                   "FORCE_STREAM_OPTION disabled",
			channelType:            constant.ChannelTypeOpenAI,
			drain:                  true,
			raw:                    `{"model":"gpt-4o","stream":true,` + messages + `}`,
			wantShouldIncludeUsage: true,
		},
		{
			name:                   "client already asked for usage",
			channelType:            constant.ChannelTypeOpenAI,
			drain:                  true,
			forceStreamOption:      true,
			raw:                    `{"model":"gpt-4o","stream":true,"stream_options":{"include_usage":true},` + messages + `}`,
			wantShouldIncludeUsage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constant.ForceStreamOption = tt.forceStreamOption
			got, info := relayPassThroughChat(t, tt.channelType, dto.ChannelSettings{ClientDisconnectDrainEnabled: tt.drain}, tt.raw)

			want := tt.want
			if want == "" {
				want = tt.raw
			}
			assert.Equal(t, want, string(got))
			// What the client sees still follows the client's own choice.
			assert.Equal(t, tt.wantShouldIncludeUsage, info.ShouldIncludeUsage)
		})
	}
}

func TestTextHelperPassThroughStreamUsageComposesWithReasoningEffort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevForceStreamOption := constant.ForceStreamOption
	constant.ForceStreamOption = true
	t.Cleanup(func() { constant.ForceStreamOption = prevForceStreamOption })

	got, _ := relayPassThroughChat(t, constant.ChannelTypeOpenAI, dto.ChannelSettings{
		ClientDisconnectDrainEnabled:  true,
		AutoSetReasoningEffortByModel: true,
	}, `{"model":"gpt-5-high","stream":true,"messages":[{"role":"user","content":"hi"}]}`)

	assert.Equal(t, "high", gjson.GetBytes(got, "reasoning_effort").String())
	assert.True(t, gjson.GetBytes(got, "stream_options.include_usage").Bool())
	assert.Equal(t, "hi", gjson.GetBytes(got, "messages.0.content").String())
}

// The pass-through include_usage patch relies on OaiStreamHandler showing the
// upstream usage chunk only to clients that asked for usage, as it does on the
// non-pass-through path where FORCE_STREAM_OPTION also forces include_usage.
func TestOaiStreamHandlerShowsForcedUsageChunkOnlyWhenClientAsked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 60
	t.Cleanup(func() { constant.StreamingTimeout = prevStreamingTimeout })

	upstreamStream := strings.Join([]string{
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":null}],"usage":null}`,
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":null}`,
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[],"usage":{"prompt_tokens":11,"completion_tokens":22,"total_tokens":33}}`,
		`data: [DONE]`,
	}, "\n\n") + "\n\n"

	tests := []struct {
		name               string
		shouldIncludeUsage bool
	}{
		{name: "client asked for usage", shouldIncludeUsage: true},
		{name: "client opted out of usage", shouldIncludeUsage: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			info := &relaycommon.RelayInfo{
				RelayFormat:        types.RelayFormatOpenAI,
				RelayMode:          relayconstant.RelayModeChatCompletions,
				IsStream:           true,
				ShouldIncludeUsage: tt.shouldIncludeUsage,
				ChannelMeta:        &relaycommon.ChannelMeta{UpstreamModelName: "gpt-4o"},
			}
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(bytes.NewBufferString(upstreamStream)),
			}

			usage, apiErr := openai.OaiStreamHandler(c, info, resp)
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, 11, usage.PromptTokens)
			assert.Equal(t, 22, usage.CompletionTokens)
			assert.Equal(t, 33, usage.TotalTokens)

			clientStream := recorder.Body.String()
			assert.Contains(t, clientStream, `"content":"hi"`)
			assert.Equal(t, tt.shouldIncludeUsage, strings.Contains(clientStream, `"prompt_tokens":11`))
			assert.True(t, strings.HasSuffix(strings.TrimSpace(clientStream), "data: [DONE]"))
		})
	}
}
