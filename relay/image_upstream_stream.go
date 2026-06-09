package relay

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"

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
	if info.ChannelSetting.PassThroughBodyEnabled {
		return false
	}
	if !isOpenAICompatibleImageApiType(info.ApiType) {
		return false
	}
	if info.RelayMode != relayconstant.RelayModeImagesGenerations &&
		info.RelayMode != relayconstant.RelayModeImagesEdits {
		return false
	}
	if imageReq.IsStream(c) {
		return false
	}
	return true
}

func applyImageUpstreamStreamSynthesize(info *relaycommon.RelayInfo, request *dto.ImageRequest) {
	request.Stream = common.GetPointer(true)
	info.ImageUpstreamStreamSynthesize = true
}
