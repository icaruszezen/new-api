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
    name: 'client-disconnect-drain',
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

const LEGACY_KEY = 'responses_client_disconnect_drain_enabled'

// Loads a saved setting into the form and builds the JSON the form would save.
function resaveChannelSetting(setting: string): Record<string, unknown> {
  const form = transformChannelToFormDefaults(channelWithSetting(setting))
  return JSON.parse(buildSettingJSON(form)) as Record<string, unknown>
}

describe('channel form client disconnect drain', () => {
  test('defaults off and omits both keys from saved JSON', () => {
    expect(CHANNEL_FORM_DEFAULT_VALUES.client_disconnect_drain_enabled).toBe(
      false
    )

    const saved = JSON.parse(
      buildSettingJSON({
        ...CHANNEL_FORM_DEFAULT_VALUES,
        name: 'client-disconnect-drain',
        key: 'test-key',
        models: 'gpt-4',
      })
    ) as Record<string, unknown>

    expect(saved).not.toHaveProperty('client_disconnect_drain_enabled')
    expect(saved).not.toHaveProperty(LEGACY_KEY)
  })

  test('loads a channel saved with the legacy Responses-only key as on', () => {
    const form = transformChannelToFormDefaults(
      channelWithSetting(JSON.stringify({ [LEGACY_KEY]: true }))
    )

    expect(form.client_disconnect_drain_enabled).toBe(true)
  })

  test('saving a channel with the legacy key writes the new key and drops the legacy one', () => {
    const saved = resaveChannelSetting(JSON.stringify({ [LEGACY_KEY]: true }))

    expect(saved.client_disconnect_drain_enabled).toBe(true)
    expect(saved).not.toHaveProperty(LEGACY_KEY)
  })

  test('round-trips an enabled drain setting under the new key', () => {
    const saved = resaveChannelSetting(
      JSON.stringify({ client_disconnect_drain_enabled: true })
    )

    expect(saved.client_disconnect_drain_enabled).toBe(true)
  })

  test('switching off a legacy-enabled channel omits both keys so the backend stops draining', () => {
    const form = transformChannelToFormDefaults(
      channelWithSetting(JSON.stringify({ [LEGACY_KEY]: true }))
    )

    const saved = JSON.parse(
      buildSettingJSON({ ...form, client_disconnect_drain_enabled: false })
    ) as Record<string, unknown>

    expect(saved).not.toHaveProperty('client_disconnect_drain_enabled')
    expect(saved).not.toHaveProperty(LEGACY_KEY)
  })
})
