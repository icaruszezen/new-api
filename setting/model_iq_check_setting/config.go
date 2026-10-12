package model_iq_check_setting

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/QuantumNous/new-api/setting/config"
)

// ModelIQCheckSetting 控制顶部导航的「智商检查」入口：开启并填写外部网站地址后，
// 入口跳转到该网站，由它展示模型智商检查。new-api 自身不再对模型发起检查。
type ModelIQCheckSetting struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
}

const (
	EnabledOptionKey = "model_iq_check_setting.enabled"
	URLOptionKey     = "model_iq_check_setting.url"

	// MaxURLLength 限制外部网站地址的长度。
	MaxURLLength = 2048
)

var modelIQCheckSetting = ModelIQCheckSetting{}

func init() {
	config.GlobalConfig.Register("model_iq_check_setting", &modelIQCheckSetting)
}

// GetSetting 返回当前配置的副本。
func GetSetting() ModelIQCheckSetting {
	return modelIQCheckSetting
}

// EntryURL 返回导航入口跳转的外部网站地址；未开启或未填写地址时为空，前端据此隐藏入口。
func EntryURL() string {
	if !modelIQCheckSetting.Enabled {
		return ""
	}
	return modelIQCheckSetting.URL
}

// ValidateURL 校验外部网站地址。地址会作为导航链接展示给所有访客，因此只接受
// http / https 的完整网址，拒绝 javascript: 等可执行脚本的地址；空值表示未填写。
func ValidateURL(value string) error {
	if value == "" {
		return nil
	}
	if len(value) > MaxURLLength {
		return fmt.Errorf("model IQ check url must be at most %d characters", MaxURLLength)
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.New("model IQ check url must be a full http or https address")
	}
	return nil
}
