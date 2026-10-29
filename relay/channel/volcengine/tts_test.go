package volcengine

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ttsClient is the downstream HTTP client of a streamed TTS request. It
// records the audio it received and leaves after the first chunk, either by
// disconnecting (the request context is cancelled, as net/http does when the
// connection drops) or by failing every later write.
type ttsClient struct {
	*httptest.ResponseRecorder
	leave  string // "", "disconnect" or "write_error"
	cancel context.CancelFunc
	writes int
}

func (w *ttsClient) Write(p []byte) (int, error) {
	w.writes++
	if w.leave == "write_error" && w.writes > 1 {
		return 0, errors.New("write: broken pipe")
	}
	n, err := w.ResponseRecorder.Write(p)
	if w.leave == "disconnect" {
		w.cancel()
	}
	return n, err
}

func writeTTSAudioFrame(conn *websocket.Conn, sequence int32, payload string) error {
	flag := MsgTypeFlagPositiveSeq
	if sequence < 0 {
		flag = MsgTypeFlagNegativeSeq
	}
	msg, err := NewMessage(MsgTypeAudioOnlyServer, flag)
	if err != nil {
		return err
	}
	msg.Sequence = sequence
	msg.Payload = []byte(payload)
	frame, err := msg.Marshal()
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.BinaryMessage, frame)
}

func TestHandleTTSWebSocketResponseBillsAfterClientDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevStreamingTimeout, prevDrainMax := constant.StreamingTimeout, constant.StreamDrainMaxSeconds
	constant.StreamingTimeout = 60
	constant.StreamDrainMaxSeconds = 1800
	t.Cleanup(func() {
		constant.StreamingTimeout, constant.StreamDrainMaxSeconds = prevStreamingTimeout, prevDrainMax
	})

	tests := []struct {
		name  string
		leave string
		drain bool
		// wantBody is the audio the client received.
		wantBody string
		// wantFinalSent: the upstream reached its final frame because the
		// relay was still receiving; otherwise the relay closed the websocket
		// mid-synthesis.
		wantFinalSent bool
	}{
		{name: "connected client receives all audio", wantBody: "abc", wantFinalSent: true},
		{name: "client disconnect without drain closes the upstream and bills", leave: "disconnect", wantBody: "a"},
		{name: "client write error without drain closes the upstream and bills", leave: "write_error", wantBody: "a"},
		{name: "client disconnect with drain receives to the final frame and bills", leave: "disconnect", drain: true, wantBody: "a", wantFinalSent: true},
		{name: "client write error with drain receives to the final frame and bills", leave: "write_error", drain: true, wantBody: "a", wantFinalSent: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var finalSent atomic.Bool
			upstreamDone := make(chan struct{})
			upgrader := websocket.Upgrader{}
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer close(upstreamDone)
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close()
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
				// The final frame goes out only once the relay answers a ping,
				// i.e. only while it is still receiving after the second frame.
				conn.SetPongHandler(func(string) error {
					finalSent.Store(true)
					return writeTTSAudioFrame(conn, -3, "c")
				})
				if writeTTSAudioFrame(conn, 1, "a") != nil || writeTTSAudioFrame(conn, 2, "b") != nil {
					return
				}
				if conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second)) != nil {
					return
				}
				for {
					if _, _, err := conn.ReadMessage(); err != nil {
						return
					}
				}
			}))
			defer upstream.Close()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := &ttsClient{ResponseRecorder: httptest.NewRecorder(), leave: tt.leave, cancel: cancel}
			c, _ := gin.CreateTestContext(client)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", nil).WithContext(ctx)

			info := &relaycommon.RelayInfo{
				ChannelMeta:                     &relaycommon.ChannelMeta{ApiKey: "appid|token"},
				DrainUpstreamOnClientDisconnect: tt.drain,
			}
			info.SetEstimatePromptTokens(7)

			usage, apiErr := handleTTSWebSocketResponse(c, "ws"+strings.TrimPrefix(upstream.URL, "http"), VolcengineTTSRequest{}, info, "mp3")
			require.Nil(t, apiErr)
			assert.Equal(t, &dto.Usage{PromptTokens: 7, TotalTokens: 7}, usage)
			assert.Equal(t, tt.wantBody, client.Body.String())

			select {
			case <-upstreamDone:
			case <-time.After(5 * time.Second):
				require.FailNow(t, "upstream websocket was not closed")
			}
			assert.Equal(t, tt.wantFinalSent, finalSent.Load())
		})
	}
}
