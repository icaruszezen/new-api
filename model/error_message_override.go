package model

import (
	"github.com/QuantumNous/new-api/common"
)

// ErrorMessageOverride 把命中某段文本的上游报错整体替换为自定义文案。
// ChannelId 为 0 时对全部渠道生效，大于 0 时只对该渠道生效。
// Priority 越小越先匹配，同级按 Id 先后。
type ErrorMessageOverride struct {
	Id                 int    `json:"id"`
	MatchSubstring     string `json:"match_substring" gorm:"type:varchar(512);not null"`
	ReplacementMessage string `json:"replacement_message" gorm:"type:text;not null"`
	ChannelId          int    `json:"channel_id" gorm:"index;default:0"`
	Priority           int    `json:"priority" gorm:"default:0"`
	Enabled            bool   `json:"enabled"`
	CreatedTime        int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime        int64  `json:"updated_time" gorm:"bigint"`
	ChannelName        string `json:"channel_name,omitempty" gorm:"-"`
}

func (ErrorMessageOverride) TableName() string {
	return "error_message_overrides"
}

// GetAllErrorMessageOverrides 按匹配顺序返回全部规则（含已禁用的，供管理页展示）。
func GetAllErrorMessageOverrides() ([]*ErrorMessageOverride, error) {
	var overrides []*ErrorMessageOverride
	err := DB.Model(&ErrorMessageOverride{}).
		Order("priority ASC, id ASC").
		Find(&overrides).Error
	if err != nil {
		return nil, err
	}
	return overrides, nil
}

func GetErrorMessageOverrideById(id int) (*ErrorMessageOverride, error) {
	var override ErrorMessageOverride
	if err := DB.First(&override, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &override, nil
}

func (o *ErrorMessageOverride) Insert() error {
	now := common.GetTimestamp()
	o.CreatedTime = now
	o.UpdatedTime = now
	return DB.Create(o).Error
}

func (o *ErrorMessageOverride) Update() error {
	o.UpdatedTime = common.GetTimestamp()
	return DB.Model(o).Select(
		"match_substring",
		"replacement_message",
		"channel_id",
		"priority",
		"enabled",
		"updated_time",
	).Updates(o).Error
}

func DeleteErrorMessageOverrideById(id int) error {
	return DB.Delete(&ErrorMessageOverride{}, "id = ?", id).Error
}

// FillErrorMessageOverrideChannelNames 为渠道专属规则补上渠道名，供管理页展示。
// 渠道被删除时留空，前端按渠道 ID 兜底。
func FillErrorMessageOverrideChannelNames(overrides []*ErrorMessageOverride) error {
	channelIds := make([]int, 0, len(overrides))
	seen := make(map[int]struct{}, len(overrides))
	for _, override := range overrides {
		if override.ChannelId <= 0 {
			continue
		}
		if _, ok := seen[override.ChannelId]; ok {
			continue
		}
		seen[override.ChannelId] = struct{}{}
		channelIds = append(channelIds, override.ChannelId)
	}
	if len(channelIds) == 0 {
		return nil
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
	for _, override := range overrides {
		override.ChannelName = channelNameById[override.ChannelId]
	}
	return nil
}
