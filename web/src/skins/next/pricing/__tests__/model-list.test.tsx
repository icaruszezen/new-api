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

const models: PricingModel[] = [
  {
    id: 1,
    model_name: 'gpt-4o',
    quota_type: 0,
    model_ratio: 1.25,
    completion_ratio: 4,
    cache_ratio: 0.5,
    enable_groups: ['default', 'vip'],
    group_ratio: { default: 1, vip: 0.8 },
  },
  {
    id: 2,
    model_name: 'claude-sonnet-4',
    quota_type: 0,
    model_ratio: 1.5,
    completion_ratio: 5,
    enable_groups: ['default'],
    group_ratio: { default: 1 },
  },
]

const usableGroup = {
  default: { desc: 'Default', ratio: 1 },
  vip: { desc: 'VIP', ratio: 0.8 },
}

describe('next model square list', () => {
  test('renders column headers and one row per model name', () => {
    render(
      <ModelList
        models={models}
        usableGroup={usableGroup}
        groupRatio={{ default: 1, vip: 0.8 }}
        tokenUnit='M'
        showRechargePrice={false}
        priceRate={1}
        usdExchangeRate={1}
      />
    )

    expect(screen.getByText('Model')).toBeInTheDocument()
    expect(screen.getByText('Input (per 1M tokens)')).toBeInTheDocument()
    expect(screen.getByText('Output (per 1M tokens)')).toBeInTheDocument()
    expect(
      screen.getByText('Cached input (per 1M tokens)')
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Expand gpt-4o pricing' })
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Expand claude-sonnet-4 pricing' })
    ).toBeInTheDocument()
  })

  test('opens a nested group drawer for the selected model only', async () => {
    const user = userEvent.setup()
    render(
      <ModelList
        models={models}
        usableGroup={usableGroup}
        groupRatio={{ default: 1, vip: 0.8 }}
        tokenUnit='M'
        showRechargePrice={false}
        priceRate={1}
        usdExchangeRate={1}
      />
    )

    await user.click(
      screen.getByRole('button', { name: 'Expand gpt-4o pricing' })
    )

    expect(screen.getByText('Pricing by Group')).toBeInTheDocument()
    expect(screen.getByText('1x')).toBeInTheDocument()
    expect(screen.getByText('0.8x')).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Expand claude-sonnet-4 pricing' })
    ).toHaveAttribute('aria-expanded', 'false')
  })
})
