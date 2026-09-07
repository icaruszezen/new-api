package channelmonitor

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPingTargetURLNormalizesChannelBaseURL(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		expect  string
	}{
		{name: "https kept", baseURL: "https://api.openai.com", expect: "https://api.openai.com"},
		{name: "http kept", baseURL: "http://127.0.0.1:8001", expect: "http://127.0.0.1:8001"},
		{name: "path preserved", baseURL: "https://api.example.com/v1", expect: "https://api.example.com/v1"},
		// Channel base URLs are admin-entered and frequently omit the scheme.
		{name: "scheme omitted assumes https", baseURL: "api.example.com", expect: "https://api.example.com"},
		{name: "surrounding whitespace trimmed", baseURL: "  https://api.example.com  ", expect: "https://api.example.com"},
		{name: "empty base url", baseURL: "", expect: ""},
		{name: "host missing", baseURL: "https:///v1", expect: ""},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			target, err := pingTargetURL(testCase.baseURL)
			require.NoError(t, err)
			assert.Equal(t, testCase.expect, target)
		})
	}
}

// 回归：accept 一个 TCP 连接几乎是瞬时的（本机透明代理就是这样把 ping 压成 1ms 的），
// 只有等上游回响应头才能测出真实链路时延。
func TestMeasureEndpointPingWaitsForUpstreamResponse(t *testing.T) {
	const upstreamDelay = 60 * time.Millisecond

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(upstreamDelay)
		// 端点根路径通常不接受探测用的 GET，任意状态码都应算作一次成功往返。
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	pingMs := MeasureEndpointPing(t.Context(), server.Client(), server.URL)

	// 下界留出调度余量：慢机器只会让测量值更大，不会更小。
	assert.GreaterOrEqual(t, pingMs, 40)
}

func TestMeasureEndpointPingReturnsZeroWhenUnmeasurable(t *testing.T) {
	unreachable := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client := unreachable.Client()
	unreachableURL := unreachable.URL
	unreachable.Close()

	// A zero ping is the "unknown" signal the status page renders as "--", so
	// unusable input and unreachable endpoints must never surface as a latency number.
	assert.Zero(t, MeasureEndpointPing(t.Context(), client, ""))
	assert.Zero(t, MeasureEndpointPing(t.Context(), nil, "https://api.example.com"))
	assert.Zero(t, MeasureEndpointPing(t.Context(), client, unreachableURL))
}

// 回归：共享 keep-alive 客户端上若再取多次最小值，第 2、3 次瞬时响应会把第一次
// 的真实时延压成 1ms。探测必须只保留第一次成功往返。
func TestMeasureEndpointPingKeepsFirstSuccessfulRoundTrip(t *testing.T) {
	const firstDelay = 50 * time.Millisecond
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			time.Sleep(firstDelay)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	pingMs := MeasureEndpointPing(t.Context(), server.Client(), server.URL)

	assert.Equal(t, int32(1), requests.Load())
	assert.GreaterOrEqual(t, pingMs, 40)
}
