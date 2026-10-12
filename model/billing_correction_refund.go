package model

import (
	"errors"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 退费记录的类型。
const (
	// BillingCorrectionRefundKindMark 管理员标记为已另行退费，不改动余额。
	BillingCorrectionRefundKindMark = "mark"
	// BillingCorrectionRefundKindCredit 管理员手动补费：把这次扣的额度退回了用户钱包。
	BillingCorrectionRefundKindCredit = "credit"
	// BillingCorrectionRefundKindAuto 后台核对发现上游没有扣费，自动把额度退回了用户钱包。
	BillingCorrectionRefundKindAuto = "auto"
)

var ErrBillingCorrectionRefundUserNotFound = errors.New("billing correction refund user not found")

// BillingCorrectionRefund 记录一条消费日志已经退费：要么是管理员标记为另行退费（只是记录，不改余额），
// 要么是补费，额度已经退回用户钱包。每个请求最多一条，同一请求不会重复退费。
// 用 request_id 关联消费日志，因为日志可能存放在独立的日志库里（ClickHouse 日志没有自增 ID），
// 而每条消费日志写入时都会带上 request_id。
type BillingCorrectionRefund struct {
	Id           int    `json:"id"`
	RequestId    string `json:"request_id" gorm:"type:varchar(64);uniqueIndex"`
	UserId       int    `json:"user_id" gorm:"index"`
	ChannelId    int    `json:"channel_id"`
	Quota        int    `json:"quota"`
	Kind         string `json:"kind" gorm:"type:varchar(16);default:'mark'"`
	LogCreatedAt int64  `json:"log_created_at" gorm:"bigint"`
	OperatorId   int    `json:"operator_id"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint"`
}

func (BillingCorrectionRefund) TableName() string {
	return "billing_correction_refunds"
}

// billingCorrectionRequestIdChunk 控制 IN 查询的参数个数，远低于各数据库的占位符上限。
const billingCorrectionRequestIdChunk = 500

// BillingCorrectionLogFilter 是错误计费记录查询的筛选条件，含义与使用日志查询一致。
type BillingCorrectionLogFilter struct {
	StartTimestamp int64
	EndTimestamp   int64
	Username       string
	TokenName      string
	ModelName      string
	ChannelId      int
	RequestIds     []string
	// RuleMatchedOnly 只要符合修正规则条件的记录：输入超过渠道阈值，并且日志里带有本地估算标记、
	// 估算出的计费 usage，或已被规则修正。
	RuleMatchedOnly bool
}

// FindBillingCorrectionCandidateLogs 在启用了规则的渠道上找出输出为 0、并且扣了费或已被规则修正的消费日志。
// thresholds 以渠道 ID 为键；filter.RuleMatchedOnly 时再按阈值和本地估算标记预筛。
// 调用方再按 other 字段做精确判定，这里的 LIKE 只是预筛。最多返回 limit 条，调用方据此判断结果是否被截断。
func FindBillingCorrectionCandidateLogs(thresholds map[int]int, filter BillingCorrectionLogFilter, limit int) ([]*Log, error) {
	if filter.ChannelId != 0 {
		threshold, ok := thresholds[filter.ChannelId]
		if !ok {
			return nil, nil
		}
		thresholds = map[int]int{filter.ChannelId: threshold}
	}
	if len(thresholds) == 0 {
		return nil, nil
	}

	channelIds := make([]int, 0, len(thresholds))
	for channelId := range thresholds {
		channelIds = append(channelIds, channelId)
	}
	sort.Ints(channelIds)

	tx := LOG_DB.Model(&Log{}).
		Select("logs.id, logs.created_at, logs.user_id, logs.username, logs.token_name, logs.token_id, logs.model_name, logs.quota, logs.prompt_tokens, logs.completion_tokens, logs.use_time, logs.is_stream, logs.channel_id, logs.request_id, logs.upstream_request_id, logs.other").
		Where("logs.type = ?", LogTypeConsume).
		Where("logs.completion_tokens = 0").
		Where("(logs.quota > 0 OR logs.other LIKE ?)", `%"billing_corrected":true%`)

	if filter.RuleMatchedOnly {
		ruleConditions := make([]string, 0, len(channelIds))
		ruleArgs := make([]any, 0, len(channelIds)*2)
		for _, channelId := range channelIds {
			ruleConditions = append(ruleConditions, "(logs.channel_id = ? AND logs.prompt_tokens > ?)")
			ruleArgs = append(ruleArgs, channelId, thresholds[channelId])
		}
		tx = tx.Where("("+strings.Join(ruleConditions, " OR ")+")", ruleArgs...).
			Where("(logs.other LIKE ? OR logs.other LIKE ? OR logs.other LIKE ?)",
				`%"billing_corrected":true%`, `%"local_count_tokens":true%`, `%"usage_billing_path":"billing-usage-%-estimated"%`)
	} else {
		tx = tx.Where("logs.channel_id IN ?", channelIds)
	}

	var err error
	if tx, err = applyExplicitLogTextFilter(tx, "logs.username", filter.Username); err != nil {
		return nil, err
	}
	if tx, err = applyExplicitLogTextFilter(tx, "logs.model_name", filter.ModelName); err != nil {
		return nil, err
	}
	if filter.TokenName != "" {
		tx = tx.Where("logs.token_name = ?", filter.TokenName)
	}
	if filter.StartTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", filter.StartTimestamp)
	}
	if filter.EndTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", filter.EndTimestamp)
	}
	if len(filter.RequestIds) > 0 {
		tx = tx.Where("logs.request_id IN ?", filter.RequestIds)
	}

	order := "logs.created_at desc, logs.id desc"
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		order = clickHouseLogOrder("logs.")
	}
	var logs []*Log
	if err := tx.Order(order).Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// GetBillingCorrectionRefundsByRequestIds 返回这些请求已有的退费标记，以 request_id 为键。
func GetBillingCorrectionRefundsByRequestIds(requestIds []string) (map[string]*BillingCorrectionRefund, error) {
	refunds := make(map[string]*BillingCorrectionRefund, len(requestIds))
	for start := 0; start < len(requestIds); start += billingCorrectionRequestIdChunk {
		end := min(start+billingCorrectionRequestIdChunk, len(requestIds))
		var chunk []*BillingCorrectionRefund
		if err := DB.Where("request_id IN ?", requestIds[start:end]).Find(&chunk).Error; err != nil {
			return nil, err
		}
		for _, refund := range chunk {
			refunds[refund.RequestId] = refund
		}
	}
	return refunds, nil
}

// CreateBillingCorrectionRefunds 写入退费标记，已有退费记录的请求保持原样（包括原来的时间、操作人和类型）。
func CreateBillingCorrectionRefunds(refunds []*BillingCorrectionRefund) error {
	if len(refunds) == 0 {
		return nil
	}
	now := common.GetTimestamp()
	for _, refund := range refunds {
		refund.Kind = BillingCorrectionRefundKindMark
		refund.CreatedAt = now
	}
	return DB.Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(refunds, billingCorrectionRequestIdChunk).Error
}

// CreditBillingCorrectionRefund 把一次扣费退回用户钱包，并写下补费记录。补费记录和余额变动在同一个事务里，
// request_id 唯一索引保证同一请求只退一次：已经标记退费或补过费的请求返回 false，余额不变。
// 这是退回当初扣掉的额度，不是新增额度，所以不受钱包上限约束。
func CreditBillingCorrectionRefund(refund *BillingCorrectionRefund) (bool, error) {
	if refund.Quota <= 0 {
		return false, nil
	}
	refund.CreatedAt = common.GetTimestamp()
	credited := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(refund)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		result = tx.Model(&User{}).Where("id = ?", refund.UserId).Update("quota", gorm.Expr("quota + ?", refund.Quota))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrBillingCorrectionRefundUserNotFound
		}
		credited = true
		return nil
	})
	if err != nil || !credited {
		return false, err
	}
	userId, quota := refund.UserId, int64(refund.Quota)
	gopool.Go(func() {
		if err := cacheIncrUserQuota(userId, quota); err != nil {
			common.SysLog("failed to increase user quota cache: " + err.Error())
		}
	})
	return true, nil
}

// DeleteBillingCorrectionRefunds 撤销这些请求的退费标记，返回实际撤销的条数。
// 补费已经把额度退回了用户，撤销不了，这类记录保持原样。
func DeleteBillingCorrectionRefunds(requestIds []string) (int64, error) {
	var deleted int64
	for start := 0; start < len(requestIds); start += billingCorrectionRequestIdChunk {
		end := min(start+billingCorrectionRequestIdChunk, len(requestIds))
		result := DB.Where("request_id IN ? AND kind = ?", requestIds[start:end], BillingCorrectionRefundKindMark).
			Delete(&BillingCorrectionRefund{})
		if result.Error != nil {
			return deleted, result.Error
		}
		deleted += result.RowsAffected
	}
	return deleted, nil
}
