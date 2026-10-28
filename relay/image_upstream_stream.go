package relay

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"

	"github.com/gin-gonic/gin"
)

func isOpenAICompatibleImageApiType(apiType int) bool {
	switch apiType {
	case constant.APITypeOpenAI,
		constant.APITypeOpenRouter,
		constant.APITypeXinference,
		constant.APITypeVolcEngine,
		constant.APITypeXai:
		return true
	default:
		return false
	}
}

func shouldImageUpstreamStreamSynthesize(c *gin.Context, info *relaycommon.RelayInfo, imageReq *dto.ImageRequest) bool {
	if info == nil || imageReq == nil {
		return false
	}
	if !info.ChannelSetting.ImageNonStreamViaUpstreamStreamEnabled {
		return false
	}
	// 透传（全局或渠道）原样转发请求体，补不上 stream:true，不能走上游流式聚合
	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		return false
	}
	if !isOpenAICompatibleImageApiType(info.ApiType) {
		return false
	}
	if info.RelayMode != relayconstant.RelayModeImagesGenerations &&
		info.RelayMode != relayconstant.RelayModeImagesEdits {
		return false
	}
	if imageReq.IsStream(c.Request) {
		return false
	}
	return true
}

func applyImageUpstreamStreamSynthesize(info *relaycommon.RelayInfo, request *dto.ImageRequest) {
	request.Stream = common.GetPointer(true)
	info.ImageUpstreamStreamSynthesize = true
}
