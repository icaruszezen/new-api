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
import { renderHook } from '@testing-library/react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

vi.mock('@tanstack/react-router', () => ({
  useLocation: (options?: { select?: (location: { pathname: string }) => unknown }) => {
    const location = { pathname: '/usage-logs/common' }
    return options?.select ? options.select(location) : location
  },
}))

vi.mock('../use-sidebar-config', () => ({
  useSidebarConfig: (groups: unknown) => groups,
}))

const { useSidebarView } = await import('../use-sidebar-view')

function setUser(role: number) {
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'tester',
    role,
  })
}

function generalUrls() {
  const { result } = renderHook(() => useSidebarView())
  const general = result.current.navGroups.find((group) => group.id === 'general')
  return (general?.items ?? [])
    .map((item) => ('url' in item ? item.url : undefined))
    .filter((url): url is string => typeof url === 'string')
}

describe('useSidebarView first-token errors', () => {
  afterEach(() => {
    useAuthStore.getState().auth.reset()
  })

  test('hides first-token errors from regular users', () => {
    setUser(ROLE.USER)
    expect(generalUrls()).not.toContain('/first-token-errors')
  })

  test('shows first-token errors to admins', () => {
    setUser(ROLE.ADMIN)
    expect(generalUrls()).toContain('/first-token-errors')
  })
})
