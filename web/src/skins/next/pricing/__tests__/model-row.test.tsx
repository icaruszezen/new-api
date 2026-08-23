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

import { Accordion } from '@/components/ui/accordion'
import { formatPrice } from '@/features/pricing/lib/price'
import type { PricingModel } from '@/features/pricing/types'

import { ModelRow } from '../components/model-row'

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { to: string; children: ReactNode; className?: string }) => (
    <a href={props.to} className={props.className}>
      {props.children}
    </a>
  ),
}))

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: (iconName?: string | null) =>
    iconName ? <span data-testid={`model-icon-${iconName}`} /> : null,
}))

const usableGroup = {
  default: { desc: 'Default', ratio: 1 },
  vip: { desc: 'VIP', ratio: 0.8 },
}

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

function renderRow(model: PricingModel) {
  return render(
    <Accordion>
      <ModelRow
        model={model}
        usableGroup={usableGroup}
        groupRatio={model.group_ratio || {}}
        tokenUnit='M'
        showRechargePrice={false}
        priceRate={1}
        usdExchangeRate={1}
      />
    </Accordion>
  )
}

describe('next model square row', () => {
  test('shows the model logo before the name', () => {
    renderRow(tokenModel({ icon: 'OpenAI.Color' }))

    expect(screen.getByTestId('model-icon-OpenAI.Color')).toBeInTheDocument()
  })

  test('falls back to the vendor icon when the model has no logo', () => {
    renderRow(tokenModel({ vendor_icon: 'Claude' }))

    expect(screen.getByTestId('model-icon-Claude')).toBeInTheDocument()
  })

  test('shows the lowest available input, output and cache prices while collapsed', () => {
    const model = tokenModel()
    renderRow(model)

    const trigger = screen.getByRole('button', {
      name: 'Expand gpt-4o pricing',
    })
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    expect(trigger).toHaveTextContent(
      formatPrice(model, 'input', 'M', false, 1, 1)
    )
    expect(trigger).toHaveTextContent(
      formatPrice(model, 'output', 'M', false, 1, 1)
    )
    expect(trigger).toHaveTextContent(
      formatPrice(model, 'cache', 'M', false, 1, 1)
    )
    expect(trigger).toHaveTextContent('lowest')
    expect(trigger).toHaveTextContent('0.8x')
    expect(trigger).not.toHaveTextContent('1x')
    expect(screen.queryByText('Pricing by Group')).toBeNull()
  })

  test('keeps the lowest ratio tag out of the model-name cell so the list can align it as a column', () => {
    renderRow(tokenModel())

    const name = screen.getByText('gpt-4o')
    const tag = screen.getByText('0.8x')
    expect(name.parentElement).not.toContainElement(tag)
    expect(tag).not.toHaveClass('w-full')
    expect(tag.parentElement).toHaveClass('justify-center')
  })

  test('omits the lowest badge when a model belongs to a single group', () => {
    renderRow(
      tokenModel({ enable_groups: ['default'], group_ratio: { default: 1 } })
    )

    const trigger = screen.getByRole('button', {
      name: 'Expand gpt-4o pricing',
    })
    expect(trigger).not.toHaveTextContent('lowest')
    expect(trigger).toHaveTextContent('1x')
  })

  test('omits a cache price when the model has no cache ratio', () => {
    renderRow(tokenModel({ cache_ratio: null }))

    const trigger = screen.getByRole('button', {
      name: 'Expand gpt-4o pricing',
    })
    expect(trigger).toHaveTextContent('—')
  })

  test('expands a group pricing drawer with multipliers and per-group prices', async () => {
    const user = userEvent.setup()
    const model = tokenModel()
    renderRow(model)

    const trigger = screen.getByRole('button', {
      name: 'Expand gpt-4o pricing',
    })
    await user.click(trigger)

    expect(trigger).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Pricing by Group')).toBeInTheDocument()
    expect(screen.getByText('default')).toBeInTheDocument()
    expect(screen.getByText('vip')).toBeInTheDocument()
    expect(screen.getByText('1x')).toBeInTheDocument()
    expect(screen.getAllByText('0.8x')).toHaveLength(2)
    expect(screen.getByRole('link', { name: 'Model details' })).toBeVisible()
  })

  test('toggles the drawer with the keyboard', async () => {
    const user = userEvent.setup()
    renderRow(tokenModel())

    const trigger = screen.getByRole('button', {
      name: 'Expand gpt-4o pricing',
    })
    trigger.focus()
    await user.keyboard('{Enter}')

    expect(trigger).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Pricing by Group')).toBeInTheDocument()
  })
})
