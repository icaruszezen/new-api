package service

import (
	"context"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// ChannelRetryPolicy is a channel's own retry setting: a request that fails
// with one of the listed upstream statuses is re-sent to the same channel up to
// Times times, waiting Interval before each attempt. It runs before, and
// independently of, the global retry that switches channels.
type ChannelRetryPolicy struct {
	Times       int
	Interval    time.Duration
	statusCodes []operation_setting.StatusCodeRange
}

// GetChannelRetryPolicy returns the retry policy of the channel selected for
// the current attempt. Times is 0 when the channel does not retry.
func GetChannelRetryPolicy(c *gin.Context) ChannelRetryPolicy {
	settings, _ := common.GetContextKeyType[dto.ChannelSettings](c, constant.ContextKeyChannelSetting)
	if settings.RetryTimes <= 0 {
		return ChannelRetryPolicy{}
	}
	ranges, err := operation_setting.ParseHTTPStatusCodeRanges(settings.RetryStatusCodes)
	if err != nil {
		logger.LogWarn(c, fmt.Sprintf("channel retry disabled, invalid retry_status_codes %q: %s", settings.RetryStatusCodes, err.Error()))
		return ChannelRetryPolicy{}
	}
	if len(ranges) == 0 {
		ranges, _ = operation_setting.ParseHTTPStatusCodeRanges(dto.DefaultChannelRetryStatusCodes)
	}
	intervalMs := dto.DefaultChannelRetryIntervalMs
	if settings.RetryIntervalMs != nil {
		intervalMs = *settings.RetryIntervalMs
	}
	// Saved settings are validated; the clamps only guard rows written around
	// that validation.
	return ChannelRetryPolicy{
		Times:       min(settings.RetryTimes, dto.MaxChannelRetryTimes),
		Interval:    time.Duration(min(max(intervalMs, 0), dto.MaxChannelRetryIntervalMs)) * time.Millisecond,
		statusCodes: ranges,
	}
}

// RetriesUpstreamStatus reports whether an upstream answer with this HTTP
// status is retried on the same channel. 0 (no upstream status) never is.
func (p ChannelRetryPolicy) RetriesUpstreamStatus(statusCode int) bool {
	return operation_setting.MatchHTTPStatusCodeRanges(p.statusCodes, statusCode)
}

// WaitBeforeRetry waits Interval before the next same-channel attempt and
// reports false when ctx ends first, i.e. the client went away.
func (p ChannelRetryPolicy) WaitBeforeRetry(ctx context.Context) bool {
	if p.Interval <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(p.Interval)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
