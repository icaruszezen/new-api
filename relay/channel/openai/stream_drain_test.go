package openai

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

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// disconnectingStream runs a stream handler against an upstream the test feeds
// through a pipe. The client disconnects right after a write containing
// disconnectAfter reaches it (see cancelAfterWriter).
//
// A cancelled request context alone does not mean the wait loop in
// helper.scanUpstreamStream has acted on it. With drain on, the loop must take its clientGone case before an
// upstream terminal ([DONE] or EOF) closes stopChan: when both are ready,
// select picks one at random, and if it picks the stop the stream ends as a
// plain done/eof that never drained. So with drain on, disconnectSeen closes
// only when the wait loop logs, inside that case, that it has started
// draining. With drain off it is the request context: those tests send
// nothing more, so clientGone is the only case the loop can take.
type disconnectingStream struct {
	c              *gin.Context
	upstream       *io.PipeWriter
	disconnectSeen <-chan struct{}
	done           chan struct{}
	usage          *dto.Usage
	apiErr         *types.NewAPIError
}

// drainLogWriter passes log output through and closes draining once the stream
// scanner logs that it keeps reading the upstream after the client left.
type drainLogWriter struct {
	io.Writer
	draining chan struct{}
	once     sync.Once
}

func (w *drainLogWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "stream drain: client disconnected") {
		w.once.Do(func() { close(w.draining) })
	}
	return w.Writer.Write(p)
}

func startDisconnectingStream(t *testing.T, path, disconnectAfter string, info *relaycommon.RelayInfo,
	handler func(*gin.Context, *relaycommon.RelayInfo, *http.Response) (*dto.Usage, *types.NewAPIError)) *disconnectingStream {
	t.Helper()
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30

	ctx, cancel := context.WithCancel(context.Background())
	pr, pw := io.Pipe()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, path, nil).WithContext(ctx)
	c.Set(common.RequestIdKey, "drain-test")
	c.Writer = &cancelAfterWriter{ResponseWriter: c.Writer, needle: disconnectAfter, cancel: cancel}

	s := &disconnectingStream{c: c, upstream: pw, disconnectSeen: ctx.Done(), done: make(chan struct{})}
	if info.DrainUpstreamOnClientDisconnect {
		common.LogWriterMu.Lock()
		logs := &drainLogWriter{Writer: gin.DefaultWriter, draining: make(chan struct{})}
		gin.DefaultWriter = logs
		common.LogWriterMu.Unlock()
		s.disconnectSeen = logs.draining
		// Registered before the cleanup below, so it runs after the handler returned.
		t.Cleanup(func() {
			common.LogWriterMu.Lock()
			gin.DefaultWriter = logs.Writer
			common.LogWriterMu.Unlock()
		})
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       pr,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}
	go func() {
		defer close(s.done)
		s.usage, s.apiErr = handler(c, info, resp)
	}()
	t.Cleanup(func() {
		cancel()
		_ = pw.Close()
		<-s.done
		gin.SetMode(oldMode)
		constant.StreamingTimeout = oldTimeout
	})
	return s
}

// send writes each event as an SSE data line; a write returns once the
// handler's scanner has read it.
func (s *disconnectingStream) send(t *testing.T, events ...string) {
	t.Helper()
	for _, event := range events {
		_, err := fmt.Fprint(s.upstream, "data: "+event+"\n")
		require.NoError(t, err)
	}
}

// waitClientGone returns once the handler has seen the disconnect (see
// disconnectingStream); only then may the test send an upstream terminal.
func (s *disconnectingStream) waitClientGone(t *testing.T) {
	t.Helper()
	select {
	case <-s.disconnectSeen:
	case <-time.After(2 * time.Second):
		t.Fatal("client did not disconnect, or with drain on the handler did not start draining")
	}
}

func (s *disconnectingStream) wait(t *testing.T) (*dto.Usage, *types.NewAPIError) {
	t.Helper()
	select {
	case <-s.done:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not return")
	}
	return s.usage, s.apiErr
}

// 客户端断开后：开启排水时计费用断开后上游才发出的 usage chunk；未开启时立即关闭上游，
// 按已收到的文本本地估算，且都不能把写入失败记成流错误。
func TestOaiStreamHandlerClientDisconnect(t *testing.T) {
	const estimatedPrompt = 9
	beforeDisconnect := []string{
		`{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1710000000,"model":"gpt-test","choices":[{"index":0,"delta":{"role":"assistant","content":"hello"}}]}`,
		`{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1710000000,"model":"gpt-test","choices":[{"index":0,"delta":{"content":" world"}}]}`,
	}
	afterDisconnect := []string{
		`{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1710000000,"model":"gpt-test","choices":[{"index":0,"delta":{"content":" and the rest"},"finish_reason":"stop"}]}`,
		`{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1710000000,"model":"gpt-test","choices":[],"usage":{"prompt_tokens":11,"completion_tokens":42,"total_tokens":53}}`,
		`[DONE]`,
	}
	estimatedCompletion := service.EstimateTokenByModel("gpt-test", "hello world")
	require.Positive(t, estimatedCompletion)

	tests := []struct {
		name            string
		drain           bool
		afterDisconnect []string
		wantPrompt      int
		wantCompletion  int
		wantEndReason   relaycommon.StreamEndReason
		wantLocalCount  bool
	}{
		{
			name:            "drain bills the usage chunk sent after the disconnect",
			drain:           true,
			afterDisconnect: afterDisconnect,
			wantPrompt:      11,
			wantCompletion:  42,
			wantEndReason:   relaycommon.StreamEndReasonDrained,
		},
		{
			name:           "without drain the upstream is closed and the text received is estimated",
			wantPrompt:     estimatedPrompt,
			wantCompletion: estimatedCompletion,
			wantEndReason:  relaycommon.StreamEndReasonClientGone,
			wantLocalCount: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				RelayFormat:                     types.RelayFormatOpenAI,
				RelayMode:                       relayconstant.RelayModeChatCompletions,
				IsStream:                        true,
				ShouldIncludeUsage:              true,
				DisablePing:                     true,
				DrainUpstreamOnClientDisconnect: tt.drain,
				ChannelMeta:                     &relaycommon.ChannelMeta{UpstreamModelName: "gpt-test"},
			}
			info.SetEstimatePromptTokens(estimatedPrompt)

			// The handler forwards a chunk when the next one arrives, so the
			// "hello" chunk reaches the client (and disconnects it) on the second.
			s := startDisconnectingStream(t, "/v1/chat/completions", "hello", info, OaiStreamHandler)
			s.send(t, beforeDisconnect...)
			s.waitClientGone(t)
			s.send(t, tt.afterDisconnect...)
			usage, apiErr := s.wait(t)

			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, tt.wantPrompt, usage.PromptTokens)
			assert.Equal(t, tt.wantCompletion, usage.CompletionTokens)
			assert.Equal(t, tt.wantPrompt+tt.wantCompletion, usage.TotalTokens)
			assert.Equal(t, tt.wantLocalCount, common.GetContextKeyBool(s.c, constant.ContextKeyLocalCountTokens))
			require.NotNil(t, info.StreamStatus)
			assert.Equal(t, tt.wantEndReason, info.StreamStatus.EndReason)
			assert.True(t, info.StreamStatus.ClientGone)
			assert.False(t, info.StreamStatus.HasErrors())

			_, err := fmt.Fprint(s.upstream, "data: late\n")
			assert.ErrorIs(t, err, io.ErrClosedPipe, "the upstream body must be closed")
		})
	}
}
