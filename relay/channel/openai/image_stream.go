package openai

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const (
	imageStreamInitialBufferSize = 64 << 10
	imageStreamMaxBufferSize     = 64 << 20
)

type imageGenerationStreamEvent struct {
	Type              string             `json:"type"`
	B64Json           string             `json:"b64_json,omitempty"`
	Url               string             `json:"url,omitempty"`
	RevisedPrompt     string             `json:"revised_prompt,omitempty"`
	PartialImageIndex *int               `json:"partial_image_index,omitempty"`
	Usage             *dto.Usage         `json:"usage,omitempty"`
	Error             *types.OpenAIError `json:"error,omitempty"`
}

func imageStreamMaxScannerBuffer() int {
	if constant.StreamScannerMaxBufferMB > 0 {
		return constant.StreamScannerMaxBufferMB << 20
	}
	return imageStreamMaxBufferSize
}

func OpenaiImageStreamToNonStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	if resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("empty upstream response body"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}

	images, usage, streamErr := consumeImageGenerationStream(c, resp.Body)
	if streamErr != nil {
		return nil, streamErr
	}

	imageResponse := dto.ImageResponse{
		Created: time.Now().Unix(),
		Data:    images,
	}
	responseBody, err := common.Marshal(imageResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
	}

	if usage == nil {
		usage = &dto.Usage{}
	}
	service.ApplyChannelCacheReadBillingRatio(info, usage, &responseBody)
	c.Data(resp.StatusCode, "application/json", responseBody)
	return usage, nil
}

func imageStreamIdleTimeout() time.Duration {
	timeoutSec := constant.StreamingTimeout
	if timeoutSec <= 0 {
		timeoutSec = 300
	}
	return time.Duration(timeoutSec) * time.Second
}

func consumeImageGenerationStream(c *gin.Context, body io.Reader) ([]dto.ImageData, *dto.Usage, *types.NewAPIError) {
	streamingTimeout := imageStreamIdleTimeout()
	lineCh := make(chan streamLineResult, 1)

	go func() {
		defer close(lineCh)
		scanner := bufio.NewScanner(body)
		scanner.Buffer(make([]byte, imageStreamInitialBufferSize), imageStreamMaxScannerBuffer())
		for scanner.Scan() {
			lineCh <- streamLineResult{line: scanner.Text()}
		}
		if scanErr := scanner.Err(); scanErr != nil && scanErr != io.EOF {
			lineCh <- streamLineResult{err: scanErr}
		}
	}()

	ticker := time.NewTimer(streamingTimeout)
	defer ticker.Stop()

	var (
		images        []dto.ImageData
		usage         dto.Usage
		gotCompleted  bool
		hasUsage      bool
	)

	for {
		select {
		case <-ticker.C:
			return nil, nil, types.NewOpenAIError(
				fmt.Errorf("upstream image stream idle timeout after %s", streamingTimeout),
				types.ErrorCodeReadResponseBodyFailed,
				http.StatusGatewayTimeout,
			)
		case result, ok := <-lineCh:
			if !ok {
				if !gotCompleted || len(images) == 0 {
					return nil, nil, types.NewOpenAIError(
						fmt.Errorf("upstream image stream ended without completed event"),
						types.ErrorCodeBadResponseBody,
						http.StatusBadGateway,
					)
				}
				if !hasUsage {
					return images, &usage, nil
				}
				return images, &usage, nil
			}
			if result.err != nil {
				return nil, nil, types.NewOpenAIError(result.err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
			}

			ticker.Reset(streamingTimeout)
			event, parseErr := parseImageGenerationStreamLine(result.line)
			if parseErr != nil {
				logger.LogDebug(c, "skip image stream line: %s", parseErr.Error())
				continue
			}
			if event == nil {
				continue
			}

			switch event.Type {
			case "error":
				if event.Error != nil && event.Error.Type != "" {
					return nil, nil, types.WithOpenAIError(*event.Error, http.StatusBadRequest)
				}
				return nil, nil, types.NewOpenAIError(
					fmt.Errorf("upstream image stream returned error event"),
					types.ErrorCodeBadResponseBody,
					http.StatusBadGateway,
				)
			case "image_generation.partial_image":
				continue
			case "image_generation.completed":
				gotCompleted = true
				images = append(images, dto.ImageData{
					B64Json:       event.B64Json,
					Url:           event.Url,
					RevisedPrompt: event.RevisedPrompt,
				})
				if event.Usage != nil {
					usage = *event.Usage
					hasUsage = true
				}
			}
		}
	}
}

type streamLineResult struct {
	line string
	err  error
}

func parseImageGenerationStreamLine(line string) (*imageGenerationStreamEvent, error) {
	if len(line) < 6 || !strings.HasPrefix(line, "data:") {
		return nil, nil
	}
	data := strings.TrimSpace(line[5:])
	if data == "" || data == "[DONE]" {
		return nil, nil
	}

	var event imageGenerationStreamEvent
	if err := common.UnmarshalJsonStr(data, &event); err != nil {
		return nil, err
	}
	if event.Type == "" {
		return nil, fmt.Errorf("missing event type")
	}
	return &event, nil
}
