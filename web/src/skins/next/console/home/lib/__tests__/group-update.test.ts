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

import type { ApiKey } from '@/features/keys/types'

import { buildApiKeyGroupUpdatePayload } from '../group-update'

const sampleKey: ApiKey = {
  id: 7,
  name: 'plus',
  key: 'd7af123443e0',
  status: 1,
  remain_quota: 120,
  used_quota: 10,
  unlimited_quota: false,
  expired_time: 99,
  created_time: 1,
  accessed_time: 0,
  group: 'special',
  auto_groups: ['vip'],
  cross_group_retry: true,
  model_limits_enabled: true,
  model_limits: 'gpt-4',
  allow_ips: '127.0.0.1',
}

describe('buildApiKeyGroupUpdatePayload', () => {
  test('keeps the existing key fields when switching to another group', () => {
    expect(buildApiKeyGroupUpdatePayload(sampleKey, 'vip')).toEqual({
      id: 7,
      name: 'plus',
      remain_quota: 120,
      expired_time: 99,
      unlimited_quota: false,
      model_limits_enabled: true,
      model_limits: 'gpt-4',
      allow_ips: '127.0.0.1',
      group: 'vip',
      auto_groups: [],
      cross_group_retry: false,
    })
  })

  test('turns on auto-group retry and keeps the auto order when switching to auto', () => {
    const payload = buildApiKeyGroupUpdatePayload(sampleKey, 'auto')

    expect(payload.group).toBe('auto')
    expect(payload.auto_groups).toEqual(['vip'])
    expect(payload.cross_group_retry).toBe(true)
  })

  test('normalizes missing optional fields to empty values', () => {
    const payload = buildApiKeyGroupUpdatePayload(
      {
        ...sampleKey,
        model_limits: null,
        allow_ips: null,
        auto_groups: null,
      },
      'auto'
    )

    expect(payload.model_limits).toBe('')
    expect(payload.allow_ips).toBe('')
    expect(payload.auto_groups).toEqual([])
  })
})
