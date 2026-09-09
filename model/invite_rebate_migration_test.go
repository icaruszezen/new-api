package model

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// useInviteRebateMigrationEnv isolates both the option table and the in-memory
// option map, because the migration writes through UpdateOption.
func useInviteRebateMigrationEnv(t *testing.T) {
	t.Helper()
	useInviteRebateDB(t)

	previousOptionMap := common.OptionMap
	previousSetting := *operation_setting.GetInviteRebateSetting()
	previousQuotaPerUnit := common.QuotaPerUnit
	previousLegacyInviter := common.QuotaForInviter
	general := operation_setting.GetGeneralSetting()
	previousDisplayType := general.QuotaDisplayType

	common.OptionMap = map[string]string{}
	common.QuotaPerUnit = 500_000
	general.QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD

	t.Cleanup(func() {
		common.OptionMap = previousOptionMap
		*operation_setting.GetInviteRebateSetting() = previousSetting
		common.QuotaPerUnit = previousQuotaPerUnit
		common.QuotaForInviter = previousLegacyInviter
		general.QuotaDisplayType = previousDisplayType
	})
}

// 升级到新返利机制时，老站点配置的 QuotaForInviter 必须换算成展示币种的注册返利金额，
// 否则邀请人会在升级后突然停发返利。
func TestMigrateInviteRebateSettingsSeedsRegisterAmountFromLegacyQuota(t *testing.T) {
	useInviteRebateMigrationEnv(t)
	require.NoError(t, DB.Create(&Option{Key: "QuotaForInviter", Value: "1000000"}).Error)

	require.NoError(t, MigrateInviteRebateSettings())

	var stored Option
	require.NoError(t, DB.Where(&Option{Key: operation_setting.InviteRebateRegisterAmountOptionKey}).First(&stored).Error)
	amount, err := strconv.ParseFloat(stored.Value, 64)
	require.NoError(t, err)
	assert.InDelta(t, 2.0, amount, 1e-9)
	assert.InDelta(t, 2.0, operation_setting.GetInviteRebateSetting().RegisterAmount, 1e-9)
	// 充值返利没有旧配置，保持关闭等管理员显式开启。
	assert.Zero(t, operation_setting.GetInviteRebateSetting().TopupAmount)
}

// 管理员保存过的值就是权威值：把注册返利设为 0（明确关闭）后重启不能被旧配置覆盖。
func TestMigrateInviteRebateSettingsKeepsExplicitAdminValue(t *testing.T) {
	useInviteRebateMigrationEnv(t)
	require.NoError(t, DB.Create(&Option{Key: "QuotaForInviter", Value: "1000000"}).Error)
	require.NoError(t, DB.Create(&Option{
		Key:   operation_setting.InviteRebateRegisterAmountOptionKey,
		Value: "0",
	}).Error)

	require.NoError(t, MigrateInviteRebateSettings())

	var stored Option
	require.NoError(t, DB.Where(&Option{Key: operation_setting.InviteRebateRegisterAmountOptionKey}).First(&stored).Error)
	assert.Equal(t, "0", stored.Value)
}

func TestMigrateInviteRebateSettingsSkipsWhenLegacyRewardWasDisabled(t *testing.T) {
	useInviteRebateMigrationEnv(t)
	common.QuotaForInviter = 0
	require.NoError(t, DB.Create(&Option{Key: "QuotaForInviter", Value: "0"}).Error)

	require.NoError(t, MigrateInviteRebateSettings())

	var count int64
	require.NoError(t, DB.Model(&Option{}).
		Where(&Option{Key: operation_setting.InviteRebateRegisterAmountOptionKey}).
		Count(&count).Error)
	assert.Zero(t, count)
}

func TestMigrateInviteRebateSettingsIsIdempotent(t *testing.T) {
	useInviteRebateMigrationEnv(t)
	require.NoError(t, DB.Create(&Option{Key: "QuotaForInviter", Value: "1000000"}).Error)

	require.NoError(t, MigrateInviteRebateSettings())
	first, err := AllOption()
	require.NoError(t, err)

	require.NoError(t, MigrateInviteRebateSettings())
	second, err := AllOption()
	require.NoError(t, err)

	assert.ElementsMatch(t, first, second)
}
