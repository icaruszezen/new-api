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

import { GroupPriceDrawer } from '../components/group-price-drawer'

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { to: string; children: ReactNode; className?: string }) => (
    <a href={props.to} className={props.className}>
      {props.children}
    </a>
  ),
}))

const usableGroup = {
  default: { desc: 'Default', ratio: 1 },
  vip: { desc: 'VIP', ratio: 0.8 },
}

const MULTI_TIER_EXPR =
  'len <= 200000 ? tier("standard", p * 3 + c * 15 + cr * 0.3) : tier("long_context", p * 6 + c * 22.5 + cr * 0.6)'

function tokenModel(overrides?: Partial<PricingModel>): PricingModel {
  return {
    id: 1,
    model_name: 'gpt-4o',
    quota_type: 0,
    model_ratio: 1.25,
    completion_ratio: 4,
    cache_ratio: 0.5,
    enable_groups: ['default', 'vip'],
    group_ratio: { default: 1, vip: 0.8 },
    ...overrides,
  }
}

function renderDrawer(model: PricingModel) {
  return render(
    <GroupPriceDrawer
      model={model}
      usableGroup={usableGroup}
      groupRatio={model.group_ratio || {}}
      tokenUnit='M'
      showRechargePrice={false}
      priceRate={1}
      usdExchangeRate={1}
    />
  )
}

describe('next model square group price drawer', () => {
  test('renders each group multiplier as a ratio tag', () => {
    renderDrawer(tokenModel())

    expect(screen.getByText('1x')).toBeInTheDocument()
    expect(screen.getByText('0.8x')).toBeInTheDocument()
  })

  test('shows only the lowest tier until a group tier menu is opened', async () => {
    const user = userEvent.setup()
    renderDrawer(
      tokenModel({
        billing_mode: 'tiered_expr',
        billing_expr: MULTI_TIER_EXPR,
      })
    )

    expect(screen.getByText('Group')).toBeInTheDocument()
    expect(screen.getByText('Multiplier')).toBeInTheDocument()
    expect(screen.getByText('default')).toBeInTheDocument()
    expect(screen.getByText('vip')).toBeInTheDocument()
    expect(screen.getByText('1x')).toBeInTheDocument()
    expect(screen.getByText('0.8x')).toBeInTheDocument()
    expect(screen.queryByText('standard')).toBeNull()
    expect(screen.queryByText('long_context')).toBeNull()

    const defaultTrigger = screen.getByRole('button', {
      name: 'Expand default tier prices',
    })
    expect(defaultTrigger).toHaveAttribute('aria-expanded', 'false')
    expect(
      screen.getByRole('button', { name: 'Expand vip tier prices' })
    ).toHaveAttribute('aria-expanded', 'false')

    await user.click(defaultTrigger)

    expect(defaultTrigger).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('standard')).toBeVisible()
    expect(screen.getByText('long_context')).toBeVisible()
    expect(
      screen.getByRole('button', { name: 'Expand vip tier prices' })
    ).toHaveAttribute('aria-expanded', 'false')
  })

  test('omits the secondary tier menu when a model has a single tier', () => {
    renderDrawer(
      tokenModel({
        billing_mode: 'tiered_expr',
        billing_expr: 'tier("base", p * 2 + c * 4)',
      })
    )

    expect(screen.getByText('default')).toBeInTheDocument()
    expect(screen.getByText('vip')).toBeInTheDocument()
    expect(screen.queryByText('base')).toBeNull()
    expect(
      screen.queryByRole('button', { name: 'Expand default tier prices' })
    ).toBeNull()
    expect(screen.queryByText('Tiered pricing')).toBeNull()
  })
})
