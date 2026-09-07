package channelmonitor

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// pingTimeout 单次测量的上限，超时即视为不可测。
	pingTimeout = 5 * time.Second
)

// MeasureEndpointPing 测量上游端点的往返时延（发出请求到收到响应头），单位毫秒；无法测量时返回 0。
//
// 这里刻意不用 TCP 握手计时：本机的透明代理与 TUN 模式会在本地就完成三次握手，渠道
// 配置的 HTTP 代理也只握手到代理，两种情况都会把结果压成 1ms，与实际体验完全脱节。
// 只有等上游真的回了响应头，数字才反映用户感受到的链路时延。client 必须是该渠道
// relay 在用的那一个，这样代理与 TLS 策略和真实流量保持一致。
//
// 只测第一次成功的往返。共享 keep-alive 客户端上再取多次最小值，会被复用连接或
// 本地代理对 HEAD 的缓存秒回压成 1ms，把第一次的真实时延盖掉。
func MeasureEndpointPing(ctx context.Context, client *http.Client, baseURL string) int {
	if client == nil || ctx.Err() != nil {
		return 0
	}
	target, err := pingTargetURL(baseURL)
	if err != nil || target == "" {
		return 0
	}
	elapsed, ok := measureRoundTrip(ctx, client, target)
	if !ok {
		return 0
	}
	return elapsed
}

// measureRoundTrip 计时到响应头到达为止。任意状态码都算一次成功往返：这里只关心
// 链路时延，端点是否接受 GET 并不重要。
func measureRoundTrip(ctx context.Context, client *http.Client, target string) (int, bool) {
	requestCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, target, nil)
	if err != nil {
		return 0, false
	}
	request.Close = true
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("Pragma", "no-cache")
	query := request.URL.Query()
	query.Set("_cm_ping", strconv.FormatInt(time.Now().UnixNano(), 10))
	request.URL.RawQuery = query.Encode()

	start := time.Now()
	response, err := client.Do(request)
	if err != nil {
		return 0, false
	}
	elapsed := int(time.Since(start).Milliseconds())
	_ = response.Body.Close()

	if elapsed <= 0 {
		elapsed = 1
	}
	return elapsed, true
}

// pingTargetURL 归一化渠道 base URL：管理员填写时经常省略协议，缺省按 https 补全。
func pingTargetURL(baseURL string) (string, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return "", nil
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}
	if parsed.Host == "" {
		return "", nil
	}
	return parsed.String(), nil
}
