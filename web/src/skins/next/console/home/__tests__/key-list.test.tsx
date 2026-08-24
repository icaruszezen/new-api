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
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type { ApiKey } from '@/features/keys/types'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

const apiMocks = vi.hoisted(() => ({
  getUserQuotaDates: vi.fn(),
  getApiKeys: vi.fn(),
  getApiKey: vi.fn(),
  deleteApiKey: vi.fn(),
  updateApiKey: vi.fn(),
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
}))

vi.mock('@/features/keys/api', () => ({
  getApiKeys: apiMocks.getApiKeys,
  getApiKey: apiMocks.getApiKey,
  deleteApiKey: apiMocks.deleteApiKey,
  updateApiKey: apiMocks.updateApiKey,
  getTokenAutoGroups: apiMocks.getTokenAutoGroups,
}))

vi.mock('@/lib/api', () => ({
  getUserGroups: apiMocks.getUserGroups,
  getUserModels: apiMocks.getUserModels,
  getStatus: apiMocks.getStatus,
}))

vi.mock('@/features/dashboard/hooks/use-status-data', () => ({
  useApiInfo: () => ({ items: [], loading: false }),
}))

const { NextConsoleHome } = await import('../index')

const longGroupName =
  'very-long-billing-group-name-that-should-truncate-on-narrow-cards'

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

function stubMatchMedia(isMobile: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: (query: string): MediaQueryList => ({
      matches: isMobile && query.includes('max-width: 640px'),
      media: query,
      onchange: null,
      addListener: () => undefined,
      removeListener: () => undefined,
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      dispatchEvent: () => false,
    }),
  })
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

describe('next console key list', () => {
  beforeEach(() => {
    stubMatchMedia(false)
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
    apiMocks.getApiKey.mockResolvedValue({
      success: true,
      data: sampleKey,
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
    apiMocks.deleteApiKey.mockResolvedValue({ success: true })
    apiMocks.updateApiKey.mockResolvedValue({ success: true, data: sampleKey })
  })

  afterEach(() => {
    stubMatchMedia(false)
    useAuthStore.getState().auth.reset()
    vi.clearAllMocks()
  })

  test('shows console key columns with group ratio and hides created, last used, and preview-only columns', async () => {
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    expect(screen.getByRole('columnheader', { name: 'Name' })).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'Status' })).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'API Key' })).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'Quota' })).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'Group' })).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'Models' })).toBeVisible()
    expect(
      screen.getByRole('columnheader', { name: 'IP Restriction' })
    ).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'Expires' })).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'Actions' })).toBeVisible()
    expect(screen.getByRole('button', { name: 'Delete' })).toBeVisible()
    expect(screen.queryByRole('button', { name: 'Open menu' })).toBeNull()
    const groupTrigger = screen.getByRole('combobox', { name: 'Switch group' })
    const ratio = screen.getByText('0.12x')
    expect(groupTrigger).toHaveClass('border', 'rounded-lg')
    expect(
      groupTrigger.querySelector('[data-slot="console-key-group-chevrons"]')
    ).not.toBeNull()
    expect(ratio).toHaveClass('rounded-[4px]', 'backdrop-blur-md')
    expect(ratio.className).toMatch(/bg-green-(500|400)\/15/)
    expect(screen.queryByRole('columnheader', { name: 'Created' })).toBeNull()
    expect(screen.queryByRole('columnheader', { name: 'Last Used' })).toBeNull()
    expect(screen.queryByRole('columnheader', { name: 'Platform' })).toBeNull()
    expect(screen.queryByRole('columnheader', { name: 'Usage' })).toBeNull()
    expect(
      document.querySelector('[data-slot="console-key-desktop-table"]')
    ).not.toBeNull()
  })

  test('renders a mobile card with quota, group, and ratio instead of table headers', async () => {
    stubMatchMedia(true)
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    expect(
      document.querySelector('[data-slot="api-key-mobile-list"]')
    ).not.toBeNull()
    expect(screen.getByText('Enabled')).toBeVisible()
    expect(screen.getByText('Quota')).toBeVisible()
    expect(screen.getByText('Group')).toBeVisible()
    expect(screen.getByText('special')).toBeVisible()
    const groupRow = screen.getByText('Group').parentElement
    expect(groupRow).toHaveAttribute(
      'data-slot',
      'console-key-mobile-group-row'
    )
    expect(groupRow).toHaveClass('flex', 'items-center', 'justify-between')
    expect(groupRow).toContainElement(
      screen.getByRole('combobox', { name: 'Switch group' })
    )
    const mobileRatio = screen.getByText('0.12x')
    expect(mobileRatio).toBeVisible()
    expect(mobileRatio).toHaveClass('rounded-[4px]', 'backdrop-blur-md')
    expect(screen.getByRole('button', { name: 'Delete' })).toBeVisible()
    expect(screen.queryByRole('button', { name: 'Open menu' })).toBeNull()
    expect(screen.queryByRole('columnheader', { name: 'Models' })).toBeNull()
    expect(
      screen.queryByRole('columnheader', { name: 'IP Restriction' })
    ).toBeNull()
    expect(
      document.querySelector('[data-slot="console-key-desktop-table"]')
    ).toBeNull()
  })

  test('keeps the ratio visible when the group name is very long on mobile', async () => {
    stubMatchMedia(true)
    const longGroupKey = {
      ...sampleKey,
      group: longGroupName,
    }
    apiMocks.getApiKeys.mockResolvedValue({
      success: true,
      data: { items: [longGroupKey], total: 1, page: 1, page_size: 50 },
    })
    apiMocks.getUserGroups.mockResolvedValue({
      success: true,
      data: { [longGroupName]: { desc: 'Sale', ratio: 0.12 } },
    })

    renderHome()

    await waitFor(() => {
      expect(screen.getByText('0.12x')).toBeVisible()
    })

    const groupTrigger = screen.getByRole('combobox', { name: 'Switch group' })
    const groupName = screen.getByText(longGroupName)
    const ratio = screen.getByText('0.12x')
    expect(groupName).toBeVisible()
    expect(groupName).toHaveAttribute('data-slot', 'console-key-group-name')
    expect(groupName).toHaveClass('truncate')
    expect(ratio).toHaveClass('shrink-0')
    expect(
      groupTrigger.querySelector('[data-slot="console-key-group-chevrons"]')
    ).not.toBeNull()
    expect(groupTrigger).toContainElement(ratio)
    expect(screen.getByText('Group').parentElement).toContainElement(
      groupTrigger
    )
  })

  test('shows the full mobile group name instead of truncating it beside the ratio', async () => {
    stubMatchMedia(true)
    apiMocks.getApiKeys.mockResolvedValue({
      success: true,
      data: {
        items: [
          { ...sampleKey, id: 1, name: 'plus', group: 'plus-special' },
          {
            ...sampleKey,
            id: 2,
            name: 'kiro',
            group: 'kiro-claude-official',
          },
        ],
        total: 2,
        page: 1,
        page_size: 50,
      },
    })
    apiMocks.getUserGroups.mockResolvedValue({
      success: true,
      data: {
        'plus-special': { desc: 'Plus', ratio: 0.08 },
        'kiro-claude-official': { desc: 'Kiro', ratio: 0.2 },
      },
    })

    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus-special')).toBeVisible()
    })

    expect(screen.getByText('plus-special')).toBeVisible()
    expect(screen.getByText('kiro-claude-official')).toBeVisible()
    expect(screen.getByText('0.08x')).toBeVisible()
    expect(screen.getByText('0.2x')).toBeVisible()
    expect(
      screen.getAllByRole('combobox', { name: 'Switch group' })
    ).toHaveLength(2)
  })

  test('opens the update drawer from the row edit action without navigating', async () => {
    const user = userEvent.setup()
    const pushState = vi.spyOn(window.history, 'pushState')
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    await user.click(screen.getByRole('button', { name: 'Edit' }))

    await waitFor(() => {
      expect(screen.getByText('Update API Key')).toBeVisible()
    })
    expect(pushState).not.toHaveBeenCalled()
    expect(window.location.pathname).toBe('/')
    pushState.mockRestore()
  })

  test('opens the delete confirmation from the row delete button without a more-actions menu', async () => {
    const user = userEvent.setup()
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    expect(screen.queryByRole('button', { name: 'Open menu' })).toBeNull()
    await user.click(screen.getByRole('button', { name: 'Delete' }))

    const dialog = await screen.findByRole('alertdialog')
    expect(within(dialog).getByText('Are you sure?')).toBeVisible()
    expect(within(dialog).getByText('plus')).toBeVisible()
    expect(apiMocks.deleteApiKey).not.toHaveBeenCalled()
  })

  test('deletes the API key after the confirmation is accepted', async () => {
    const user = userEvent.setup()
    apiMocks.deleteApiKey.mockResolvedValue({ success: true })
    apiMocks.getApiKeys
      .mockResolvedValueOnce({
        success: true,
        data: { items: [sampleKey], total: 1, page: 1, page_size: 50 },
      })
      .mockResolvedValueOnce({
        success: true,
        data: { items: [], total: 0, page: 1, page_size: 50 },
      })

    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    await user.click(screen.getByRole('button', { name: 'Delete' }))
    const dialog = await screen.findByRole('alertdialog')
    await user.click(within(dialog).getByRole('button', { name: 'Delete' }))

    await waitFor(() => {
      expect(apiMocks.deleteApiKey).toHaveBeenCalledWith(1)
    })
    await waitFor(() => {
      expect(screen.queryByRole('alertdialog')).toBeNull()
    })
    await waitFor(() => {
      expect(screen.getByText('No API keys yet')).toBeVisible()
    })
  })

  test('keeps the API key when delete confirmation is cancelled', async () => {
    const user = userEvent.setup()
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    await user.click(screen.getByRole('button', { name: 'Delete' }))
    const dialog = await screen.findByRole('alertdialog')
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))

    await waitFor(() => {
      expect(screen.queryByRole('alertdialog')).toBeNull()
    })
    expect(apiMocks.deleteApiKey).not.toHaveBeenCalled()
    expect(screen.getByText('plus')).toBeVisible()
  })

  test('keeps the confirmation open when deleting the API key fails', async () => {
    const user = userEvent.setup()
    apiMocks.deleteApiKey.mockResolvedValue({
      success: false,
      message: 'cannot delete this key',
    })
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    await user.click(screen.getByRole('button', { name: 'Delete' }))
    const dialog = await screen.findByRole('alertdialog')
    await user.click(within(dialog).getByRole('button', { name: 'Delete' }))

    await waitFor(() => {
      expect(apiMocks.deleteApiKey).toHaveBeenCalledWith(1)
    })
    expect(screen.getByRole('alertdialog')).toBeVisible()
    expect(
      within(screen.getByRole('alertdialog')).getByText('plus')
    ).toBeVisible()
    expect(
      document.querySelector('[data-slot="console-key-desktop-table"]')
    ).toHaveTextContent('plus')
  })

  test('opens a compact group picker from the group cell and lists available groups', async () => {
    const user = userEvent.setup()
    apiMocks.getUserGroups.mockResolvedValue({
      success: true,
      data: {
        special: { desc: 'Sale', ratio: 0.12 },
        vip: { desc: 'VIP', ratio: 3 },
      },
    })
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    await user.click(screen.getByRole('combobox', { name: 'Switch group' }))

    const picker = await waitFor(() => {
      const node = document.querySelector(
        '[data-slot="console-key-group-picker"]'
      )
      expect(node).not.toBeNull()
      return node as HTMLElement
    })
    expect(
      within(picker).getByPlaceholderText('Search groups...')
    ).toBeVisible()
    expect(
      within(picker).getByRole('option', { name: /special/ })
    ).toBeVisible()
    expect(within(picker).getByRole('option', { name: /vip/ })).toBeVisible()
    expect(within(picker).getByText('3x')).toBeVisible()
    expect(document.querySelector('[data-slot="sheet-content"]')).toBeNull()
    expect(apiMocks.updateApiKey).not.toHaveBeenCalled()
  })

  test('updates the API key with the full payload when another group is selected', async () => {
    const user = userEvent.setup()
    apiMocks.getUserGroups.mockResolvedValue({
      success: true,
      data: {
        special: { desc: 'Sale', ratio: 0.12 },
        vip: { desc: 'VIP', ratio: 3 },
      },
    })
    apiMocks.getApiKey.mockResolvedValue({
      success: true,
      data: { ...sampleKey, remain_quota: 88 },
    })
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    await user.click(screen.getByRole('combobox', { name: 'Switch group' }))
    const vipOption = await screen.findByRole('option', { name: /vip/ })
    apiMocks.getApiKeys.mockResolvedValue({
      success: true,
      data: {
        items: [{ ...sampleKey, group: 'vip' }],
        total: 1,
        page: 1,
        page_size: 50,
      },
    })
    await user.click(vipOption)

    await waitFor(() => {
      expect(apiMocks.getApiKey).toHaveBeenCalledWith(1)
      expect(apiMocks.updateApiKey).toHaveBeenCalledWith({
        id: 1,
        name: 'plus',
        remain_quota: 88,
        expired_time: -1,
        unlimited_quota: true,
        model_limits_enabled: false,
        model_limits: '',
        allow_ips: '',
        group: 'vip',
        auto_groups: [],
        cross_group_retry: false,
      })
    })
    await waitFor(() => {
      expect(
        document.querySelector('[data-slot="console-key-group-picker"]')
      ).toBeNull()
    })
    expect(
      document.querySelector('[data-slot="console-key-group-cell"]')
    ).toHaveTextContent('vip')
  })

  test('closes the group picker without a request when the current group is selected', async () => {
    const user = userEvent.setup()
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    await user.click(screen.getByRole('combobox', { name: 'Switch group' }))
    await user.click(await screen.findByRole('option', { name: /special/ }))

    await waitFor(() => {
      expect(
        document.querySelector('[data-slot="console-key-group-picker"]')
      ).toBeNull()
    })
    expect(apiMocks.getApiKey).not.toHaveBeenCalled()
    expect(apiMocks.updateApiKey).not.toHaveBeenCalled()
  })

  test('keeps the group picker open when switching groups fails', async () => {
    const user = userEvent.setup()
    apiMocks.getUserGroups.mockResolvedValue({
      success: true,
      data: {
        special: { desc: 'Sale', ratio: 0.12 },
        vip: { desc: 'VIP', ratio: 3 },
      },
    })
    apiMocks.updateApiKey.mockResolvedValue({
      success: false,
      message: 'cannot move this key',
    })
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    await user.click(screen.getByRole('combobox', { name: 'Switch group' }))
    await user.click(await screen.findByRole('option', { name: /vip/ }))

    await waitFor(() => {
      expect(apiMocks.updateApiKey).toHaveBeenCalled()
    })
    expect(
      document.querySelector('[data-slot="console-key-group-picker"]')
    ).not.toBeNull()
    expect(
      document.querySelector('[data-slot="console-key-group-cell"]')
    ).toHaveTextContent('special')
  })

  test('opens the group picker from the mobile group control', async () => {
    const user = userEvent.setup()
    stubMatchMedia(true)
    apiMocks.getUserGroups.mockResolvedValue({
      success: true,
      data: {
        special: { desc: 'Sale', ratio: 0.12 },
        vip: { desc: 'VIP', ratio: 3 },
      },
    })
    renderHome()

    await waitFor(() => {
      expect(screen.getByText('plus')).toBeVisible()
    })

    await user.click(screen.getByRole('combobox', { name: 'Switch group' }))

    expect(
      document.querySelector('[data-slot="console-key-group-picker"]')
    ).not.toBeNull()
    expect(await screen.findByRole('option', { name: /vip/ })).toBeVisible()
    expect(document.querySelector('[data-slot="sheet-content"]')).toBeNull()
  })
})
