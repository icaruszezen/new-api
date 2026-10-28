package claude

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const claudeStreamTestModel = "claude-sonnet-4-5"

// lockedRecorder lets the test read what the client has received while the
// stream goroutines are still writing to it.
type lockedRecorder struct {
	*httptest.ResponseRecorder
	mu sync.Mutex
}

func (r *lockedRecorder) Write(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ResponseRecorder.Write(b)
}

func (r *lockedRecorder) WriteString(s string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ResponseRecorder.WriteString(s)
}

func (r *lockedRecorder) received() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.Body.String()
}

func claudeSSE(events ...string) string {
	var b strings.Builder
	for _, event := range events {
		b.WriteString("data: " + event + "\n\n")
	}
	return b.String()
}

func claudeTextDelta(text string) string {
	return `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"` + text + `"}}`
}

const (
	claudeMessageStart      = `{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"` + claudeStreamTestModel + `","content":[],"usage":{"input_tokens":10,"output_tokens":1}}}`
	claudeTextBlockStart    = `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`
	claudeContentBlockStop  = `{"type":"content_block_stop","index":0}`
	claudeMessageDelta77Out = `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":77}}`
	claudeMessageStop       = `{"type":"message_stop"}`
)

func newClaudeStreamTestInfo(format types.RelayFormat, drain bool) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayFormat:                     format,
		OriginModelName:                 claudeStreamTestModel,
		DisablePing:                     true,
		DrainUpstreamOnClientDisconnect: drain,
		ChannelMeta:                     &relaycommon.ChannelMeta{UpstreamModelName: claudeStreamTestModel},
	}
}

// message_start 自带 output_tokens=1；流在 message_delta 之前结束（截断、断开、排水超时）时，
// 结算读的 BillingUsage 必须是本地补估的输出，而不是 message_start 的 1。
func TestClaudeStreamHandlerBillsEstimatedOutputWithoutMessageDelta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	const streamedText = "Hello, this stream was cut off before its message_delta ever arrived."
	wantOutput := service.EstimateTokenByModel(claudeStreamTestModel, streamedText)
	require.Greater(t, wantOutput, 1)

	for _, format := range []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI} {
		t.Run(string(format), func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
			info := newClaudeStreamTestInfo(format, false)
			body := claudeSSE(claudeMessageStart, claudeTextBlockStart, claudeTextDelta(streamedText), claudeContentBlockStop)

			usage, apiErr := ClaudeStreamHandler(c, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, info)

			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, 10, usage.PromptTokens)
			assert.Equal(t, wantOutput, usage.CompletionTokens)
			require.NotNil(t, usage.BillingUsage)
			require.NotNil(t, usage.BillingUsage.ClaudeUsage)
			assert.Equal(t, 10, usage.BillingUsage.ClaudeUsage.InputTokens)
			assert.Equal(t, wantOutput, usage.BillingUsage.ClaudeUsage.OutputTokens)
			assert.True(t, usage.BillingUsage.Estimated)
			require.NotNil(t, info.StreamStatus)
			assert.Equal(t, relaycommon.StreamEndReasonEOF, info.StreamStatus.EndReason)
		})
	}
}

// 开排水时客户端在流中途断开，断开后才到的 message_delta 用上游给的数结算，不被本地估算覆盖。
func TestClaudeStreamHandlerDrainBillsMessageDeltaAfterDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	for _, format := range []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI} {
		t.Run(string(format), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			pr, pw := io.Pipe()
			t.Cleanup(func() {
				cancel()
				_ = pr.Close()
				_ = pw.Close()
			})

			recorder := &lockedRecorder{ResponseRecorder: httptest.NewRecorder()}
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil).WithContext(ctx)
			info := newClaudeStreamTestInfo(format, true)

			type result struct {
				usage  *dto.Usage
				apiErr *types.NewAPIError
			}
			done := make(chan result, 1)
			go func() {
				usage, apiErr := ClaudeStreamHandler(c, &http.Response{StatusCode: http.StatusOK, Body: pr}, info)
				done <- result{usage: usage, apiErr: apiErr}
			}()

			_, err := fmt.Fprint(pw, claudeSSE(claudeMessageStart, claudeTextBlockStart, claudeTextDelta("Hello")))
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				return strings.Contains(recorder.received(), "Hello")
			}, 2*time.Second, 5*time.Millisecond)

			cancel()

			var got result
			select {
			case got = <-done:
				t.Fatalf("handler returned right after the client disconnected: %+v", got)
			case <-time.After(200 * time.Millisecond):
			}

			_, err = fmt.Fprint(pw, claudeSSE(claudeTextDelta(" drained text"), claudeContentBlockStop, claudeMessageDelta77Out, claudeMessageStop))
			require.NoError(t, err)
			require.NoError(t, pw.Close())

			select {
			case got = <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("handler did not return after the drained upstream ended")
			}

			require.Nil(t, got.apiErr)
			require.NotNil(t, got.usage)
			assert.Equal(t, 10, got.usage.PromptTokens)
			assert.Equal(t, 77, got.usage.CompletionTokens)
			require.NotNil(t, got.usage.BillingUsage)
			require.NotNil(t, got.usage.BillingUsage.ClaudeUsage)
			assert.Equal(t, 10, got.usage.BillingUsage.ClaudeUsage.InputTokens)
			assert.Equal(t, 77, got.usage.BillingUsage.ClaudeUsage.OutputTokens)
			assert.False(t, got.usage.BillingUsage.Estimated)

			require.NotNil(t, info.StreamStatus)
			assert.Equal(t, relaycommon.StreamEndReasonDrained, info.StreamStatus.EndReason)
			assert.Equal(t, relaycommon.StreamEndReasonEOF, info.StreamStatus.DrainedFrom)
			assert.False(t, info.StreamStatus.HasErrors())

			assert.NotContains(t, recorder.received(), "drained text")
		})
	}
}
