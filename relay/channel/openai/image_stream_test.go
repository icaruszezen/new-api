package openai

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func buildImageSSE(events ...string) string {
	var b strings.Builder
	for _, event := range events {
		fmt.Fprintf(&b, "data: %s\n", event)
	}
	b.WriteString("data: [DONE]\n")
	return b.String()
}

func TestConsumeImageGenerationStreamSingleCompleted(t *testing.T) {
	body := buildImageSSE(`{"type":"image_generation.completed","b64_json":"final-image","usage":{"total_tokens":10,"prompt_tokens":4,"completion_tokens":6}}`)
	images, usage, err := consumeImageGenerationStream(nil, strings.NewReader(body))
	if err != nil {
		t.Fatalf("consumeImageGenerationStream returned error: %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}
	if images[0].B64Json != "final-image" {
		t.Fatalf("unexpected b64_json: %q", images[0].B64Json)
	}
	if usage == nil || usage.TotalTokens != 10 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}

func TestConsumeImageGenerationStreamMultipleCompleted(t *testing.T) {
	body := buildImageSSE(
		`{"type":"image_generation.completed","b64_json":"image-1"}`,
		`{"type":"image_generation.completed","b64_json":"image-2","usage":{"total_tokens":2,"prompt_tokens":1,"completion_tokens":1}}`,
	)
	images, usage, err := consumeImageGenerationStream(nil, strings.NewReader(body))
	if err != nil {
		t.Fatalf("consumeImageGenerationStream returned error: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}
	if images[0].B64Json != "image-1" || images[1].B64Json != "image-2" {
		t.Fatalf("unexpected images: %+v", images)
	}
	if usage == nil || usage.TotalTokens != 2 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}

func TestConsumeImageGenerationStreamIgnoresPartialImage(t *testing.T) {
	body := buildImageSSE(
		`{"type":"image_generation.partial_image","b64_json":"partial","partial_image_index":0}`,
		`{"type":"image_generation.completed","b64_json":"final-image"}`,
	)
	images, _, err := consumeImageGenerationStream(nil, strings.NewReader(body))
	if err != nil {
		t.Fatalf("consumeImageGenerationStream returned error: %v", err)
	}
	if len(images) != 1 || images[0].B64Json != "final-image" {
		t.Fatalf("unexpected images: %+v", images)
	}
}

func TestConsumeImageGenerationStreamErrorEvent(t *testing.T) {
	body := buildImageSSE(`{"type":"error","error":{"message":"generation failed","type":"server_error","code":"500"}}`)
	_, _, err := consumeImageGenerationStream(nil, strings.NewReader(body))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestOpenaiImageHandlerFallsBackToNonStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	responseBody := []byte(`{"created":1,"data":[{"b64_json":"sync-image"}],"usage":{"total_tokens":3,"prompt_tokens":1,"completion_tokens":2}}`)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(responseBody)),
	}
	info := &relaycommon.RelayInfo{
		ImageUpstreamStreamSynthesize: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: 1,
		},
	}

	usage, err := OpenaiImageHandler(c, info, resp)
	if err != nil {
		t.Fatalf("OpenaiImageHandler returned error: %v", err)
	}
	if usage == nil || usage.TotalTokens != 3 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
	if !strings.Contains(recorder.Body.String(), "sync-image") {
		t.Fatalf("unexpected response body: %s", recorder.Body.String())
	}
}

func TestOpenaiImageStreamToNonStreamHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	body := buildImageSSE(`{"type":"image_generation.completed","b64_json":"stream-final","usage":{"total_tokens":5,"prompt_tokens":2,"completion_tokens":3}}`)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	info := &relaycommon.RelayInfo{
		ImageUpstreamStreamSynthesize: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: 1,
		},
	}

	usage, err := OpenaiImageStreamToNonStreamHandler(c, info, resp)
	if err != nil {
		t.Fatalf("OpenaiImageStreamToNonStreamHandler returned error: %v", err)
	}
	if usage == nil || usage.TotalTokens != 5 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected application/json response, got %q", recorder.Header().Get("Content-Type"))
	}

	var imageResp dto.ImageResponse
	if unmarshalErr := common.Unmarshal(recorder.Body.Bytes(), &imageResp); unmarshalErr != nil {
		t.Fatalf("failed to unmarshal response: %v", unmarshalErr)
	}
	if len(imageResp.Data) != 1 || imageResp.Data[0].B64Json != "stream-final" {
		t.Fatalf("unexpected image response: %+v", imageResp)
	}
}

func TestParseImageGenerationStreamLine(t *testing.T) {
	event, err := parseImageGenerationStreamLine(`data: {"type":"image_generation.completed","b64_json":"x"}`)
	if err != nil {
		t.Fatalf("parseImageGenerationStreamLine returned error: %v", err)
	}
	if event == nil || event.Type != "image_generation.completed" {
		t.Fatalf("unexpected event: %+v", event)
	}

	if _, err := parseImageGenerationStreamLine("event: ping"); err != nil {
		t.Fatalf("expected nil error for non-data line, got %v", err)
	}
}

func TestOpenaiImageHandlerUsesStreamAggregatorForSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	body := buildImageSSE(`{"type":"image_generation.completed","b64_json":"via-handler"}`)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	info := &relaycommon.RelayInfo{
		ImageUpstreamStreamSynthesize: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: 1,
		},
	}

	_, err := OpenaiImageHandler(c, info, resp)
	if err != nil {
		t.Fatalf("OpenaiImageHandler returned error: %v", err)
	}
	if !strings.Contains(recorder.Body.String(), "via-handler") {
		t.Fatalf("unexpected response body: %s", recorder.Body.String())
	}
}

func TestConsumeImageGenerationStreamMissingCompleted(t *testing.T) {
	body := `data: {"type":"image_generation.partial_image","b64_json":"partial"}` + "\n"
	_, _, err := consumeImageGenerationStream(nil, strings.NewReader(body))
	if err == nil {
		t.Fatal("expected error for missing completed event")
	}
	if err.GetErrorCode() != types.ErrorCodeBadResponseBody {
		t.Fatalf("unexpected error code: %s", err.GetErrorCode())
	}
}
