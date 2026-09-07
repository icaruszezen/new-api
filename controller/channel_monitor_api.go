package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/channelmonitor"
	"github.com/QuantumNous/new-api/setting/channel_monitoring_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// GetChannelMonitoringStatus 是对外公开的状态页数据接口，无需登录。
// 响应刻意不包含分组名与渠道信息，避免向访客暴露内部路由拓扑。
func GetChannelMonitoringStatus(c *gin.Context) {
	status, err := channelmonitor.Status()
	if err != nil {
		common.ApiErrorMsg(c, "failed to load channel monitoring status")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    status,
	})
}

// adminMonitorView 是管理端的监控项视图，额外带上解析后的图标供表单预览。
type adminMonitorView struct {
	channelmonitor.Monitor
	ResolvedIcon string `json:"resolved_icon"`
}

// GetChannelMonitoringConfig 返回管理端渠道监控配置。
func GetChannelMonitoringConfig(c *gin.Context) {
	monitors := channelmonitor.Monitors()
	views := make([]adminMonitorView, 0, len(monitors))
	for _, monitor := range monitors {
		views = append(views, adminMonitorView{
			Monitor:      monitor,
			ResolvedIcon: resolveMonitorIcon(monitor),
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"enabled":  channel_monitoring_setting.IsEnabled(),
			"monitors": views,
		},
	})
}

// updateChannelMonitoringConfigRequest 用指针区分「未提交」与「提交了零值」，
// 让开关与监控项各自独立更新。
type updateChannelMonitoringConfigRequest struct {
	Enabled  *bool                     `json:"enabled"`
	Monitors *[]channelmonitor.Monitor `json:"monitors"`
}

// UpdateChannelMonitoringConfig 校验并保存管理端提交的渠道监控配置。
// 走独立的管理端接口而不是 root-only 的 /api/option/，
// 以便拥有渠道写权限的管理员就能维护监控配置。
func UpdateChannelMonitoringConfig(c *gin.Context) {
	var request updateChannelMonitoringConfigRequest
	if err := common.UnmarshalBodyReusable(c, &request); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid request body",
		})
		return
	}

	updates := map[string]string{}
	auditParams := map[string]interface{}{}

	if request.Monitors != nil {
		normalized, err := normalizeSubmittedMonitors(*request.Monitors)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		payload, err := common.Marshal(normalized)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		updates[channel_monitoring_setting.MonitorsOptionKey] = string(payload)
		auditParams["monitor_count"] = len(normalized)
	}

	if request.Enabled != nil {
		updates[channel_monitoring_setting.EnabledOptionKey] = common.Interface2String(*request.Enabled)
		auditParams["enabled"] = *request.Enabled
	}

	// 一次事务写入，避免开关已生效但监控项未保存的中间状态。
	if err := model.UpdateOptionsBulk(updates); err != nil {
		common.ApiError(c, err)
		return
	}
	channelmonitor.InvalidateStatusCache()
	recordManageAudit(c, "channel_monitoring.update", auditParams)

	GetChannelMonitoringConfig(c)
}

// ResetChannelMonitor 清空单个监控项的历史数据，配置本身保留。
func ResetChannelMonitor(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "monitor id is required",
		})
		return
	}

	found := false
	for _, monitor := range channelmonitor.Monitors() {
		if monitor.Id == id {
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "monitor not found",
		})
		return
	}

	if err := channelmonitor.ResetMonitorData(id); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "channel_monitoring.reset", map[string]interface{}{
		"monitor_id": id,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// normalizeSubmittedMonitors 校验监控项并补全自动解析出的图标。
func normalizeSubmittedMonitors(submitted []channelmonitor.Monitor) ([]channelmonitor.Monitor, error) {
	groupRatio := ratio_setting.GetGroupRatioCopy()
	groupModels := make(map[string][]string)

	normalized, err := channelmonitor.ValidateMonitors(
		submitted,
		func(group string) bool {
			_, exists := groupRatio[group]
			return exists
		},
		func(group string, modelName string) bool {
			models, cached := groupModels[group]
			if !cached {
				models = model.GetGroupEnabledModels(group)
				groupModels[group] = models
			}
			return common.StringsContains(models, modelName)
		},
	)
	if err != nil {
		return nil, err
	}

	// 图标缺省时按模型自动解析，保证状态页总能画出供应商图标。
	for index := range normalized {
		if normalized[index].Icon == "" {
			normalized[index].Icon = model.ResolveModelIconKey(normalized[index].Model)
		}
	}
	return normalized, nil
}

// GetGroupModels 返回某个分组下已启用的模型，供监控项表单联动使用。
func GetGroupModels(c *gin.Context) {
	group := c.Query("group")
	if group == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "group is required",
		})
		return
	}
	models := model.GetGroupEnabledModels(group)
	if models == nil {
		models = []string{}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    models,
	})
}

func resolveMonitorIcon(monitor channelmonitor.Monitor) string {
	if monitor.Icon != "" {
		return monitor.Icon
	}
	return model.ResolveModelIconKey(monitor.Model)
}
