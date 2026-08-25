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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type { PricingData } from '@/features/pricing/types'

import { GroupPickerDialog } from '../group-picker-dialog'

const apiMocks = vi.hoisted(() => ({
  getPricing: vi.fn(),
}))

vi.mock('@/features/pricing/api', () => ({
  getPricing: apiMocks.getPricing,
}))

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: (iconName?: string | null) =>
    iconName ? <span data-testid={`vendor-icon-${iconName}`} /> : null,
}))

const catalogFixture: PricingData = {
  success: true,
  message: '',
  data: [
    {
      id: 1,
      model_name: 'gpt-4o',
      quota_type: 0,
      model_ratio: 1,
      completion_ratio: 1,
      enable_groups: ['cheap'],
      vendor_id: 1,
    },
    {
      id: 2,
      model_name: 'claude-sonnet',
      quota_type: 0,
      model_ratio: 1,
      completion_ratio: 1,
      enable_groups: ['vip'],
      vendor_id: 2,
    },
  ],
  vendors: [
    { id: 1, name: 'OpenAI', icon: 'OpenAI' },
    { id: 2, name: 'Anthropic', icon: 'Claude' },
  ],
  group_ratio: {},
  usable_group: {},
  supported_endpoint: {},
  auto_groups: [],
}

const options = [
  { value: 'auto', label: 'auto', desc: 'Automatic routing', ratio: 'auto' },
  { value: 'cheap', label: 'cheap', desc: 'Sale', ratio: 0.2 },
  { value: 'vip', label: 'vip', desc: 'Premium', ratio: 3 },
]

function renderPicker(
  ui: ReactNode,
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
) {
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  )
}

describe('next group picker dialog', () => {
  beforeEach(() => {
    apiMocks.getPricing.mockResolvedValue(catalogFixture)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  test('opens a centered dialog instead of a sheet and keeps search unfocused', async () => {
    renderPicker(
      <GroupPickerDialog
        open
        onOpenChange={() => undefined}
        options={options}
        title='Select a group'
        onSelect={() => undefined}
      />
    )

    expect(
      await screen.findByRole('dialog', { name: 'Select a group' })
    ).toBeVisible()
    expect(
      document.querySelector('[data-slot="next-group-picker-dialog"]')
    ).not.toBeNull()
    expect(document.querySelector('[data-slot="sheet-content"]')).toBeNull()
    expect(
      await screen.findByPlaceholderText('Search groups...')
    ).not.toHaveFocus()
  })

  test('groups options under OpenAI then Claude with vendor logos', async () => {
    renderPicker(
      <GroupPickerDialog
        open
        onOpenChange={() => undefined}
        options={options}
        title='Select a group'
        onSelect={() => undefined}
      />
    )

    const openai = await screen.findByRole('heading', { name: 'OpenAI' })
    const claude = await screen.findByRole('heading', { name: 'Claude' })
    expect(openai.compareDocumentPosition(claude) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(screen.getByTestId('vendor-icon-OpenAI')).toBeVisible()
    expect(screen.getByTestId('vendor-icon-Claude')).toBeVisible()
    expect(screen.getByRole('heading', { name: 'Cross-group' })).toBeVisible()
  })

  test('filters vendor sections when the search does not match their groups', async () => {
    const user = userEvent.setup()
    renderPicker(
      <GroupPickerDialog
        open
        onOpenChange={() => undefined}
        options={options}
        title='Select a group'
        onSelect={() => undefined}
      />
    )

    await screen.findByRole('heading', { name: 'OpenAI' })
    await user.type(screen.getByPlaceholderText('Search groups...'), 'premium')

    expect(screen.queryByRole('heading', { name: 'OpenAI' })).toBeNull()
    expect(screen.getByRole('heading', { name: 'Claude' })).toBeVisible()
    expect(screen.getByRole('option', { name: /vip/ })).toBeVisible()
  })

  test('selects a group and reports the value to the caller', async () => {
    const user = userEvent.setup()
    const onSelect = vi.fn()
    const onOpenChange = vi.fn()
    renderPicker(
      <GroupPickerDialog
        open
        onOpenChange={onOpenChange}
        options={options}
        value='cheap'
        title='Select a group'
        onSelect={onSelect}
      />
    )

    await user.click(await screen.findByRole('option', { name: /vip/ }))
    expect(onSelect).toHaveBeenCalledWith('vip')
  })

  test('shows an empty state when no groups are available', async () => {
    renderPicker(
      <GroupPickerDialog
        open
        onOpenChange={() => undefined}
        options={[]}
        title='Select a group'
        onSelect={() => undefined}
      />
    )

    expect(await screen.findByText('No groups available')).toBeVisible()
  })

  test('falls back to a flat ratio-sorted list when pricing fails to load', async () => {
    apiMocks.getPricing.mockRejectedValue(new Error('pricing down'))
    renderPicker(
      <GroupPickerDialog
        open
        onOpenChange={() => undefined}
        options={options.filter((option) => option.value !== 'auto')}
        title='Select a group'
        onSelect={() => undefined}
      />
    )

    const cheap = await screen.findByRole('option', { name: /cheap/ })
    const vip = screen.getByRole('option', { name: /vip/ })
    expect(
      cheap.compareDocumentPosition(vip) & Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy()
    expect(screen.queryByRole('heading', { name: 'OpenAI' })).toBeNull()
    expect(screen.queryByRole('heading', { name: 'Claude' })).toBeNull()
  })

  test('shows a loading skeleton until pricing is ready', async () => {
    let resolvePricing: (value: PricingData) => void = () => undefined
    apiMocks.getPricing.mockReset()
    apiMocks.getPricing.mockImplementation(
      () =>
        new Promise<PricingData>((resolve) => {
          resolvePricing = resolve
        })
    )
    renderPicker(
      <GroupPickerDialog
        open
        onOpenChange={() => undefined}
        options={options}
        title='Select a group'
        onSelect={() => undefined}
      />
    )

    expect(screen.queryByRole('option', { name: /vip/ })).toBeNull()

    resolvePricing(catalogFixture)
    expect(await screen.findByRole('option', { name: /vip/ })).toBeVisible()
  })
})
