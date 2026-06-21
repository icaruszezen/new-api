package debug_capture

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetStores 重建全局存储，保证各测试之间互不影响。
func resetStores() {
	errorStore = &errorStoreT{}
	recentStore = newRecentStore(recentCapacity)
}

func newTestContext(body string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	return c
}

func TestRecentStoreRingOverwrite(t *testing.T) {
	resetStores()

	const pushed = recentCapacity + 5
	for i := 0; i < pushed; i++ {
		recentStore.Push(CaptureRecord{RequestID: fmt.Sprintf("req-%d", i)})
	}

	out := recentStore.Snapshot()
	require.Len(t, out, recentCapacity, "环形缓冲应只保留固定容量条数")

	// 最新在前：最后一次 push 的 req-(pushed-1) 应排在首位。
	assert.Equal(t, fmt.Sprintf("req-%d", pushed-1), out[0].RequestID)
	// 最旧保留项应为 req-(pushed-recentCapacity)，更早的已被覆盖。
	assert.Equal(t, fmt.Sprintf("req-%d", pushed-recentCapacity), out[len(out)-1].RequestID)
}

func TestErrorStoreAppendSnapshotClear(t *testing.T) {
	resetStores()

	for i := 0; i < 3; i++ {
		errorStore.Append(CaptureRecord{RequestID: fmt.Sprintf("e-%d", i)})
	}

	out := errorStore.Snapshot()
	require.Len(t, out, 3)
	// 最新在前。
	assert.Equal(t, "e-2", out[0].RequestID)
	assert.Equal(t, "e-0", out[2].RequestID)
	// ID 单调递增。
	assert.Equal(t, uint64(3), out[0].ID)

	errorStore.Clear()
	assert.Empty(t, errorStore.Snapshot())
}

func TestErrorReason(t *testing.T) {
	relayErr := types.NewError(fmt.Errorf("boom"), types.ErrorCodeDoRequestFailed)

	cases := []struct {
		name     string
		info     *relaycommon.RelayInfo
		relayErr *types.NewAPIError
		want     string
	}{
		{name: "error wins", info: &relaycommon.RelayInfo{}, relayErr: relayErr, want: ReasonError},
		{name: "no usage", info: &relaycommon.RelayInfo{DebugBillingIssue: relaycommon.DebugBillingIssueNoUsage}, relayErr: nil, want: ReasonBillingNoUsage},
		{name: "zero tokens", info: &relaycommon.RelayInfo{DebugBillingIssue: relaycommon.DebugBillingIssueZeroTokens}, relayErr: nil, want: ReasonBillingZeroTokens},
		{name: "no issue", info: &relaycommon.RelayInfo{}, relayErr: nil, want: ""},
		{name: "nil info no error", info: nil, relayErr: nil, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, errorReason(tc.info, tc.relayErr))
		})
	}
}

func TestUpstreamCaptureBufferTruncation(t *testing.T) {
	original := constant.MaxRequestBodyMB
	constant.MaxRequestBodyMB = 1
	defer func() { constant.MaxRequestBodyMB = original }()

	buf := relaycommon.NewUpstreamCaptureBuffer()
	limit := 1 << 20

	n, err := buf.Write(make([]byte, 2*limit))
	require.NoError(t, err)
	// Write 报告消费全部字节（即便内部截断），以免破坏 TeeReader 读取。
	assert.Equal(t, 2*limit, n)
	assert.Len(t, buf.String(), limit, "超过上限的部分应被截断")
	assert.True(t, buf.Truncated())
}

func TestBuildRecordNilInfoCapturesDownstreamOnly(t *testing.T) {
	body := `{"model":"gpt-4o","stream":false}`
	c := newTestContext(body)
	relayErr := types.NewErrorWithStatusCode(fmt.Errorf("bad gateway"), types.ErrorCodeBadResponseStatusCode, 502)

	rec := buildRecord(c, nil, relayErr)

	assert.Equal(t, body, rec.DownstreamRequest, "nil relayInfo 时仍应捕获下游请求体")
	assert.Empty(t, rec.UpstreamResponse, "nil relayInfo 时上游响应为空")
	assert.False(t, rec.UpstreamCaptureIncomplete, "nil relayInfo 不应标记 incomplete")
	assert.Equal(t, string(types.ErrorCodeBadResponseStatusCode), rec.ErrorCode)
	assert.Equal(t, 502, rec.UpstreamStatus)
	assert.NotEmpty(t, rec.ErrorMessage)
}

func TestBuildRecordSDKChannelMarkedIncomplete(t *testing.T) {
	c := newTestContext(`{"model":"claude"}`)
	// 选中了渠道但没有 UpstreamCapture，模拟 SDK / 非 HTTP 渠道。
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 7}}

	rec := buildRecord(c, info, nil)

	assert.True(t, rec.UpstreamCaptureIncomplete)
	assert.Equal(t, 7, rec.ChannelID)
}

func TestFinalizeCaptureRespectsIndependentToggles(t *testing.T) {
	setting := operation_setting.GetDebugSetting()
	originalErr, originalRecent := setting.ErrorCaptureEnabled, setting.RecentCaptureEnabled
	defer func() {
		setting.ErrorCaptureEnabled = originalErr
		setting.RecentCaptureEnabled = originalRecent
	}()

	relayErr := types.NewError(fmt.Errorf("boom"), types.ErrorCodeDoRequestFailed)

	t.Run("recent only records every request, error store untouched", func(t *testing.T) {
		resetStores()
		setting.ErrorCaptureEnabled = false
		setting.RecentCaptureEnabled = true

		FinalizeCapture(newTestContext(`{"a":1}`), &relaycommon.RelayInfo{}, relayErr)

		assert.Len(t, recentStore.Snapshot(), 1)
		assert.Empty(t, errorStore.Snapshot())
	})

	t.Run("error only records failures, not clean requests", func(t *testing.T) {
		resetStores()
		setting.ErrorCaptureEnabled = true
		setting.RecentCaptureEnabled = false

		// 干净请求（无错误、无计费异常）不应被错误捕获记录。
		FinalizeCapture(newTestContext(`{"a":1}`), &relaycommon.RelayInfo{}, nil)
		assert.Empty(t, errorStore.Snapshot())
		assert.Empty(t, recentStore.Snapshot())

		// 失败请求应被记录，reason 为 error。
		FinalizeCapture(newTestContext(`{"a":1}`), &relaycommon.RelayInfo{}, relayErr)
		errs := errorStore.Snapshot()
		require.Len(t, errs, 1)
		assert.Equal(t, ReasonError, errs[0].Reason)
		assert.Empty(t, recentStore.Snapshot())
	})

	t.Run("both disabled is a no-op", func(t *testing.T) {
		resetStores()
		setting.ErrorCaptureEnabled = false
		setting.RecentCaptureEnabled = false

		FinalizeCapture(newTestContext(`{"a":1}`), &relaycommon.RelayInfo{}, relayErr)
		assert.Empty(t, errorStore.Snapshot())
		assert.Empty(t, recentStore.Snapshot())
	})
}
