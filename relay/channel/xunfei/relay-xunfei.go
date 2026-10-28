package xunfei

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/samber/lo"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// https://console.xfyun.cn/services/cbm
// https://www.xfyun.cn/doc/spark/Web.html

func requestOpenAI2Xunfei(request dto.GeneralOpenAIRequest, xunfeiAppId string, domain string) *XunfeiChatRequest {
	messages := make([]XunfeiMessage, 0, len(request.Messages))
	shouldCovertSystemMessage := !strings.HasSuffix(request.Model, "3.5")
	for _, message := range request.Messages {
		if message.Role == "system" && shouldCovertSystemMessage {
			messages = append(messages, XunfeiMessage{
				Role:    "user",
				Content: message.StringContent(),
			})
			messages = append(messages, XunfeiMessage{
				Role:    "assistant",
				Content: "Okay",
			})
		} else {
			messages = append(messages, XunfeiMessage{
				Role:    message.Role,
				Content: message.StringContent(),
			})
		}
	}
	xunfeiRequest := XunfeiChatRequest{}
	xunfeiRequest.Header.AppId = xunfeiAppId
	xunfeiRequest.Parameter.Chat.Domain = domain
	xunfeiRequest.Parameter.Chat.Temperature = request.Temperature
	xunfeiRequest.Parameter.Chat.TopK = lo.FromPtrOr(request.N, 0)
	xunfeiRequest.Parameter.Chat.MaxTokens = request.GetMaxTokens()
	xunfeiRequest.Payload.Message.Text = messages
	return &xunfeiRequest
}

func responseXunfei2OpenAI(response *XunfeiChatResponse) *dto.OpenAITextResponse {
	if len(response.Payload.Choices.Text) == 0 {
		response.Payload.Choices.Text = []XunfeiChatResponseTextItem{
			{
				Content: "",
			},
		}
	}
	choice := dto.OpenAITextResponseChoice{
		Index: 0,
		Message: dto.Message{
			Role:    "assistant",
			Content: response.Payload.Choices.Text[0].Content,
		},
		FinishReason: constant.FinishReasonStop,
	}
	fullTextResponse := dto.OpenAITextResponse{
		Object:  "chat.completion",
		Created: common.GetTimestamp(),
		Choices: []dto.OpenAITextResponseChoice{choice},
		Usage:   response.Payload.Usage.Text,
	}
	return &fullTextResponse
}

func streamResponseXunfei2OpenAI(xunfeiResponse *XunfeiChatResponse) *dto.ChatCompletionsStreamResponse {
	if len(xunfeiResponse.Payload.Choices.Text) == 0 {
		xunfeiResponse.Payload.Choices.Text = []XunfeiChatResponseTextItem{
			{
				Content: "",
			},
		}
	}
	var choice dto.ChatCompletionsStreamResponseChoice
	choice.Delta.SetContentString(xunfeiResponse.Payload.Choices.Text[0].Content)
	if xunfeiResponse.Payload.Choices.Status == 2 {
		choice.FinishReason = &constant.FinishReasonStop
	}
	response := dto.ChatCompletionsStreamResponse{
		Object:  "chat.completion.chunk",
		Created: common.GetTimestamp(),
		Model:   "SparkDesk",
		Choices: []dto.ChatCompletionsStreamResponseChoice{choice},
	}
	return &response
}

func buildXunfeiAuthUrl(hostUrl string, apiKey, apiSecret string) string {
	HmacWithShaToBase64 := func(algorithm, data, key string) string {
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write([]byte(data))
		encodeData := mac.Sum(nil)
		return base64.StdEncoding.EncodeToString(encodeData)
	}
	ul, err := url.Parse(hostUrl)
	if err != nil {
		fmt.Println(err)
	}
	date := time.Now().UTC().Format(time.RFC1123)
	signString := []string{"host: " + ul.Host, "date: " + date, "GET " + ul.Path + " HTTP/1.1"}
	sign := strings.Join(signString, "\n")
	sha := HmacWithShaToBase64("hmac-sha256", sign, apiSecret)
	authUrl := fmt.Sprintf("hmac username=\"%s\", algorithm=\"%s\", headers=\"%s\", signature=\"%s\"", apiKey,
		"hmac-sha256", "host date request-line", sha)
	authorization := base64.StdEncoding.EncodeToString([]byte(authUrl))
	v := url.Values{}
	v.Add("host", ul.Host)
	v.Add("date", date)
	v.Add("authorization", authorization)
	callUrl := hostUrl + "?" + v.Encode()
	return callUrl
}

// xunfeiStreamHandler 在当前 goroutine 里逐帧读上游 websocket 并转成 SSE，usage 累加自
// 每帧的 payload.usage（实际只在 status==2 的最后一帧里）。每次读取受 STREAMING_TIMEOUT
// 空闲超时约束。客户端断开：未开启排水时立即关闭 websocket 让阻塞中的读取返回；开启排水时
// 跳过写客户端，继续读到最后一帧，另以 STREAM_DRAIN_MAX_SECONDS 封顶（从读取返回后发现
// 断开时算起，断开时正在进行的那次读取仍只受空闲超时约束）。
func xunfeiStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, conn *websocket.Conn) *dto.Usage {
	defer conn.Close()
	info.StreamStatus = relaycommon.NewStreamStatus()
	helper.SetEventStreamHeaders(c)

	clientCtx := c.Request.Context()
	if !info.DrainUpstreamOnClientDisconnect {
		stopClosingOnDisconnect := context.AfterFunc(clientCtx, func() { _ = conn.Close() })
		defer stopClosingOnDisconnect()
	}
	streamingTimeout := time.Duration(constant.StreamingTimeout) * time.Second
	maxDrain := time.Duration(constant.StreamDrainMaxSeconds) * time.Second

	var (
		usage         dto.Usage
		draining      bool
		drainDeadline time.Time
		drainStart    time.Time
	)
	for {
		readDeadline := time.Now().Add(streamingTimeout)
		if draining && maxDrain > 0 && drainDeadline.Before(readDeadline) {
			readDeadline = drainDeadline
		}
		_ = conn.SetReadDeadline(readDeadline)
		_, msg, err := conn.ReadMessage()

		if clientCtx.Err() != nil && !draining {
			info.StreamStatus.ClientGone = true
			if !info.DrainUpstreamOnClientDisconnect {
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, clientCtx.Err())
				break
			}
			draining = true
			drainStart = time.Now()
			drainDeadline = drainStart.Add(maxDrain)
			logger.LogInfo(c, "stream drain: client disconnected, reading xunfei websocket to its final frame for billing")
		}
		if err != nil {
			var netErr net.Error
			switch {
			case draining && errors.As(err, &netErr) && netErr.Timeout():
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, clientCtx.Err())
				if maxDrain > 0 && !time.Now().Before(drainDeadline) {
					info.StreamStatus.RecordError("drain max duration reached")
				} else {
					info.StreamStatus.RecordError("drain idle timeout")
				}
			case errors.As(err, &netErr) && netErr.Timeout():
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonTimeout, nil)
			case websocket.IsCloseError(err, websocket.CloseNormalClosure):
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonEOF, nil)
			default:
				common.SysLog("error reading stream response: " + err.Error())
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonScannerErr, err)
			}
			break
		}

		var xunfeiResponse XunfeiChatResponse
		if err := common.Unmarshal(msg, &xunfeiResponse); err != nil {
			common.SysLog("error unmarshalling stream response: " + err.Error())
			info.StreamStatus.RecordError(err.Error())
			info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonHandlerStop, err)
			break
		}
		info.SetFirstResponseTime()
		info.ReceivedResponseCount++
		usage.PromptTokens += xunfeiResponse.Payload.Usage.Text.PromptTokens
		usage.CompletionTokens += xunfeiResponse.Payload.Usage.Text.CompletionTokens
		usage.TotalTokens += xunfeiResponse.Payload.Usage.Text.TotalTokens
		helper.ExtendWriteDeadline(c)
		_ = helper.ObjectData(c, streamResponseXunfei2OpenAI(&xunfeiResponse))
		if xunfeiResponse.Payload.Choices.Status == 2 {
			info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonDone, nil)
			break
		}
	}
	if draining {
		info.StreamStatus.MarkDrained(time.Since(drainStart))
	}
	helper.Done(c)
	return &usage
}

func xunfeiHandler(c *gin.Context, conn *websocket.Conn) (*dto.Usage, *types.NewAPIError) {
	defer conn.Close()
	var usage dto.Usage
	var content string
	var xunfeiResponse XunfeiChatResponse
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			common.SysLog("error reading stream response: " + err.Error())
			break
		}
		var response XunfeiChatResponse
		if err := common.Unmarshal(msg, &response); err != nil {
			common.SysLog("error unmarshalling stream response: " + err.Error())
			break
		}
		xunfeiResponse = response
		if len(response.Payload.Choices.Text) > 0 {
			content += response.Payload.Choices.Text[0].Content
			usage.PromptTokens += response.Payload.Usage.Text.PromptTokens
			usage.CompletionTokens += response.Payload.Usage.Text.CompletionTokens
			usage.TotalTokens += response.Payload.Usage.Text.TotalTokens
		}
		if response.Payload.Choices.Status == 2 {
			break
		}
	}
	if len(xunfeiResponse.Payload.Choices.Text) == 0 {
		xunfeiResponse.Payload.Choices.Text = []XunfeiChatResponseTextItem{
			{
				Content: "",
			},
		}
	}
	xunfeiResponse.Payload.Choices.Text[0].Content = content

	response := responseXunfei2OpenAI(&xunfeiResponse)
	jsonResponse, err := common.Marshal(response)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	_, _ = c.Writer.Write(jsonResponse)
	return &usage, nil
}

// xunfeiMakeRequest 建立到星火的 websocket 并发出请求，调用方负责关闭返回的连接。
// 连接不绑定客户端请求的 context，客户端断开后是否继续读由调用方决定。
func xunfeiMakeRequest(textRequest dto.GeneralOpenAIRequest, domain, authUrl, appId string) (*websocket.Conn, error) {
	d := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	conn, _, err := d.Dial(authUrl, nil)
	if err != nil {
		return nil, err
	}

	data := requestOpenAI2Xunfei(textRequest, appId, domain)
	if err := conn.WriteJSON(data); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func apiVersion2domain(apiVersion string) string {
	switch apiVersion {
	case "v1.1":
		return "lite"
	case "v2.1":
		return "generalv2"
	case "v3.1":
		return "generalv3"
	case "v3.5":
		return "generalv3.5"
	case "v4.0":
		return "4.0Ultra"
	}
	return "general" + apiVersion
}

func getXunfeiAuthUrl(c *gin.Context, apiKey string, apiSecret string, modelName string) (string, string) {
	apiVersion := getAPIVersion(c, modelName)
	domain := apiVersion2domain(apiVersion)
	authUrl := buildXunfeiAuthUrl(fmt.Sprintf("wss://spark-api.xf-yun.com/%s/chat", apiVersion), apiKey, apiSecret)
	return domain, authUrl
}

func getAPIVersion(c *gin.Context, modelName string) string {
	query := c.Request.URL.Query()
	apiVersion := query.Get("api-version")
	if apiVersion != "" {
		return apiVersion
	}
	parts := strings.Split(modelName, "-")
	if len(parts) == 2 {
		apiVersion = parts[1]
		return apiVersion

	}
	apiVersion = c.GetString("api_version")
	if apiVersion != "" {
		return apiVersion
	}
	apiVersion = "v1.1"
	common.SysLog("api_version not found, using default: " + apiVersion)
	return apiVersion
}
