package model

import (
	"gorm.io/gorm/clause"
)

// BillingCorrectionUpstreamCheck 记录一次 0 输出请求在上游 new-api 的扣费核对结果，每个本地请求一条。
// 用本地 request_id 关联消费日志，原因与 BillingCorrectionRefund 相同。
type BillingCorrectionUpstreamCheck struct {
	Id                int    `json:"id"`
	RequestId         string `json:"request_id" gorm:"type:varchar(64);uniqueIndex"`
	ChannelId         int    `json:"channel_id" gorm:"index"`
	UpstreamRequestId string `json:"upstream_request_id" gorm:"type:varchar(128);default:''"`
	Status            string `json:"status" gorm:"type:varchar(16);default:''"`
	UpstreamQuota     int    `json:"upstream_quota" gorm:"default:0"`
	Message           string `json:"message" gorm:"type:varchar(255);default:''"`
	Attempts          int    `json:"attempts" gorm:"default:0"`
	CheckedAt         int64  `json:"checked_at" gorm:"bigint"`
}

func (BillingCorrectionUpstreamCheck) TableName() string {
	return "billing_correction_upstream_checks"
}

// GetBillingCorrectionUpstreamChecksByRequestIds 返回这些请求已有的上游核对结果，以 request_id 为键。
func GetBillingCorrectionUpstreamChecksByRequestIds(requestIds []string) (map[string]*BillingCorrectionUpstreamCheck, error) {
	checks := make(map[string]*BillingCorrectionUpstreamCheck, len(requestIds))
	for start := 0; start < len(requestIds); start += billingCorrectionRequestIdChunk {
		end := min(start+billingCorrectionRequestIdChunk, len(requestIds))
		var chunk []*BillingCorrectionUpstreamCheck
		if err := DB.Where("request_id IN ?", requestIds[start:end]).Find(&chunk).Error; err != nil {
			return nil, err
		}
		for _, check := range chunk {
			checks[check.RequestId] = check
		}
	}
	return checks, nil
}

// SaveBillingCorrectionUpstreamCheck 写入或覆盖一个请求的上游核对结果。
func SaveBillingCorrectionUpstreamCheck(check *BillingCorrectionUpstreamCheck) error {
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "request_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"channel_id", "upstream_request_id", "status", "upstream_quota", "message", "attempts", "checked_at",
		}),
	}).Create(check).Error
}
