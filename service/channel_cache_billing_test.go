package service

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
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

func TestCacheReadBillingRatioStepsInclusiveHundredths(t *testing.T) {
	require.Equal(t, []float64{0.95, 0.96, 0.97, 0.98, 0.99}, cacheReadBillingRatioSteps(0.95, 0.99))
}

func TestEffectiveCacheReadBillingRatioLegacyFixedWithoutRangeField(t *testing.T) {
	require.Equal(t, 0.8, EffectiveCacheReadBillingRatio(dto.ChannelSettings{
		CacheBillingRatioEnabled: true,
		CacheBillingRatio:        0.8,
	}))
}

func TestEffectiveCacheReadBillingRatioIgnoresMinMaxWhenRangeOff(t *testing.T) {
	require.Equal(t, 0.8, EffectiveCacheReadBillingRatio(dto.ChannelSettings{
		CacheBillingRatioEnabled: true,
		CacheBillingRatio:        0.8,
		CacheBillingRatioMin:     0.95,
		CacheBillingRatioMax:     0.99,
	}))
}

func TestEffectiveCacheReadBillingRatioRange(t *testing.T) {
	require.Equal(t, 0.97, EffectiveCacheReadBillingRatio(dto.ChannelSettings{
		CacheBillingRatioEnabled: true,
		CacheBillingRatioRange:   true,
		CacheBillingRatioMin:     0.97,
		CacheBillingRatioMax:     0.97,
	}))
	require.Equal(t, 1.0, EffectiveCacheReadBillingRatio(dto.ChannelSettings{
		CacheBillingRatioEnabled: true,
		CacheBillingRatioRange:   true,
		CacheBillingRatioMin:     0.99,
		CacheBillingRatioMax:     0.95,
	}))
	require.Equal(t, 1.0, EffectiveCacheReadBillingRatio(dto.ChannelSettings{
		CacheBillingRatioEnabled: true,
		CacheBillingRatioRange:   true,
		CacheBillingRatioMin:     0,
		CacheBillingRatioMax:     1,
	}))
	require.Equal(t, 1.0, EffectiveCacheReadBillingRatio(dto.ChannelSettings{
		CacheBillingRatioEnabled: true,
		CacheBillingRatioRange:   true,
		CacheBillingRatioMin:     0.5,
		CacheBillingRatioMax:     11,
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
		InputTokensDetails:          inputDetails,
		ClaudeCacheCreation5mTokens: 100,
		ClaudeCacheCreation1hTokens: 50,
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

func TestApplyChannelCacheReadBillingRatioPinsRangeSample(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelSetting: dto.ChannelSettings{
				CacheBillingRatioEnabled: true,
				CacheBillingRatioRange:   true,
				CacheBillingRatioMin:     0.95,
				CacheBillingRatioMax:     0.99,
			},
		},
	}
	usage := &dto.Usage{
		PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 10000},
	}

	ApplyChannelCacheReadBillingRatio(info, usage, nil)

	ratio, ok := info.ResolvedCacheReadBillingRatio()
	require.True(t, ok)
	require.Contains(t, cacheReadBillingRatioSteps(0.95, 0.99), ratio)
	first := usage.PromptTokensDetails.CachedTokens
	require.Equal(t, int(ratio*10000+0.5), first)

	usage.PromptTokensDetails.CachedTokens = 10000
	ApplyChannelCacheReadBillingRatio(info, usage, nil)

	second, ok := info.ResolvedCacheReadBillingRatio()
	require.True(t, ok)
	require.Equal(t, ratio, second)
	require.Equal(t, first, usage.PromptTokensDetails.CachedTokens)
}

func TestSnapshotRestoreCacheReadUsage(t *testing.T) {
	inputDetails := &dto.InputTokenDetails{CachedTokens: 600}
	usage := &dto.Usage{
		PromptCacheHitTokens: 800,
		PromptTokensDetails:  dto.InputTokenDetails{CachedTokens: 1000},
		InputTokensDetails:   inputDetails,
	}

	snap := snapshotCacheReadUsage(usage)
	ApplyCacheReadBillingRatioToUsage(usage, 0.5)
	restoreCacheReadUsage(usage, snap)

	require.Equal(t, 1000, usage.PromptTokensDetails.CachedTokens)
	require.Equal(t, 800, usage.PromptCacheHitTokens)
	require.Equal(t, 600, usage.InputTokensDetails.CachedTokens)
}

func TestApplyCacheReadBillingRatioWithSettingRollsBackUsageOnPatchFailure(t *testing.T) {
	usage := &dto.Usage{
		PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 1000},
	}
	body := []byte(`{`)
	setting := dto.ChannelSettings{
		CacheBillingRatioEnabled: true,
		CacheBillingRatio:        0.5,
	}

	ApplyCacheReadBillingRatioWithSetting(setting, usage, &body)

	require.Equal(t, 1000, usage.PromptTokensDetails.CachedTokens)
}

func TestApplyCacheReadBillingRatioWithSettingRatioOneLeavesBodyUnchanged(t *testing.T) {
	usage := &dto.Usage{
		PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 1000},
	}
	body := []byte(`{"usage":{"prompt_tokens_details":{"cached_tokens":1000}}}`)
	original := string(body)
	setting := dto.ChannelSettings{
		CacheBillingRatioEnabled: true,
		CacheBillingRatio:        1,
	}

	ApplyCacheReadBillingRatioWithSetting(setting, usage, &body)

	require.Equal(t, 1000, usage.PromptTokensDetails.CachedTokens)
	require.Equal(t, original, string(body))
}
