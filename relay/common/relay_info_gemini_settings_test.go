package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/stretchr/testify/assert"
)

func TestRelayInfoConvOptionsUsesNormalizedGeminiSafetySettings(t *testing.T) {
	settings := model_setting.GetGeminiSettings()
	original := settings.SafetySettings
	t.Cleanup(func() {
		settings.SafetySettings = original
	})
	settings.SafetySettings = map[string]string{
		"HARM_CATEGORY_HATE_SPEECH":       "",
		"HARM_CATEGORY_DANGEROUS_CONTENT": "BLOCK_ONLY_HIGH",
	}

	options := (&RelayInfo{}).ConvOptions()

	assert.Equal(t, "OFF", options.Gemini.SafetySetting("HARM_CATEGORY_HATE_SPEECH"))
	assert.Equal(t, "BLOCK_ONLY_HIGH", options.Gemini.SafetySetting("HARM_CATEGORY_DANGEROUS_CONTENT"))
}

// TestRelayInfoConvOptionsGeminiJSONSchemaSupport covers which upstreams get
// parametersJsonSchema and responseJsonSchema: every channel except one whose
// Gemini API version for the model is v1, and Vertex AI regardless of it.
func TestRelayInfoConvOptionsGeminiJSONSchemaSupport(t *testing.T) {
	settings := model_setting.GetGeminiSettings()
	original := settings.VersionSettings
	originalAdapter := settings.ThinkingAdapterEnabled
	t.Cleanup(func() {
		settings.VersionSettings = original
		settings.ThinkingAdapterEnabled = originalAdapter
	})
	settings.VersionSettings = map[string]string{"default": "v1beta", "gemini-v1-model": "v1"}

	tests := []struct {
		name            string
		channelType     int
		model           string
		thinkingAdapter bool
		want            bool
	}{
		{name: "Gemini on the default version", channelType: constant.ChannelTypeGemini, model: "gemini-3.7-flash", want: true},
		{name: "Gemini on v1", channelType: constant.ChannelTypeGemini, model: "gemini-v1-model", want: false},
		{name: "Vertex AI", channelType: constant.ChannelTypeVertexAi, model: "gemini-v1-model", want: true},
		{name: "other channel on v1", channelType: constant.ChannelTypeAdvancedCustom, model: "gemini-v1-model", want: false},
		{name: "other channel on the default version", channelType: constant.ChannelTypeAdvancedCustom, model: "gemini-3.7-flash", want: true},
		// The request URL drops these suffixes before it picks the version.
		{name: "thinking suffix on a v1 model", channelType: constant.ChannelTypeGemini, model: "gemini-v1-model-thinking", thinkingAdapter: true, want: false},
		{name: "thinking budget suffix on a v1 model", channelType: constant.ChannelTypeGemini, model: "gemini-v1-model-thinking-1024", thinkingAdapter: true, want: false},
		{name: "nothinking suffix on a v1 model", channelType: constant.ChannelTypeGemini, model: "gemini-v1-model-nothinking", thinkingAdapter: true, want: false},
		{name: "effort suffix on a v1 model", channelType: constant.ChannelTypeGemini, model: "gemini-v1-model-high", thinkingAdapter: true, want: false},
		{name: "suffix kept when the thinking adapter is off", channelType: constant.ChannelTypeGemini, model: "gemini-v1-model-thinking", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings.ThinkingAdapterEnabled = tt.thinkingAdapter
			info := &RelayInfo{ChannelMeta: &ChannelMeta{ChannelType: tt.channelType}}

			assert.Equal(t, tt.want, info.ConvOptions().Gemini.SupportsJSONSchemaModel(tt.model))
		})
	}
}
