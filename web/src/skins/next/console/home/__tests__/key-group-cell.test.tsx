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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactElement } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type { ApiKey } from '@/features/keys/types'

import { ConsoleKeyGroupCell } from '../components/key-group-cell'

const apiMocks = vi.hoisted(() => ({
  getApiKey: vi.fn(),
  updateApiKey: vi.fn(),
  getUserGroups: vi.fn(),
  getPricing: vi.fn(),
}))

vi.mock('@/features/keys/api', () => ({
  getApiKey: apiMocks.getApiKey,
  updateApiKey: apiMocks.updateApiKey,
}))

vi.mock('@/lib/api', () => ({
  getUserGroups: apiMocks.getUserGroups,
}))

vi.mock('@/features/pricing/api', () => ({
  getPricing: apiMocks.getPricing,
}))

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: (iconName?: string | null) =>
    iconName ? <span data-testid={`vendor-icon-${iconName}`} /> : null,
}))

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

function renderGroupCell(ui: ReactElement, onSwitched = vi.fn()) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return {
    onSwitched,
    ...render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>),
  }
}

function renderSampleCell(overrides?: Partial<ConsoleKeyGroupCellPropsLike>) {
  const onSwitched = vi.fn()
  return renderGroupCell(
    <ConsoleKeyGroupCell
      apiKey={overrides?.apiKey ?? sampleKey}
      ratio={overrides?.ratio ?? 0.12}
      onSwitched={onSwitched}
      className={overrides?.className}
    />,
    onSwitched
  )
}

type ConsoleKeyGroupCellPropsLike = {
  apiKey: ApiKey
  ratio?: number | string | null
  className?: string
}

describe('next console key group cell', () => {
  beforeEach(() => {
    apiMocks.getUserGroups.mockResolvedValue({
      success: true,
      data: {
        special: { desc: 'Sale', ratio: 0.12 },
        vip: { desc: 'VIP', ratio: 3 },
        auto: { desc: 'Automatic routing', ratio: 'auto' },
      },
    })
    apiMocks.getApiKey.mockResolvedValue({
      success: true,
      data: sampleKey,
    })
    apiMocks.updateApiKey.mockResolvedValue({
      success: true,
      data: { ...sampleKey, group: 'vip' },
    })
    apiMocks.getPricing.mockResolvedValue({
      success: true,
      data: [],
      vendors: [],
      group_ratio: {},
      usable_group: {},
      supported_endpoint: {},
      auto_groups: [],
    })
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  test('renders the group as a bordered dialog trigger with chevrons', () => {
    renderSampleCell()

    const trigger = screen.getByRole('button', { name: 'Switch group' })
    expect(trigger).toHaveAttribute('data-slot', 'console-key-group-cell')
    expect(trigger).toHaveAttribute('aria-haspopup', 'dialog')
    expect(trigger).toHaveClass('border', 'rounded-lg', 'cursor-pointer')
    expect(
      trigger.querySelector('[data-slot="console-key-group-chevrons"]')
    ).not.toBeNull()
    expect(screen.getByText('special')).toBeVisible()
  })

  test('keeps every group name in the same trigger style', () => {
    const { rerender } = renderSampleCell()
    const special = screen.getByRole('button', { name: 'Switch group' })

    rerender(
      <QueryClientProvider
        client={
          new QueryClient({ defaultOptions: { queries: { retry: false } } })
        }
      >
        <ConsoleKeyGroupCell
          apiKey={{ ...sampleKey, group: 'vip' }}
          ratio={3}
          onSwitched={() => undefined}
        />
      </QueryClientProvider>
    )

    const vip = screen.getByRole('button', { name: 'Switch group' })
    expect(vip.className).toBe(special.className)
  })

  test('shows the model-square ratio tag beside a numeric group ratio', () => {
    renderSampleCell()

    const ratio = screen.getByText('0.12x')
    expect(ratio).toHaveClass(
      'inline-flex',
      'shrink-0',
      'rounded-[4px]',
      'text-sm',
      'backdrop-blur-md',
      'backdrop-saturate-150'
    )
    expect(ratio.className).toMatch(/bg-green-(500|400)\/15/)
    expect(ratio.className).not.toMatch(/rounded-full/)
  })

  test('keeps the auto group monochrome and hides a non-numeric ratio', () => {
    renderSampleCell({
      apiKey: { ...sampleKey, group: 'auto' },
      ratio: 'Auto',
    })

    expect(screen.getByText('Cross-group')).toBeVisible()
    expect(screen.queryByText(/x$/)).toBeNull()
    expect(screen.queryByText('Auto Ratio')).toBeNull()
  })

  test('opens a centered group dialog instead of a side drawer', async () => {
    const user = userEvent.setup()
    renderSampleCell()

    await user.click(screen.getByRole('button', { name: 'Switch group' }))

    expect(
      await screen.findByRole('dialog', { name: 'Switch group' })
    ).toBeVisible()
    expect(await screen.findByPlaceholderText('Search groups...')).toBeVisible()
    expect(
      document.querySelector('[data-slot="next-group-picker-dialog"]')
    ).not.toBeNull()
    expect(
      document.querySelector('[data-slot="console-key-group-drawer"]')
    ).toBeNull()
    expect(document.querySelector('[data-slot="sheet-content"]')).toBeNull()
    expect(await screen.findByRole('option', { name: /vip/ })).toBeVisible()
  })

  test('does not scroll the page when the group picker opens', async () => {
    const scrollIntoView = vi.fn()
    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
      configurable: true,
      value: scrollIntoView,
    })
    const scrollTo = vi.spyOn(window, 'scrollTo')
    const user = userEvent.setup()
    renderSampleCell()

    await user.click(screen.getByRole('button', { name: 'Switch group' }))
    await screen.findByRole('option', { name: /vip/ })

    expect(scrollIntoView).not.toHaveBeenCalled()
    expect(scrollTo).not.toHaveBeenCalled()
    expect(screen.getByPlaceholderText('Search groups...')).not.toHaveFocus()

    scrollTo.mockRestore()
    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
      configurable: true,
      value: () => undefined,
    })
  })

  test('shows an empty state when no groups are available', async () => {
    const user = userEvent.setup()
    apiMocks.getUserGroups.mockResolvedValue({
      success: true,
      data: {},
    })
    renderSampleCell()

    await user.click(screen.getByRole('button', { name: 'Switch group' }))

    expect(await screen.findByText('No groups available')).toBeVisible()
  })

  test('falls back to the list row when fetching the latest key fails', async () => {
    const user = userEvent.setup()
    apiMocks.getApiKey.mockResolvedValue({
      success: false,
      message: 'not found',
    })
    const { onSwitched } = renderSampleCell()

    await user.click(screen.getByRole('button', { name: 'Switch group' }))
    await user.click(await screen.findByRole('option', { name: /vip/ }))

    await waitFor(() => {
      expect(apiMocks.updateApiKey).toHaveBeenCalledWith({
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
    expect(onSwitched).toHaveBeenCalledTimes(1)
    await waitFor(() => {
      expect(
        document.querySelector('[data-slot="next-group-picker-dialog"]')
      ).toBeNull()
    })
  })

  test('keeps auto groups and turns on retry when switching to auto', async () => {
    const user = userEvent.setup()
    renderSampleCell()

    await user.click(screen.getByRole('button', { name: 'Switch group' }))
    await user.click(await screen.findByRole('option', { name: /Cross-group/ }))

    await waitFor(() => {
      expect(apiMocks.updateApiKey).toHaveBeenCalledWith(
        expect.objectContaining({
          group: 'auto',
          auto_groups: ['vip'],
          cross_group_retry: true,
        })
      )
    })
  })
})
