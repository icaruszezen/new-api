package common

import (
	"strings"

	"github.com/QuantumNous/new-api/relaykit/relayconvert/reasoning"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

// GeminiUpstreamModelName returns the model name a Gemini channel sends
// upstream. With the Gemini thinking adapter on (and the suffix not preserved
// for originModelName), the -thinking-<budget>, -thinking, -nothinking and
// reasoning-effort suffixes only steer the request and are not part of the
// upstream name. The request URL and the API version choice must both use
// this name, so a version configured for the base model applies to its
// suffixed variants too.
func GeminiUpstreamModelName(originModelName, upstreamModelName string) string {
	if !model_setting.GetGeminiSettings().ThinkingAdapterEnabled ||
		model_setting.ShouldPreserveThinkingSuffix(originModelName) {
		return upstreamModelName
	}
	if base, _, found := strings.Cut(upstreamModelName, "-thinking-"); found {
		return base
	}
	if base, found := strings.CutSuffix(upstreamModelName, "-thinking"); found {
		return base
	}
	if base, found := strings.CutSuffix(upstreamModelName, "-nothinking"); found {
		return base
	}
	if base, level, ok := reasoning.TrimEffortSuffix(upstreamModelName); ok && level != "" {
		return base
	}
	return upstreamModelName
}
