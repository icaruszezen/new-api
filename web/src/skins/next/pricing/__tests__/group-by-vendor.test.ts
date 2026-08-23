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

import type { PricingModel } from '@/features/pricing/types'

import {
  OTHER_VENDOR_GROUP_ID,
  groupModelsByVendor,
} from '../lib/group-by-vendor'

function model(
  name: string,
  overrides?: Partial<PricingModel>
): PricingModel {
  return {
    id: 1,
    model_name: name,
    quota_type: 0,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default'],
    ...overrides,
  }
}

describe('groupModelsByVendor', () => {
  test('returns an empty list when there are no models', () => {
    expect(groupModelsByVendor([])).toEqual([])
  })

  test('puts models without a vendor name into Other', () => {
    const groups = groupModelsByVendor([
      model('orphan-a'),
      model('orphan-b', { vendor_id: 9 }),
    ])

    expect(groups).toHaveLength(1)
    expect(groups[0]?.id).toBe(OTHER_VENDOR_GROUP_ID)
    expect(groups[0]?.name).toBe('')
    expect(groups[0]?.models.map((item) => item.model_name)).toEqual([
      'orphan-a',
      'orphan-b',
    ])
  })

  test('pins OpenAI, Anthropic, Gemini and Grok first and keeps auto-route vendors later', () => {
    const groups = groupModelsByVendor([
      model('local-model'),
      model('kimi', { vendor_name: 'Moonshot', vendor_icon: 'Moonshot' }),
      model('grok-4.6', { vendor_name: 'Grok', vendor_icon: 'Grok' }),
      model('gemini-3-pro', { vendor_name: 'Gemini', vendor_icon: 'Google' }),
      model('claude-sonnet-4', {
        vendor_name: 'Anthropic',
        vendor_icon: 'Claude',
      }),
      model('g5.5', {
        vendor_name: '自动路由（GPT）',
        vendor_icon: 'OpenAI',
      }),
      model('claude-alias', {
        vendor_name: '自动路由（Claude）',
        vendor_icon: 'Claude',
      }),
      model('gpt-4o', { vendor_name: 'OpenAI', vendor_icon: 'OpenAI' }),
    ])

    const ids = groups.map((group) => group.id)
    expect(ids.slice(0, 4)).toEqual([
      'OpenAI',
      'Anthropic',
      'Gemini',
      'Grok',
    ])
    expect(ids.at(-1)).toBe(OTHER_VENDOR_GROUP_ID)
    expect(ids.slice(4, -1)).toEqual(
      ['Moonshot', '自动路由（Claude）', '自动路由（GPT）'].sort((a, b) =>
        a.localeCompare(b)
      )
    )
  })

  test('keeps input order inside a vendor and fills a missing icon later', () => {
    const groups = groupModelsByVendor([
      model('gpt-4o-mini', { vendor_name: 'OpenAI' }),
      model('gpt-4o', { vendor_name: 'OpenAI', vendor_icon: 'OpenAI' }),
    ])

    expect(groups).toHaveLength(1)
    expect(groups[0]?.models.map((item) => item.model_name)).toEqual([
      'gpt-4o-mini',
      'gpt-4o',
    ])
    expect(groups[0]?.icon).toBe('OpenAI')
  })
})
