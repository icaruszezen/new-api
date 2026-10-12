package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 新建、回复、关闭工单分别挂载 TicketWriteRateLimit，但共用同一个按用户计数的窗口。
func TestTicketWriteRateLimitSharesOneBucketPerUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer, _ := useRateLimitMiniRedis(t)

	router := gin.New()
	setUser := func(c *gin.Context) {
		id, err := strconv.Atoi(c.GetHeader("X-Test-User"))
		require.NoError(t, err)
		c.Set("id", id)
	}
	ok := func(c *gin.Context) { c.Status(http.StatusNoContent) }
	for _, path := range []string{"/ticket", "/ticket/reply", "/ticket/close"} {
		router.POST(path, setUser, TicketWriteRateLimit(), ok)
	}
	send := func(path string, userID int) int {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, path, nil)
		request.Header.Set("X-Test-User", strconv.Itoa(userID))
		router.ServeHTTP(recorder, request)
		return recorder.Code
	}

	paths := []string{"/ticket", "/ticket/reply", "/ticket/close"}
	for i := 0; i < ticketWriteRateLimitNum; i++ {
		require.Equal(t, http.StatusNoContent, send(paths[i%len(paths)], 501), "request %d", i+1)
	}
	assert.Equal(t, http.StatusTooManyRequests, send("/ticket/close", 501))
	assert.Equal(t, http.StatusTooManyRequests, send("/ticket", 501))
	assert.Equal(t, http.StatusNoContent, send("/ticket", 502), "other users have their own bucket")

	key := redisUserRateLimitKey("TK", 501)
	assert.Equal(t, 600*time.Second, redisServer.TTL(key))
}
