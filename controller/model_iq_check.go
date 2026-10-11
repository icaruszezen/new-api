package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/model_iq_check_setting"

	"github.com/gin-gonic/gin"
)

// GetModelIQCheckConfig 返回智商检查入口的开关与外部网站地址。
func GetModelIQCheckConfig(c *gin.Context) {
	setting := model_iq_check_setting.GetSetting()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"enabled": setting.Enabled,
			"url":     setting.URL,
		},
	})
}

type updateModelIQCheckConfigRequest struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
}

// UpdateModelIQCheckConfig 保存智商检查入口的开关与外部网站地址，开启入口时必须填写地址。
func UpdateModelIQCheckConfig(c *gin.Context) {
	var request updateModelIQCheckConfigRequest
	if err := common.UnmarshalBodyReusable(c, &request); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	entryURL := strings.TrimSpace(request.URL)
	if err := model_iq_check_setting.ValidateURL(entryURL); err != nil {
		common.ApiErrorI18n(c, i18n.MsgModelIQCheckURLInvalid, map[string]any{"Max": model_iq_check_setting.MaxURLLength})
		return
	}
	if request.Enabled && entryURL == "" {
		common.ApiErrorI18n(c, i18n.MsgModelIQCheckURLRequired)
		return
	}

	// 一次事务写入，避免入口已开启但地址未保存的中间状态。
	if err := model.UpdateOptionsBulk(map[string]string{
		model_iq_check_setting.EnabledOptionKey: strconv.FormatBool(request.Enabled),
		model_iq_check_setting.URLOptionKey:     entryURL,
	}); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "model_iq_check.update", map[string]interface{}{
		"enabled": request.Enabled,
		"url":     entryURL,
	})

	GetModelIQCheckConfig(c)
}
