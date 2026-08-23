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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { describe, expect, test, vi } from 'vitest'

import type { PricingModel } from '@/features/pricing/types'

import { ModelList } from '../components/model-list'

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { to: string; children: ReactNode; className?: string }) => (
    <a href={props.to} className={props.className}>
      {props.children}
    </a>
  ),
}))

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: (iconName?: string | null) =>
    iconName ? <span data-testid={`vendor-icon-${iconName}`} /> : null,
}))

function tokenModel(
  name: string,
  overrides?: Partial<PricingModel>
): PricingModel {
  return {
    id: 1,
    model_name: name,
    quota_type: 0,
    model_ratio: 1.25,
    completion_ratio: 4,
    cache_ratio: 0.5,
    enable_groups: ['default'],
    group_ratio: { default: 1 },
    ...overrides,
  }
}

const models: PricingModel[] = [
  tokenModel('gpt-4o', {
    vendor_name: 'OpenAI',
    vendor_icon: 'OpenAI',
    enable_groups: ['default', 'vip'],
    group_ratio: { default: 1, vip: 0.8 },
  }),
  tokenModel('claude-sonnet-4', {
    vendor_name: 'Anthropic',
    vendor_icon: 'Claude',
  }),
]

const usableGroup = {
  default: { desc: 'Default', ratio: 1 },
  vip: { desc: 'VIP', ratio: 0.8 },
}

function renderList(listModels: PricingModel[] = models) {
  return render(
    <ModelList
      models={listModels}
      usableGroup={usableGroup}
      groupRatio={{ default: 1, vip: 0.8 }}
      tokenUnit='M'
      showRechargePrice={false}
      priceRate={1}
      usdExchangeRate={1}
    />
  )
}

describe('next model square list', () => {
  test('splits models into vendor sections with icons and column headers', () => {
    renderList()

    const vendorHeadings = screen
      .getAllByRole('heading', { level: 2 })
      .map((heading) => heading.textContent?.trim())
    expect(vendorHeadings).toEqual(['OpenAI', 'Anthropic'])
    expect(screen.getAllByTestId('vendor-icon-Claude').length).toBeGreaterThan(0)
    expect(screen.getAllByTestId('vendor-icon-OpenAI').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Model')).toHaveLength(2)
    expect(screen.getAllByText('Multiplier')).toHaveLength(2)
    expect(screen.getAllByText('Input (per 1M tokens)')).toHaveLength(2)
    expect(screen.getAllByText('Output (per 1M tokens)')).toHaveLength(2)
    expect(screen.getAllByText('Cached input (per 1M tokens)')).toHaveLength(2)
    expect(
      screen.getByRole('button', { name: 'Expand gpt-4o pricing' })
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Expand claude-sonnet-4 pricing' })
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Previous page' })
    ).toBeNull()
    expect(screen.queryByRole('button', { name: 'Next page' })).toBeNull()
  })

  test('renders every filtered model on one page when the catalog is longer than the old page size', () => {
    const manyModels = Array.from({ length: 25 }, (_, index) =>
      tokenModel(`model-${index + 1}`, {
        vendor_name: index < 12 ? 'OpenAI' : 'Anthropic',
      })
    )

    renderList(manyModels)

    expect(
      screen.getAllByRole('button', { name: /Expand .+ pricing/ })
    ).toHaveLength(25)
    expect(screen.queryByText(/Page \d+ of \d+/)).toBeNull()
  })

  test('opens a nested group drawer for the selected model only', async () => {
    const user = userEvent.setup()
    renderList()

    await user.click(
      screen.getByRole('button', { name: 'Expand gpt-4o pricing' })
    )

    expect(screen.getByText('Pricing by Group')).toBeInTheDocument()
    expect(screen.getAllByText('1x')).toHaveLength(2)
    expect(screen.getAllByText('0.8x')).toHaveLength(2)
    expect(
      screen.getByRole('button', { name: 'Expand claude-sonnet-4 pricing' })
    ).toHaveAttribute('aria-expanded', 'false')
  })
})
