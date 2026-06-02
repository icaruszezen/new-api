package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestEffectiveCacheReadBillingRatio(t *testing.T) {
	require.Equal(t, 1.0, EffectiveCacheReadBillingRatio(dto.ChannelSettings{}))
	require.Equal(t, 1.0, EffectiveCacheReadBillingRatio(dto.ChannelSettings{
		CacheBillingRatioEnabled: true,
		CacheBillingRatio:        0,
	}))
	require.Equal(t, 1.0, EffectiveCacheReadBillingRatio(dto.ChannelSettings{
		CacheBillingRatioEnabled: true,
		CacheBillingRatio:        11,
	}))
	require.Equal(t, 0.8, EffectiveCacheReadBillingRatio(dto.ChannelSettings{
		CacheBillingRatioEnabled: true,
		CacheBillingRatio:        0.8,
	}))
}

func TestApplyCacheReadBillingRatioToUsage(t *testing.T) {
	inputDetails := &dto.InputTokenDetails{
		CachedTokens:         1000,
		CachedCreationTokens: 500,
	}
	usage := &dto.Usage{
		PromptCacheHitTokens: 2000,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         10000,
			CachedCreationTokens: 500,
		},
		InputTokensDetails:             inputDetails,
		ClaudeCacheCreation5mTokens:    100,
		ClaudeCacheCreation1hTokens:    50,
	}

	ApplyCacheReadBillingRatioToUsage(usage, 0.5)

	require.Equal(t, 5000, usage.PromptTokensDetails.CachedTokens)
	require.Equal(t, 1000, usage.PromptCacheHitTokens)
	require.Equal(t, 500, usage.InputTokensDetails.CachedTokens)
	require.Equal(t, 500, usage.PromptTokensDetails.CachedCreationTokens)
	require.Equal(t, 500, usage.InputTokensDetails.CachedCreationTokens)
	require.Equal(t, 100, usage.ClaudeCacheCreation5mTokens)
	require.Equal(t, 50, usage.ClaudeCacheCreation1hTokens)
}

func TestApplyCacheReadBillingRatioToUsageRatioOneNoOp(t *testing.T) {
	usage := &dto.Usage{
		PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 1234},
	}
	ApplyCacheReadBillingRatioToUsage(usage, 1)
	require.Equal(t, 1234, usage.PromptTokensDetails.CachedTokens)
}

func TestPatchCacheReadBillingRatioInJSON(t *testing.T) {
	body := []byte(`{
		"usage": {
			"prompt_tokens_details": {"cached_tokens": 10000, "cached_creation_tokens": 900},
			"prompt_cache_hit_tokens": 8000,
			"cache_read_input_tokens": 7000,
			"input_tokens_details": {"cached_tokens": 6000, "cached_creation_tokens": 500}
		},
		"choices": [{"usage": {"cached_tokens": 4000}}]
	}`)

	patched, err := PatchCacheReadBillingRatioInJSON(body, 0.5)
	require.NoError(t, err)
	require.Contains(t, string(patched), `"cached_tokens": 5000`)
	require.Contains(t, string(patched), `"prompt_cache_hit_tokens": 4000`)
	require.Contains(t, string(patched), `"cache_read_input_tokens": 3500`)
	require.Contains(t, string(patched), `"cached_creation_tokens": 900`)
	require.Contains(t, string(patched), `"cached_creation_tokens": 500`)
	require.Contains(t, string(patched), `"cached_tokens": 2000`)
}

func TestPatchCacheReadBillingRatioInJSONClaudeAndGeminiPaths(t *testing.T) {
	body := []byte(`{
		"message": {"usage": {"cache_read_input_tokens": 1000, "cache_creation_input_tokens": 200}},
		"usageMetadata": {"cachedContentTokenCount": 3000}
	}`)

	patched, err := PatchCacheReadBillingRatioInJSON(body, 0.5)
	require.NoError(t, err)
	require.Contains(t, string(patched), `"cache_read_input_tokens": 500`)
	require.Contains(t, string(patched), `"cache_creation_input_tokens": 200`)
	require.Contains(t, string(patched), `"cachedContentTokenCount": 1500`)
}

func TestApplyChannelCacheReadBillingRatio(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelSetting: dto.ChannelSettings{
				CacheBillingRatioEnabled: true,
				CacheBillingRatio:        0.5,
			},
		},
	}
	usage := &dto.Usage{
		PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 10000},
	}
	body := []byte(`{"usage":{"prompt_tokens_details":{"cached_tokens":10000}}}`)

	ApplyChannelCacheReadBillingRatio(info, usage, &body)

	require.Equal(t, 5000, usage.PromptTokensDetails.CachedTokens)
	require.Contains(t, string(body), `"cached_tokens":5000`)
}
