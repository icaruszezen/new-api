package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/channelmonitor"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/channel_monitoring_setting"

	"github.com/gin-gonic/gin"
)

// channelMonitorProbePath 是探测使用的入口路径，与选渠时的 RequestPath 保持一致。
const channelMonitorProbePath = "/v1/chat/completions"

type channelMonitorProbeSummary struct {
	Monitors int `json:"monitors"`
	Probed   int `json:"probed"`
	Failed   int `json:"failed"`
	Pinged   int `json:"pinged"`
}

// runChannelMonitorProbeTask 每个周期做两件事：
// 为所有启用的监控刷新端点 ping；只对超过探测间隔没有用户流量的监控补发一次合成流式请求。
func runChannelMonitorProbeTask(ctx context.Context, report func(processed, total int)) (channelMonitorProbeSummary, error) {
	summary := channelMonitorProbeSummary{}
	setting := channel_monitoring_setting.GetSetting()
	if !setting.Enabled {
		return summary, nil
	}

	monitors := channelmonitor.EnabledMonitors()
	summary.Monitors = len(monitors)
	if len(monitors) == 0 {
		return summary, nil
	}

	probeUserID, err := resolveChannelTestUserID(nil)
	if err != nil {
		return summary, err
	}

	monitorIds := make([]string, 0, len(monitors))
	for _, monitor := range monitors {
		monitorIds = append(monitorIds, monitor.Id)
	}
	states, err := model.GetChannelMonitorStates(monitorIds)
	if err != nil {
		return summary, err
	}
	lastBeatById := make(map[string]int64, len(states))
	for _, state := range states {
		lastBeatById[state.MonitorId] = state.LastBeatTs
	}

	idleCutoff := time.Now().Unix() - int64(setting.ProbeIntervalSeconds)
	for index, monitor := range monitors {
		if ctx.Err() != nil {
			break
		}

		// 用户流量优先：只有该监控在一个探测周期内没有任何 beat 时才补发合成请求。
		// 未刷盘的热桶也算活跃，避免长采样窗口下对正在使用的模型重复打上游。
		isIdle := lastBeatById[monitor.Id] < idleCutoff &&
			!channelmonitor.HasActivitySince(monitor.Id, idleCutoff)

		channel, selectErr := selectChannelForMonitor(ctx, monitor)
		if selectErr != nil {
			common.SysError(fmt.Sprintf("channel monitor %s has no available channel: %s", monitor.Id, selectErr.Error()))
			if isIdle {
				// 分组内没有可用渠道时用户请求同样会失败，记一次失败 beat。
				channelmonitor.RecordProbeResult(monitor.Id, false, 0, 0)
				summary.Failed++
			}
			if report != nil {
				report(index+1, len(monitors))
			}
			continue
		}

		// A zero ping is recorded too: it means "measured but unreachable", which
		// the status page renders as "--" rather than a stale previous value.
		// 复用该渠道 relay 用的客户端，让测量与真实流量走同一套代理和 TLS 策略。
		pingMs := 0
		channelSettings := channel.GetSetting()
		pingClient, clientErr := service.GetHttpClientWithProxySettings(channelSettings.Proxy, channelSettings)
		if clientErr != nil {
			common.SysError("failed to build channel monitor ping client: " + clientErr.Error())
		} else {
			pingMs = channelmonitor.MeasureEndpointPing(ctx, pingClient, channel.GetBaseURL())
		}
		if pingMs > 0 {
			summary.Pinged++
		}
		if err := model.UpsertChannelMonitorPing(monitor.Id, pingMs, channel.Id, time.Now().Unix()); err != nil {
			common.SysError("failed to persist channel monitor ping: " + err.Error())
		}

		if !isIdle || shouldSkipChannelMonitorSyntheticProbe(channel, monitor.Model) {
			if report != nil {
				report(index+1, len(monitors))
			}
			continue
		}

		result := probeChannelMonitor(ctx, monitor, channel, probeUserID)
		summary.Probed++
		if !result.success {
			summary.Failed++
		}
		channelmonitor.RecordProbeResult(monitor.Id, result.success, result.ttftMs, channel.Id)

		if report != nil {
			report(index+1, len(monitors))
		}
		if common.RequestInterval > 0 {
			time.Sleep(common.RequestInterval)
		}
	}

	return summary, nil
}

type channelMonitorProbeResult struct {
	success bool
	ttftMs  int
}

// probeChannelMonitor 复用渠道测试链路发一次流式请求，不写消费日志、不扣配额。
func probeChannelMonitor(ctx context.Context, monitor channelmonitor.Monitor, channel *model.Channel, probeUserID int) channelMonitorProbeResult {
	result := testChannel(ctx, channel, probeUserID, monitor.Model, "", true, channelTestOptions{
		group:          monitor.Group,
		skipConsumeLog: true,
		maxTokens:      common.GetPointer(uint(16)),
	})
	if result.localErr != nil || result.newAPIError != nil {
		return channelMonitorProbeResult{}
	}
	return channelMonitorProbeResult{success: true, ttftMs: result.ttftMs}
}

// selectChannelForMonitor 按监控项配置的分组走生产选渠逻辑，
// 保证探测命中的渠道与真实用户请求一致。
func selectChannelForMonitor(ctx context.Context, monitor channelmonitor.Monitor) (*model.Channel, error) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(ctx, http.MethodPost, channelMonitorProbePath, nil)
	common.SetContextKey(c, constant.ContextKeyUserGroup, monitor.Group)
	common.SetContextKey(c, constant.ContextKeyUsingGroup, monitor.Group)

	channel, _, err := service.CacheGetRandomSatisfiedChannel(&service.RetryParam{
		Ctx:         c,
		ModelName:   monitor.Model,
		TokenGroup:  monitor.Group,
		RequestPath: channelMonitorProbePath,
		Retry:       common.GetPointer(0),
	})
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, fmt.Errorf("no channel available for group %s model %s", monitor.Group, monitor.Model)
	}
	return channel, nil
}

// shouldSkipChannelMonitorSyntheticProbe 跳过 embedding / 出图 / rerank / 视频类合成请求。
// 这些路径复用渠道测试时会打出高价上游调用；状态条仍靠用户流量和 ping。
func shouldSkipChannelMonitorSyntheticProbe(channel *model.Channel, modelName string) bool {
	modelName = strings.TrimSpace(modelName)
	lower := strings.ToLower(modelName)
	if strings.Contains(lower, "rerank") {
		return true
	}
	if strings.Contains(lower, "embedding") ||
		strings.HasPrefix(modelName, "m3e") ||
		strings.Contains(modelName, "bge-") ||
		strings.Contains(lower, "embed") ||
		(channel != nil && channel.Type == constant.ChannelTypeMokaAI) {
		return true
	}
	if channel != nil && channel.Type == constant.ChannelTypeVolcEngine && strings.Contains(modelName, "seedream") {
		return true
	}
	if channel == nil {
		return false
	}
	switch channel.Type {
	case constant.ChannelTypeMidjourney,
		constant.ChannelTypeMidjourneyPlus,
		constant.ChannelTypeSunoAPI,
		constant.ChannelTypeKling,
		constant.ChannelTypeJimeng,
		constant.ChannelTypeDoubaoVideo,
		constant.ChannelTypeVidu:
		return true
	default:
		return false
	}
}
