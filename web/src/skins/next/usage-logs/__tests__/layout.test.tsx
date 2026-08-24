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
import { describe, expect, test, vi } from 'vitest'

import type { LogsViewScope } from '@/features/usage-logs/components/usage-logs-provider'
import type { UsageLogsSectionId } from '@/features/usage-logs/section-registry'

const pageState: {
  activeCategory: UsageLogsSectionId
  titleKey: string
  canManageScope: boolean
  viewScope: LogsViewScope
  showTaskSwitcher: boolean
  visibleSectionTabs: { id: UsageLogsSectionId; titleKey: string }[]
} = {
  activeCategory: 'common',
  titleKey: 'Common Logs',
  canManageScope: false,
  viewScope: 'all',
  showTaskSwitcher: false,
  visibleSectionTabs: [],
}

vi.mock('@/features/usage-logs/hooks', () => ({
  useUsageLogsPage: () => ({
    activeCategory: pageState.activeCategory,
    titleKey: pageState.titleKey,
    canManageScope: pageState.canManageScope,
    viewScope: pageState.viewScope,
    handleViewScopeChange: vi.fn(),
    showTaskSwitcher: pageState.showTaskSwitcher,
    visibleSectionTabs: pageState.visibleSectionTabs,
    handleSectionChange: vi.fn(),
    selectedUserId: null,
    userInfoDialogOpen: false,
    setUserInfoDialogOpen: vi.fn(),
    affinityDialogOpen: false,
    setAffinityDialogOpen: vi.fn(),
    affinityDialogTarget: null,
  }),
}))

vi.mock('@/features/usage-logs/components/usage-logs-table', () => ({
  UsageLogsTable: () => <div data-slot='usage-logs-table'>logs table</div>,
}))

vi.mock('@/features/usage-logs/components/dialogs/user-info-dialog', () => ({
  UserInfoDialog: () => null,
}))

vi.mock(
  '@/features/system-settings/general/channel-affinity/cache-stats-dialog',
  () => ({
    CacheStatsDialog: () => null,
  })
)

const { NextUsageLogs } = await import('../index')

describe('next usage logs layout', () => {
  test('renders the next page chrome without sidebar or classic layout', () => {
    pageState.activeCategory = 'common'
    pageState.titleKey = 'Common Logs'
    pageState.canManageScope = false
    pageState.viewScope = 'all'
    pageState.showTaskSwitcher = false
    pageState.visibleSectionTabs = []

    render(<NextUsageLogs />)

    const page = document.querySelector('[data-slot="next-usage-logs"]')
    expect(page).toBeInTheDocument()
    expect(page).toHaveClass('flex')
    expect(page).toHaveClass('flex-col')
    expect(page).toHaveClass('min-h-0')
    expect(page).toHaveClass('flex-1')
    expect(page).toHaveClass('w-full')
    expect(page).toHaveClass('h-full')
    expect(page).not.toHaveClass('max-w-7xl')
    expect(page).not.toHaveClass('px-6')
    expect(page).toHaveClass('px-3')
    expect(page).toHaveClass('sm:px-4')

    const heading = screen.getByRole('heading', { name: 'Common Logs' })
    expect(heading.tagName).toBe('H1')
    expect(heading).toHaveClass('text-3xl')
    expect(heading).toHaveClass('font-medium')

    const card = document.querySelector('[data-slot="next-usage-logs-card"]')
    expect(card).toBeInTheDocument()
    expect(card).toHaveClass('rounded-xl')
    expect(card).toHaveClass('border')
    expect(card).toContainElement(
      document.querySelector('[data-slot="usage-logs-table"]')
    )
    expect(
      document.querySelector('[data-slot="next-usage-logs-footer"]')
    ).toBeInTheDocument()

    expect(screen.queryByRole('tab', { name: 'All' })).toBeNull()
    expect(screen.queryByRole('tab', { name: 'Only Mine' })).toBeNull()
    expect(screen.queryByText('Export')).toBeNull()
    expect(document.querySelector('main')).toBeNull()
    expect(document.querySelector('[data-slot="sidebar"]')).toBeNull()
    expect(document.querySelector('h2')).toBeNull()
  })

  test('keeps All and Only Mine on the title row for admins', () => {
    pageState.activeCategory = 'common'
    pageState.titleKey = 'Common Logs'
    pageState.canManageScope = true
    pageState.viewScope = 'all'
    pageState.showTaskSwitcher = false
    pageState.visibleSectionTabs = []

    render(<NextUsageLogs />)

    const heading = document.querySelector(
      '[data-slot="next-usage-logs-heading"]'
    )
    expect(heading).toHaveClass('justify-between')
    expect(heading).toContainElement(
      screen.getByRole('heading', { name: 'Common Logs' })
    )
    expect(heading).toContainElement(screen.getByRole('tab', { name: 'All' }))
    expect(heading).toContainElement(
      screen.getByRole('tab', { name: 'Only Mine' })
    )
  })

  test('keeps drawing and task tabs above the table', () => {
    pageState.activeCategory = 'task'
    pageState.titleKey = 'Task Logs'
    pageState.canManageScope = false
    pageState.viewScope = 'self'
    pageState.showTaskSwitcher = true
    pageState.visibleSectionTabs = [
      { id: 'drawing', titleKey: 'Drawing Logs' },
      { id: 'task', titleKey: 'Task Logs' },
    ]

    render(<NextUsageLogs />)

    expect(screen.getByRole('heading', { name: 'Task Logs' })).toBeVisible()
    const switcher = document.querySelector(
      '[data-slot="next-usage-logs-switcher"]'
    )
    const card = document.querySelector('[data-slot="next-usage-logs-card"]')
    const table = document.querySelector('[data-slot="usage-logs-table"]')
    expect(switcher).toBeInstanceOf(HTMLElement)
    expect(card).toBeInstanceOf(HTMLElement)
    expect(table).toBeInstanceOf(HTMLElement)
    if (
      !(switcher instanceof HTMLElement) ||
      !(card instanceof HTMLElement) ||
      !(table instanceof HTMLElement)
    ) {
      return
    }
    expect(switcher).toContainElement(
      screen.getByRole('tab', { name: 'Drawing Logs' })
    )
    expect(switcher).toContainElement(
      screen.getByRole('tab', { name: 'Task Logs' })
    )
    expect(card).toContainElement(table)
    expect(switcher.compareDocumentPosition(card)).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING
    )
  })
})
