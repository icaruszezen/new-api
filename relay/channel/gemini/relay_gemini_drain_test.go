package gemini

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
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// 客户端在流中途断开：开排水时断开后才到的最后一块 usageMetadata 必须进计费，
// 不开排水时上游要立即关闭、按 client_gone 结束。
func TestGeminiStreamHandlersClientDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })

	const (
		beforeDisconnect = `{"candidates":[{"content":{"role":"model","parts":[{"text":"Hello"}]}}],"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":2,"totalTokenCount":6}}`
		afterDisconnect  = `{"candidates":[{"content":{"role":"model","parts":[{"text":" drained text"}]}}],"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":20,"totalTokenCount":24}}`
		finalChunk       = `{"candidates":[{"content":{"role":"model","parts":[{"text":" after the client left"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":50,"totalTokenCount":54}}`
	)

	tests := []struct {
		name            string
		handler         func(*gin.Context, *relaycommon.RelayInfo, *http.Response) (*dto.Usage, *types.NewAPIError)
		relayFormat     types.RelayFormat
		drain           bool
		wantCompletion  int
		wantTotal       int
		wantEndReason   relaycommon.StreamEndReason
		wantDrainedFrom relaycommon.StreamEndReason
	}{
		{
			name:            "native stream drains to final usage",
			handler:         GeminiTextGenerationStreamHandler,
			relayFormat:     types.RelayFormatGemini,
			drain:           true,
			wantCompletion:  50,
			wantTotal:       54,
			wantEndReason:   relaycommon.StreamEndReasonDrained,
			wantDrainedFrom: relaycommon.StreamEndReasonEOF,
		},
		{
			name:           "native stream without drain closes upstream",
			handler:        GeminiTextGenerationStreamHandler,
			relayFormat:    types.RelayFormatGemini,
			drain:          false,
			wantCompletion: 2,
			wantTotal:      6,
			wantEndReason:  relaycommon.StreamEndReasonClientGone,
		},
		{
			name:            "openai client drains to final usage",
			handler:         GeminiChatStreamHandler,
			relayFormat:     types.RelayFormatOpenAI,
			drain:           true,
			wantCompletion:  50,
			wantTotal:       54,
			wantEndReason:   relaycommon.StreamEndReasonDrained,
			wantDrainedFrom: relaycommon.StreamEndReasonEOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			pr, pw := io.Pipe()
			t.Cleanup(func() {
				cancel()
				_ = pr.Close()
				_ = pw.Close()
			})

			recorder := &lockedRecorder{ResponseRecorder: httptest.NewRecorder()}
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:streamGenerateContent", nil).WithContext(ctx)

			info := &relaycommon.RelayInfo{
				RelayFormat:                     tt.relayFormat,
				OriginModelName:                 "gemini-2.5-flash",
				DisablePing:                     true,
				DrainUpstreamOnClientDisconnect: tt.drain,
				ChannelMeta:                     &relaycommon.ChannelMeta{UpstreamModelName: "gemini-2.5-flash"},
			}

			type result struct {
				usage  *dto.Usage
				apiErr *types.NewAPIError
			}
			done := make(chan result, 1)
			go func() {
				usage, apiErr := tt.handler(c, info, &http.Response{StatusCode: http.StatusOK, Body: pr})
				done <- result{usage: usage, apiErr: apiErr}
			}()

			_, err := fmt.Fprint(pw, "data: "+beforeDisconnect+"\n")
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				return strings.Contains(recorder.received(), "Hello")
			}, 2*time.Second, 5*time.Millisecond)

			cancel()

			var got result
			if tt.drain {
				select {
				case got = <-done:
					t.Fatalf("handler returned right after the client disconnected: %+v", got)
				case <-time.After(200 * time.Millisecond):
				}
				_, err = fmt.Fprint(pw, "data: "+afterDisconnect+"\n"+"data: "+finalChunk+"\n")
				require.NoError(t, err)
				require.NoError(t, pw.Close())
			}

			select {
			case got = <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("handler did not return")
			}

			if !tt.drain {
				_, err = fmt.Fprint(pw, "data: "+afterDisconnect+"\n")
				require.ErrorIs(t, err, io.ErrClosedPipe, "upstream body should be closed once the client is gone")
			}

			require.Nil(t, got.apiErr)
			require.NotNil(t, got.usage)
			assert.Equal(t, 4, got.usage.PromptTokens)
			assert.Equal(t, tt.wantCompletion, got.usage.CompletionTokens)
			assert.Equal(t, tt.wantTotal, got.usage.TotalTokens)
			// 结算读的是 BillingUsage
			require.NotNil(t, got.usage.BillingUsage)
			require.NotNil(t, got.usage.BillingUsage.GeminiUsageMetadata)
			assert.Equal(t, tt.wantCompletion, got.usage.BillingUsage.GeminiUsageMetadata.CandidatesTokenCount)
			assert.False(t, got.usage.BillingUsage.Estimated)

			require.NotNil(t, info.StreamStatus)
			assert.Equal(t, tt.wantEndReason, info.StreamStatus.EndReason)
			assert.Equal(t, tt.wantDrainedFrom, info.StreamStatus.DrainedFrom)
			assert.True(t, info.StreamStatus.ClientGone)
			assert.False(t, info.StreamStatus.HasErrors())

			assert.NotContains(t, recorder.received(), "drained text")
			assert.NotContains(t, recorder.received(), "after the client left")
		})
	}
}
