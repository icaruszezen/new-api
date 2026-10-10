package model

import (
	"github.com/QuantumNous/new-api/common"
)

// BillingCorrectionRule 是按渠道配置的超大上下文错误计费修正规则。
// 上游在只发心跳、没有任何输出也没有返回计费信息的情况下断流时，new-api 会按本地估算的
// 输入 token 计费；启用规则后，输入超过 PromptTokensThreshold 且输出为 0 的这类请求改为不计费。
// 每个渠道最多一条规则。
type BillingCorrectionRule struct {
	Id                    int    `json:"id"`
	ChannelId             int    `json:"channel_id" gorm:"uniqueIndex"`
	PromptTokensThreshold int    `json:"prompt_tokens_threshold" gorm:"default:0"`
	Enabled               bool   `json:"enabled"`
	CreatedTime           int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime           int64  `json:"updated_time" gorm:"bigint"`
	ChannelName           string `json:"channel_name,omitempty" gorm:"-"`
}

func (BillingCorrectionRule) TableName() string {
	return "billing_correction_rules"
}

// GetAllBillingCorrectionRules 返回全部规则（含已禁用的，供管理页展示）。
func GetAllBillingCorrectionRules() ([]*BillingCorrectionRule, error) {
	var rules []*BillingCorrectionRule
	if err := DB.Model(&BillingCorrectionRule{}).Order("id ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func GetBillingCorrectionRuleById(id int) (*BillingCorrectionRule, error) {
	var rule BillingCorrectionRule
	if err := DB.First(&rule, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// IsBillingCorrectionChannelTaken 判断渠道是否已被除 excludeId 以外的规则占用。
func IsBillingCorrectionChannelTaken(channelId int, excludeId int) (bool, error) {
	var count int64
	err := DB.Model(&BillingCorrectionRule{}).
		Where("channel_id = ? AND id <> ?", channelId, excludeId).
		Count(&count).Error
	return count > 0, err
}

func (r *BillingCorrectionRule) Insert() error {
	now := common.GetTimestamp()
	r.CreatedTime = now
	r.UpdatedTime = now
	return DB.Create(r).Error
}

func (r *BillingCorrectionRule) Update() error {
	r.UpdatedTime = common.GetTimestamp()
	return DB.Model(r).Select(
		"channel_id",
		"prompt_tokens_threshold",
		"enabled",
		"updated_time",
	).Updates(r).Error
}

func DeleteBillingCorrectionRuleById(id int) error {
	return DB.Delete(&BillingCorrectionRule{}, "id = ?", id).Error
}

// FillBillingCorrectionRuleChannelNames 为规则补上渠道名，供管理页展示。
// 渠道被删除时留空，前端按渠道 ID 兜底。
func FillBillingCorrectionRuleChannelNames(rules []*BillingCorrectionRule) error {
	if len(rules) == 0 {
		return nil
	}
	channelIds := make([]int, 0, len(rules))
	for _, rule := range rules {
		channelIds = append(channelIds, rule.ChannelId)
	}

	var channels []struct {
		Id   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if err := DB.Table("channels").Select("id, name").Where("id IN ?", channelIds).Find(&channels).Error; err != nil {
		return err
	}
	channelNameById := make(map[int]string, len(channels))
	for _, channel := range channels {
		channelNameById[channel.Id] = channel.Name
	}
	for _, rule := range rules {
		rule.ChannelName = channelNameById[rule.ChannelId]
	}
	return nil
}
