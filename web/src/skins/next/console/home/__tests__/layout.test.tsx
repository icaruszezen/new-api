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
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type { ApiKey } from '@/features/keys/types'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

const apiMocks = vi.hoisted(() => ({
  getUserQuotaDates: vi.fn(),
  getApiKeys: vi.fn(),
  getUserGroups: vi.fn(),
  getUserModels: vi.fn(),
  getStatus: vi.fn(),
  getTokenAutoGroups: vi.fn(),
}))

vi.mock('@/features/dashboard/api', () => ({
  getUserQuotaDates: apiMocks.getUserQuotaDates,
}))

vi.mock('@/features/keys/api', () => ({
  getApiKeys: apiMocks.getApiKeys,
  getTokenAutoGroups: apiMocks.getTokenAutoGroups,
}))

vi.mock('@/lib/api', () => ({
  getUserGroups: apiMocks.getUserGroups,
  getUserModels: apiMocks.getUserModels,
  getStatus: apiMocks.getStatus,
}))

vi.mock('@/features/dashboard/hooks/use-status-data', () => ({
  useApiInfo: () => ({
    items: [
      {
        url: 'https://api.example.com',
        route: 'default',
        description: 'Default',
        color: '',
      },
    ],
    loading: false,
  }),
}))

const { NextConsoleHome } = await import('../index')

const sampleKey: ApiKey = {
  id: 1,
  name: 'plus',
  key: 'd7af123443e0',
  status: 1,
  remain_quota: 0,
  used_quota: 0,
  unlimited_quota: true,
  expired_time: -1,
  created_time: 1,
  accessed_time: 0,
  group: 'special',
  auto_groups: null,
  cross_group_retry: false,
  model_limits_enabled: false,
  model_limits: '',
  allow_ips: '',
}

function renderHome() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <NextConsoleHome />
    </QueryClientProvider>
  )
}

describe('next console homepage layout', () => {
  beforeEach(() => {
    useAuthStore.getState().auth.setUser({
      id: 1,
      username: 'tester',
      role: ROLE.USER,
      request_count: 71703,
    })
    apiMocks.getUserQuotaDates.mockResolvedValue({
      success: true,
      data: [{ created_at: 1, count: 107, token_used: 16100000 }],
    })
    apiMocks.getApiKeys.mockResolvedValue({
      success: true,
      data: {
        items: [sampleKey],
        total: 1,
        page: 1,
        page_size: 50,
      },
    })
    apiMocks.getUserGroups.mockResolvedValue({
      success: true,
      data: { special: { desc: 'Sale', ratio: 0.12 } },
    })
    apiMocks.getUserModels.mockResolvedValue({ success: true, data: [] })
    apiMocks.getStatus.mockResolvedValue({})
    apiMocks.getTokenAutoGroups.mockResolvedValue({
      success: true,
      data: { groups: [], max_count: 3 },
    })
  })

  afterEach(() => {
    useAuthStore.getState().auth.reset()
    vi.clearAllMocks()
  })

  test('renders four stats, three shortcuts, and the key section', async () => {
    renderHome()

    expect(screen.getByRole('heading', { name: "Today's calls" })).toBeVisible()
    expect(
      screen.getByRole('heading', { name: "Today's tokens" })
    ).toBeVisible()
    expect(
      screen.getByRole('heading', { name: 'Lifetime tokens' })
    ).toBeVisible()
    expect(
      screen.getByRole('heading', { name: 'Wallet balance' })
    ).toBeVisible()
    expect(screen.getByText('Balance depleted')).toBeVisible()
    expect(
      screen.getByRole('button', { name: 'Personal Center' })
    ).toBeVisible()
    expect(screen.getByRole('button', { name: 'Usage records' })).toBeVisible()
    expect(screen.getByRole('button', { name: 'Recharge' })).toBeVisible()
    expect(
      screen.getByRole('heading', { name: 'Key Management' })
    ).toBeVisible()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })
    expect(screen.getByText('sk-d7af123443e0')).toBeVisible()
    expect(screen.getByText('special')).toBeVisible()
    expect(screen.getByText('0.12x')).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'Status' })).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'Quota' })).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'Group' })).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'Expires' })).toBeVisible()
    expect(screen.queryByRole('columnheader', { name: 'Created' })).toBeNull()
    expect(screen.queryByRole('columnheader', { name: 'Last Used' })).toBeNull()
    expect(screen.queryByRole('columnheader', { name: 'Platform' })).toBeNull()
    expect(screen.queryByRole('columnheader', { name: 'Usage' })).toBeNull()
    expect(
      screen.queryByRole('columnheader', { name: 'Billing rate' })
    ).toBeNull()
    const endpoint = screen.getByText('https://api.example.com')
    expect(endpoint).toBeVisible()
    expect(
      endpoint.closest('[data-slot="console-endpoint-chip"]')
    ).toHaveAttribute('data-tone')
  })

  test('places stats and shortcuts above key management', () => {
    renderHome()

    const stats = screen.getByRole('region', { name: "Today's usage" })
    const shortcuts = screen.getByRole('navigation', { name: 'Quick actions' })
    const keys = screen
      .getByRole('heading', { name: 'Key Management' })
      .closest('section')
    const topRow = stats.parentElement?.parentElement

    expect(stats.parentElement?.parentElement).toBe(
      shortcuts.parentElement?.parentElement
    )
    expect(stats.parentElement).toHaveClass('lg:col-span-9')
    expect(shortcuts.parentElement).toHaveClass('lg:col-span-3')
    expect(stats.parentElement?.nextElementSibling).toBe(
      shortcuts.parentElement
    )
    expect(topRow).toHaveClass('lg:grid-cols-12')
    expect(topRow).toHaveClass('lg:items-stretch')
    expect(stats).toHaveAttribute('data-slot', 'console-stat-panel')
    expect(stats).toHaveClass('grid-cols-2')
    expect(stats).toHaveClass('rounded-xl')
    expect(stats).toHaveClass('border')
    expect(stats).toHaveClass('overflow-hidden')
    expect(stats).not.toHaveClass('gap-3')
    expect(stats).not.toHaveClass('xl:grid-cols-4')
    expect(shortcuts).toHaveClass('grid-cols-1')
    expect(shortcuts).toHaveClass('lg:h-full')
    expect(shortcuts).toHaveClass('lg:grid-rows-3')
    expect(shortcuts).not.toHaveClass('sm:grid-cols-3')

    expect(keys).toBeTruthy()
    expect(keys?.parentElement).toHaveClass('flex-col')
    expect(keys?.parentElement).not.toHaveClass('lg:col-span-8')
    expect(keys?.previousElementSibling).toBe(topRow)
  })

  test('renders the four stats as quadrants of one panel', () => {
    renderHome()

    const stats = screen.getByRole('region', { name: "Today's usage" })
    const cells = stats.querySelectorAll('[data-slot="console-stat-cell"]')

    expect(cells).toHaveLength(4)
    expect(stats.children[0]).toBe(cells[0])
    expect(stats.children[1]).toBe(cells[1])
    expect(stats.children[2]).toBe(cells[2])
    expect(stats.children[3]).toBe(cells[3])

    for (const cell of cells) {
      expect(cell).not.toHaveClass('rounded-xl')
      expect(cell).not.toHaveClass('border')
    }

    expect(cells[0]).toHaveClass('border-r')
    expect(cells[0]).toHaveClass('border-b')
    expect(cells[1]).toHaveClass('border-b')
    expect(cells[1]).not.toHaveClass('border-r')
    expect(cells[2]).toHaveClass('border-r')
    expect(cells[2]).not.toHaveClass('border-b')
    expect(cells[3]).not.toHaveClass('border-r')
    expect(cells[3]).not.toHaveClass('border-b')
  })

  test('places the test connection button immediately before usage docs', () => {
    renderHome()

    const testConnection = screen.getByRole('button', {
      name: 'Test Connection',
    })
    const usageDocs = screen.getByRole('button', { name: 'Usage docs' })
    const newKey = screen.getByRole('button', { name: 'New key' })

    expect(testConnection.nextElementSibling).toBe(usageDocs)
    expect(usageDocs.nextElementSibling).toBe(newKey)
    expect(testConnection.parentElement).toBe(usageDocs.parentElement)
  })

  test('estimates remaining days from wallet quota and today spend', async () => {
    useAuthStore.getState().auth.setUser({
      id: 1,
      username: 'tester',
      role: ROLE.USER,
      quota: 500,
    })
    apiMocks.getUserQuotaDates.mockResolvedValue({
      success: true,
      data: [{ created_at: 1, count: 1, token_used: 10, quota: 100 }],
    })

    renderHome()

    await waitFor(() => {
      expect(screen.getByText('About 5 days left')).toBeVisible()
    })
    expect(
      screen.getByRole('heading', { name: 'Wallet balance' })
    ).toBeVisible()
  })

  test('shows missing marks for metrics the current APIs cannot supply', async () => {
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    expect(screen.getAllByText('--').length).toBeGreaterThan(0)
    expect(
      screen.queryByRole('heading', { name: 'Average response' })
    ).toBeNull()
  })

  test('keeps the key table structure when the user has no keys', async () => {
    apiMocks.getApiKeys.mockResolvedValue({
      success: true,
      data: { items: [], total: 0, page: 1, page_size: 50 },
    })

    renderHome()

    await waitFor(() => {
      expect(screen.getByText('No API keys yet')).toBeVisible()
    })
    expect(
      screen.getByRole('heading', { name: 'Key Management' })
    ).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'Name' })).toBeVisible()
  })
})
