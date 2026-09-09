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
import { getCurrencyDisplay } from '@/lib/currency'

import type { InviteRebateRules } from '../types'

/**
 * Upper bound the backend enforces on every rebate amount
 * (`operation_setting.MaxInviteRebateAmount`). Mirrored here so the admin form
 * rejects out-of-range input before it reaches quota conversion.
 */
export const MAX_INVITE_REBATE_AMOUNT = 1_000_000

/**
 * Symbol used for rebate amounts. Amounts are stored in the site display
 * currency, so they must never be run through an exchange rate again. Token-only
 * sites fall back to the dollar sign, matching how the backend treats them.
 */
export function getInviteRebateCurrencySymbol(): string {
  const { meta } = getCurrencyDisplay()
  if (meta.kind === 'tokens') return '$'
  return meta.symbol
}

export function formatInviteRebateAmount(
  amount: number,
  symbol: string
): string {
  if (!Number.isFinite(amount)) return `${symbol}0.00`
  return `${symbol}${amount.toFixed(2)}`
}

/**
 * Rule chips shared by the user referral page and the admin preview, so both
 * always describe the payout rules with the same wording. A tier whose amount is
 * 0 is disabled and produces no label.
 */
export function buildInviteRebateRuleLabels(
  rules: InviteRebateRules,
  symbol: string,
  t: (key: string, options?: Record<string, unknown>) => string
): string[] {
  const labels: string[] = []

  if (rules.register_amount > 0) {
    labels.push(
      t('Sign-up rebate {{amount}}', {
        amount: formatInviteRebateAmount(rules.register_amount, symbol),
      })
    )
  }

  if (rules.topup_amount > 0 && rules.topup_threshold > 0) {
    labels.push(
      t('Top up {{threshold}} to earn {{amount}}', {
        threshold: formatInviteRebateAmount(rules.topup_threshold, symbol),
        amount: formatInviteRebateAmount(rules.topup_amount, symbol),
      })
    )
  }

  return labels
}

export function isInviteRebateActive(rules: InviteRebateRules): boolean {
  if (rules.register_amount > 0) return true
  return rules.topup_amount > 0 && rules.topup_threshold > 0
}
