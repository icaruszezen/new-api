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
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type { ApiKey } from '@/features/keys/types'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

const apiMocks = vi.hoisted(() => ({
  getUserQuotaDates: vi.fn(),
  getApiKeys: vi.fn(),
  getUserGroups: vi.fn(),
}))

vi.mock('@/features/dashboard/api', () => ({
  getUserQuotaDates: apiMocks.getUserQuotaDates,
}))

vi.mock('@/features/keys/api', () => ({
  getApiKeys: apiMocks.getApiKeys,
}))

vi.mock('@/lib/api', () => ({
  getUserGroups: apiMocks.getUserGroups,
}))

vi.mock('@/features/dashboard/hooks/use-status-data', () => ({
  useApiInfo: () => ({ items: [], loading: false }),
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

describe('next console homepage inert actions', () => {
  beforeEach(() => {
    useAuthStore.getState().auth.setUser({
      id: 1,
      username: 'tester',
      role: ROLE.USER,
    })
    apiMocks.getUserQuotaDates.mockResolvedValue({
      success: true,
      data: [],
    })
    apiMocks.getApiKeys.mockResolvedValue({
      success: true,
      data: { items: [sampleKey], total: 1, page: 1, page_size: 50 },
    })
    apiMocks.getUserGroups.mockResolvedValue({
      success: true,
      data: {},
    })
  })

  afterEach(() => {
    useAuthStore.getState().auth.reset()
    vi.clearAllMocks()
  })

  test('does not navigate when homepage action buttons are clicked', async () => {
    const user = userEvent.setup()
    const pushState = vi.spyOn(window.history, 'pushState')
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    const labels = [
      'Personal Center',
      'Usage records',
      'Recharge',
      'Usage docs',
      'New key',
      'Test Connection',
      'Copy',
      'Test',
      'Edit',
      'Disable',
      'Delete',
    ]

    for (const label of labels) {
      const button = screen.getAllByRole('button', { name: label })[0]
      await user.click(button)
    }

    expect(pushState).not.toHaveBeenCalled()
    expect(window.location.pathname).toBe('/')
    pushState.mockRestore()
  })
})
