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
import { describe, expect, test } from 'vitest'

import zhTW from '@/i18n/locales/zh-TW.json'
import zh from '@/i18n/locales/zh.json'

const BILLING_CORRECTION_KEYS = [
  'Billing Correction',
  'Add Correction Rule',
  'Edit Correction Rule',
  'Input token threshold',
  'No billing correction rules yet',
  'Delete this billing correction rule?',
  'A request on a channel with an enabled rule is not charged when its input tokens are strictly greater than the threshold, it produced zero output tokens, and the upstream returned no usage information, so tokens were estimated locally.',
  'Only requests made after a rule is enabled are affected; existing logs are not changed.',
  'Threshold must be between {{min}} and {{max}} tokens',
  'Waived',
  'Charge waived',
  'Not charged: the upstream returned no output and no usage information.',
  'Estimated input tokens',
  'Original cost',
  'Rule ID',
] as const

describe('billing correction Chinese locales', () => {
  test('zh and zh-TW translate the feature keys instead of falling back to English keys', () => {
    for (const key of BILLING_CORRECTION_KEYS) {
      expect(zh.translation[key], `zh is missing "${key}"`).toEqual(
        expect.any(String)
      )
      expect(zh.translation[key], `zh left "${key}" in English`).not.toBe(key)

      expect(zhTW.translation[key], `zh-TW is missing "${key}"`).toEqual(
        expect.any(String)
      )
      expect(zhTW.translation[key], `zh-TW left "${key}" in English`).not.toBe(
        key
      )
    }
  })

  test('the range message keeps its interpolation placeholders in every Chinese locale', () => {
    const key = 'Threshold must be between {{min}} and {{max}} tokens'

    for (const translation of [zh.translation[key], zhTW.translation[key]]) {
      expect(translation).toContain('{{min}}')
      expect(translation).toContain('{{max}}')
    }
  })

  test('zh uses the agreed names for the feature and the waived marker', () => {
    expect(zh.translation['Billing Correction']).toBe('计费修正')
    expect(zh.translation['Waived']).toBe('已免扣费')
  })
})
