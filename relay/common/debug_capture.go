package common

import (
	"bytes"
	"io"
	"net/http"
	"sync"

	"github.com/QuantumNous/new-api/constant"
)

// 计费异常原因，由 Post*ConsumeQuota 在上游未返回有效计费信息时写入 RelayInfo.DebugBillingIssue。
const (
	// DebugBillingIssueNoUsage 表示上游未返回任何 usage（计费信息）。
	DebugBillingIssueNoUsage = "billing_no_usage"
	// DebugBillingIssueZeroTokens 表示 total tokens 为 0，无法扣费（可能是上游超时）。
	DebugBillingIssueZeroTokens = "billing_zero_tokens"
)

// UpstreamCaptureBuffer 线程安全地累积上游响应的原始字节，带最大容量限制，
// 供调试捕获功能记录「上游返回的内容」。超过上限后停止追加并标记 truncated。
type UpstreamCaptureBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func upstreamCaptureLimitBytes() int {
	mb := constant.MaxRequestBodyMB
	if mb <= 0 {
		mb = 128
	}
	return mb << 20
}

// NewUpstreamCaptureBuffer 创建一个带默认上限（复用 MAX_REQUEST_BODY_MB 数值）的缓冲。
func NewUpstreamCaptureBuffer() *UpstreamCaptureBuffer {
	return &UpstreamCaptureBuffer{limit: upstreamCaptureLimitBytes()}
}

// Write 实现 io.Writer，配合 io.TeeReader 在读取响应体时同步累积字节。
func (b *UpstreamCaptureBuffer) Write(p []byte) (int, error) {
	if b == nil {
		return len(p), nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.limit > 0 {
		remaining := b.limit - b.buf.Len()
		if remaining <= 0 {
			b.truncated = true
			return len(p), nil
		}
		if len(p) > remaining {
			b.buf.Write(p[:remaining])
			b.truncated = true
			return len(p), nil
		}
	}
	b.buf.Write(p)
	return len(p), nil
}

func (b *UpstreamCaptureBuffer) String() string {
	if b == nil {
		return ""
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (b *UpstreamCaptureBuffer) Truncated() bool {
	if b == nil {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.truncated
}

// ResetUpstreamCapture 在每次重试迭代开始时调用，丢弃上一次尝试累积的上游响应，
// 只保留最终一次尝试的内容，避免多次重试的响应叠加。
func (info *RelayInfo) ResetUpstreamCapture() {
	info.UpstreamCapture = NewUpstreamCaptureBuffer()
}

// WrapUpstreamBody 用 io.TeeReader 包装 resp.Body，使后续读取响应体（流式扫描或
// io.ReadAll）时同步把原始字节累积到 info.UpstreamCapture。仅在调试捕获开启时调用。
func WrapUpstreamBody(resp *http.Response, info *RelayInfo) {
	if resp == nil || resp.Body == nil || info == nil {
		return
	}
	if info.UpstreamCapture == nil {
		info.UpstreamCapture = NewUpstreamCaptureBuffer()
	}
	resp.Body = &captureReadCloser{
		reader: io.TeeReader(resp.Body, info.UpstreamCapture),
		closer: resp.Body,
	}
}

type captureReadCloser struct {
	reader io.Reader
	closer io.Closer
}

func (t *captureReadCloser) Read(p []byte) (int, error) { return t.reader.Read(p) }

func (t *captureReadCloser) Close() error { return t.closer.Close() }
