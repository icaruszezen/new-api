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

import type { PricingModel, PricingVendor } from '@/features/pricing/types'

import {
  GROUP_PICKER_AUTO_SECTION_ID,
  GROUP_PICKER_OTHER_SECTION_ID,
  filterGroupPickerSections,
  groupUserGroupsByVendor,
  normalizeVendor,
  resolvePricingModels,
  sortGroupsByRatio,
  type GroupPickerOption,
} from '../group-picker'

function option(
  value: string,
  ratio?: number | string,
  desc?: string
): GroupPickerOption {
  return { value, label: value, desc, ratio }
}

function model(
  groups: string[],
  vendorName?: string,
  vendorIcon?: string
): Pick<PricingModel, 'enable_groups' | 'vendor_name' | 'vendor_icon'> {
  return {
    enable_groups: groups,
    vendor_name: vendorName,
    vendor_icon: vendorIcon,
  }
}

describe('normalizeVendor', () => {
  test('maps Anthropic and Claude icons onto the Claude section title', () => {
    expect(normalizeVendor('Anthropic', 'Claude')).toEqual({
      id: 'claude',
      title: 'Claude',
      icon: 'Claude',
    })
    expect(normalizeVendor('Claude').title).toBe('Claude')
  })

  test('pins OpenAI ahead of Claude, Gemini, and Grok by featured id', () => {
    expect(normalizeVendor('OpenAI').id).toBe('openai')
    expect(normalizeVendor('Gemini', 'Google').id).toBe('gemini')
    expect(normalizeVendor('Grok').id).toBe('grok')
  })
})

describe('groupUserGroupsByVendor', () => {
  test('pins auto first, then OpenAI before Claude, and Other last', () => {
    const sections = groupUserGroupsByVendor(
      [
        option('auto', 'auto'),
        option('moon', 2),
        option('cheap', 0.2),
        option('orphan', 1),
      ],
      [
        model(['moon'], 'Moonshot', 'Moonshot'),
        model(['cheap'], 'OpenAI', 'OpenAI'),
        model(['cheap'], 'Anthropic', 'Claude'),
      ]
    )

    expect(sections.map((section) => section.id)).toEqual([
      GROUP_PICKER_AUTO_SECTION_ID,
      'openai',
      'claude',
      'Moonshot',
      GROUP_PICKER_OTHER_SECTION_ID,
    ])
    expect(sections[1]?.title).toBe('OpenAI')
    expect(sections[2]?.title).toBe('Claude')
    expect(sections[4]?.groups.map((group) => group.value)).toEqual(['orphan'])
  })

  test('repeats a multi-vendor group under every vendor it enables', () => {
    const sections = groupUserGroupsByVendor(
      [option('default', 1)],
      [
        model(['default'], 'OpenAI', 'OpenAI'),
        model(['default'], 'Anthropic', 'Claude'),
      ]
    )

    expect(sections).toHaveLength(2)
    expect(sections[0]?.groups.map((group) => group.value)).toEqual(['default'])
    expect(sections[1]?.groups.map((group) => group.value)).toEqual(['default'])
  })

  test('sorts groups inside a vendor by numeric ratio then name', () => {
    const sections = groupUserGroupsByVendor(
      [option('vip', 3), option('alpha', 0.5), option('beta', 0.5)],
      [
        model(['vip', 'alpha', 'beta'], 'OpenAI', 'OpenAI'),
      ]
    )

    expect(sections[0]?.groups.map((group) => group.value)).toEqual([
      'alpha',
      'beta',
      'vip',
    ])
  })

  test('puts groups without pricing models into Other', () => {
    const sections = groupUserGroupsByVendor([option('ghost', 1.2)], [])

    expect(sections).toHaveLength(1)
    expect(sections[0]?.id).toBe(GROUP_PICKER_OTHER_SECTION_ID)
    expect(sections[0]?.groups.map((group) => group.value)).toEqual(['ghost'])
  })
})

describe('sortGroupsByRatio', () => {
  test('orders numeric ratios ascending and keeps non-numeric last', () => {
    expect(
      sortGroupsByRatio([
        option('auto', 'auto'),
        option('vip', 3),
        option('sale', 0.12),
      ]).map((group) => group.value)
    ).toEqual(['sale', 'vip', 'auto'])
  })
})

describe('filterGroupPickerSections', () => {
  test('hides vendor sections that have no matching groups', () => {
    const sections = groupUserGroupsByVendor(
      [option('default', 1, 'Standard'), option('vip', 2, 'Premium')],
      [
        model(['default'], 'OpenAI', 'OpenAI'),
        model(['vip'], 'Anthropic', 'Claude'),
      ]
    )

    const filtered = filterGroupPickerSections(sections, 'premium')
    expect(filtered).toHaveLength(1)
    expect(filtered[0]?.id).toBe('claude')
    expect(filtered[0]?.groups.map((group) => group.value)).toEqual(['vip'])
  })
})

describe('resolvePricingModels', () => {
  test('copies vendor name and icon from the vendors list', () => {
    const vendors: PricingVendor[] = [
      { id: 7, name: 'Anthropic', icon: 'Claude' },
    ]
    const models: PricingModel[] = [
      {
        id: 1,
        model_name: 'claude-sonnet',
        quota_type: 0,
        model_ratio: 1,
        completion_ratio: 1,
        enable_groups: ['default'],
        vendor_id: 7,
      },
    ]

    expect(resolvePricingModels(models, vendors)[0]).toMatchObject({
      vendor_name: 'Anthropic',
      vendor_icon: 'Claude',
    })
  })
})
