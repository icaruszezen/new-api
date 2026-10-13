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
import { afterEach, describe, expect, test, vi } from 'vitest'

import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { useConsolePreviewStore } from '../console-preview-store'

vi.mock('@tanstack/react-router', () => ({
  Navigate: (props: { to: string; replace?: boolean }) => (
    <div data-testid='navigate' data-to={props.to} />
  ),
  getRouteApi: () => ({
    useParams: () => ({ ticketId: '42' }),
  }),
}))

vi.mock('../next/tickets', () => ({
  NextTickets: () => <div data-testid='next-tickets' />,
  NextTicketDetail: (props: { ticketId: number }) => (
    <div data-testid='next-ticket-detail' data-ticket-id={props.ticketId} />
  ),
}))

const { SkinnedTicketDetail, SkinnedTickets } = await import('../tickets')

function setUser(role: number, userId = 1) {
  useAuthStore.getState().auth.setUser({
    id: userId,
    username: 'tester',
    role,
  })
}

function setSiteSkin(uiSkin: 'classic' | 'next') {
  useSystemConfigStore.setState({
    config: { ...useSystemConfigStore.getState().config, uiSkin },
  })
}

describe('support ticket skin dispatch', () => {
  afterEach(() => {
    setSiteSkin('classic')
    useConsolePreviewStore.getState().exitPreview()
    useAuthStore.getState().auth.reset()
  })

  test('renders the next ticket list for a regular user', () => {
    setSiteSkin('next')
    setUser(ROLE.USER)
    render(<SkinnedTickets />)

    expect(screen.getByTestId('next-tickets')).toBeInTheDocument()
    expect(screen.queryByTestId('navigate')).toBeNull()
  })

  test('passes the numeric ticket id to the next detail page', () => {
    setSiteSkin('next')
    setUser(ROLE.USER)
    render(<SkinnedTicketDetail />)

    expect(screen.getByTestId('next-ticket-detail')).toHaveAttribute(
      'data-ticket-id',
      '42'
    )
  })

  test('sends a classic user back to the dashboard', () => {
    setSiteSkin('classic')
    setUser(ROLE.USER)
    render(<SkinnedTickets />)

    expect(screen.getByTestId('navigate')).toHaveAttribute(
      'data-to',
      '/dashboard'
    )
    expect(screen.queryByTestId('next-tickets')).toBeNull()
  })

  test('sends an administrator on the next site to the ticket console', () => {
    setSiteSkin('next')
    setUser(ROLE.ADMIN, 8)
    render(<SkinnedTicketDetail />)

    expect(screen.getByTestId('navigate')).toHaveAttribute(
      'data-to',
      '/ticket-management'
    )
    expect(screen.queryByTestId('next-ticket-detail')).toBeNull()
  })

  test('renders the next ticket list when an administrator previews', () => {
    setSiteSkin('next')
    setUser(ROLE.ADMIN, 8)
    useConsolePreviewStore.getState().enterPreview(8)
    render(<SkinnedTickets />)

    expect(screen.getByTestId('next-tickets')).toBeInTheDocument()
  })
})
