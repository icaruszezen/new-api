package service

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	maxCacheBillingRatio       = 10
	cacheBillingRatioStepCenti = 100
)

// Cache read billing scales only cache-read token fields in the final usage object used for
// settlement. Stream handlers may forward intermediate SSE chunks with upstream cache-read
// values until the terminal event (e.g. OpenAI stream last chunk, Responses response.completed,
// Claude message_start/message_delta, Gemini final usage chunk).

// EffectiveCacheReadBillingRatio returns the multiplier for cache read tokens (1 = no change).
// Fixed mode (default, including legacy settings without a range flag) uses CacheBillingRatio.
// Range mode samples once per call from [min, max] in 0.01 steps; callers that apply the
// ratio more than once on the same request must pin the result themselves.
func EffectiveCacheReadBillingRatio(setting dto.ChannelSettings) float64 {
	if !setting.CacheBillingRatioEnabled {
		return 1
	}
	if setting.CacheBillingRatioRange {
		return sampleCacheReadBillingRatio(setting.CacheBillingRatioMin, setting.CacheBillingRatioMax)
	}
	if setting.CacheBillingRatio <= 0 || setting.CacheBillingRatio > maxCacheBillingRatio {
		return 1
	}
	return setting.CacheBillingRatio
}

func cacheReadBillingRatioCents(min, max float64) (minCents int, maxCents int, ok bool) {
	if min <= 0 || max <= 0 || min > maxCacheBillingRatio || max > maxCacheBillingRatio {
		return 0, 0, false
	}
	minCents = int(math.Round(min * float64(cacheBillingRatioStepCenti)))
	maxCents = int(math.Round(max * float64(cacheBillingRatioStepCenti)))
	if minCents <= 0 || maxCents <= 0 {
		return 0, 0, false
	}
	if minCents > maxCacheBillingRatio*cacheBillingRatioStepCenti || maxCents > maxCacheBillingRatio*cacheBillingRatioStepCenti {
		return 0, 0, false
	}
	if minCents > maxCents {
		return 0, 0, false
	}
	return minCents, maxCents, true
}

func cacheReadBillingRatioFromCents(cents int) float64 {
	return float64(cents) / float64(cacheBillingRatioStepCenti)
}

func cacheReadBillingRatioSteps(min, max float64) []float64 {
	minCents, maxCents, ok := cacheReadBillingRatioCents(min, max)
	if !ok {
		return nil
	}
	steps := make([]float64, 0, maxCents-minCents+1)
	for cents := minCents; cents <= maxCents; cents++ {
		steps = append(steps, cacheReadBillingRatioFromCents(cents))
	}
	return steps
}

func sampleCacheReadBillingRatio(min, max float64) float64 {
	minCents, maxCents, ok := cacheReadBillingRatioCents(min, max)
	if !ok {
		return 1
	}
	if minCents == maxCents {
		return cacheReadBillingRatioFromCents(minCents)
	}
	picked := minCents + rand.Intn(maxCents-minCents+1)
	return cacheReadBillingRatioFromCents(picked)
}

func scaleCacheReadTokenCount(tokens int, ratio float64) int {
	if tokens <= 0 || ratio == 1 {
		return tokens
	}
	return int(math.Round(float64(tokens) * ratio))
}

type cacheReadUsageSnapshot struct {
	cachedTokens      int
	promptCacheHit    int
	inputCachedTokens int
	hasInputCached    bool
}

func snapshotCacheReadUsage(usage *dto.Usage) cacheReadUsageSnapshot {
	snap := cacheReadUsageSnapshot{
		cachedTokens:   usage.PromptTokensDetails.CachedTokens,
		promptCacheHit: usage.PromptCacheHitTokens,
	}
	if usage.InputTokensDetails != nil {
		snap.hasInputCached = true
		snap.inputCachedTokens = usage.InputTokensDetails.CachedTokens
	}
	return snap
}

func restoreCacheReadUsage(usage *dto.Usage, snap cacheReadUsageSnapshot) {
	if usage == nil {
		return
	}
	usage.PromptTokensDetails.CachedTokens = snap.cachedTokens
	usage.PromptCacheHitTokens = snap.promptCacheHit
	if snap.hasInputCached {
		if usage.InputTokensDetails == nil {
			usage.InputTokensDetails = &dto.InputTokenDetails{}
		}
		usage.InputTokensDetails.CachedTokens = snap.inputCachedTokens
	}
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
	if !gjson.ValidBytes(body) {
		return body, fmt.Errorf("invalid JSON for cache read billing patch")
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
	applyCacheReadBillingRatio(EffectiveCacheReadBillingRatio(setting), usage, body)
}

func applyCacheReadBillingRatio(ratio float64, usage *dto.Usage, body *[]byte) {
	if usage == nil {
		return
	}
	if ratio == 1 {
		return
	}
	snap := snapshotCacheReadUsage(usage)
	ApplyCacheReadBillingRatioToUsage(usage, ratio)
	if body == nil || len(*body) == 0 {
		return
	}
	patched, err := PatchCacheReadBillingRatioInJSON(*body, ratio)
	if err != nil {
		restoreCacheReadUsage(usage, snap)
		common.SysLog(fmt.Sprintf("cache read billing JSON patch failed, usage rolled back: %v", err))
		return
	}
	*body = patched
}

// ApplyChannelCacheReadBillingRatio reads channel setting, scales usage, and patches response body when provided.
// ChannelMeta is nil until InitChannelMeta runs, so requests that reach a response
// handler without channel context (channel tests, converter unit paths) keep upstream values.
func ApplyChannelCacheReadBillingRatio(info *relaycommon.RelayInfo, usage *dto.Usage, body *[]byte) {
	if info == nil || info.ChannelMeta == nil {
		return
	}
	ratio, ok := info.ResolvedCacheReadBillingRatio()
	if !ok {
		ratio = EffectiveCacheReadBillingRatio(info.ChannelSetting)
		info.SetResolvedCacheReadBillingRatio(ratio)
	}
	applyCacheReadBillingRatio(ratio, usage, body)
}
