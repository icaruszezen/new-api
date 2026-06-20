package common

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"

	"github.com/stretchr/testify/assert"
)

func newFakeFrtInfo(stream bool, enabled bool, minSeconds, maxSeconds int, startedAgo time.Duration) *RelayInfo {
	return &RelayInfo{
		IsStream:  stream,
		StartTime: time.Now().Add(-startedAgo),
		ChannelMeta: &ChannelMeta{
			ChannelSetting: dto.ChannelSettings{
				StreamPreludeEnabled:         enabled,
				StreamPreludeDelayMinSeconds: minSeconds,
				StreamPreludeDelayMaxSeconds: maxSeconds,
			},
		},
	}
}

// TestSetFirstResponseTime_FakeWithinRange 验证开启「流式假首字」时，记录的首字时间被伪造为
// StartTime + 随机延迟，使 frt 落在配置区间（这里 Min==Max==1s，结果应恰为 1000ms）。
func TestSetFirstResponseTime_FakeWithinRange(t *testing.T) {
	info := newFakeFrtInfo(true, true, 1, 1, 10*time.Second)
	info.SetFirstResponseTime()

	frtMs := info.FirstResponseTime.Sub(info.StartTime).Milliseconds()
	assert.Equal(t, int64(1000), frtMs)
	assert.True(t, info.HasSendResponse())
}

// TestSetFirstResponseTime_FakeRangeBounds 验证随机毫秒级延迟落在 [Min, Max] 秒区间内。
func TestSetFirstResponseTime_FakeRangeBounds(t *testing.T) {
	for i := 0; i < 200; i++ {
		info := newFakeFrtInfo(true, true, 0, 2, 30*time.Second)
		info.SetFirstResponseTime()
		frtMs := info.FirstResponseTime.Sub(info.StartTime).Milliseconds()
		assert.GreaterOrEqual(t, frtMs, int64(0))
		assert.LessOrEqual(t, frtMs, int64(2000))
	}
}

// TestSetFirstResponseTime_DisabledKeepsReal 验证功能关闭时记录真实首字时间（不伪造）。
func TestSetFirstResponseTime_DisabledKeepsReal(t *testing.T) {
	info := newFakeFrtInfo(true, false, 0, 1, 10*time.Second)
	info.SetFirstResponseTime()

	frtMs := info.FirstResponseTime.Sub(info.StartTime).Milliseconds()
	assert.Greater(t, frtMs, int64(5000), "disabled must record real (large) first response time")
}

// TestSetFirstResponseTime_NonStreamKeepsReal 验证非流式请求不伪造首字时间。
func TestSetFirstResponseTime_NonStreamKeepsReal(t *testing.T) {
	info := newFakeFrtInfo(false, true, 0, 1, 10*time.Second)
	info.SetFirstResponseTime()

	frtMs := info.FirstResponseTime.Sub(info.StartTime).Milliseconds()
	assert.Greater(t, frtMs, int64(5000))
}

// TestSetFirstResponseTime_NotInflatedWhenRealFaster 验证真实响应比配置延迟更快时不会被“放慢”，
// 避免首字时间晚于完成时间这类不一致。
func TestSetFirstResponseTime_NotInflatedWhenRealFaster(t *testing.T) {
	// StartTime≈现在，真实首字几乎为 0；配置延迟 5s 远大于真实耗时 → 应保留真实值。
	info := newFakeFrtInfo(true, true, 5, 5, 0)
	info.SetFirstResponseTime()

	frtMs := info.FirstResponseTime.Sub(info.StartTime).Milliseconds()
	assert.Less(t, frtMs, int64(1000), "must not inflate a fast response up to the configured delay")
}

// TestSetFirstResponseTime_Idempotent 验证仅记录一次。
func TestSetFirstResponseTime_Idempotent(t *testing.T) {
	info := newFakeFrtInfo(true, true, 1, 1, 10*time.Second)
	info.SetFirstResponseTime()
	first := info.FirstResponseTime
	info.SetFirstResponseTime()
	assert.Equal(t, first, info.FirstResponseTime)
}

// TestSetFirstResponseTime_NilChannelMetaKeepsReal 验证 ChannelMeta 缺失时安全回退到真实值，不 panic。
func TestSetFirstResponseTime_NilChannelMetaKeepsReal(t *testing.T) {
	info := &RelayInfo{IsStream: true, StartTime: time.Now().Add(-10 * time.Second)}
	info.SetFirstResponseTime()

	frtMs := info.FirstResponseTime.Sub(info.StartTime).Milliseconds()
	assert.Greater(t, frtMs, int64(5000))
}
