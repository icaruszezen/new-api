package openai

import (
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func OaiResponsesHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	// read response body
	var responsesResponse dto.OpenAIResponsesResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	err = common.Unmarshal(responseBody, &responsesResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := responsesResponse.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}

	info.ObserveResponseModel(responsesResponse.Model)
	responseBody = rewriteSGLangResponsesCreatedAt(info, responseBody, "created_at", responsesResponse.CreatedAt)

	// compute usage
	usage := &dto.Usage{}
	service.ApplyResponsesUsage(usage, responsesResponse.Usage)
	service.ApplyChannelCacheReadBillingRatio(info, usage, &responseBody)

	// 写入新的 response body
	service.IOCopyBytesGracefully(c, resp, responseBody)

	// Count actual tool invocations from Output (not tool declarations).
	for _, output := range responsesResponse.Output {
		switch output.Type {
		case dto.BuildInCallWebSearchCall:
			info.CountBillableToolCall(dto.BuildInCallWebSearchCall, "")
		case dto.BuildInCallFileSearchCall:
			info.CountBillableToolCall(dto.BuildInCallFileSearchCall, "")
		case dto.BuildInCallFunctionCall:
			info.CountBillableToolCall(dto.BuildInCallFunctionCall, output.Name)
		}
	}
	info.ApplyVendorToolUsage(responseBody)

	imageCounter := &relaycommon.ImageGenerationCallCounter{}
	for i := range responsesResponse.Output {
		imageCounter.Observe(&responsesResponse.Output[i], &i)
	}
	imageCounter.Commit(info)

	return usage, nil
}

func OaiResponsesStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		logger.LogError(c, common.LogText("invalid response or response body"))
		return nil, types.NewError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse)
	}

	defer service.CloseResponseBodyGracefully(resp)

	info.ApplyResponsesClientDisconnectDrain()

	accumulator := service.NewResponsesUsageAccumulator(info)
	// upstreamUsage keeps only the token counts upstream reported, so the
	// first-token error handling below can tell them from Finish's estimates.
	upstreamUsage := &dto.Usage{}
	var firstTokenScan responsesFirstTokenScan

	helper.StreamScannerHandler(c, resp, info, func(data string, sr *helper.StreamResult) {

		// 检查当前数据是否包含 completed 状态和 usage 信息
		var streamResponse dto.ResponsesStreamResponse
		if err := common.UnmarshalJsonStr(data, &streamResponse); err != nil {
			logger.LogError(c, common.LogText("failed to unmarshal stream response: %s", err.Error()))
			sr.Error(err)
			return
		}
		observeResponsesFirstTokenEvent(&firstTokenScan, &streamResponse)
		if streamResponse.Response != nil {
			data = string(rewriteSGLangResponsesCreatedAt(info, []byte(data), "response.created_at", streamResponse.Response.CreatedAt))
		}
		switch streamResponse.Type {
		case "response.completed", "response.done":
			if streamResponse.Response != nil {
				service.ApplyResponsesUsage(upstreamUsage, streamResponse.Response.Usage)
			}
			// Patch the client-visible cache-read usage with the channel billing
			// ratio. The throwaway usage keeps the accumulator's settlement usage
			// unscaled until it is scaled once after Finish with the same pinned ratio.
			bodyBytes := []byte(data)
			service.ApplyChannelCacheReadBillingRatio(info, &dto.Usage{}, &bodyBytes)
			data = string(bodyBytes)
		case "response.failed", "response.incomplete", "response.cancelled", "response.canceled":
			if streamResponse.Response != nil {
				service.ApplyResponsesUsage(upstreamUsage, streamResponse.Response.Usage)
			}
		}
		sendResponsesStreamData(c, streamResponse, data)
		accumulator.Observe(&streamResponse, common.StringToByteSlice(data))
	})

	common.SetContextKey(c, constant.ContextKeyResponseStreamStatus, info.StreamStatus)
	info.StreamStatus.RequireTerminal()

	usage := accumulator.Finish()
	// After an actual upstream error, output estimated from streamed text is not
	// billed unless it is exactly one token, which first-token error correction
	// inspects below.
	if upstreamUsage.CompletionTokens == 0 && firstTokenScan.actualError && usage.CompletionTokens != 1 {
		usage.CompletionTokens = 0
		// Without output, Finish's rule bills the estimated prompt only when
		// upstream did not report an explicit failure.
		if info.StreamStatus.ResponseFailed() {
			usage.PromptTokens = upstreamUsage.PromptTokens
		}
		usage.TotalTokens = usage.PromptTokens
		// Finish also copies the estimate into BillingUsage, which settlement prefers.
		usage.BillingUsage = dto.CloneBillingUsage(upstreamUsage.BillingUsage)
	}
	service.ApplyChannelCacheReadBillingRatio(info, usage, nil)

	if common.FirstTokenErrorCorrectionEnabled {
		endReason := relaycommon.StreamEndReasonNone
		if info.StreamStatus != nil {
			endReason = info.StreamStatus.EndReason
		}
		decision := decideResponsesFirstTokenError(firstTokenScan, usage.CompletionTokens, endReason, common.FirstTokenErrorTreatAllOutputOneEnabled)
		if decision.Hit {
			service.HandleResponsesFirstTokenError(c, info, cloneResponsesUsage(usage), decision.ErrorMessage)
			resetResponsesUsage(usage)
			return usage, types.NewError(fmt.Errorf("%s", decision.ErrorMessage), types.ErrorCodeBadResponse, types.ErrOptionWithSkipRetry())
		}
	}

	return usage, nil
}

func rewriteSGLangResponsesCreatedAt(info *relaycommon.RelayInfo, payload []byte, path string, createdAt dto.IntValue) []byte {
	if info.GetChannelType() != constant.ChannelTypeSGLang {
		return payload
	}
	if !gjson.GetBytes(payload, path).Exists() {
		return payload
	}
	patched, err := sjson.SetBytes(payload, path, int(createdAt))
	if err != nil {
		return payload
	}
	return patched
}
