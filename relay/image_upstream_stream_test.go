package relay

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"

	"github.com/gin-gonic/gin"
)

func TestShouldImageUpstreamStreamSynthesize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	baseInfo := func() *relaycommon.RelayInfo {
		return &relaycommon.RelayInfo{
			RelayMode: relayconstant.RelayModeImagesGenerations,
			ChannelMeta: &relaycommon.ChannelMeta{
				ApiType: constant.APITypeOpenAI,
				ChannelSetting: dto.ChannelSettings{
					ImageNonStreamViaUpstreamStreamEnabled: true,
				},
			},
		}
	}
	baseReq := &dto.ImageRequest{Model: "gpt-image-1", Prompt: "test"}

	if !shouldImageUpstreamStreamSynthesize(c, baseInfo(), baseReq) {
		t.Fatal("expected synthesize to be enabled")
	}

	disabledSetting := baseInfo()
	disabledSetting.ChannelSetting.ImageNonStreamViaUpstreamStreamEnabled = false
	if shouldImageUpstreamStreamSynthesize(c, disabledSetting, baseReq) {
		t.Fatal("expected synthesize to be disabled when channel setting is off")
	}

	passThrough := baseInfo()
	passThrough.ChannelSetting.PassThroughBodyEnabled = true
	if shouldImageUpstreamStreamSynthesize(c, passThrough, baseReq) {
		t.Fatal("expected synthesize to be disabled for pass-through channels")
	}

	aliInfo := baseInfo()
	aliInfo.ApiType = constant.APITypeAli
	if shouldImageUpstreamStreamSynthesize(c, aliInfo, baseReq) {
		t.Fatal("expected synthesize to be disabled for non OpenAI-compatible api type")
	}

	streamReq := &dto.ImageRequest{Model: "gpt-image-1", Prompt: "test", Stream: common.GetPointer(true)}
	if shouldImageUpstreamStreamSynthesize(c, baseInfo(), streamReq) {
		t.Fatal("expected synthesize to be disabled when client requests stream")
	}
}

func TestApplyImageUpstreamStreamSynthesize(t *testing.T) {
	info := &relaycommon.RelayInfo{}
	request := &dto.ImageRequest{Model: "gpt-image-1", Prompt: "test"}

	applyImageUpstreamStreamSynthesize(info, request)

	if !info.ImageUpstreamStreamSynthesize {
		t.Fatal("expected synthesize flag to be set")
	}
	if info.IsStream {
		t.Fatal("expected client IsStream to remain false for upstream-only streaming")
	}
	if request.Stream == nil || !*request.Stream {
		t.Fatal("expected upstream stream to be true")
	}
}
