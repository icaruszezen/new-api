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
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type { ApiKey } from '@/features/keys/types'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

const apiMocks = vi.hoisted(() => ({
  getUserQuotaDates: vi.fn(),
  getUserQuotaSummary: vi.fn(),
  getApiKeys: vi.fn(),
  getApiKey: vi.fn(),
  createApiKey: vi.fn(),
  getUserGroups: vi.fn(),
  getUserModels: vi.fn(),
  getStatus: vi.fn(),
  getTokenAutoGroups: vi.fn(),
}))

vi.mock('@tanstack/react-router', () => ({
  Link: (props: {
    to: string
    params?: Record<string, string>
    children: ReactNode
    className?: string
  }) => {
    let href = props.to
    if (props.params) {
      href = Object.entries(props.params).reduce(
        (path, [key, value]) => path.replace(`$${key}`, value),
        props.to
      )
    }
    return (
      <a href={href} className={props.className}>
        {props.children}
      </a>
    )
  },
}))

vi.mock('@/features/dashboard/api', () => ({
  getUserQuotaDates: apiMocks.getUserQuotaDates,
  getUserQuotaSummary: apiMocks.getUserQuotaSummary,
}))

vi.mock('@/features/keys/api', () => ({
  getApiKeys: apiMocks.getApiKeys,
  getApiKey: apiMocks.getApiKey,
  createApiKey: apiMocks.createApiKey,
  getTokenAutoGroups: apiMocks.getTokenAutoGroups,
}))

vi.mock('@/lib/api', () => ({
  getUserGroups: apiMocks.getUserGroups,
  getUserModels: apiMocks.getUserModels,
  getStatus: apiMocks.getStatus,
}))

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: (iconName?: string | null) =>
    iconName ? <span data-testid={`vendor-icon-${iconName}`} /> : null,
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

describe('next console homepage navigation', () => {
  beforeEach(() => {
    window.localStorage.removeItem('status')
    useAuthStore.getState().auth.setUser({
      id: 1,
      username: 'tester',
      role: ROLE.USER,
    })
    useAuthStore.setState((state) => ({
      auth: { ...state.auth, accessToken: 'test-token' },
    }))
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
    apiMocks.getUserModels.mockResolvedValue({ success: true, data: [] })
    apiMocks.getStatus.mockResolvedValue({})
    apiMocks.getTokenAutoGroups.mockResolvedValue({
      success: true,
      data: { groups: [], max_count: 3 },
    })
  })

  afterEach(() => {
    useAuthStore.getState().auth.reset()
    window.localStorage.removeItem('status')
    vi.clearAllMocks()
  })

  test('points personal center, usage records, and recharge at the existing console routes', async () => {
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    expect(
      screen.getByRole('link', { name: 'Personal Center' })
    ).toHaveAttribute('href', '/profile')
    expect(screen.getByRole('link', { name: 'Usage records' })).toHaveAttribute(
      'href',
      '/usage-logs/common'
    )
    expect(screen.getByRole('link', { name: 'Recharge' })).toHaveAttribute(
      'href',
      '/wallet'
    )
  })

  test('points usage docs at /docs when no external docs link is configured', async () => {
    renderHome()

    await waitFor(() => {
      expect(screen.getByRole('link', { name: 'Usage docs' })).toBeVisible()
    })

    expect(screen.getByRole('link', { name: 'Usage docs' })).toHaveAttribute(
      'href',
      '/docs'
    )
    expect(
      screen.getByRole('link', { name: 'Usage docs' })
    ).not.toHaveAttribute('target')
  })

  test('opens the configured docs link in a new tab', async () => {
    apiMocks.getStatus.mockResolvedValue({
      docs_link: 'https://docs.example.com/guide',
    })

    renderHome()

    await waitFor(() => {
      expect(screen.getByRole('link', { name: 'Usage docs' })).toHaveAttribute(
        'href',
        'https://docs.example.com/guide'
      )
    })

    const usageDocs = screen.getByRole('link', { name: 'Usage docs' })
    expect(usageDocs).toHaveAttribute('target', '_blank')
    expect(usageDocs).toHaveAttribute('rel', 'noopener noreferrer')
  })

  test('keeps the wallet balance card and test connection button from navigating', async () => {
    const user = userEvent.setup()
    const pushState = vi.spyOn(window.history, 'pushState')
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    expect(
      screen.getByRole('heading', { name: 'Wallet balance' })
    ).toBeVisible()
    expect(screen.queryByRole('link', { name: /Wallet balance/ })).toBeNull()

    await user.click(screen.getByRole('heading', { name: 'Wallet balance' }))
    await user.click(screen.getByRole('button', { name: 'Test Connection' }))

    expect(pushState).not.toHaveBeenCalled()
    expect(window.location.pathname).toBe('/')
    expect(screen.getByRole('dialog')).toBeVisible()
    expect(screen.getByText('Test request link')).toBeVisible()
    expect(
      document.querySelector('[data-slot="next-test-connection-dialog"]')
    ).toBeTruthy()
    pushState.mockRestore()
  })

  test('opens the create dialog when New key is clicked', async () => {
    const user = userEvent.setup()
    const pushState = vi.spyOn(window.history, 'pushState')
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    await user.click(screen.getByRole('button', { name: 'New key' }))

    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeVisible()
    })
    expect(screen.getByText('Create API Key')).toBeVisible()
    expect(
      document.querySelector('[data-slot="next-api-key-dialog"]')
    ).toBeTruthy()
    expect(document.querySelector('[data-slot="sheet-content"]')).toBeNull()
    expect(pushState).not.toHaveBeenCalled()
    expect(window.location.pathname).toBe('/')
    pushState.mockRestore()
  })
})
