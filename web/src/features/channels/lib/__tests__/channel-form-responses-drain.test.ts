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
  transformChannelToFormDefaults,
} from '../channel-form'

function channelWithSetting(setting: string): Channel {
  return {
    id: 1,
    type: 1,
    key: '',
    status: 1,
    name: 'responses-drain',
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

describe('channel form responses client disconnect drain', () => {
  test('defaults off and omits the setting from saved JSON', () => {
    expect(
      CHANNEL_FORM_DEFAULT_VALUES.responses_client_disconnect_drain_enabled
    ).toBe(false)

    const saved = JSON.parse(
      buildSettingJSON({
        ...CHANNEL_FORM_DEFAULT_VALUES,
        name: 'responses-drain',
        key: 'test-key',
        models: 'gpt-4',
      })
    ) as Record<string, unknown>

    expect(saved).not.toHaveProperty(
      'responses_client_disconnect_drain_enabled'
    )
  })

  test('round-trips an enabled drain setting', () => {
    const form = transformChannelToFormDefaults(
      channelWithSetting(
        JSON.stringify({
          responses_client_disconnect_drain_enabled: true,
        })
      )
    )

    expect(form.responses_client_disconnect_drain_enabled).toBe(true)

    const saved = JSON.parse(buildSettingJSON(form)) as Record<string, unknown>
    expect(saved.responses_client_disconnect_drain_enabled).toBe(true)
  })
})
