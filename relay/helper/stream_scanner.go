package helper

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"

	"github.com/gin-gonic/gin"
)

const (
	InitialScannerBufferSize    = 64 << 10  // 64KB (64*1024)
	DefaultMaxScannerBufferSize = 128 << 20 // 64MB (64*1024*1024) default SSE buffer size
	DefaultPingInterval         = 10 * time.Second
	// streamWriteTimeout bounds a single blocked write to a slow client so the
	// unconditional wg.Wait() in cleanup can always finish. Without it, a slow
	// but connected client (full TCP buffer, no server WriteTimeout) could hang
	// the handler forever.
	streamWriteTimeout = 30 * time.Second
)

func getScannerBufferSize() int {
	if constant.StreamScannerMaxBufferMB > 0 {
		return constant.StreamScannerMaxBufferMB << 20
	}
	return DefaultMaxScannerBufferSize
}

func NewStreamScanner(reader io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, InitialScannerBufferSize), getScannerBufferSize())
	return scanner
}

func copyCodexSSEHeaders(c *gin.Context, resp *http.Response) {
	if c == nil || c.Writer == nil || resp == nil {
		return
	}
	// codex
	for _, name := range []string{"X-Reasoning-Included", "X-Codex-Turn-State"} {
		values := resp.Header.Values(name)
		if !service.ShouldCopyUpstreamHeader(c, name, values) {
			continue
		}
		for _, value := range values {
			if value != "" {
				c.Writer.Header().Add(name, value)
			}
		}
	}
}

// ExtendWriteDeadline pushes the connection write deadline forward before each
// stream write. Best-effort: writers that don't support deadlines (e.g.
// httptest recorders) are silently ignored.
func ExtendWriteDeadline(c *gin.Context) {
	if c == nil || c.Writer == nil {
		return
	}
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(streamWriteTimeout))
}

// StreamScannerHandler reads an SSE upstream and hands every `data:` payload
// to dataHandler; `data: [DONE]` ends the stream. When the client disconnects
// it closes the upstream at once, unless info.DrainUpstreamOnClientDisconnect
// is set: then it keeps reading until the upstream ends on its own, so the
// handler still sees the final usage event, bounded by the STREAMING_TIMEOUT
// idle watchdog and STREAM_DRAIN_MAX_SECONDS.
func StreamScannerHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo, dataHandler func(data string, sr *StreamResult)) {
	scanUpstreamStream(c, resp, info, false, dataHandler)
}

// StreamLineScannerHandler is StreamScannerHandler for upstreams that do not
// frame events as SSE `data:` lines (NDJSON, `event:`/`meta:` lines). Every
// non-blank line reaches dataHandler untouched, and no line ends the stream
// by itself: dataHandler calls sr.Done() on the protocol's final event, or
// the stream ends at EOF.
func StreamLineScannerHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo, dataHandler func(line string, sr *StreamResult)) {
	scanUpstreamStream(c, resp, info, true, dataHandler)
}

func scanUpstreamStream(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo, rawLines bool, dataHandler func(data string, sr *StreamResult)) {

	if resp == nil || dataHandler == nil {
		return
	}

	// 无条件新建 StreamStatus
	info.StreamStatus = relaycommon.NewStreamStatus()

	ctx, cancel := context.WithCancel(context.Background())

	streamingTimeout := time.Duration(constant.StreamingTimeout) * time.Second
	maxDrain := time.Duration(constant.StreamDrainMaxSeconds) * time.Second

	var (
		stopChan   = make(chan bool, 3) // 增加缓冲区避免阻塞
		scanner    = NewStreamScanner(resp.Body)
		ticker     = time.NewTicker(streamingTimeout)
		pingTicker *time.Ticker
		// 复用 RelayInfo 上的共享锁，使上游转发、Ping、prelude 共用同一把写锁
		writeMutex  = info.StreamWriteMutex()
		wg          sync.WaitGroup // 用于等待所有 goroutine 退出
		cleanupOnce sync.Once
		stopOnce    sync.Once
	)

	stop := func() {
		stopOnce.Do(func() {
			close(stopChan)
		})
	}

	generalSettings := operation_setting.GetGeneralSetting()
	pingEnabled := generalSettings.PingIntervalEnabled && !info.DisablePing
	pingInterval := time.Duration(generalSettings.PingIntervalSeconds) * time.Second
	if pingInterval <= 0 {
		pingInterval = DefaultPingInterval
	}

	if pingEnabled {
		pingTicker = time.NewTicker(pingInterval)
	}

	logger.LogDebug(c, "relay timeout seconds: %d", common.RelayTimeout)
	logger.LogDebug(c, "relay max idle conns: %d", common.RelayMaxIdleConns)
	logger.LogDebug(c, "relay max idle conns per host: %d", common.RelayMaxIdleConnsPerHost)
	logger.LogDebug(c, "streaming timeout seconds: %d", int64(streamingTimeout.Seconds()))
	logger.LogDebug(c, "ping interval seconds: %d", int64(pingInterval.Seconds()))

	cleanup := func() {
		cleanupOnce.Do(func() {
			cancel()
			stop()
			if resp.Body != nil {
				_ = resp.Body.Close()
			}

			ticker.Stop()
			if pingTicker != nil {
				pingTicker.Stop()
			}

			wg.Wait()
		})
	}
	// Ensure gin.Context is not returned to Gin's pool while any stream goroutine can still use it.
	defer cleanup()

	scanner.Split(bufio.ScanLines)
	copyCodexSSEHeaders(c, resp)
	SetEventStreamHeaders(c)

	ctx = context.WithValue(ctx, "stop_chan", stopChan)

	// Handle ping data sending with improved error handling
	if pingEnabled && pingTicker != nil {
		wg.Add(1)
		gopool.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					logger.LogError(c, fmt.Sprintf("ping goroutine panic: %v", r))
					info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPanic, fmt.Errorf("ping panic: %v", r))
					stop()
				}
				logger.LogDebug(c, "ping goroutine exited")
				wg.Done()
			}()

			// 添加超时保护，防止 goroutine 无限运行
			maxPingDuration := 30 * time.Minute // 最大 ping 持续时间
			pingTimeout := time.NewTimer(maxPingDuration)
			defer pingTimeout.Stop()

			for {
				select {
				case <-pingTicker.C:
					var err error
					func() {
						writeMutex.Lock()
						defer writeMutex.Unlock()
						ExtendWriteDeadline(c)
						err = PingData(c)
					}()
					if err != nil {
						// A ping that fails because the client left must not claim
						// the first-wins end reason: the main loop records the
						// disconnect, and a drain still has to reach its own end.
						if requestContextDone(c) {
							return
						}
						logger.LogError(c, "ping data error: "+err.Error())
						info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPingFail, err)
						return
					}
					logger.LogDebug(c, "ping data sent")
				case <-ctx.Done():
					return
				case <-stopChan:
					return
				case <-c.Request.Context().Done():
					// 监听客户端断开连接
					return
				case <-pingTimeout.C:
					logger.LogError(c, "ping goroutine max duration reached")
					return
				}
			}
		})
	}

	dataChan := make(chan string, 10)

	wg.Add(1)
	gopool.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				logger.LogError(c, fmt.Sprintf("data handler goroutine panic: %v", r))
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPanic, fmt.Errorf("handler panic: %v", r))
			}
			stop()
			wg.Done()
		}()
		sr := newStreamResult(info.StreamStatus)
		for data := range dataChan {
			sr.reset()
			func() {
				writeMutex.Lock()
				defer writeMutex.Unlock()
				ExtendWriteDeadline(c)
				dataHandler(data, sr)
			}()
			if sr.IsStopped() {
				return
			}
		}
	})

	// Scanner goroutine with improved error handling
	wg.Add(1)
	common.RelayCtxGo(ctx, func() {
		defer func() {
			close(dataChan)
			if r := recover(); r != nil {
				logger.LogError(c, fmt.Sprintf("scanner goroutine panic: %v", r))
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPanic, fmt.Errorf("scanner panic: %v", r))
			}
			stop()
			logger.LogDebug(c, "scanner goroutine exited")
			wg.Done()
		}()

		for scanner.Scan() {
			// 检查是否需要停止
			select {
			case <-stopChan:
				return
			case <-ctx.Done():
				return
			default:
			}

			// Any upstream line, comments and event: lines included, proves the
			// upstream is alive; the same watchdog bounds a drain's idle time.
			ticker.Reset(streamingTimeout)
			line := scanner.Text()
			logger.LogDebug(c, "stream scanner data: %s", line)

			var data string
			switch {
			case rawLines:
				if strings.TrimSpace(line) != "" {
					data = line
				}
			case strings.HasPrefix(line, "data:"):
				data = strings.TrimSpace(line[len("data:"):])
			case strings.HasPrefix(line, "[DONE]"):
				data = "[DONE]"
			}
			if data == "" {
				continue
			}
			if !rawLines && strings.HasPrefix(data, "[DONE]") {
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonDone, nil)
				logger.LogDebug(c, "received [DONE], stopping scanner")
				return
			}

			info.SetFirstResponseTime()
			info.ReceivedResponseCount++

			select {
			case dataChan <- data:
			case <-ctx.Done():
				return
			case <-stopChan:
				return
			}
		}

		if err := scanner.Err(); err != nil {
			if err != io.EOF {
				logger.LogError(c, "scanner error: "+err.Error())
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonScannerErr, err)
			}
		}
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonEOF, nil)
	})

	// 主循环等待完成或超时。客户端断开：默认立即 cleanup 关闭上游 resp.Body，
	// 解除 scanner 阻塞并让上游停止生成。DrainUpstreamOnClientDisconnect 时不关，
	// 继续读到上游自己结束（[DONE]/EOF/错误），期间仍受同一个空闲看门狗约束，
	// 并以 STREAM_DRAIN_MAX_SECONDS 封顶；成功的终态在所有 goroutine 退出后改记为 drained。
	var (
		clientGone   = c.Request.Context().Done()
		draining     bool
		drainStart   time.Time
		drainTimer   *time.Timer
		drainExpired <-chan time.Time
	)
waitLoop:
	for {
		select {
		case <-ticker.C:
			if draining {
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, c.Request.Context().Err())
				info.StreamStatus.RecordError("drain idle timeout")
			} else {
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonTimeout, nil)
			}
			break waitLoop
		case <-stopChan:
			// EndReason already set by the goroutine that triggered stopChan.
			// If the client left in the same instant, select may still pick
			// this case; record the disconnect so the outcome does not depend
			// on which ready case won.
			if clientGone != nil && c.Request.Context().Err() != nil {
				info.StreamStatus.ClientGone = true
				if info.DrainUpstreamOnClientDisconnect {
					draining = true
					drainStart = time.Now()
				}
			}
			break waitLoop
		case <-clientGone:
			info.StreamStatus.ClientGone = true
			if !info.DrainUpstreamOnClientDisconnect {
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, c.Request.Context().Err())
				break waitLoop
			}
			draining = true
			drainStart = time.Now()
			clientGone = nil
			if maxDrain > 0 {
				drainTimer = time.NewTimer(maxDrain)
				drainExpired = drainTimer.C
			}
			logger.LogInfo(c, fmt.Sprintf("stream drain: client disconnected, reading upstream to its end for billing (format=%s, path=%s)", info.RelayFormat, info.RequestURLPath))
		case <-drainExpired:
			info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, c.Request.Context().Err())
			info.StreamStatus.RecordError("drain max duration reached")
			break waitLoop
		}
	}
	if drainTimer != nil {
		drainTimer.Stop()
	}

	cleanup()
	if draining && info.StreamStatus.MarkDrained(time.Since(drainStart)) {
		logger.LogInfo(c, fmt.Sprintf("stream drain: upstream reached %s after client disconnect", info.StreamStatus.DrainedFrom))
	}
	if info.StreamStatus.IsNormalEnd() && !info.StreamStatus.HasErrors() {
		logger.LogInfo(c, fmt.Sprintf("stream ended: %s", info.StreamStatus.Summary()))
	} else {
		logger.LogError(c, fmt.Sprintf("stream ended: %s, received=%d", info.StreamStatus.Summary(), info.ReceivedResponseCount))
	}
}
