package relay

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// alphaSearchLeavingClient receives the first write and then disconnects:
// like net/http after the connection drops, the request context is cancelled
// and every later write fails.
type alphaSearchLeavingClient struct {
	*httptest.ResponseRecorder
	cancel context.CancelFunc
	gone   bool
}

func (w *alphaSearchLeavingClient) Write(p []byte) (int, error) {
	if w.gone {
		return 0, errors.New("write: broken pipe")
	}
	w.gone = true
	w.cancel()
	return w.ResponseRecorder.Write(p)
}

// alphaSearchUpstreamBody records whether the relay read the upstream
// response to EOF.
type alphaSearchUpstreamBody struct {
	io.Reader
	eof bool
}

func (b *alphaSearchUpstreamBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	if err == io.EOF {
		b.eof = true
	}
	return n, err
}

func TestRelayAlphaSearchResponseBillsDrainedClientDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		clientLeaves  bool
		drain         bool
		upstreamFails bool
		wantBilled    bool
		wantBody      string
		wantEOF       bool
	}{
		{name: "connected client gets the whole response", wantBilled: true, wantBody: `{"results":[{"title":"x"}]}`, wantEOF: true},
		{name: "client disconnect with drain reads the rest and bills", clientLeaves: true, drain: true, wantBilled: true, wantBody: `{"results":[`, wantEOF: true},
		{name: "client disconnect without drain refunds", clientLeaves: true, wantBody: `{"results":[`},
		{name: "upstream failure during the drain refunds", clientLeaves: true, drain: true, upstreamFails: true, wantBody: `{"results":[`},
		{name: "upstream failure with a connected client refunds even with drain", drain: true, upstreamFails: true, wantBody: `{"results":[{"title":"x"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			recorder := httptest.NewRecorder()
			var writer http.ResponseWriter = recorder
			if tt.clientLeaves {
				writer = &alphaSearchLeavingClient{ResponseRecorder: recorder, cancel: cancel}
			}
			c, _ := gin.CreateTestContext(writer)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/alpha/search", nil).WithContext(ctx)

			tail := io.Reader(strings.NewReader(`]}`))
			if tt.upstreamFails {
				tail = iotest.ErrReader(errors.New("upstream connection reset"))
			}
			body := &alphaSearchUpstreamBody{Reader: io.MultiReader(strings.NewReader(`{"results":[`), strings.NewReader(`{"title":"x"}`), tail)}
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(body),
			}

			err := relayAlphaSearchResponse(c, &relaycommon.RelayInfo{DrainUpstreamOnClientDisconnect: tt.drain}, resp)
			if tt.wantBilled {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			assert.Equal(t, tt.wantBody, recorder.Body.String())
			assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
			assert.Equal(t, tt.wantEOF, body.eof)
		})
	}
}
