package palm

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// https://developers.generativeai.google/api/rest/generativelanguage/models/generateMessage#request-body
// https://developers.generativeai.google/api/rest/generativelanguage/models/generateMessage#response-body

func responsePaLM2OpenAI(response *PaLMChatResponse) *dto.OpenAITextResponse {
	fullTextResponse := dto.OpenAITextResponse{
		Choices: make([]dto.OpenAITextResponseChoice, 0, len(response.Candidates)),
	}
	for i, candidate := range response.Candidates {
		choice := dto.OpenAITextResponseChoice{
			Index: i,
			Message: dto.Message{
				Role:    "assistant",
				Content: candidate.Content,
			},
			FinishReason: "stop",
		}
		fullTextResponse.Choices = append(fullTextResponse.Choices, choice)
	}
	return &fullTextResponse
}

func streamResponsePaLM2OpenAI(palmResponse *PaLMChatResponse) *dto.ChatCompletionsStreamResponse {
	var choice dto.ChatCompletionsStreamResponseChoice
	if len(palmResponse.Candidates) > 0 {
		choice.Delta.SetContentString(palmResponse.Candidates[0].Content)
	}
	choice.FinishReason = &constant.FinishReasonStop
	var response dto.ChatCompletionsStreamResponse
	response.Object = "chat.completion.chunk"
	response.Model = "palm2"
	response.Choices = []dto.ChatCompletionsStreamResponseChoice{choice}
	return &response
}

// palmStreamHandler: PaLM 的"流式"上游其实只返回一个 JSON。先同步读完再转成一个
// SSE 块，计费只依赖读到的全文。客户端断开后写入一律跳过；未开启排水时立即关闭
// 上游让读取返回，开启排水时照常读完，但最多再等 STREAMING_TIMEOUT 与
// STREAM_DRAIN_MAX_SECONDS 中较短的一个（整个回复是一次读取，空闲时长即剩余等待时长）。
func palmStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) string {
	responseId := helper.GetResponseID(c)
	createdTime := common.GetTimestamp()
	info.StreamStatus = relaycommon.NewStreamStatus()
	helper.SetEventStreamHeaders(c)
	defer helper.Done(c)

	drainLimit := time.Duration(constant.StreamingTimeout) * time.Second
	drainCutReason := "drain idle timeout"
	if maxDrain := time.Duration(constant.StreamDrainMaxSeconds) * time.Second; maxDrain > 0 && (drainLimit <= 0 || maxDrain < drainLimit) {
		drainLimit = maxDrain
		drainCutReason = "drain max duration reached"
	}
	type clientLeft struct {
		at time.Time
		// drainCut closes the upstream once a drain has waited drainLimit.
		drainCut *time.Timer
	}
	left := make(chan clientLeft, 1)
	stopWatchingClient := context.AfterFunc(c.Request.Context(), func() {
		if !info.DrainUpstreamOnClientDisconnect {
			left <- clientLeft{at: time.Now()}
			_ = resp.Body.Close()
			return
		}
		event := clientLeft{at: time.Now()}
		if drainLimit > 0 {
			event.drainCut = time.AfterFunc(drainLimit, func() { _ = resp.Body.Close() })
		}
		left <- event
	})
	responseBody, err := io.ReadAll(resp.Body)
	clientGone := !stopWatchingClient()
	var gone clientLeft
	if clientGone {
		gone = <-left
	}
	// Stop reports false once the timer has fired, i.e. the drain was cut.
	drainWasCut := gone.drainCut != nil && !gone.drainCut.Stop()
	service.CloseResponseBodyGracefully(resp)
	info.StreamStatus.ClientGone = clientGone
	if err != nil {
		if clientGone && (!info.DrainUpstreamOnClientDisconnect || drainWasCut) {
			info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, c.Request.Context().Err())
			if drainWasCut {
				info.StreamStatus.RecordError(drainCutReason)
			}
			return ""
		}
		common.SysLog("error reading stream response: " + err.Error())
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonScannerErr, err)
		return ""
	}
	info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonEOF, nil)
	if clientGone && info.DrainUpstreamOnClientDisconnect {
		info.StreamStatus.MarkDrained(time.Since(gone.at))
	}

	var palmResponse PaLMChatResponse
	if err := common.Unmarshal(responseBody, &palmResponse); err != nil {
		common.SysLog("error unmarshalling stream response: " + err.Error())
		return ""
	}
	responseText := ""
	if len(palmResponse.Candidates) > 0 {
		responseText = palmResponse.Candidates[0].Content
	}
	fullTextResponse := streamResponsePaLM2OpenAI(&palmResponse)
	fullTextResponse.Id = responseId
	fullTextResponse.Created = createdTime
	_ = helper.ObjectData(c, fullTextResponse)
	return responseText
}

func palmHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)
	var palmResponse PaLMChatResponse
	err = common.Unmarshal(responseBody, &palmResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if palmResponse.Error.Code != 0 || len(palmResponse.Candidates) == 0 {
		return nil, types.WithOpenAIError(types.OpenAIError{
			Message: palmResponse.Error.Message,
			Type:    palmResponse.Error.Status,
			Param:   "",
			Code:    palmResponse.Error.Code,
		}, resp.StatusCode)
	}
	fullTextResponse := responsePaLM2OpenAI(&palmResponse)
	usage := service.ResponseText2Usage(c, palmResponse.Candidates[0].Content, info.UpstreamModelName, info.GetEstimatePromptTokens())
	fullTextResponse.Usage = *usage
	jsonResponse, err := common.Marshal(fullTextResponse)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	service.IOCopyBytesGracefully(c, resp, jsonResponse)
	return usage, nil
}
