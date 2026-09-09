package operation_setting

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

// InviteRebateSetting 邀请返利配置。三个金额都以站点展示币种计价（USD 展示为美元，
// CNY 展示为人民币，TOKENS 展示按美元），只在发放那一刻换算成额度，因此管理员之后
// 调整汇率不会改写已发放的历史记录。
type InviteRebateSetting struct {
	// RegisterAmount 好友注册成功后给邀请人的返利，0 表示关闭注册返利。
	RegisterAmount float64 `json:"register_amount"`
	// TopupAmount 好友累计充值达标后给邀请人的返利，0 表示关闭充值返利。
	TopupAmount float64 `json:"topup_amount"`
	// TopupThreshold 触发充值返利所需的累计充值金额。
	TopupThreshold float64 `json:"topup_threshold"`
}

const (
	InviteRebateRegisterAmountOptionKey = "invite_rebate_setting.register_amount"
	InviteRebateTopupAmountOptionKey    = "invite_rebate_setting.topup_amount"
	InviteRebateTopupThresholdOptionKey = "invite_rebate_setting.topup_threshold"

	// MaxInviteRebateAmount 限制单笔返利与充值门槛的录入上限。返利金额会被换算成
	// 32 位的额度列，没有上限的输入可以直接把额度推到 int32 饱和边界。
	MaxInviteRebateAmount = 1_000_000
)

var inviteRebateSetting = InviteRebateSetting{}

func init() {
	config.GlobalConfig.Register("invite_rebate_setting", &inviteRebateSetting)
}

func GetInviteRebateSetting() *InviteRebateSetting {
	return &inviteRebateSetting
}

// IsInviteRebateAmountOptionKey 判断 key 是否为邀请返利的金额配置项。
func IsInviteRebateAmountOptionKey(key string) bool {
	switch key {
	case InviteRebateRegisterAmountOptionKey, InviteRebateTopupAmountOptionKey, InviteRebateTopupThresholdOptionKey:
		return true
	}
	return false
}

// ValidateInviteRebateAmount 校验管理员录入的返利金额。返利金额是计费乘数，必须在
// 进入额度换算前被限定范围，负数与超大值都要在设置层直接拒绝。
func ValidateInviteRebateAmount(value string) error {
	amount, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return errors.New("邀请返利金额必须是数字")
	}
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return errors.New("邀请返利金额必须是有效数字")
	}
	if amount < 0 {
		return errors.New("邀请返利金额不能为负数")
	}
	if amount > MaxInviteRebateAmount {
		return fmt.Errorf("邀请返利金额不能超过 %d", MaxInviteRebateAmount)
	}
	return nil
}

// ValidateInviteRebateTopupPair 在开启充值返利时要求门槛能换算出正额度。尘埃门槛
// 会让 InviteRebateQuota(threshold) 变成 0，发放侧的「累计 < 门槛」判断会恒为假，
// 任意一笔成功充值都会发奖。
func ValidateInviteRebateTopupPair(topupAmount, threshold float64) error {
	if topupAmount <= 0 {
		return nil
	}
	if InviteRebateQuota(threshold) <= 0 {
		return errors.New("开启充值返利时，充值门槛换算后的额度必须大于 0")
	}
	return nil
}

// ValidateInviteRebateSettingUpdate 把即将写入的金额叠到当前内存配置上，再校验
// 充值返利档位。单键保存与批量保存都走这里，避免先写返利金额、门槛仍为 0 时漏检。
func ValidateInviteRebateSettingUpdate(values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	current := GetInviteRebateSetting()
	topupAmount := current.TopupAmount
	threshold := current.TopupThreshold
	sawInviteKey := false
	for key, value := range values {
		if !IsInviteRebateAmountOptionKey(key) {
			continue
		}
		sawInviteKey = true
		if err := ValidateInviteRebateAmount(value); err != nil {
			return err
		}
		amount, _ := strconv.ParseFloat(value, 64)
		switch key {
		case InviteRebateTopupAmountOptionKey:
			topupAmount = amount
		case InviteRebateTopupThresholdOptionKey:
			threshold = amount
		}
	}
	if !sawInviteKey {
		return nil
	}
	return ValidateInviteRebateTopupPair(topupAmount, threshold)
}

// InviteRebateCurrencyRate 返回 1 USD 折合多少展示币种，用于返利金额与额度互转。
func InviteRebateCurrencyRate() float64 {
	rate := GetUsdToCurrencyRate(USDExchangeRate)
	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		return 1
	}
	return rate
}

// InviteRebateQuota 把展示币种金额换算成额度。金额已在设置层被限定在
// [0, MaxInviteRebateAmount]，这里再走统一的饱和转换以防汇率或 QuotaPerUnit 异常。
func InviteRebateQuota(amount float64) int {
	if amount <= 0 || math.IsNaN(amount) {
		return 0
	}
	if amount > MaxInviteRebateAmount {
		amount = MaxInviteRebateAmount
	}
	quota := common.QuotaFromFloat(amount / InviteRebateCurrencyRate() * common.QuotaPerUnit)
	if quota < 0 {
		return 0
	}
	return quota
}

// InviteRebateAmountUSD 把展示币种金额换算成美元，用于发放记录的审计字段。
func InviteRebateAmountUSD(amount float64) float64 {
	if amount <= 0 || math.IsNaN(amount) {
		return 0
	}
	return amount / InviteRebateCurrencyRate()
}

// InviteRebateAmountFromQuota 把额度换算回展示币种金额，供旧 QuotaForInviter 迁移出
// 默认值使用。
func InviteRebateAmountFromQuota(quota int) float64 {
	if quota <= 0 || common.QuotaPerUnit <= 0 {
		return 0
	}
	amount := float64(quota) / common.QuotaPerUnit * InviteRebateCurrencyRate()
	if amount > MaxInviteRebateAmount {
		return MaxInviteRebateAmount
	}
	return amount
}
