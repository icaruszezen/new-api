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
    name: 'channel-retry',
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
    name: 'channel-retry',
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

function savedSetting(form: ReturnType<typeof validForm>) {
  return JSON.parse(buildSettingJSON(form)) as Record<string, unknown>
}

function issuesAt(form: ReturnType<typeof validForm>, field: string) {
  const result = channelFormSchema.safeParse(form)
  if (result.success) return []
  return result.error.issues
    .filter((issue) => issue.path[0] === field)
    .map((issue) => issue.message)
}

describe('channel form same-channel retry', () => {
  test('new channel starts with retry disabled and saves no retry fields', () => {
    expect(CHANNEL_FORM_DEFAULT_VALUES.retry_times).toBeUndefined()
    expect(CHANNEL_FORM_DEFAULT_VALUES.retry_status_codes).toBe('429,503')
    expect(CHANNEL_FORM_DEFAULT_VALUES.retry_interval_ms).toBe(200)

    const saved = savedSetting(validForm())

    expect(saved).not.toHaveProperty('retry_times')
    expect(saved).not.toHaveProperty('retry_status_codes')
    expect(saved).not.toHaveProperty('retry_interval_ms')
  })

  test('zero retry times omits the status codes and interval', () => {
    const saved = savedSetting(
      validForm({
        retry_times: 0,
        retry_status_codes: '500-599',
        retry_interval_ms: 1000,
      })
    )

    expect(saved).not.toHaveProperty('retry_times')
    expect(saved).not.toHaveProperty('retry_status_codes')
    expect(saved).not.toHaveProperty('retry_interval_ms')
  })

  test('cleared retry times disables retry', () => {
    const saved = savedSetting(
      validForm({
        retry_times: null,
        retry_status_codes: '500-599',
        retry_interval_ms: 1000,
      })
    )

    expect(saved).not.toHaveProperty('retry_times')
    expect(saved).not.toHaveProperty('retry_status_codes')
    expect(saved).not.toHaveProperty('retry_interval_ms')
  })

  test('channel without retry settings loads the documented defaults', () => {
    const form = transformChannelToFormDefaults(
      channelWithSetting(JSON.stringify({ force_format: false }))
    )

    expect(form.retry_times).toBeUndefined()
    expect(form.retry_status_codes).toBe('429,503')
    expect(form.retry_interval_ms).toBe(200)
  })

  test('configured retry policy round-trips through load and save', () => {
    const form = transformChannelToFormDefaults(
      channelWithSetting(
        JSON.stringify({
          retry_times: 3,
          retry_status_codes: '500-599',
          retry_interval_ms: 1000,
        })
      )
    )

    expect(form.retry_times).toBe(3)
    expect(form.retry_status_codes).toBe('500-599')
    expect(form.retry_interval_ms).toBe(1000)

    const saved = savedSetting(form)
    expect(saved.retry_times).toBe(3)
    expect(saved.retry_status_codes).toBe('500-599')
    expect(saved.retry_interval_ms).toBe(1000)
  })

  test('explicit zero retry interval is kept instead of the 200 ms default', () => {
    const form = transformChannelToFormDefaults(
      channelWithSetting(
        JSON.stringify({ retry_times: 2, retry_interval_ms: 0 })
      )
    )

    expect(form.retry_interval_ms).toBe(0)
    expect(savedSetting(form).retry_interval_ms).toBe(0)
  })

  test('status codes are saved in normalized form', () => {
    const saved = savedSetting(
      validForm({
        retry_times: 1,
        retry_status_codes: '503，429, 500-599',
      })
    )

    expect(saved.retry_status_codes).toBe('429,500-599')
  })

  test('empty status codes and interval are omitted so the backend defaults apply', () => {
    const saved = savedSetting(
      validForm({
        retry_times: 2,
        retry_status_codes: '  ',
        // a cleared number input stores null
        retry_interval_ms: null,
      })
    )

    expect(saved.retry_times).toBe(2)
    expect(saved).not.toHaveProperty('retry_status_codes')
    expect(saved).not.toHaveProperty('retry_interval_ms')
  })

  test.each([
    { retryTimes: undefined, valid: true },
    { retryTimes: null, valid: true },
    { retryTimes: 0, valid: true },
    { retryTimes: 10, valid: true },
    { retryTimes: 11, valid: false },
    { retryTimes: -1, valid: false },
    { retryTimes: 1.5, valid: false },
  ])('retry times $retryTimes is accepted: $valid', ({ retryTimes, valid }) => {
    expect(
      issuesAt(validForm({ retry_times: retryTimes }), 'retry_times')
    ).toEqual(valid ? [] : ['Retry times must be an integer between 0 and 10'])
  })

  test.each([
    { retryIntervalMs: undefined, valid: true },
    { retryIntervalMs: 0, valid: true },
    { retryIntervalMs: 60000, valid: true },
    { retryIntervalMs: 60001, valid: false },
    { retryIntervalMs: -1, valid: false },
    { retryIntervalMs: 0.5, valid: false },
  ])(
    'retry interval $retryIntervalMs ms is accepted: $valid',
    ({ retryIntervalMs, valid }) => {
      expect(
        issuesAt(
          validForm({ retry_times: 1, retry_interval_ms: retryIntervalMs }),
          'retry_interval_ms'
        )
      ).toEqual(
        valid
          ? []
          : ['Retry interval must be an integer between 0 and 60000 ms']
      )
    }
  )

  test('invalid status codes are rejected with the offending tokens when retry is enabled', () => {
    expect(
      issuesAt(
        validForm({ retry_times: 1, retry_status_codes: '429,abc,700' }),
        'retry_status_codes'
      )
    ).toEqual(['Invalid status code rules: abc, 700'])
  })

  test('status codes and interval are not validated while retry is disabled', () => {
    const result = channelFormSchema.safeParse(
      validForm({
        retry_times: undefined,
        retry_status_codes: 'abc',
        retry_interval_ms: -1,
      })
    )

    expect(result.success).toBe(true)
  })
})
