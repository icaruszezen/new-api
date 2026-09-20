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

const OVERRIDE_PAGE_KEYS = [
  'Error Message Override',
  'Add Override Rule',
  'Match text',
  'No override rules yet',
  'When an upstream error message contains the match text, the whole message returned to the user is replaced.',
] as const

describe('error message override Chinese locales', () => {
  test('zh and zh-TW translate the page chrome instead of falling back to English keys', () => {
    for (const key of OVERRIDE_PAGE_KEYS) {
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
})
