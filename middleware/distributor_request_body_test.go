package middleware

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributeReportsRequestBodyFailureCause(t *testing.T) {
	require.NoError(t, i18n.Init())
	gin.SetMode(gin.TestMode)
	previous := constant.MaxRequestBodyMB
	constant.MaxRequestBodyMB = 1
	t.Cleanup(func() { constant.MaxRequestBodyMB = previous })

	reached := func(c *gin.Context) { c.Status(http.StatusNoContent) }
	router := gin.New()
	router.POST("/v1/chat/completions", Distribute(), reached)
	router.POST("/mj/submit/imagine", Distribute(), reached)
	oversized := `{"model":"gpt-x","prompt":"` + strings.Repeat("a", 1<<20) + `"}`
	tests := []struct {
		name    string
		path    string
		body    io.Reader
		status  int
		message string
	}{
		{
			name:    "invalid JSON is reported once",
			path:    "/v1/chat/completions",
			body:    strings.NewReader(`{"model":"gpt-x",`),
			status:  http.StatusBadRequest,
			message: "Invalid request: invalid JSON request body",
		},
		{
			name:    "body cut off mid-upload",
			path:    "/v1/chat/completions",
			body:    io.MultiReader(strings.NewReader(`{"model":"gpt-x","messages":`), iotest.ErrReader(io.ErrUnexpectedEOF)),
			status:  http.StatusBadRequest,
			message: "Invalid request: unexpected EOF",
		},
		{
			name:    "oversized body on an ordinary path",
			path:    "/v1/chat/completions",
			body:    strings.NewReader(oversized),
			status:  http.StatusRequestEntityTooLarge,
			message: "Invalid request: request body exceeds 1 MB: request body too large",
		},
		{
			name:    "oversized Midjourney body",
			path:    "/mj/submit/imagine",
			body:    strings.NewReader(oversized),
			status:  http.StatusRequestEntityTooLarge,
			message: "Invalid request: request body exceeds 1 MB: request body too large",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, testCase.path, testCase.body)
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			assert.Equal(t, testCase.status, recorder.Code)
			assert.Contains(t, recorder.Body.String(), `"message":"`+testCase.message+` (request id:`)
		})
	}
}

// The Kling adapter swallows a failed body read and lets Distribute read the
// body again. A server request body is closed after the first read, so the
// second reader must get the first cause instead of a closed-body error.
func TestDistributeKeepsFailedBodyReadCauseForLaterReaders(t *testing.T) {
	require.NoError(t, i18n.Init())
	gin.SetMode(gin.TestMode)
	previous := constant.MaxRequestBodyMB
	constant.MaxRequestBodyMB = 1
	t.Cleanup(func() { constant.MaxRequestBodyMB = previous })

	router := gin.New()
	router.POST("/kling/v1/videos/text2video", KlingRequestConvert(), Distribute(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	body := `{"model_name":"kling-v1","prompt":"` + strings.Repeat("a", 1<<20) + `"}`
	raw := fmt.Sprintf("POST /kling/v1/videos/text2video HTTP/1.1\r\nHost: example.com\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
	request, err := http.ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	require.NoError(t, err)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"message":"Invalid request: request body exceeds 1 MB: request body too large (request id:`)
	assert.NotContains(t, recorder.Body.String(), "closed Body")
}
