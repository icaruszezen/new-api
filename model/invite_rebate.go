/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"gorm.io/gorm"
)

// 邀请返利的两个档位。每位被邀请人对邀请人最多各触发一次。
const (
	InviteRebateTypeRegister = "register"
	InviteRebateTypeTopup    = "topup"
)

// 被邀请人在邀请页上的状态。
const (
	InviteeStatusRegistered = "registered"
	InviteeStatusRecharged  = "recharged"
)

// InviteRebate 是一次邀请返利的发放记录。(invitee_id, type) 唯一索引是「每位好友
// 各档只发一次」的硬约束，也是并发充值回调的幂等边界：第二次插入会被数据库拒绝，
// 而不是依赖读-判-写。
type InviteRebate struct {
	Id        int     `json:"id"`
	InviterId int     `json:"inviter_id" gorm:"type:int;index"`
	InviteeId int     `json:"invitee_id" gorm:"type:int;uniqueIndex:idx_invite_rebate_invitee_type"`
	Type      string  `json:"type" gorm:"type:varchar(16);uniqueIndex:idx_invite_rebate_invitee_type"`
	Quota     int     `json:"quota" gorm:"type:int"`
	AmountUsd float64 `json:"amount_usd" gorm:"column:amount_usd"`
	CreatedAt int64   `json:"created_at" gorm:"bigint"`
}

func (InviteRebate) TableName() string {
	return "invite_rebates"
}

// InviteeSummary 是邀请页列表的一行，用户名已脱敏。
type InviteeSummary struct {
	Username     string `json:"username"`
	RegisteredAt int64  `json:"registered_at"`
	Status       string `json:"status"`
	RebateQuota  int    `json:"rebate_quota"`
}

// InviteOverview 是邀请页需要的全部数据。
type InviteOverview struct {
	AffCode        string           `json:"aff_code"`
	RegisterAmount float64          `json:"register_amount"`
	TopupAmount    float64          `json:"topup_amount"`
	TopupThreshold float64          `json:"topup_threshold"`
	InviteeCount   int              `json:"invitee_count"`
	RechargedCount int              `json:"recharged_count"`
	RebateQuota    int              `json:"rebate_quota"`
	Invitees       []InviteeSummary `json:"invitees"`
}

// IssueInviteRegisterRebate 在好友注册成功后给邀请人发放注册返利。发放前再读库确认
// 被邀请人的 inviter_id 与参数一致，参数被错传时不发钱；失败不影响注册本身。
func IssueInviteRegisterRebate(inviterId int, inviteeId int) {
	if inviterId <= 0 || inviteeId <= 0 || inviterId == inviteeId {
		return
	}
	if !operation_setting.IsPaymentComplianceConfirmed() {
		return
	}
	var invitee User
	if err := DB.Select("id", "inviter_id").Where("id = ?", inviteeId).First(&invitee).Error; err != nil {
		common.SysError(fmt.Sprintf("failed to load invitee %d for register rebate: %s", inviteeId, err.Error()))
		return
	}
	if invitee.InviterId != inviterId {
		return
	}

	setting := operation_setting.GetInviteRebateSetting()
	quota := operation_setting.InviteRebateQuota(setting.RegisterAmount)
	if quota <= 0 {
		return
	}
	rebate := &InviteRebate{
		InviterId: inviterId,
		InviteeId: inviteeId,
		Type:      InviteRebateTypeRegister,
		Quota:     quota,
		AmountUsd: operation_setting.InviteRebateAmountUSD(setting.RegisterAmount),
	}
	if err := creditInviteRebate(rebate); err != nil {
		common.SysError(fmt.Sprintf("failed to issue invite register rebate to user %d for invitee %d: %s", inviterId, inviteeId, err.Error()))
	}
}

// IssueInviteTopupRebate 在被邀请人一笔充值成功入账后判断累计充值是否达到门槛，达标
// 则给邀请人发放一次充值返利。返利失败只记录日志，绝不回滚已经完成的充值；下一笔
// 充值会重新判定门槛。
func IssueInviteTopupRebate(inviteeId int) {
	setting := operation_setting.GetInviteRebateSetting()
	quota := operation_setting.InviteRebateQuota(setting.TopupAmount)
	thresholdQuota := operation_setting.InviteRebateQuota(setting.TopupThreshold)
	if quota <= 0 || thresholdQuota <= 0 {
		return
	}
	if !operation_setting.IsPaymentComplianceConfirmed() {
		return
	}

	var invitee User
	if err := DB.Select("id", "inviter_id").Where("id = ?", inviteeId).First(&invitee).Error; err != nil {
		common.SysError(fmt.Sprintf("failed to load invitee %d for topup rebate: %s", inviteeId, err.Error()))
		return
	}
	if invitee.InviterId <= 0 || invitee.InviterId == inviteeId {
		return
	}

	issued, err := hasInviteRebate(inviteeId, InviteRebateTypeTopup)
	if err != nil {
		common.SysError(fmt.Sprintf("failed to check invite topup rebate for invitee %d: %s", inviteeId, err.Error()))
		return
	}
	if issued {
		return
	}

	creditedQuota, err := sumSuccessTopUpQuota(inviteeId)
	if err != nil {
		common.SysError(fmt.Sprintf("failed to sum topups for invitee %d: %s", inviteeId, err.Error()))
		return
	}
	if creditedQuota < int64(thresholdQuota) {
		return
	}

	rebate := &InviteRebate{
		InviterId: invitee.InviterId,
		InviteeId: inviteeId,
		Type:      InviteRebateTypeTopup,
		Quota:     quota,
		AmountUsd: operation_setting.InviteRebateAmountUSD(setting.TopupAmount),
	}
	if err := creditInviteRebate(rebate); err != nil {
		common.SysError(fmt.Sprintf("failed to issue invite topup rebate to user %d for invitee %d: %s", invitee.InviterId, inviteeId, err.Error()))
	}
}

// creditInviteRebate 在一个事务里写入发放记录并给邀请人加额度。记录先插入，让唯一
// 索引在并发下裁决谁真正发放；冲突方回滚，不会重复加钱。加额走与充值入账相同的
// int32 天花板：超限或邀请人不存在时整单回滚，唯一索引不会被占用。
func creditInviteRebate(rebate *InviteRebate) error {
	if _, err := topUpQuotaMaxCurrent(rebate.Quota); err != nil {
		return err
	}
	rebate.CreatedAt = common.GetTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(rebate).Error; err != nil {
			return err
		}
		return creditTopUpQuota(tx, rebate.InviterId, rebate.Quota, map[string]interface{}{
			"aff_history": gorm.Expr("aff_history + ?", rebate.Quota),
		})
	})
	if err != nil {
		return err
	}

	syncCreditUserQuotaCache(rebate.InviterId, rebate.Quota, "invite rebate")
	content := fmt.Sprintf("好友注册邀请返利 %s", logger.LogQuota(rebate.Quota))
	if rebate.Type == InviteRebateTypeTopup {
		content = fmt.Sprintf("好友充值邀请返利 %s", logger.LogQuota(rebate.Quota))
	}
	RecordLog(rebate.InviterId, LogTypeSystem, content)
	return nil
}

// hasInviteRebate is a fast pre-check that avoids doing threshold work for a
// friend who already paid out. It is not the correctness boundary: the unique
// index on (invitee_id, type) is what rejects a concurrent duplicate.
func hasInviteRebate(inviteeId int, rebateType string) (bool, error) {
	var count int64
	err := DB.Model(&InviteRebate{}).
		Where("invitee_id = ? AND type = ?", inviteeId, rebateType).
		Count(&count).Error
	return count > 0, err
}

// sumSuccessTopUpQuota 累计某用户所有成功在线支付订单的入账额度。用入账额度而不是
// 支付金额，是因为各支付渠道的 Money/Amount 口径并不统一；兑换码、签到与管理员直接
// 加额都不写 top_ups，因此天然不计入门槛。
func sumSuccessTopUpQuota(userId int) (int64, error) {
	var total int64
	err := DB.Model(&TopUp{}).
		Where("user_id = ? AND status = ?", userId, common.TopUpStatusSuccess).
		Select("COALESCE(SUM(credited_quota), 0)").
		Scan(&total).Error
	return total, err
}

// GetInviteOverview 组装邀请页数据：邀请码、当前规则、统计与脱敏后的被邀请人列表。
// status 为 InviteeStatusRegistered / InviteeStatusRecharged 时按状态筛选列表，其余
// 值返回全部；统计始终基于全部被邀请人，筛选不影响卡片数字。
func GetInviteOverview(userId int, status string) (*InviteOverview, error) {
	if userId <= 0 {
		return nil, errors.New("用户不存在")
	}

	affCode, err := GetUserAffCode(userId)
	if err != nil {
		return nil, err
	}

	setting := operation_setting.GetInviteRebateSetting()
	overview := &InviteOverview{
		AffCode:        affCode,
		RegisterAmount: setting.RegisterAmount,
		TopupAmount:    setting.TopupAmount,
		TopupThreshold: setting.TopupThreshold,
		Invitees:       []InviteeSummary{},
	}

	var inviteeCount int64
	if err := DB.Model(&User{}).Where("inviter_id = ?", userId).Count(&inviteeCount).Error; err != nil {
		return nil, err
	}
	overview.InviteeCount = int(inviteeCount)

	var rebateQuota int64
	if err := DB.Model(&InviteRebate{}).
		Where("inviter_id = ?", userId).
		Select("COALESCE(SUM(quota), 0)").
		Scan(&rebateQuota).Error; err != nil {
		return nil, err
	}
	if rebateQuota > int64(common.MaxQuota) {
		overview.RebateQuota = common.MaxQuota
	} else if rebateQuota > 0 {
		overview.RebateQuota = int(rebateQuota)
	}

	creditedTopUpUsers := DB.Model(&TopUp{}).
		Select("user_id").
		Where("status = ? AND credited_quota > ?", common.TopUpStatusSuccess, 0)
	topupRebateInvitees := DB.Model(&InviteRebate{}).
		Select("invitee_id").
		Where("inviter_id = ? AND type = ?", userId, InviteRebateTypeTopup)
	var rechargedCount int64
	if err := DB.Model(&User{}).
		Where("inviter_id = ?", userId).
		Where("id IN (?) OR id IN (?)", creditedTopUpUsers, topupRebateInvitees).
		Count(&rechargedCount).Error; err != nil {
		return nil, err
	}
	overview.RechargedCount = int(rechargedCount)

	var invitees []User
	if err := DB.Select("id", "username", "created_at").
		Where("inviter_id = ?", userId).
		Order("created_at desc").
		Order("id desc").
		Limit(maxInviteeListSize).
		Find(&invitees).Error; err != nil {
		return nil, err
	}
	if len(invitees) == 0 {
		return overview, nil
	}

	inviteeIds := make([]int, 0, len(invitees))
	for _, invitee := range invitees {
		inviteeIds = append(inviteeIds, invitee.Id)
	}

	var rebates []InviteRebate
	if err := DB.Select("invitee_id", "type", "quota").
		Where("inviter_id = ? AND invitee_id IN ?", userId, inviteeIds).
		Find(&rebates).Error; err != nil {
		return nil, err
	}

	rebateQuotaByInvitee := make(map[int]int, len(rebates))
	topupIssued := make(map[int]bool, len(rebates))
	for _, rebate := range rebates {
		rebateQuotaByInvitee[rebate.InviteeId] += rebate.Quota
		if rebate.Type == InviteRebateTypeTopup {
			topupIssued[rebate.InviteeId] = true
		}
	}

	rechargedInvitees, err := rechargedInviteeIds(inviteeIds)
	if err != nil {
		return nil, err
	}

	for _, invitee := range invitees {
		inviteeStatus := InviteeStatusRegistered
		if rechargedInvitees[invitee.Id] || topupIssued[invitee.Id] {
			inviteeStatus = InviteeStatusRecharged
		}
		if status == InviteeStatusRegistered || status == InviteeStatusRecharged {
			if inviteeStatus != status {
				continue
			}
		}
		overview.Invitees = append(overview.Invitees, InviteeSummary{
			Username:     maskUsername(invitee.Username),
			RegisteredAt: invitee.CreatedAt,
			Status:       inviteeStatus,
			RebateQuota:  rebateQuotaByInvitee[invitee.Id],
		})
	}

	return overview, nil
}

// maxInviteeListSize 限制邀请页一次返回的被邀请人数量，避免头部推广账号把整张表拉进
// 内存。卡片上的人数与累计返利按全表聚合，不受此上限影响。
const maxInviteeListSize = 500

func rechargedInviteeIds(inviteeIds []int) (map[int]bool, error) {
	recharged := make(map[int]bool, len(inviteeIds))
	if len(inviteeIds) == 0 {
		return recharged, nil
	}
	var userIds []int
	if err := DB.Model(&TopUp{}).
		Where("user_id IN ? AND status = ? AND credited_quota > ?", inviteeIds, common.TopUpStatusSuccess, 0).
		Distinct().
		Pluck("user_id", &userIds).Error; err != nil {
		return nil, err
	}
	for _, id := range userIds {
		recharged[id] = true
	}
	return recharged, nil
}

// maskUsername 脱敏用户名，保留首尾各一个字符。邀请人不应该看到好友的完整账号。
func maskUsername(username string) string {
	runes := []rune(username)
	switch len(runes) {
	case 0:
		return ""
	case 1:
		return string(runes) + "***"
	default:
		return string(runes[0]) + "***" + string(runes[len(runes)-1])
	}
}

const (
	defaultInviteAdminRankLimit = 20
	maxInviteAdminRankLimit     = 50
	maxInviteAdminRecentSize    = 50
)

// InviteAdminSummary 是全站邀请关系与已发放返利的汇总。人数来自 users.inviter_id，
// 额度只统计 invite_rebates，不含旧 aff_quota / aff_history。
type InviteAdminSummary struct {
	InviterCount        int64 `json:"inviter_count"`
	InviteeCount        int64 `json:"invitee_count"`
	RechargedCount      int64 `json:"recharged_count"`
	RebateQuota         int64 `json:"rebate_quota"`
	RebatePayoutCount   int64 `json:"rebate_payout_count"`
	RegisterRebateQuota int64 `json:"register_rebate_quota"`
	RegisterRebateCount int64 `json:"register_rebate_count"`
	TopupRebateQuota    int64 `json:"topup_rebate_quota"`
	TopupRebateCount    int64 `json:"topup_rebate_count"`
}

// InviteCountRank 是按成功邀请人数排序的一行。
type InviteCountRank struct {
	Rank         int    `json:"rank"`
	UserId       int    `json:"user_id"`
	Username     string `json:"username"`
	InviteeCount int64  `json:"invitee_count"`
	RebateQuota  int64  `json:"rebate_quota"`
}

// InviteRebateRank 是按已发放返利额度排序的一行。
type InviteRebateRank struct {
	Rank         int    `json:"rank"`
	UserId       int    `json:"user_id"`
	Username     string `json:"username"`
	RebateQuota  int64  `json:"rebate_quota"`
	PayoutCount  int64  `json:"payout_count"`
	InviteeCount int64  `json:"invitee_count"`
}

// InviteAdminInvitee 是管理员最近邀请记录的一行，用户名为明文。
type InviteAdminInvitee struct {
	InviteeId       int    `json:"invitee_id"`
	InviteeUsername string `json:"invitee_username"`
	InviterId       int    `json:"inviter_id"`
	InviterUsername string `json:"inviter_username"`
	RegisteredAt    int64  `json:"registered_at"`
	Status          string `json:"status"`
	RebateQuota     int64  `json:"rebate_quota"`
}

// InviteAdminStats 是邀请返利设置页上的全站快照。
type InviteAdminStats struct {
	Summary        InviteAdminSummary   `json:"summary"`
	InviteRankings []InviteCountRank    `json:"invite_rankings"`
	RebateRankings []InviteRebateRank   `json:"rebate_rankings"`
	RecentInvitees []InviteAdminInvitee `json:"recent_invitees"`
}

// GetInviteAdminStats 汇总全站邀请转化、已发放额度，以及邀请人数 / 返利额度排行。
// limit 只作用于两套排行，默认 20、上限 50；KPI 始终按全表聚合。
func GetInviteAdminStats(limit int) (*InviteAdminStats, error) {
	if limit <= 0 {
		limit = defaultInviteAdminRankLimit
	}
	if limit > maxInviteAdminRankLimit {
		limit = maxInviteAdminRankLimit
	}

	stats := &InviteAdminStats{
		InviteRankings: []InviteCountRank{},
		RebateRankings: []InviteRebateRank{},
		RecentInvitees: []InviteAdminInvitee{},
	}

	if err := DB.Model(&User{}).
		Where("inviter_id > ?", 0).
		Select("COUNT(DISTINCT inviter_id)").
		Scan(&stats.Summary.InviterCount).Error; err != nil {
		return nil, err
	}

	var inviteeCount int64
	if err := DB.Model(&User{}).Where("inviter_id > ?", 0).Count(&inviteeCount).Error; err != nil {
		return nil, err
	}
	stats.Summary.InviteeCount = inviteeCount

	creditedTopUpUsers := DB.Model(&TopUp{}).
		Select("user_id").
		Where("status = ? AND credited_quota > ?", common.TopUpStatusSuccess, 0)
	topupRebateInvitees := DB.Model(&InviteRebate{}).
		Select("invitee_id").
		Where("type = ?", InviteRebateTypeTopup)
	var rechargedCount int64
	if err := DB.Model(&User{}).
		Where("inviter_id > ?", 0).
		Where("id IN (?) OR id IN (?)", creditedTopUpUsers, topupRebateInvitees).
		Count(&rechargedCount).Error; err != nil {
		return nil, err
	}
	stats.Summary.RechargedCount = rechargedCount

	var rebateTotals []struct {
		Type  string `gorm:"column:type"`
		Quota int64  `gorm:"column:quota"`
		Count int64  `gorm:"column:count"`
	}
	if err := DB.Model(&InviteRebate{}).
		Select("type, COALESCE(SUM(quota), 0) as quota, COUNT(*) as count").
		Group("type").
		Scan(&rebateTotals).Error; err != nil {
		return nil, err
	}
	for _, total := range rebateTotals {
		stats.Summary.RebateQuota += total.Quota
		stats.Summary.RebatePayoutCount += total.Count
		switch total.Type {
		case InviteRebateTypeRegister:
			stats.Summary.RegisterRebateQuota = total.Quota
			stats.Summary.RegisterRebateCount = total.Count
		case InviteRebateTypeTopup:
			stats.Summary.TopupRebateQuota = total.Quota
			stats.Summary.TopupRebateCount = total.Count
		}
	}

	var inviteRanks []struct {
		InviterId    int   `gorm:"column:inviter_id"`
		InviteeCount int64 `gorm:"column:invitee_count"`
	}
	if err := DB.Model(&User{}).
		Select("inviter_id, COUNT(*) as invitee_count").
		Where("inviter_id > ?", 0).
		Group("inviter_id").
		Order("invitee_count DESC").
		Order("inviter_id ASC").
		Limit(limit).
		Scan(&inviteRanks).Error; err != nil {
		return nil, err
	}

	var rebateRanks []struct {
		InviterId   int   `gorm:"column:inviter_id"`
		RebateQuota int64 `gorm:"column:rebate_quota"`
		PayoutCount int64 `gorm:"column:payout_count"`
	}
	if err := DB.Model(&InviteRebate{}).
		Select("inviter_id, COALESCE(SUM(quota), 0) as rebate_quota, COUNT(*) as payout_count").
		Group("inviter_id").
		Order("rebate_quota DESC").
		Order("inviter_id ASC").
		Limit(limit).
		Scan(&rebateRanks).Error; err != nil {
		return nil, err
	}

	rankUserIDs := make([]int, 0, len(inviteRanks)+len(rebateRanks))
	for _, row := range inviteRanks {
		rankUserIDs = append(rankUserIDs, row.InviterId)
	}
	for _, row := range rebateRanks {
		rankUserIDs = append(rankUserIDs, row.InviterId)
	}
	usernames, err := inviteUsernamesByIDs(rankUserIDs)
	if err != nil {
		return nil, err
	}

	inviteRankIDs := make([]int, 0, len(inviteRanks))
	for _, row := range inviteRanks {
		inviteRankIDs = append(inviteRankIDs, row.InviterId)
	}
	rebateQuotaByInviter := make(map[int]int64, len(inviteRankIDs))
	if len(inviteRankIDs) > 0 {
		var rebateSums []struct {
			InviterId   int   `gorm:"column:inviter_id"`
			RebateQuota int64 `gorm:"column:rebate_quota"`
		}
		if err := DB.Model(&InviteRebate{}).
			Select("inviter_id, COALESCE(SUM(quota), 0) as rebate_quota").
			Where("inviter_id IN ?", inviteRankIDs).
			Group("inviter_id").
			Scan(&rebateSums).Error; err != nil {
			return nil, err
		}
		for _, row := range rebateSums {
			rebateQuotaByInviter[row.InviterId] = row.RebateQuota
		}
	}

	rebateRankIDs := make([]int, 0, len(rebateRanks))
	for _, row := range rebateRanks {
		rebateRankIDs = append(rebateRankIDs, row.InviterId)
	}
	inviteeCountByInviter := make(map[int]int64, len(rebateRankIDs))
	if len(rebateRankIDs) > 0 {
		var inviteeCounts []struct {
			InviterId    int   `gorm:"column:inviter_id"`
			InviteeCount int64 `gorm:"column:invitee_count"`
		}
		if err := DB.Model(&User{}).
			Select("inviter_id, COUNT(*) as invitee_count").
			Where("inviter_id IN ?", rebateRankIDs).
			Group("inviter_id").
			Scan(&inviteeCounts).Error; err != nil {
			return nil, err
		}
		for _, row := range inviteeCounts {
			inviteeCountByInviter[row.InviterId] = row.InviteeCount
		}
	}

	for i, row := range inviteRanks {
		stats.InviteRankings = append(stats.InviteRankings, InviteCountRank{
			Rank:         i + 1,
			UserId:       row.InviterId,
			Username:     usernames[row.InviterId],
			InviteeCount: row.InviteeCount,
			RebateQuota:  rebateQuotaByInviter[row.InviterId],
		})
	}
	for i, row := range rebateRanks {
		stats.RebateRankings = append(stats.RebateRankings, InviteRebateRank{
			Rank:         i + 1,
			UserId:       row.InviterId,
			Username:     usernames[row.InviterId],
			RebateQuota:  row.RebateQuota,
			PayoutCount:  row.PayoutCount,
			InviteeCount: inviteeCountByInviter[row.InviterId],
		})
	}

	var recent []User
	if err := DB.Select("id", "username", "inviter_id", "created_at").
		Where("inviter_id > ?", 0).
		Order("created_at desc").
		Order("id desc").
		Limit(maxInviteAdminRecentSize).
		Find(&recent).Error; err != nil {
		return nil, err
	}
	if len(recent) == 0 {
		return stats, nil
	}

	recentIDs := make([]int, 0, len(recent)*2)
	inviteeIDs := make([]int, 0, len(recent))
	for _, invitee := range recent {
		inviteeIDs = append(inviteeIDs, invitee.Id)
		recentIDs = append(recentIDs, invitee.Id, invitee.InviterId)
	}
	recentNames, err := inviteUsernamesByIDs(recentIDs)
	if err != nil {
		return nil, err
	}

	var rebates []InviteRebate
	if err := DB.Select("invitee_id", "type", "quota").
		Where("invitee_id IN ?", inviteeIDs).
		Find(&rebates).Error; err != nil {
		return nil, err
	}
	rebateQuotaByInvitee := make(map[int]int64, len(rebates))
	topupIssued := make(map[int]bool, len(rebates))
	for _, rebate := range rebates {
		rebateQuotaByInvitee[rebate.InviteeId] += int64(rebate.Quota)
		if rebate.Type == InviteRebateTypeTopup {
			topupIssued[rebate.InviteeId] = true
		}
	}
	rechargedInvitees, err := rechargedInviteeIds(inviteeIDs)
	if err != nil {
		return nil, err
	}

	for _, invitee := range recent {
		status := InviteeStatusRegistered
		if rechargedInvitees[invitee.Id] || topupIssued[invitee.Id] {
			status = InviteeStatusRecharged
		}
		stats.RecentInvitees = append(stats.RecentInvitees, InviteAdminInvitee{
			InviteeId:       invitee.Id,
			InviteeUsername: invitee.Username,
			InviterId:       invitee.InviterId,
			InviterUsername: recentNames[invitee.InviterId],
			RegisteredAt:    invitee.CreatedAt,
			Status:          status,
			RebateQuota:     rebateQuotaByInvitee[invitee.Id],
		})
	}

	return stats, nil
}

func inviteUsernamesByIDs(ids []int) (map[int]string, error) {
	names := make(map[int]string)
	if len(ids) == 0 {
		return names, nil
	}
	unique := make([]int, 0, len(ids))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return names, nil
	}

	var users []User
	if err := DB.Select("id", "username").Where("id IN ?", unique).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, user := range users {
		names[user.Id] = user.Username
	}
	return names, nil
}
