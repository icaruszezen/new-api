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
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"gorm.io/gorm"
)

// MigrateInviteRebateSettings 为已上线站点写出邀请返利的初始注册返利金额。旧配置
// QuotaForInviter 以额度计价，新配置以站点展示币种计价，缺少这一步升级后邀请人会突然
// 停发注册返利。只有新配置行完全不存在时才写入，所以管理员之后保存 0 就是明确关闭。
//
// 充值返利没有对应的旧配置，保持 0（关闭），由管理员显式开启。
//
// 必须在 InitOptionMap 之后调用：金额换算依赖已从数据库加载的展示币种与汇率配置。
func MigrateInviteRebateSettings() error {
	if DB == nil {
		return errors.New("database is not initialized")
	}

	var count int64
	if err := DB.Model(&Option{}).
		Where(&Option{Key: operation_setting.InviteRebateRegisterAmountOptionKey}).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	legacyQuota, err := legacyInviterQuota()
	if err != nil {
		return err
	}
	amount := operation_setting.InviteRebateAmountFromQuota(legacyQuota)
	if amount <= 0 {
		return nil
	}

	// 走 UpdateOption 而不是直接写表，这样内存中的 OptionMap 与已注册配置同步生效，
	// 迁移后无需重启即可发放返利。
	return UpdateOption(
		operation_setting.InviteRebateRegisterAmountOptionKey,
		strconv.FormatFloat(amount, 'f', -1, 64),
	)
}

// legacyInviterQuota 读取旧的 QuotaForInviter 额度。数据库没有该行时回落到编译期默认
// 值，与 InitOptionMap 的取值顺序保持一致。
func legacyInviterQuota() (int, error) {
	var option Option
	err := DB.Where(&Option{Key: "QuotaForInviter"}).First(&option).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return common.QuotaForInviter, nil
	}
	if err != nil {
		return 0, err
	}
	quota, convErr := strconv.Atoi(option.Value)
	if convErr != nil {
		return 0, nil
	}
	return quota, nil
}
