package helper

import (
	"math/rand"
	"time"

	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// preludeSupportedFormat 报告该下游客户端格式是否支持「流式假首字」prelude。
// 仅 OpenAI chat、Claude messages、OpenAI Responses 三种流式协议受支持。
func preludeSupportedFormat(format types.RelayFormat) bool {
	switch format {
	case types.RelayFormatOpenAI, types.RelayFormatClaude, types.RelayFormatOpenAIResponses:
		return true
	default:
		return false
	}
}

// preludeEnabledForFormat 判断当前请求是否启用「流式假首字」prelude：
// 渠道开启且下游客户端格式为受支持的三种流式协议之一。
func preludeEnabledForFormat(info *relaycommon.RelayInfo) bool {
	if info == nil || info.ChannelMeta == nil {
		return false
	}
	if !info.ChannelSetting.StreamPreludeEnabled {
		return false
	}
	return preludeSupportedFormat(info.RelayFormat)
}

// streamPreludeDelay 计算本次请求的随机延迟，delay = Min + rand(0, Max-Min) 秒。
func streamPreludeDelay(info *relaycommon.RelayInfo) time.Duration {
	minSeconds := info.ChannelSetting.StreamPreludeDelayMinSeconds
	maxSeconds := info.ChannelSetting.StreamPreludeDelayMaxSeconds
	if minSeconds < 0 {
		minSeconds = 0
	}
	if maxSeconds < minSeconds {
		maxSeconds = minSeconds
	}
	delaySeconds := minSeconds
	if maxSeconds > minSeconds {
		delaySeconds += rand.Intn(maxSeconds - minSeconds + 1)
	}
	return time.Duration(delaySeconds) * time.Second
}

// deliverStreamPrelude 在持有共享写锁的前提下按下游协议发送一次 prelude。
// 同步、纯逻辑，便于单测；返回是否实际发送了 prelude。
// 持锁后会二次确认上游尚未开始，避免与上游首条业务内容产生竞态写。
// 发送成功后会标记 FirstResponseTime，使上游自身日志的首字时间反映 prelude 下发时刻。
func deliverStreamPrelude(c *gin.Context, info *relaycommon.RelayInfo) bool {
	if !preludeSupportedFormat(info.RelayFormat) {
		return false
	}

	mutex := info.StreamWriteMutex()
	mutex.Lock()
	defer mutex.Unlock()

	if info.StreamUpstreamStarted() {
		return false
	}
	// 保证每请求最多发送一次 prelude
	if !info.TryMarkStreamPreludeSent() {
		return false
	}

	var err error
	switch info.RelayFormat {
	case types.RelayFormatOpenAI:
		// Chat Completions：发送一个空首字 chunk（delta.role=assistant、content=""）。
		// 客户端可容忍多个 chunk，安全。
		prelude := GenerateStartEmptyResponse(GetResponseID(c), time.Now().Unix(), info.OriginModelName, nil)
		err = ObjectData(c, prelude)
	case types.RelayFormatClaude, types.RelayFormatOpenAIResponses:
		// 无法安全插入空起始事件（会与真实 message_start/response.created 冲突或 ID 对不上），
		// 退化为仅发送 SSE 注释行保活，不发伪造业务事件。
		err = CommentData(c, "")
	}

	if err != nil {
		logger.LogDebug(c, "stream prelude send failed: %s", err.Error())
		return false
	}
	// prelude 是上游真正发给客户端的第一个字节，记入首字时间。
	info.SetFirstResponseTime()
	return true
}
