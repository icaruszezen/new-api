package palm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const palmBody = `{"candidates":[{"author":"1","content":"Hello there"}]}`

func newPalmContext(t *testing.T, drain bool) (*gin.Context, *httptest.ResponseRecorder, context.CancelFunc, *relaycommon.RelayInfo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil).WithContext(ctx)
	info := &relaycommon.RelayInfo{
		DrainUpstreamOnClientDisconnect: drain,
		ChannelMeta:                     &relaycommon.ChannelMeta{UpstreamModelName: "chat-bison-001"},
	}
	return c, recorder, cancel, info
}

// fakePaLMUpstream sends the response headers at once and the JSON body only
// after the test closes release. upstreamClosed is closed when the gateway
// drops the connection before that.
type fakePaLMUpstream struct {
	release        chan struct{}
	upstreamClosed chan struct{}
}

func startFakePaLMUpstream(t *testing.T) (*http.Response, *fakePaLMUpstream) {
	t.Helper()
	upstream := &fakePaLMUpstream{release: make(chan struct{}), upstreamClosed: make(chan struct{})}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		select {
		case <-upstream.release:
			_, _ = io.WriteString(w, palmBody)
		case <-r.Context().Done():
			close(upstream.upstreamClosed)
		}
	}))
	t.Cleanup(func() {
		// Cut the gateway side first so a hung handler fails the test instead of blocking Close.
		server.CloseClientConnections()
		server.Close()
	})
	resp, err := server.Client().Get(server.URL)
	require.NoError(t, err)
	return resp, upstream
}

func waitClosed(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal(msg)
	}
}

func TestPalmStreamHandlerRelaysSingleChunk(t *testing.T) {
	c, recorder, _, info := newPalmContext(t, false)
	resp, upstream := startFakePaLMUpstream(t)
	close(upstream.release)

	responseText := palmStreamHandler(c, info, resp)

	assert.Equal(t, "Hello there", responseText)
	assert.Equal(t, relaycommon.StreamEndReasonEOF, info.StreamStatus.EndReason)
	events := strings.Split(strings.TrimSpace(recorder.Body.String()), "\n\n")
	require.Len(t, events, 2)
	assert.Equal(t, "data: [DONE]", events[1])
	var chunk dto.ChatCompletionsStreamResponse
	require.NoError(t, common.UnmarshalJsonStr(strings.TrimPrefix(events[0], "data: "), &chunk))
	require.Len(t, chunk.Choices, 1)
	assert.Equal(t, "palm2", chunk.Model)
	assert.Equal(t, "Hello there", chunk.Choices[0].Delta.GetContentString())
	assert.Equal(t, "stop", lo.FromPtr(chunk.Choices[0].FinishReason))
}

func TestPalmStreamHandlerClientDisconnect(t *testing.T) {
	tests := []struct {
		name           string
		drain          bool
		wantText       string
		wantEndReason  relaycommon.StreamEndReason
		upstreamCutOff bool
	}{
		{
			name:          "drain reads the whole reply for billing",
			drain:         true,
			wantText:      "Hello there",
			wantEndReason: relaycommon.StreamEndReasonDrained,
		},
		{
			name:           "no drain closes the upstream at once",
			drain:          false,
			wantText:       "",
			wantEndReason:  relaycommon.StreamEndReasonClientGone,
			upstreamCutOff: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, recorder, cancel, info := newPalmContext(t, tt.drain)
			resp, upstream := startFakePaLMUpstream(t)

			var responseText string
			done := make(chan struct{})
			go func() {
				defer close(done)
				responseText = palmStreamHandler(c, info, resp)
			}()
			cancel()
			if tt.upstreamCutOff {
				waitClosed(t, done, "handler did not return after the client disconnected")
				waitClosed(t, upstream.upstreamClosed, "upstream connection was not closed after the client disconnected")
			} else {
				select {
				case <-done:
					t.Fatal("handler returned before the upstream replied although drain is on")
				case <-time.After(100 * time.Millisecond):
				}
				close(upstream.release)
				waitClosed(t, done, "handler did not return after the upstream replied")
			}

			assert.Equal(t, tt.wantText, responseText)
			assert.Equal(t, tt.wantEndReason, info.StreamStatus.EndReason)
			assert.True(t, info.StreamStatus.ClientGone)
			// The client is gone: neither the chunk nor [DONE] is written.
			assert.Empty(t, recorder.Body.String())
		})
	}
}
