package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldSkipChannelMonitorSyntheticProbe(t *testing.T) {
	cases := []struct {
		name    string
		channel *model.Channel
		model   string
		skip    bool
	}{
		{name: "chat model", model: "gpt-5", skip: false},
		{name: "embedding by name", model: "text-embedding-3-large", skip: true},
		{name: "bge embedding", model: "bge-m3", skip: true},
		{name: "rerank", model: "jina-rerank-v2", skip: true},
		{name: "moka embeddings channel", channel: &model.Channel{Type: constant.ChannelTypeMokaAI}, model: "m3e-base", skip: true},
		{name: "seedream image", channel: &model.Channel{Type: constant.ChannelTypeVolcEngine}, model: "seedream-4", skip: true},
		{name: "kling video", channel: &model.Channel{Type: constant.ChannelTypeKling}, model: "kling-v1", skip: true},
		{name: "gemini chat still probes", model: "gemini-2.5-pro", skip: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.skip, shouldSkipChannelMonitorSyntheticProbe(testCase.channel, testCase.model))
		})
	}
}

func TestBuildTestRequestHonorsCheapMonitorProbeBudget(t *testing.T) {
	gemini := buildTestRequest("gemini-2.5-pro", "", nil, true, common.GetPointer(uint(16)))
	req, ok := gemini.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.NotNil(t, req.MaxTokens)
	assert.Equal(t, uint(16), *req.MaxTokens)

	native := buildTestRequest("gemini-2.5-pro", string(constant.EndpointTypeGemini), nil, true, common.GetPointer(uint(16)))
	geminiReq, ok := native.(*dto.GeminiChatRequest)
	require.True(t, ok)
	require.NotNil(t, geminiReq.GenerationConfig.MaxOutputTokens)
	assert.Equal(t, uint(16), *geminiReq.GenerationConfig.MaxOutputTokens)

	unbounded := buildTestRequest("gemini-2.5-pro", "", nil, true, nil)
	defaultReq, ok := unbounded.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.NotNil(t, defaultReq.MaxTokens)
	assert.Equal(t, uint(3000), *defaultReq.MaxTokens)
}
