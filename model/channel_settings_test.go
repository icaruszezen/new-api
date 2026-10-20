package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelValidateSettingsRejectsInvalidHTTPTransport(t *testing.T) {
	tests := []struct {
		name    string
		setting dto.ChannelSettings
		wantErr string
	}{
		{
			name:    "auto with shards is valid",
			setting: dto.ChannelSettings{HTTPProtocol: "auto", HTTP2ConnectionShards: 4},
		},
		{
			name:    "http1 with shards greater than one rejected",
			setting: dto.ChannelSettings{HTTPProtocol: "http1", HTTP2ConnectionShards: 2},
			wantErr: "http2_connection_shards",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := &Channel{}
			channel.SetSetting(tt.setting)
			err := channel.ValidateSettings()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestChannelValidateSettingsChannelRetry(t *testing.T) {
	intPtr := func(v int) *int { return &v }
	tests := []struct {
		name    string
		setting dto.ChannelSettings
		wantErr string
	}{
		{name: "retry not configured", setting: dto.ChannelSettings{}},
		{name: "retry_times negative", setting: dto.ChannelSettings{RetryTimes: -1}, wantErr: "retry_times must be between 0 and 10"},
		{name: "retry_times at the maximum", setting: dto.ChannelSettings{RetryTimes: 10}},
		{name: "retry_times above the maximum", setting: dto.ChannelSettings{RetryTimes: 11}, wantErr: "retry_times must be between 0 and 10"},
		{name: "retry_status_codes list", setting: dto.ChannelSettings{RetryTimes: 1, RetryStatusCodes: "429,503"}},
		{name: "retry_status_codes range and full-width comma", setting: dto.ChannelSettings{RetryTimes: 1, RetryStatusCodes: "429，500-599"}},
		{name: "retry_status_codes invalid token", setting: dto.ChannelSettings{RetryTimes: 1, RetryStatusCodes: "429,abc"}, wantErr: "invalid retry_status_codes"},
		{name: "retry_status_codes out of range", setting: dto.ChannelSettings{RetryTimes: 1, RetryStatusCodes: "600"}, wantErr: "invalid retry_status_codes"},
		{name: "retry_interval_ms unset", setting: dto.ChannelSettings{RetryTimes: 1, RetryIntervalMs: nil}},
		{name: "retry_interval_ms zero", setting: dto.ChannelSettings{RetryTimes: 1, RetryIntervalMs: intPtr(0)}},
		{name: "retry_interval_ms at the maximum", setting: dto.ChannelSettings{RetryTimes: 1, RetryIntervalMs: intPtr(60000)}},
		{name: "retry_interval_ms above the maximum", setting: dto.ChannelSettings{RetryTimes: 1, RetryIntervalMs: intPtr(60001)}, wantErr: "retry_interval_ms must be between 0 and 60000"},
		{name: "retry_interval_ms negative", setting: dto.ChannelSettings{RetryTimes: 1, RetryIntervalMs: intPtr(-1)}, wantErr: "retry_interval_ms must be between 0 and 60000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := &Channel{}
			channel.SetSetting(tt.setting)
			err := channel.ValidateSettings()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestChannelValidateSettingsCacheBillingRatio(t *testing.T) {
	tests := []struct {
		name    string
		setting dto.ChannelSettings
		wantErr string
	}{
		{
			name: "legacy fixed ratio remains valid",
			setting: dto.ChannelSettings{
				CacheBillingRatioEnabled: true,
				CacheBillingRatio:        0.8,
			},
		},
		{
			name: "range min max valid",
			setting: dto.ChannelSettings{
				CacheBillingRatioEnabled: true,
				CacheBillingRatioRange:   true,
				CacheBillingRatioMin:     0.95,
				CacheBillingRatioMax:     0.99,
			},
		},
		{
			name: "range min greater than max rejected",
			setting: dto.ChannelSettings{
				CacheBillingRatioEnabled: true,
				CacheBillingRatioRange:   true,
				CacheBillingRatioMin:     0.99,
				CacheBillingRatioMax:     0.95,
			},
			wantErr: "cache_billing_ratio_min must not exceed cache_billing_ratio_max",
		},
		{
			name: "range min missing rejected",
			setting: dto.ChannelSettings{
				CacheBillingRatioEnabled: true,
				CacheBillingRatioRange:   true,
				CacheBillingRatioMax:     0.99,
			},
			wantErr: "cache_billing_ratio_min must be greater than 0",
		},
		{
			name: "range max exceeds limit rejected",
			setting: dto.ChannelSettings{
				CacheBillingRatioEnabled: true,
				CacheBillingRatioRange:   true,
				CacheBillingRatioMin:     0.5,
				CacheBillingRatioMax:     11,
			},
			wantErr: "cache_billing_ratio_max must not exceed 10",
		},
		{
			name: "fixed ratio still required when range is off",
			setting: dto.ChannelSettings{
				CacheBillingRatioEnabled: true,
			},
			wantErr: "cache_billing_ratio must be greater than 0 when enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := &Channel{}
			channel.SetSetting(tt.setting)
			err := channel.ValidateSettings()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestAdvancedCustomChannelRequiresModelListRouteOnlyWhenUpdateChecksEnabled(t *testing.T) {
	inferenceRoute := dto.AdvancedCustomRoute{
		IncomingPath: "/v1/chat/completions",
		UpstreamPath: "/v1/chat/completions",
		Converter:    "none",
	}

	tests := []struct {
		name          string
		checksEnabled bool
		routes        []dto.AdvancedCustomRoute
		wantErr       string
	}{
		{
			name:   "legacy channel without discovery route remains valid",
			routes: []dto.AdvancedCustomRoute{inferenceRoute},
		},
		{
			name:          "enabled checks require discovery route",
			checksEnabled: true,
			routes:        []dto.AdvancedCustomRoute{inferenceRoute},
			wantErr:       dto.AdvancedCustomModelListPath,
		},
		{
			name:          "enabled checks accept discovery route",
			checksEnabled: true,
			routes: []dto.AdvancedCustomRoute{
				inferenceRoute,
				{
					IncomingPath: dto.AdvancedCustomModelListPath,
					UpstreamPath: dto.AdvancedCustomModelListPath,
					Converter:    "none",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := &Channel{Type: constant.ChannelTypeAdvancedCustom}
			channel.SetOtherSettings(dto.ChannelOtherSettings{
				UpstreamModelUpdateCheckEnabled: tt.checksEnabled,
				AdvancedCustom: &dto.AdvancedCustomConfig{
					Routes: tt.routes,
				},
			})

			err := channel.ValidateSettings()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
