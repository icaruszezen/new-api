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

import type { Channel } from '../../types'
import {
  CHANNEL_FORM_DEFAULT_VALUES,
  buildSettingJSON,
  channelFormSchema,
  transformChannelToFormDefaults,
} from '../channel-form'

function validForm(
  overrides: Partial<typeof CHANNEL_FORM_DEFAULT_VALUES> = {}
) {
  return {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'cache-billing',
    key: 'test-key',
    models: 'gpt-4',
    ...overrides,
  }
}

function channelWithSetting(setting: string): Channel {
  return {
    id: 1,
    type: 1,
    key: '',
    status: 1,
    name: 'legacy-cache',
    created_time: 0,
    test_time: 0,
    response_time: 0,
    other: '',
    balance: 0,
    balance_updated_time: 0,
    models: 'gpt-4',
    group: 'default',
    used_quota: 0,
    other_info: '',
    remark: '',
    max_input_tokens: 0,
    setting,
    settings: '{}',
    channel_info: {
      is_multi_key: false,
      multi_key_size: 0,
      multi_key_polling_index: 0,
      multi_key_mode: 'random',
    },
  }
}

describe('channel form cache billing ratio', () => {
  test('defaults to fixed mode without a range switch', () => {
    expect(CHANNEL_FORM_DEFAULT_VALUES.cache_billing_ratio_range).toBe(false)
    expect(CHANNEL_FORM_DEFAULT_VALUES.cache_billing_ratio).toBe(1)
  })

  test('accepts a legacy fixed ratio when range mode is off', () => {
    const result = channelFormSchema.safeParse(
      validForm({
        cache_billing_ratio_enabled: true,
        cache_billing_ratio: 0.8,
      })
    )

    expect(result.success).toBe(true)
  })

  test('accepts an inclusive range when range mode is on', () => {
    const result = channelFormSchema.safeParse(
      validForm({
        cache_billing_ratio_enabled: true,
        cache_billing_ratio_range: true,
        cache_billing_ratio_min: 0.95,
        cache_billing_ratio_max: 0.99,
      })
    )

    expect(result.success).toBe(true)
  })

  test('rejects a range whose min exceeds max', () => {
    const result = channelFormSchema.safeParse(
      validForm({
        cache_billing_ratio_enabled: true,
        cache_billing_ratio_range: true,
        cache_billing_ratio_min: 0.99,
        cache_billing_ratio_max: 0.95,
      })
    )

    expect(result.success).toBe(false)
    if (!result.success) {
      expect(
        result.error.issues.some(
          (issue) =>
            issue.path[0] === 'cache_billing_ratio_min' &&
            issue.message === 'Cache billing ratio min must not exceed max'
        )
      ).toBe(true)
    }
  })

  test('loads legacy setting JSON without a range field as fixed mode', () => {
    const form = transformChannelToFormDefaults(
      channelWithSetting(
        JSON.stringify({
          cache_billing_ratio_enabled: true,
          cache_billing_ratio: 0.8,
        })
      )
    )

    expect(form.cache_billing_ratio_enabled).toBe(true)
    expect(form.cache_billing_ratio_range).toBe(false)
    expect(form.cache_billing_ratio).toBe(0.8)

    const saved = JSON.parse(buildSettingJSON(form)) as Record<string, unknown>
    expect(saved.cache_billing_ratio_enabled).toBe(true)
    expect(saved.cache_billing_ratio).toBe(0.8)
    expect(saved).not.toHaveProperty('cache_billing_ratio_range')
    expect(saved).not.toHaveProperty('cache_billing_ratio_min')
    expect(saved).not.toHaveProperty('cache_billing_ratio_max')
  })

  test('round-trips an explicit range setting', () => {
    const form = transformChannelToFormDefaults(
      channelWithSetting(
        JSON.stringify({
          cache_billing_ratio_enabled: true,
          cache_billing_ratio_range: true,
          cache_billing_ratio_min: 0.95,
          cache_billing_ratio_max: 0.99,
        })
      )
    )

    expect(form.cache_billing_ratio_enabled).toBe(true)
    expect(form.cache_billing_ratio_range).toBe(true)
    expect(form.cache_billing_ratio_min).toBe(0.95)
    expect(form.cache_billing_ratio_max).toBe(0.99)

    const saved = JSON.parse(buildSettingJSON(form)) as Record<string, unknown>
    expect(saved.cache_billing_ratio_range).toBe(true)
    expect(saved.cache_billing_ratio_min).toBe(0.95)
    expect(saved.cache_billing_ratio_max).toBe(0.99)
    expect(saved).not.toHaveProperty('cache_billing_ratio')
  })
})
