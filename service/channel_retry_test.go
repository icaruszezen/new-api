package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChannelRetryPolicy(t *testing.T) {
	tests := []struct {
		name         string
		setting      string // channel "setting" JSON; empty means no settings in context
		wantTimes    int
		wantInterval time.Duration
		retried      []int
		notRetried   []int
	}{
		{
			name:       "no channel settings",
			notRetried: []int{429, 503},
		},
		{
			name:       "retry_times unset",
			setting:    `{"retry_status_codes":"429"}`,
			notRetried: []int{429},
		},
		{
			name:       "retry_times zero",
			setting:    `{"retry_times":0,"retry_status_codes":"429"}`,
			notRetried: []int{429},
		},
		{
			name:       "retry_times negative",
			setting:    `{"retry_times":-1}`,
			notRetried: []int{429},
		},
		{
			name:         "defaults to 429 and 503 after 200ms",
			setting:      `{"retry_times":2}`,
			wantTimes:    2,
			wantInterval: 200 * time.Millisecond,
			retried:      []int{429, 503},
			notRetried:   []int{0, 400, 500, 502},
		},
		{
			name:         "blank status codes use the defaults",
			setting:      `{"retry_times":1,"retry_status_codes":"  "}`,
			wantTimes:    1,
			wantInterval: 200 * time.Millisecond,
			retried:      []int{429, 503},
			notRetried:   []int{500},
		},
		{
			name:         "explicit zero interval retries immediately",
			setting:      `{"retry_times":1,"retry_interval_ms":0}`,
			wantTimes:    1,
			wantInterval: 0,
			retried:      []int{429, 503},
		},
		{
			name:         "explicit interval",
			setting:      `{"retry_times":1,"retry_interval_ms":1500}`,
			wantTimes:    1,
			wantInterval: 1500 * time.Millisecond,
			retried:      []int{429},
		},
		{
			name:         "status code range",
			setting:      `{"retry_times":1,"retry_status_codes":"500-599","retry_interval_ms":0}`,
			wantTimes:    1,
			wantInterval: 0,
			retried:      []int{500, 503, 599},
			notRetried:   []int{0, 429, 499},
		},
		{
			name:         "full-width comma separator",
			setting:      `{"retry_times":1,"retry_status_codes":"429，502","retry_interval_ms":0}`,
			wantTimes:    1,
			wantInterval: 0,
			retried:      []int{429, 502},
			notRetried:   []int{503},
		},
		{
			name:       "invalid status codes disable the retry",
			setting:    `{"retry_times":3,"retry_status_codes":"429,abc","retry_interval_ms":0}`,
			notRetried: []int{429},
		},
		{
			name:         "out-of-range values are clamped",
			setting:      `{"retry_times":99,"retry_interval_ms":999999}`,
			wantTimes:    dto.MaxChannelRetryTimes,
			wantInterval: time.Duration(dto.MaxChannelRetryIntervalMs) * time.Millisecond,
			retried:      []int{429},
		},
		{
			name:         "negative interval is clamped to zero",
			setting:      `{"retry_times":1,"retry_interval_ms":-5}`,
			wantTimes:    1,
			wantInterval: 0,
			retried:      []int{429},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			if tt.setting != "" {
				var settings dto.ChannelSettings
				require.NoError(t, common.UnmarshalJsonStr(tt.setting, &settings))
				common.SetContextKey(c, constant.ContextKeyChannelSetting, settings)
			}

			policy := GetChannelRetryPolicy(c)

			assert.Equal(t, tt.wantTimes, policy.Times)
			assert.Equal(t, tt.wantInterval, policy.Interval)
			for _, code := range tt.retried {
				assert.True(t, policy.RetriesUpstreamStatus(code), "status %d should be retried", code)
			}
			for _, code := range tt.notRetried {
				assert.False(t, policy.RetriesUpstreamStatus(code), "status %d should not be retried", code)
			}
		})
	}
}

func TestChannelRetryPolicyWaitBeforeRetryStopsWhenClientLeaves(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name     string
		interval time.Duration
		ctx      context.Context
		want     bool
	}{
		{name: "no wait with live client", interval: 0, ctx: context.Background(), want: true},
		{name: "short wait with live client", interval: time.Millisecond, ctx: context.Background(), want: true},
		{name: "no wait after client left", interval: 0, ctx: canceled, want: false},
		{name: "long wait after client left", interval: time.Hour, ctx: canceled, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := ChannelRetryPolicy{Times: 1, Interval: tt.interval}
			assert.Equal(t, tt.want, policy.WaitBeforeRetry(tt.ctx))
		})
	}
}
