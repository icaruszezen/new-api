package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/QuantumNous/new-api/i18n"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributeReportsRequestBodyFailureCause(t *testing.T) {
	require.NoError(t, i18n.Init())
	gin.SetMode(gin.TestMode)

	reached := func(c *gin.Context) { c.Status(http.StatusNoContent) }
	router := gin.New()
	router.POST("/v1/chat/completions", Distribute(), reached)
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
