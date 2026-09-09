package operation_setting

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withCurrencyDisplay(t *testing.T, displayType string, usdExchangeRate float64, quotaPerUnit float64) {
	t.Helper()
	general := GetGeneralSetting()
	previousDisplayType := general.QuotaDisplayType
	previousUSDRate := USDExchangeRate
	previousQuotaPerUnit := common.QuotaPerUnit

	general.QuotaDisplayType = displayType
	USDExchangeRate = usdExchangeRate
	common.QuotaPerUnit = quotaPerUnit

	t.Cleanup(func() {
		general.QuotaDisplayType = previousDisplayType
		USDExchangeRate = previousUSDRate
		common.QuotaPerUnit = previousQuotaPerUnit
	})
}

// 管理员录入的金额是计费乘数，必须在换算成额度前被限定范围：负数、非数字与超大值都要
// 在设置层直接拒绝，不能进入额度转换。
func TestValidateInviteRebateAmount(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "zero disables the tier", value: "0"},
		{name: "decimal amount", value: "2.5"},
		{name: "upper bound", value: "1000000"},
		{name: "above upper bound", value: "1000000.01", wantErr: true},
		{name: "negative", value: "-1", wantErr: true},
		{name: "not a number", value: "abc", wantErr: true},
		{name: "empty", value: "", wantErr: true},
		{name: "NaN", value: "NaN", wantErr: true},
		{name: "infinity", value: "Inf", wantErr: true},
		{name: "wrapped unsigned", value: "18446744073686646784", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateInviteRebateAmount(test.value)
			if test.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestValidateInviteRebateTopupPairRejectsDustThreshold(t *testing.T) {
	withCurrencyDisplay(t, QuotaDisplayTypeUSD, 7.3, 500_000)

	assert.NoError(t, ValidateInviteRebateTopupPair(0, 1e-10))
	assert.NoError(t, ValidateInviteRebateTopupPair(5, 50))
	assert.Error(t, ValidateInviteRebateTopupPair(5, 0))
	assert.Error(t, ValidateInviteRebateTopupPair(5, 1e-10))
}

func TestValidateInviteRebateSettingUpdateMergesIncomingKeys(t *testing.T) {
	withCurrencyDisplay(t, QuotaDisplayTypeUSD, 7.3, 500_000)
	setting := GetInviteRebateSetting()
	previous := *setting
	setting.TopupAmount = 0
	setting.TopupThreshold = 0
	t.Cleanup(func() { *setting = previous })

	require.NoError(t, ValidateInviteRebateSettingUpdate(map[string]string{
		InviteRebateTopupAmountOptionKey:    "5",
		InviteRebateTopupThresholdOptionKey: "50",
	}))
	require.Error(t, ValidateInviteRebateSettingUpdate(map[string]string{
		InviteRebateTopupAmountOptionKey: "5",
	}))

	setting.TopupAmount = 5
	setting.TopupThreshold = 50
	require.Error(t, ValidateInviteRebateSettingUpdate(map[string]string{
		InviteRebateTopupThresholdOptionKey: "0.0000000001",
	}))
	require.NoError(t, ValidateInviteRebateSettingUpdate(map[string]string{
		InviteRebateRegisterAmountOptionKey: "2",
	}))
}

func TestIsInviteRebateAmountOptionKey(t *testing.T) {
	assert.True(t, IsInviteRebateAmountOptionKey(InviteRebateRegisterAmountOptionKey))
	assert.True(t, IsInviteRebateAmountOptionKey(InviteRebateTopupAmountOptionKey))
	assert.True(t, IsInviteRebateAmountOptionKey(InviteRebateTopupThresholdOptionKey))
	assert.False(t, IsInviteRebateAmountOptionKey("QuotaForInviter"))
	assert.False(t, IsInviteRebateAmountOptionKey("invite_rebate_setting"))
}

// 返利金额按站点展示币种录入，换算成额度时必须先折回美元，否则 CNY 站点会按人民币数字
// 直接发美元额度。
func TestInviteRebateQuotaConvertsFromDisplayCurrency(t *testing.T) {
	t.Run("USD display treats the amount as dollars", func(t *testing.T) {
		withCurrencyDisplay(t, QuotaDisplayTypeUSD, 7.3, 500_000)
		assert.Equal(t, 1_000_000, InviteRebateQuota(2))
	})

	t.Run("CNY display divides by the exchange rate", func(t *testing.T) {
		withCurrencyDisplay(t, QuotaDisplayTypeCNY, 7.3, 500_000)
		assert.Equal(t, common.QuotaFromFloat(2/7.3*500_000), InviteRebateQuota(2))
	})

	t.Run("token display treats the amount as dollars", func(t *testing.T) {
		withCurrencyDisplay(t, QuotaDisplayTypeTokens, 7.3, 500_000)
		assert.Equal(t, 1_000_000, InviteRebateQuota(2))
	})
}

// 汇率被配置成 0 或负数时必须回落到 1，不能产生除零、无穷大或负额度。
func TestInviteRebateQuotaSurvivesBrokenExchangeRate(t *testing.T) {
	withCurrencyDisplay(t, QuotaDisplayTypeCNY, 0, 500_000)

	assert.Equal(t, 1.0, InviteRebateCurrencyRate())
	assert.Equal(t, 1_000_000, InviteRebateQuota(2))
}

func TestInviteRebateQuotaRejectsNonPositiveAndUnboundedAmounts(t *testing.T) {
	withCurrencyDisplay(t, QuotaDisplayTypeUSD, 7.3, 500_000)

	assert.Zero(t, InviteRebateQuota(0))
	assert.Zero(t, InviteRebateQuota(-5))
	assert.Zero(t, InviteRebateQuota(math.NaN()))

	// 超出录入上限的输入被夹到上限后再换算，绝不能返回负额度。
	clamped := InviteRebateQuota(math.MaxFloat64)
	assert.Positive(t, clamped)
	assert.LessOrEqual(t, clamped, common.MaxQuota)
}

func TestInviteRebateAmountFromQuotaRoundTripsLegacyInviterQuota(t *testing.T) {
	withCurrencyDisplay(t, QuotaDisplayTypeCNY, 7.3, 500_000)

	amount := InviteRebateAmountFromQuota(500_000)
	require.InDelta(t, 7.3, amount, 1e-9)
	assert.Equal(t, 500_000, InviteRebateQuota(amount))
}

func TestInviteRebateAmountFromQuotaHandlesEmptyLegacySetting(t *testing.T) {
	withCurrencyDisplay(t, QuotaDisplayTypeUSD, 7.3, 500_000)

	assert.Zero(t, InviteRebateAmountFromQuota(0))
	assert.Zero(t, InviteRebateAmountFromQuota(-1))
}
