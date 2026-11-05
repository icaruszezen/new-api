package middleware

import (
	"fmt"
	"net/url"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

const RouteTagKey = "route_tag"

func RouteTag(tag string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(RouteTagKey, tag)
		c.Next()
	}
}

func SetUpLogger(server *gin.Engine) {
	server.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		var requestID string
		if param.Keys != nil {
			requestID, _ = param.Keys[common.RequestIdKey].(string)
		}
		tag, _ := param.Keys[RouteTagKey].(string)
		if tag == "" {
			tag = "web"
		}
		// Gemini-style clients send the API key as ?key=. Redact the log value
		// only; the handler still sees the original query.
		path := param.Path
		if requestURL := param.Request.URL; requestURL.RawQuery != "" {
			// The /jimeng and /kling adapters rewrite the path in place, so log
			// the URL the client sent, as param.Path did before.
			if clientURL, err := url.ParseRequestURI(param.Request.RequestURI); err == nil {
				requestURL = clientURL
			}
			path = requestURL.EscapedPath() + relaycommon.SanitizeURLForLog("?"+requestURL.Query().Encode())
		}
		return fmt.Sprintf("[GIN] %s | %s | %s | %3d | %13v | %15s | %7s %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			tag,
			requestID,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			path,
		)
	}))
}
