package model

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// useInviteRebateDB gives each test its own schema so the shared package DB from
// TestMain cannot leak users, topups or rebates between cases.
func useInviteRebateDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &TopUp{}, &InviteRebate{}, &Log{}, &Option{}))

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(4)

	DB, LOG_DB = db, db
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		_ = sqlDB.Close()
	})
	return db
}

// withInviteRebateSetting installs a rebate configuration plus the compliance
// confirmation that gates every payout, and restores both afterwards.
func withInviteRebateSetting(t *testing.T, registerAmount, topupAmount, topupThreshold float64) {
	t.Helper()
	setting := operation_setting.GetInviteRebateSetting()
	previous := *setting
	payment := operation_setting.GetPaymentSetting()
	previousPayment := *payment
	previousQuotaPerUnit := common.QuotaPerUnit

	setting.RegisterAmount = registerAmount
	setting.TopupAmount = topupAmount
	setting.TopupThreshold = topupThreshold
	payment.ComplianceConfirmed = true
	payment.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	common.QuotaPerUnit = 500_000

	t.Cleanup(func() {
		*setting = previous
		*payment = previousPayment
		common.QuotaPerUnit = previousQuotaPerUnit
	})
}

func createInviteTestUser(t *testing.T, username string, inviterId int) *User {
	t.Helper()
	user := &User{
		Username:  username,
		Password:  "unused-password-hash",
		AffCode:   username + "-aff",
		InviterId: inviterId,
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func createSuccessfulTopUp(t *testing.T, userId int, creditedQuota int) {
	t.Helper()
	require.NoError(t, DB.Create(&TopUp{
		UserId:        userId,
		TradeNo:       fmt.Sprintf("trade-%d-%d", userId, creditedQuota),
		Status:        common.TopUpStatusSuccess,
		CreditedQuota: creditedQuota,
	}).Error)
}

func requireUserQuota(t *testing.T, userId int) int {
	t.Helper()
	var user User
	require.NoError(t, DB.Select("id", "quota", "aff_history").Where("id = ?", userId).First(&user).Error)
	return user.Quota
}

func countRebates(t *testing.T, inviteeId int, rebateType string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, DB.Model(&InviteRebate{}).
		Where("invitee_id = ? AND type = ?", inviteeId, rebateType).
		Count(&count).Error)
	return count
}

func TestInviteRegisterRebateIsIssuedOncePerInvitee(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 2, 5, 50)

	inviter := createInviteTestUser(t, "inviter", 0)
	invitee := createInviteTestUser(t, "invitee", inviter.Id)

	IssueInviteRegisterRebate(inviter.Id, invitee.Id)
	IssueInviteRegisterRebate(inviter.Id, invitee.Id)

	// 2 元 / 7.3 元每美元 * 500000 额度每美元，截断到整数额度。
	expectedQuota := operation_setting.InviteRebateQuota(2)
	require.Positive(t, expectedQuota)
	assert.Equal(t, int64(1), countRebates(t, invitee.Id, InviteRebateTypeRegister))
	assert.Equal(t, expectedQuota, requireUserQuota(t, inviter.Id))

	var stored InviteRebate
	require.NoError(t, DB.Where("invitee_id = ?", invitee.Id).First(&stored).Error)
	assert.Equal(t, inviter.Id, stored.InviterId)
	assert.Equal(t, expectedQuota, stored.Quota)
	assert.InDelta(t, 2/operation_setting.InviteRebateCurrencyRate(), stored.AmountUsd, 1e-9)
}

func TestInviteRegisterRebateSkippedWhenAmountIsZero(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 0, 5, 50)

	inviter := createInviteTestUser(t, "inviter", 0)
	invitee := createInviteTestUser(t, "invitee", inviter.Id)

	IssueInviteRegisterRebate(inviter.Id, invitee.Id)

	assert.Zero(t, countRebates(t, invitee.Id, InviteRebateTypeRegister))
	assert.Zero(t, requireUserQuota(t, inviter.Id))
}

func TestInviteRegisterRebateSkippedWithoutPaymentCompliance(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 2, 5, 50)
	payment := operation_setting.GetPaymentSetting()
	payment.ComplianceConfirmed = false

	inviter := createInviteTestUser(t, "inviter", 0)
	invitee := createInviteTestUser(t, "invitee", inviter.Id)

	IssueInviteRegisterRebate(inviter.Id, invitee.Id)

	assert.Zero(t, countRebates(t, invitee.Id, InviteRebateTypeRegister))
	assert.Zero(t, requireUserQuota(t, inviter.Id))
}

func TestInviteRegisterRebateRejectsSelfInvite(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 2, 5, 50)

	user := createInviteTestUser(t, "self-inviter", 0)

	IssueInviteRegisterRebate(user.Id, user.Id)

	assert.Zero(t, countRebates(t, user.Id, InviteRebateTypeRegister))
	assert.Zero(t, requireUserQuota(t, user.Id))
}

func TestInviteTopupRebateWaitsUntilThresholdIsReached(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 0, 5, 50)

	inviter := createInviteTestUser(t, "inviter", 0)
	invitee := createInviteTestUser(t, "invitee", inviter.Id)
	thresholdQuota := operation_setting.InviteRebateQuota(50)
	require.Positive(t, thresholdQuota)

	createSuccessfulTopUp(t, invitee.Id, thresholdQuota-1)
	IssueInviteTopupRebate(invitee.Id)
	assert.Zero(t, countRebates(t, invitee.Id, InviteRebateTypeTopup))
	assert.Zero(t, requireUserQuota(t, inviter.Id))

	createSuccessfulTopUp(t, invitee.Id, 1)
	IssueInviteTopupRebate(invitee.Id)
	assert.Equal(t, int64(1), countRebates(t, invitee.Id, InviteRebateTypeTopup))
	assert.Equal(t, operation_setting.InviteRebateQuota(5), requireUserQuota(t, inviter.Id))
}

func TestInviteTopupRebateIsIssuedOncePerInviteeAcrossRepeatTopups(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 0, 5, 50)

	inviter := createInviteTestUser(t, "inviter", 0)
	invitee := createInviteTestUser(t, "invitee", inviter.Id)
	createSuccessfulTopUp(t, invitee.Id, operation_setting.InviteRebateQuota(500))

	IssueInviteTopupRebate(invitee.Id)
	IssueInviteTopupRebate(invitee.Id)
	IssueInviteTopupRebate(invitee.Id)

	assert.Equal(t, int64(1), countRebates(t, invitee.Id, InviteRebateTypeTopup))
	assert.Equal(t, operation_setting.InviteRebateQuota(5), requireUserQuota(t, inviter.Id))
}

// 并发充值回调各自判定门槛时，唯一索引必须裁决出唯一一次发放，否则邀请人会被重复加额。
func TestInviteTopupRebateUniqueIndexRejectsConcurrentPayouts(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 0, 5, 50)

	inviter := createInviteTestUser(t, "inviter", 0)
	invitee := createInviteTestUser(t, "invitee", inviter.Id)
	createSuccessfulTopUp(t, invitee.Id, operation_setting.InviteRebateQuota(500))

	quota := operation_setting.InviteRebateQuota(5)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = creditInviteRebate(&InviteRebate{
				InviterId: inviter.Id,
				InviteeId: invitee.Id,
				Type:      InviteRebateTypeTopup,
				Quota:     quota,
			})
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(1), countRebates(t, invitee.Id, InviteRebateTypeTopup))
	assert.Equal(t, quota, requireUserQuota(t, inviter.Id))
}

func TestInviteTopupRebateSkippedWhenAmountIsZero(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 2, 0, 50)

	inviter := createInviteTestUser(t, "inviter", 0)
	invitee := createInviteTestUser(t, "invitee", inviter.Id)
	createSuccessfulTopUp(t, invitee.Id, operation_setting.InviteRebateQuota(500))

	IssueInviteTopupRebate(invitee.Id)

	assert.Zero(t, countRebates(t, invitee.Id, InviteRebateTypeTopup))
	assert.Zero(t, requireUserQuota(t, inviter.Id))
}

func TestInviteTopupRebateSkippedWithoutInviter(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 0, 5, 50)

	orphan := createInviteTestUser(t, "orphan", 0)
	createSuccessfulTopUp(t, orphan.Id, operation_setting.InviteRebateQuota(500))

	IssueInviteTopupRebate(orphan.Id)

	assert.Zero(t, countRebates(t, orphan.Id, InviteRebateTypeTopup))
}

// 只有成功入账的在线支付订单才计入门槛。兑换码、签到与管理员直接加额都不写 top_ups，
// 未完成的订单也不能提前把好友推过门槛。
func TestInviteTopupThresholdCountsOnlySuccessfulCreditedTopUps(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 0, 5, 50)

	inviter := createInviteTestUser(t, "inviter", 0)
	invitee := createInviteTestUser(t, "invitee", inviter.Id)
	thresholdQuota := operation_setting.InviteRebateQuota(50)

	require.NoError(t, DB.Create(&TopUp{
		UserId:        invitee.Id,
		TradeNo:       "pending-order",
		Status:        common.TopUpStatusPending,
		CreditedQuota: thresholdQuota,
	}).Error)
	require.NoError(t, DB.Create(&TopUp{
		UserId:        invitee.Id,
		TradeNo:       "failed-order",
		Status:        common.TopUpStatusFailed,
		CreditedQuota: thresholdQuota,
	}).Error)
	// 兑换码只增加余额，不产生充值订单。
	require.NoError(t, IncreaseUserQuota(invitee.Id, thresholdQuota, true))

	IssueInviteTopupRebate(invitee.Id)

	total, err := sumSuccessTopUpQuota(invitee.Id)
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Zero(t, countRebates(t, invitee.Id, InviteRebateTypeTopup))
}

func TestInviteOverviewMasksUsernamesAndReportsStatus(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 2, 5, 50)

	inviter := createInviteTestUser(t, "promoter", 0)
	registeredOnly := createInviteTestUser(t, "alice", inviter.Id)
	recharged := createInviteTestUser(t, "bob", inviter.Id)
	createSuccessfulTopUp(t, recharged.Id, operation_setting.InviteRebateQuota(500))

	IssueInviteRegisterRebate(inviter.Id, registeredOnly.Id)
	IssueInviteRegisterRebate(inviter.Id, recharged.Id)
	IssueInviteTopupRebate(recharged.Id)

	overview, err := GetInviteOverview(inviter.Id, "")
	require.NoError(t, err)

	assert.Equal(t, "promoter-aff", overview.AffCode)
	assert.Equal(t, 2.0, overview.RegisterAmount)
	assert.Equal(t, 5.0, overview.TopupAmount)
	assert.Equal(t, 50.0, overview.TopupThreshold)
	assert.Equal(t, 2, overview.InviteeCount)
	assert.Equal(t, 1, overview.RechargedCount)

	registerQuota := operation_setting.InviteRebateQuota(2)
	topupQuota := operation_setting.InviteRebateQuota(5)
	assert.Equal(t, registerQuota*2+topupQuota, overview.RebateQuota)

	require.Len(t, overview.Invitees, 2)
	byStatus := map[string]InviteeSummary{}
	for _, invitee := range overview.Invitees {
		byStatus[invitee.Status] = invitee
		assert.NotContains(t, invitee.Username, "alice")
		assert.NotContains(t, invitee.Username, "bob")
	}
	assert.Equal(t, "a***e", byStatus[InviteeStatusRegistered].Username)
	assert.Equal(t, registerQuota, byStatus[InviteeStatusRegistered].RebateQuota)
	assert.Equal(t, "b***b", byStatus[InviteeStatusRecharged].Username)
	assert.Equal(t, registerQuota+topupQuota, byStatus[InviteeStatusRecharged].RebateQuota)
}

func TestInviteOverviewFiltersByStatusWithoutChangingTotals(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 2, 5, 50)

	inviter := createInviteTestUser(t, "promoter", 0)
	createInviteTestUser(t, "alice", inviter.Id)
	recharged := createInviteTestUser(t, "bob", inviter.Id)
	createSuccessfulTopUp(t, recharged.Id, operation_setting.InviteRebateQuota(500))

	registeredView, err := GetInviteOverview(inviter.Id, InviteeStatusRegistered)
	require.NoError(t, err)
	require.Len(t, registeredView.Invitees, 1)
	assert.Equal(t, InviteeStatusRegistered, registeredView.Invitees[0].Status)
	assert.Equal(t, 2, registeredView.InviteeCount)
	assert.Equal(t, 1, registeredView.RechargedCount)

	rechargedView, err := GetInviteOverview(inviter.Id, InviteeStatusRecharged)
	require.NoError(t, err)
	require.Len(t, rechargedView.Invitees, 1)
	assert.Equal(t, InviteeStatusRecharged, rechargedView.Invitees[0].Status)
	assert.Equal(t, 2, rechargedView.InviteeCount)
	assert.Equal(t, 1, rechargedView.RechargedCount)
}

func TestInviteOverviewWithoutInviteesReturnsEmptyList(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 2, 5, 50)

	inviter := createInviteTestUser(t, "lonely", 0)

	overview, err := GetInviteOverview(inviter.Id, "")
	require.NoError(t, err)

	assert.Equal(t, 0, overview.InviteeCount)
	assert.Equal(t, 0, overview.RechargedCount)
	assert.Equal(t, 0, overview.RebateQuota)
	assert.NotNil(t, overview.Invitees)
	assert.Empty(t, overview.Invitees)
}

// 邀请页对未生成邀请码的历史账号必须补发一个，否则用户永远拿不到可分享的链接。
func TestGetUserAffCodeGeneratesMissingCodeOnce(t *testing.T) {
	useInviteRebateDB(t)

	user := &User{Username: "legacy", Password: "unused-password-hash"}
	require.NoError(t, DB.Create(user).Error)

	affCode, err := GetUserAffCode(user.Id)
	require.NoError(t, err)
	require.NotEmpty(t, affCode)

	repeated, err := GetUserAffCode(user.Id)
	require.NoError(t, err)
	assert.Equal(t, affCode, repeated)
}

func TestMaskUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     string
	}{
		{name: "empty", username: "", want: ""},
		{name: "single character", username: "a", want: "a***"},
		{name: "two characters", username: "ab", want: "a***b"},
		{name: "long name", username: "promoter", want: "p***r"},
		{name: "multi-byte name", username: "张三丰", want: "张***丰"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, maskUsername(test.username))
		})
	}
}

// 注册奖励入口必须无条件记录邀请关系，这样即使返利金额为 0，邀请页的成功人数也不会
// 和被邀请人列表对不上。
func TestApplyRegistrationRewardsCountsInviteEvenWithoutPayout(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 0, 0, 0)

	inviter := createInviteTestUser(t, "inviter", 0)
	invitee := createInviteTestUser(t, "invitee", inviter.Id)

	applyRegistrationRewards(invitee.Id, inviter.Id)

	var stored User
	require.NoError(t, DB.Select("id", "aff_count", "quota").Where("id = ?", inviter.Id).First(&stored).Error)
	assert.Equal(t, 1, stored.AffCount)
	assert.Zero(t, stored.Quota)
}

// OAuth 注册必须把邀请关系落库，否则被邀请人日后充值时无法追溯邀请人。
func TestInsertWithTxPersistsInviterId(t *testing.T) {
	db := useInviteRebateDB(t)

	inviter := createInviteTestUser(t, "inviter", 0)
	oauthUser := &User{Username: "oauth-user", DisplayName: "OAuth User"}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return oauthUser.InsertWithTx(tx, inviter.Id)
	}))

	var stored User
	require.NoError(t, DB.Select("id", "inviter_id").Where("id = ?", oauthUser.Id).First(&stored).Error)
	assert.Equal(t, inviter.Id, stored.InviterId)
}

func TestInsertPersistsInviterId(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 0, 0, 0)

	inviter := createInviteTestUser(t, "inviter", 0)
	user := &User{Username: "password-user", Password: "secret", DisplayName: "Password User"}
	require.NoError(t, user.Insert(inviter.Id))

	var stored User
	require.NoError(t, DB.Select("id", "inviter_id").Where("id = ?", user.Id).First(&stored).Error)
	assert.Equal(t, inviter.Id, stored.InviterId)
}

func TestInviteRegisterRebateSkippedWhenInviterBindingMismatches(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 2, 5, 50)

	inviter := createInviteTestUser(t, "inviter", 0)
	invitee := createInviteTestUser(t, "invitee", 0)

	IssueInviteRegisterRebate(inviter.Id, invitee.Id)

	assert.Zero(t, countRebates(t, invitee.Id, InviteRebateTypeRegister))
	assert.Zero(t, requireUserQuota(t, inviter.Id))
}

func TestInviteTopupRebateSkippedWhenThresholdQuotaIsZero(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 0, 5, 1e-10)

	inviter := createInviteTestUser(t, "inviter", 0)
	invitee := createInviteTestUser(t, "invitee", inviter.Id)
	createSuccessfulTopUp(t, invitee.Id, operation_setting.InviteRebateQuota(500))

	IssueInviteTopupRebate(invitee.Id)

	assert.Zero(t, operation_setting.InviteRebateQuota(1e-10))
	assert.Zero(t, countRebates(t, invitee.Id, InviteRebateTypeTopup))
	assert.Zero(t, requireUserQuota(t, inviter.Id))
}

func TestCreditInviteRebateRollsBackWhenWalletWouldOverflow(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 2, 5, 50)

	inviter := createInviteTestUser(t, "inviter", 0)
	invitee := createInviteTestUser(t, "invitee", inviter.Id)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", inviter.Id).Updates(map[string]interface{}{
		"quota":       common.MaxQuota - 5,
		"aff_history": 7,
	}).Error)

	err := creditInviteRebate(&InviteRebate{
		InviterId: inviter.Id,
		InviteeId: invitee.Id,
		Type:      InviteRebateTypeRegister,
		Quota:     100,
	})
	require.ErrorIs(t, err, ErrTopUpQuotaLimitExceeded)
	assert.Zero(t, countRebates(t, invitee.Id, InviteRebateTypeRegister))

	var stored User
	require.NoError(t, DB.Select("quota", "aff_history").Where("id = ?", inviter.Id).First(&stored).Error)
	assert.Equal(t, common.MaxQuota-5, stored.Quota)
	assert.Equal(t, 7, stored.AffHistoryQuota)
}

func TestCreditInviteRebateRollsBackWhenInviterMissing(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 2, 5, 50)

	invitee := createInviteTestUser(t, "invitee", 0)
	err := creditInviteRebate(&InviteRebate{
		InviterId: 999_999,
		InviteeId: invitee.Id,
		Type:      InviteRebateTypeRegister,
		Quota:     operation_setting.InviteRebateQuota(2),
	})
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assert.Zero(t, countRebates(t, invitee.Id, InviteRebateTypeRegister))
}

func TestInviteOverviewIgnoresZeroCreditedSubscriptionTopUp(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 2, 5, 50)

	inviter := createInviteTestUser(t, "promoter", 0)
	invitee := createInviteTestUser(t, "alice", inviter.Id)
	require.NoError(t, DB.Create(&TopUp{
		UserId:        invitee.Id,
		TradeNo:       "subscription-order",
		Status:        common.TopUpStatusSuccess,
		CreditedQuota: 0,
	}).Error)

	IssueInviteTopupRebate(invitee.Id)

	overview, err := GetInviteOverview(inviter.Id, "")
	require.NoError(t, err)
	assert.Equal(t, 1, overview.InviteeCount)
	assert.Equal(t, 0, overview.RechargedCount)
	require.Len(t, overview.Invitees, 1)
	assert.Equal(t, InviteeStatusRegistered, overview.Invitees[0].Status)
	assert.Zero(t, countRebates(t, invitee.Id, InviteRebateTypeTopup))
}

func TestInviteOverviewTotalsIgnoreListCap(t *testing.T) {
	useInviteRebateDB(t)
	withInviteRebateSetting(t, 2, 5, 50)

	inviter := createInviteTestUser(t, "promoter", 0)
	oldest := createInviteTestUser(t, "oldest", inviter.Id)
	for i := 0; i < maxInviteeListSize; i++ {
		createInviteTestUser(t, fmt.Sprintf("invitee-%03d", i), inviter.Id)
	}
	createSuccessfulTopUp(t, oldest.Id, operation_setting.InviteRebateQuota(500))
	IssueInviteRegisterRebate(inviter.Id, oldest.Id)
	IssueInviteTopupRebate(oldest.Id)

	overview, err := GetInviteOverview(inviter.Id, "")
	require.NoError(t, err)

	registerQuota := operation_setting.InviteRebateQuota(2)
	topupQuota := operation_setting.InviteRebateQuota(5)
	assert.Equal(t, maxInviteeListSize+1, overview.InviteeCount)
	assert.Equal(t, 1, overview.RechargedCount)
	assert.Equal(t, registerQuota+topupQuota, overview.RebateQuota)
	require.Len(t, overview.Invitees, maxInviteeListSize)
	for _, invitee := range overview.Invitees {
		assert.NotEqual(t, "o***t", invitee.Username)
	}
}

func createInviteRebateRecord(t *testing.T, inviterId int, inviteeId int, rebateType string, quota int) {
	t.Helper()
	require.NoError(t, DB.Create(&InviteRebate{
		InviterId: inviterId,
		InviteeId: inviteeId,
		Type:      rebateType,
		Quota:     quota,
		CreatedAt: common.GetTimestamp(),
	}).Error)
}

func TestInviteAdminStatsEmptySite(t *testing.T) {
	useInviteRebateDB(t)

	stats, err := GetInviteAdminStats(0)
	require.NoError(t, err)

	assert.Zero(t, stats.Summary.InviterCount)
	assert.Zero(t, stats.Summary.InviteeCount)
	assert.Zero(t, stats.Summary.RechargedCount)
	assert.Zero(t, stats.Summary.RebateQuota)
	assert.Zero(t, stats.Summary.RebatePayoutCount)
	assert.Empty(t, stats.InviteRankings)
	assert.Empty(t, stats.RebateRankings)
	assert.Empty(t, stats.RecentInvitees)
}

func TestInviteAdminStatsCountsBindingsWithoutPayouts(t *testing.T) {
	useInviteRebateDB(t)

	inviter := createInviteTestUser(t, "promoter", 0)
	first := createInviteTestUser(t, "alice", inviter.Id)
	second := createInviteTestUser(t, "bob", inviter.Id)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", first.Id).Update("created_at", 100).Error)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", second.Id).Update("created_at", 200).Error)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", inviter.Id).Updates(map[string]interface{}{
		"aff_history": 9_000,
	}).Error)

	stats, err := GetInviteAdminStats(0)
	require.NoError(t, err)

	assert.Equal(t, int64(1), stats.Summary.InviterCount)
	assert.Equal(t, int64(2), stats.Summary.InviteeCount)
	assert.Zero(t, stats.Summary.RechargedCount)
	assert.Zero(t, stats.Summary.RebateQuota)
	assert.Zero(t, stats.Summary.RebatePayoutCount)
	require.Len(t, stats.InviteRankings, 1)
	assert.Equal(t, inviter.Id, stats.InviteRankings[0].UserId)
	assert.Equal(t, "promoter", stats.InviteRankings[0].Username)
	assert.Equal(t, int64(2), stats.InviteRankings[0].InviteeCount)
	assert.Zero(t, stats.InviteRankings[0].RebateQuota)
	assert.Empty(t, stats.RebateRankings)
	require.Len(t, stats.RecentInvitees, 2)
	assert.Equal(t, "bob", stats.RecentInvitees[0].InviteeUsername)
	assert.Equal(t, "alice", stats.RecentInvitees[1].InviteeUsername)
	assert.Equal(t, InviteeStatusRegistered, stats.RecentInvitees[0].Status)
}

func TestInviteAdminStatsSplitsRegisterAndTopupPayouts(t *testing.T) {
	useInviteRebateDB(t)

	inviter := createInviteTestUser(t, "promoter", 0)
	registeredOnly := createInviteTestUser(t, "alice", inviter.Id)
	recharged := createInviteTestUser(t, "bob", inviter.Id)
	createSuccessfulTopUp(t, recharged.Id, 500)
	createInviteRebateRecord(t, inviter.Id, registeredOnly.Id, InviteRebateTypeRegister, 100)
	createInviteRebateRecord(t, inviter.Id, recharged.Id, InviteRebateTypeRegister, 100)
	createInviteRebateRecord(t, inviter.Id, recharged.Id, InviteRebateTypeTopup, 250)

	stats, err := GetInviteAdminStats(0)
	require.NoError(t, err)

	assert.Equal(t, int64(1), stats.Summary.InviterCount)
	assert.Equal(t, int64(2), stats.Summary.InviteeCount)
	assert.Equal(t, int64(1), stats.Summary.RechargedCount)
	assert.Equal(t, int64(450), stats.Summary.RebateQuota)
	assert.Equal(t, int64(3), stats.Summary.RebatePayoutCount)
	assert.Equal(t, int64(200), stats.Summary.RegisterRebateQuota)
	assert.Equal(t, int64(2), stats.Summary.RegisterRebateCount)
	assert.Equal(t, int64(250), stats.Summary.TopupRebateQuota)
	assert.Equal(t, int64(1), stats.Summary.TopupRebateCount)

	require.Len(t, stats.InviteRankings, 1)
	assert.Equal(t, int64(450), stats.InviteRankings[0].RebateQuota)
	require.Len(t, stats.RebateRankings, 1)
	assert.Equal(t, int64(3), stats.RebateRankings[0].PayoutCount)
	assert.Equal(t, int64(2), stats.RebateRankings[0].InviteeCount)

	byInvitee := map[string]InviteAdminInvitee{}
	for _, row := range stats.RecentInvitees {
		byInvitee[row.InviteeUsername] = row
	}
	assert.Equal(t, InviteeStatusRegistered, byInvitee["alice"].Status)
	assert.Equal(t, int64(100), byInvitee["alice"].RebateQuota)
	assert.Equal(t, InviteeStatusRecharged, byInvitee["bob"].Status)
	assert.Equal(t, int64(350), byInvitee["bob"].RebateQuota)
	assert.Equal(t, "promoter", byInvitee["bob"].InviterUsername)
}

func TestInviteAdminStatsRanksInvitersByInviteesAndRebate(t *testing.T) {
	useInviteRebateDB(t)

	moreInvites := createInviteTestUser(t, "alice", 0)
	moreRebate := createInviteTestUser(t, "bob", 0)
	aliceInvitee := createInviteTestUser(t, "a1", moreInvites.Id)
	createInviteTestUser(t, "a2", moreInvites.Id)
	createInviteTestUser(t, "a3", moreInvites.Id)
	bobInvitee := createInviteTestUser(t, "b1", moreRebate.Id)
	createInviteRebateRecord(t, moreInvites.Id, aliceInvitee.Id, InviteRebateTypeRegister, 200)
	createInviteRebateRecord(t, moreRebate.Id, bobInvitee.Id, InviteRebateTypeRegister, 500)

	stats, err := GetInviteAdminStats(0)
	require.NoError(t, err)

	require.Len(t, stats.InviteRankings, 2)
	assert.Equal(t, "alice", stats.InviteRankings[0].Username)
	assert.Equal(t, int64(3), stats.InviteRankings[0].InviteeCount)
	assert.Equal(t, "bob", stats.InviteRankings[1].Username)
	assert.Equal(t, int64(1), stats.InviteRankings[1].InviteeCount)

	require.Len(t, stats.RebateRankings, 2)
	assert.Equal(t, "bob", stats.RebateRankings[0].Username)
	assert.Equal(t, int64(500), stats.RebateRankings[0].RebateQuota)
	assert.Equal(t, "alice", stats.RebateRankings[1].Username)
	assert.Equal(t, int64(200), stats.RebateRankings[1].RebateQuota)
}

func TestInviteAdminStatsTieBreaksByInviterId(t *testing.T) {
	useInviteRebateDB(t)

	first := createInviteTestUser(t, "first", 0)
	second := createInviteTestUser(t, "second", 0)
	firstInvitee := createInviteTestUser(t, "f1", first.Id)
	createInviteTestUser(t, "f2", first.Id)
	secondInvitee := createInviteTestUser(t, "s1", second.Id)
	createInviteTestUser(t, "s2", second.Id)
	createInviteRebateRecord(t, first.Id, firstInvitee.Id, InviteRebateTypeRegister, 80)
	createInviteRebateRecord(t, second.Id, secondInvitee.Id, InviteRebateTypeRegister, 80)

	stats, err := GetInviteAdminStats(0)
	require.NoError(t, err)

	require.Len(t, stats.InviteRankings, 2)
	assert.Equal(t, first.Id, stats.InviteRankings[0].UserId)
	assert.Equal(t, second.Id, stats.InviteRankings[1].UserId)
	require.Len(t, stats.RebateRankings, 2)
	assert.Equal(t, first.Id, stats.RebateRankings[0].UserId)
	assert.Equal(t, second.Id, stats.RebateRankings[1].UserId)
}

func TestInviteAdminStatsRecentInviteesNewestFirst(t *testing.T) {
	useInviteRebateDB(t)

	inviter := createInviteTestUser(t, "promoter", 0)
	oldest := createInviteTestUser(t, "oldest", inviter.Id)
	middle := createInviteTestUser(t, "middle", inviter.Id)
	newest := createInviteTestUser(t, "newest", inviter.Id)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", oldest.Id).Update("created_at", 10).Error)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", middle.Id).Update("created_at", 20).Error)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", newest.Id).Update("created_at", 30).Error)

	stats, err := GetInviteAdminStats(0)
	require.NoError(t, err)

	require.Len(t, stats.RecentInvitees, 3)
	assert.Equal(t, []string{"newest", "middle", "oldest"}, []string{
		stats.RecentInvitees[0].InviteeUsername,
		stats.RecentInvitees[1].InviteeUsername,
		stats.RecentInvitees[2].InviteeUsername,
	})
}

func TestInviteAdminStatsShowsEmptyUsernameWhenInviterMissing(t *testing.T) {
	useInviteRebateDB(t)

	invitee := createInviteTestUser(t, "orphan", 999_999)

	stats, err := GetInviteAdminStats(0)
	require.NoError(t, err)

	assert.Equal(t, int64(1), stats.Summary.InviterCount)
	assert.Equal(t, int64(1), stats.Summary.InviteeCount)
	require.Len(t, stats.InviteRankings, 1)
	assert.Equal(t, 999_999, stats.InviteRankings[0].UserId)
	assert.Empty(t, stats.InviteRankings[0].Username)
	require.Len(t, stats.RecentInvitees, 1)
	assert.Equal(t, invitee.Id, stats.RecentInvitees[0].InviteeId)
	assert.Empty(t, stats.RecentInvitees[0].InviterUsername)
}

func TestInviteAdminStatsCountsTopupRebateWithoutSuccessfulTopUp(t *testing.T) {
	useInviteRebateDB(t)

	inviter := createInviteTestUser(t, "promoter", 0)
	invitee := createInviteTestUser(t, "alice", inviter.Id)
	createInviteRebateRecord(t, inviter.Id, invitee.Id, InviteRebateTypeTopup, 40)

	stats, err := GetInviteAdminStats(0)
	require.NoError(t, err)

	assert.Equal(t, int64(1), stats.Summary.RechargedCount)
	require.Len(t, stats.RecentInvitees, 1)
	assert.Equal(t, InviteeStatusRecharged, stats.RecentInvitees[0].Status)
}

func TestInviteAdminStatsClampsRankLimit(t *testing.T) {
	useInviteRebateDB(t)

	for i := 0; i < 3; i++ {
		inviter := createInviteTestUser(t, fmt.Sprintf("inviter-%d", i), 0)
		invitee := createInviteTestUser(t, fmt.Sprintf("invitee-%d", i), inviter.Id)
		createInviteRebateRecord(t, inviter.Id, invitee.Id, InviteRebateTypeRegister, 10*(i+1))
	}

	limited, err := GetInviteAdminStats(1)
	require.NoError(t, err)
	require.Len(t, limited.InviteRankings, 1)
	require.Len(t, limited.RebateRankings, 1)
	assert.Equal(t, int64(3), limited.Summary.InviterCount)

	clamped, err := GetInviteAdminStats(100)
	require.NoError(t, err)
	require.Len(t, clamped.InviteRankings, 3)
	require.Len(t, clamped.RebateRankings, 3)
}
