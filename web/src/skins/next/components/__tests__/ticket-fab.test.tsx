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
import type { AnchorHTMLAttributes, ReactNode } from 'react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { useConsolePreviewStore } from '../../../console-preview-store'

const pathnameRef = vi.hoisted(() => ({ current: '/dashboard/overview' }))

const apiMocks = vi.hoisted(() => ({
  getSelfTicketUnreadCount: vi.fn(),
}))

vi.mock('@/features/tickets/api', () => apiMocks)

vi.mock('@tanstack/react-router', () => ({
  Link: (props: AnchorHTMLAttributes<HTMLAnchorElement> & { to: string }) => (
    <a {...props} href={props.to} />
  ),
  useRouterState: (options?: {
    select?: (state: { location: { pathname: string } }) => unknown
  }) => {
    const state = { location: { pathname: pathnameRef.current } }
    return options?.select ? options.select(state) : state
  },
}))

const { NextTicketFab } = await import('../ticket-fab')

function setUser(role: number, userId = 1) {
  useAuthStore.getState().auth.setUser({ id: userId, username: 'alice', role })
}

function setSiteSkin(uiSkin: 'classic' | 'next') {
  useSystemConfigStore.setState({
    config: { ...useSystemConfigStore.getState().config, uiSkin },
  })
}

function renderFab() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const Wrapper = (props: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {props.children}
    </QueryClientProvider>
  )
  return render(<NextTicketFab />, { wrapper: Wrapper })
}

describe('next support ticket floating button', () => {
  afterEach(() => {
    pathnameRef.current = '/dashboard/overview'
    setSiteSkin('classic')
    useConsolePreviewStore.getState().exitPreview()
    useAuthStore.getState().auth.reset()
  })

  test('links a signed-in Next user to the ticket list', async () => {
    setSiteSkin('next')
    setUser(ROLE.USER)
    apiMocks.getSelfTicketUnreadCount.mockResolvedValue(0)
    renderFab()

    const link = screen.getByRole('link', { name: 'Support tickets' })
    expect(link).toHaveAttribute('href', '/tickets')
    expect(link).toHaveClass('fixed', 'bottom-4', 'right-4', 'z-40')
    await waitFor(() =>
      expect(apiMocks.getSelfTicketUnreadCount).toHaveBeenCalledTimes(1)
    )
    expect(link).not.toHaveAttribute('aria-describedby')
  })

  test('shows the unread update count as a badge with an accessible description', async () => {
    setSiteSkin('next')
    setUser(ROLE.USER)
    apiMocks.getSelfTicketUnreadCount.mockResolvedValue(3)
    renderFab()

    expect(
      await screen.findByRole('link', {
        name: 'Support tickets',
        description: 'Unread updates: 3',
      })
    ).toBeInTheDocument()
    expect(screen.getByText('3')).toHaveAttribute('aria-hidden', 'true')
  })

  test('caps the badge at 99+ for large unread counts', async () => {
    setSiteSkin('next')
    setUser(ROLE.USER)
    apiMocks.getSelfTicketUnreadCount.mockResolvedValue(150)
    renderFab()

    expect(await screen.findByText('99+')).toBeInTheDocument()
  })

  test('renders nothing and never polls for a guest', () => {
    setSiteSkin('next')
    renderFab()

    expect(screen.queryByRole('link')).toBeNull()
    expect(apiMocks.getSelfTicketUnreadCount).not.toHaveBeenCalled()
  })

  test('renders nothing for a regular user when the site uses the classic skin', () => {
    setSiteSkin('classic')
    setUser(ROLE.USER)
    renderFab()

    expect(screen.queryByRole('link')).toBeNull()
    expect(apiMocks.getSelfTicketUnreadCount).not.toHaveBeenCalled()
  })

  test('renders nothing for an administrator who is not previewing the user console', () => {
    setSiteSkin('next')
    setUser(ROLE.ADMIN, 8)
    renderFab()

    expect(screen.queryByRole('link')).toBeNull()
    expect(apiMocks.getSelfTicketUnreadCount).not.toHaveBeenCalled()
  })

  test('shows for an administrator previewing the user console', () => {
    setSiteSkin('next')
    setUser(ROLE.ADMIN, 8)
    useConsolePreviewStore.getState().enterPreview(8)
    apiMocks.getSelfTicketUnreadCount.mockResolvedValue(0)
    renderFab()

    expect(
      screen.getByRole('link', { name: 'Support tickets' })
    ).toBeInTheDocument()
  })

  test.each([
    '/tickets',
    '/tickets/42',
    '/playground',
    '/chat/abc',
    '/chat2link',
  ])('hides itself and stops polling on %s', (pathname) => {
    setSiteSkin('next')
    setUser(ROLE.USER)
    pathnameRef.current = pathname
    renderFab()

    expect(screen.queryByRole('link')).toBeNull()
    expect(apiMocks.getSelfTicketUnreadCount).not.toHaveBeenCalled()
  })

  test.each(['/usage-logs/common', '/first-token-errors', '/keys'])(
    'sits above the bottom-right pagination on %s',
    (pathname) => {
      setSiteSkin('next')
      setUser(ROLE.USER)
      apiMocks.getSelfTicketUnreadCount.mockResolvedValue(0)
      pathnameRef.current = pathname
      renderFab()

      const link = screen.getByRole('link', { name: 'Support tickets' })
      expect(link).toHaveClass('bottom-20', 'sm:bottom-24')
      expect(link).not.toHaveClass('bottom-4')
    }
  )
})
