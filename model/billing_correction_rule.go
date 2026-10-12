package model

import (
	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// BillingCorrectionRule 是按渠道配置的超大上下文错误计费修正规则。
// 上游在只发心跳、没有任何输出也没有返回计费信息的情况下断流时，new-api 会按本地估算的
// 输入 token 计费；启用规则后，输入超过 PromptTokensThreshold 且输出为 0 的这类请求改为不计费。
// 每个渠道最多一条规则。
//
// 上游同样是 new-api 时，可以填上游的用户 ID 和系统访问令牌，按上游返回的请求 ID 去上游的使用日志里
// 核对这次请求有没有被扣费（实验性）。访问令牌只写不读，接口返回时只告诉前端是否已设置。
type BillingCorrectionRule struct {
	Id                    int    `json:"id"`
	ChannelId             int    `json:"channel_id" gorm:"uniqueIndex"`
	PromptTokensThreshold int    `json:"prompt_tokens_threshold" gorm:"default:0"`
	Enabled               bool   `json:"enabled"`
	UpstreamCheckEnabled  bool   `json:"upstream_check_enabled"`
	UpstreamBaseUrl       string `json:"upstream_base_url" gorm:"type:varchar(512);default:''"`
	UpstreamUserId        int    `json:"upstream_user_id" gorm:"default:0"`
	UpstreamAccessToken   string `json:"-" gorm:"type:varchar(255);default:''"`
	// UpstreamCheckSince 是开启上游核对的时间，后台自动核对只处理这之后的请求。
	UpstreamCheckSince int64 `json:"upstream_check_since" gorm:"bigint;default:0"`
	// AutoRefund 开启后，后台自动核对发现上游没有扣费的 0 输出请求时，直接把本地扣的额度退回用户。
	AutoRefund             bool   `json:"auto_refund"`
	CreatedTime            int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime            int64  `json:"updated_time" gorm:"bigint"`
	ChannelName            string `json:"channel_name,omitempty" gorm:"-"`
	UpstreamAccessTokenSet bool   `json:"upstream_access_token_set" gorm:"-"`
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
		"upstream_check_enabled",
		"upstream_base_url",
		"upstream_user_id",
		"upstream_access_token",
		"upstream_check_since",
		"auto_refund",
		"updated_time",
	).Updates(r).Error
}

func DeleteBillingCorrectionRuleById(id int) error {
	return DB.Delete(&BillingCorrectionRule{}, "id = ?", id).Error
}

// DeleteOrphanBillingCorrectionRules 删除渠道已不存在的规则。删除渠道后必须调用：
// 渠道 ID 可能被新建的渠道复用（例如 SQLite 的 rowid），残留规则会让新渠道
// 平白继承一条免扣费规则。
func DeleteOrphanBillingCorrectionRules(db *gorm.DB) error {
	return db.Where("channel_id NOT IN (?)", db.Model(&Channel{}).Select("id")).
		Delete(&BillingCorrectionRule{}).Error
}

// HasUpstreamCheck 判断规则是否已启用并配置好上游 new-api 核对。
func (r *BillingCorrectionRule) HasUpstreamCheck() bool {
	return r.Enabled && r.UpstreamCheckEnabled && r.UpstreamUserId > 0 && r.UpstreamAccessToken != ""
}

// FillBillingCorrectionRuleChannelNames 为规则补上渠道名和访问令牌是否已设置，供管理页展示。
// 渠道被删除时渠道名留空，前端按渠道 ID 兜底。
func FillBillingCorrectionRuleChannelNames(rules []*BillingCorrectionRule) error {
	if len(rules) == 0 {
		return nil
	}
	channelIds := make([]int, 0, len(rules))
	for _, rule := range rules {
		channelIds = append(channelIds, rule.ChannelId)
		rule.UpstreamAccessTokenSet = rule.UpstreamAccessToken != ""
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
