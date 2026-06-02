package service

import (
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const maxCacheBillingRatio = 10

// EffectiveCacheReadBillingRatio returns the multiplier for cache read tokens (1 = no change).
func EffectiveCacheReadBillingRatio(setting dto.ChannelSettings) float64 {
	if !setting.CacheBillingRatioEnabled {
		return 1
	}
	if setting.CacheBillingRatio <= 0 || setting.CacheBillingRatio > maxCacheBillingRatio {
		return 1
	}
	return setting.CacheBillingRatio
}

func scaleCacheReadTokenCount(tokens int, ratio float64) int {
	if tokens <= 0 || ratio == 1 {
		return tokens
	}
	return int(math.Round(float64(tokens) * ratio))
}

// ApplyCacheReadBillingRatioToUsage scales cache read token fields only.
func ApplyCacheReadBillingRatioToUsage(usage *dto.Usage, ratio float64) {
	if usage == nil || ratio == 1 {
		return
	}
	usage.PromptTokensDetails.CachedTokens = scaleCacheReadTokenCount(usage.PromptTokensDetails.CachedTokens, ratio)
	usage.PromptCacheHitTokens = scaleCacheReadTokenCount(usage.PromptCacheHitTokens, ratio)
	if usage.InputTokensDetails != nil {
		usage.InputTokensDetails.CachedTokens = scaleCacheReadTokenCount(usage.InputTokensDetails.CachedTokens, ratio)
	}
}

var cacheReadBillingJSONPaths = []string{
	"usage.prompt_tokens_details.cached_tokens",
	"usage.cached_tokens",
	"usage.prompt_cache_hit_tokens",
	"usage.input_tokens_details.cached_tokens",
	"usage.cache_read_input_tokens",
	"message.usage.cache_read_input_tokens",
	"response.usage.cache_read_input_tokens",
	"response.usage.input_tokens_details.cached_tokens",
	"usageMetadata.cachedContentTokenCount",
}

// PatchCacheReadBillingRatioInJSON updates cache read token fields in downstream response JSON.
func PatchCacheReadBillingRatioInJSON(body []byte, ratio float64) ([]byte, error) {
	if len(body) == 0 || ratio == 1 {
		return body, nil
	}
	out := body
	var err error
	for _, path := range cacheReadBillingJSONPaths {
		out, err = patchNumericPathAt(out, path, ratio)
		if err != nil {
			return body, err
		}
	}
	out, err = patchChoicesUsageCachedTokens(out, ratio)
	if err != nil {
		return body, err
	}
	return out, nil
}

func patchNumericPathAt(body []byte, path string, ratio float64) ([]byte, error) {
	result := gjson.GetBytes(body, path)
	if !result.Exists() || result.Type != gjson.Number {
		return body, nil
	}
	scaled := scaleCacheReadTokenCount(int(result.Int()), ratio)
	return sjson.SetBytes(body, path, scaled)
}

func patchChoicesUsageCachedTokens(body []byte, ratio float64) ([]byte, error) {
	choices := gjson.GetBytes(body, "choices")
	if !choices.IsArray() {
		return body, nil
	}
	out := body
	var err error
	for i := range choices.Array() {
		path := fmt.Sprintf("choices.%d.usage.cached_tokens", i)
		item := gjson.GetBytes(out, path)
		if !item.Exists() || item.Type != gjson.Number {
			continue
		}
		scaled := scaleCacheReadTokenCount(int(item.Int()), ratio)
		out, err = sjson.SetBytes(out, path, scaled)
		if err != nil {
			return body, err
		}
	}
	return out, nil
}

// ApplyCacheReadBillingRatioWithSetting scales usage and patches body using channel settings directly.
func ApplyCacheReadBillingRatioWithSetting(setting dto.ChannelSettings, usage *dto.Usage, body *[]byte) {
	if usage == nil {
		return
	}
	ratio := EffectiveCacheReadBillingRatio(setting)
	if ratio == 1 {
		return
	}
	ApplyCacheReadBillingRatioToUsage(usage, ratio)
	if body == nil || len(*body) == 0 {
		return
	}
	patched, err := PatchCacheReadBillingRatioInJSON(*body, ratio)
	if err != nil {
		return
	}
	*body = patched
}

// ApplyChannelCacheReadBillingRatio reads channel setting, scales usage, and patches response body when provided.
func ApplyChannelCacheReadBillingRatio(info *relaycommon.RelayInfo, usage *dto.Usage, body *[]byte) {
	if info == nil {
		return
	}
	ApplyCacheReadBillingRatioWithSetting(info.ChannelSetting, usage, body)
}
