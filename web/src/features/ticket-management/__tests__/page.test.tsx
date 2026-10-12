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
import { Toaster, toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type {
  AdminTicketDetail,
  Ticket,
  TicketPage,
  TicketStats,
} from '@/features/tickets/types'

import type { TicketManagementSearch } from '../lib/ticket-management'

const apiMocks = vi.hoisted(() => ({
  getAdminTickets: vi.fn(),
  getAdminTicketStats: vi.fn(),
  getAdminTicket: vi.fn(),
  replyAdminTicket: vi.fn(),
  updateAdminTicketStatus: vi.fn(),
  getTicketAttachmentBlob: vi.fn(),
}))

const routeState = vi.hoisted(() => ({
  search: {} as TicketManagementSearch,
  navigate: vi.fn(),
}))

vi.mock('@/features/tickets/api', () => apiMocks)

vi.mock('@tanstack/react-router', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@tanstack/react-router')>()),
  getRouteApi: () => ({
    useSearch: () => routeState.search,
    useNavigate: () => routeState.navigate,
  }),
}))

const { TicketManagement } = await import('..')

function buildTicket(overrides: Partial<Ticket> = {}): Ticket {
  return {
    id: 12,
    user_id: 7,
    subject: 'Top-up not credited',
    status: 1,
    last_reply_role: 'user',
    message_count: 1,
    user_unread: false,
    admin_unread: false,
    created_time: 1_760_000_000,
    updated_time: 1_760_000_600,
    closed_time: 0,
    username: 'alice',
    ...overrides,
  }
}

function buildPage(items: Ticket[], total = items.length): TicketPage {
  return { page: 1, page_size: 20, total, items }
}

const STATS: TicketStats = {
  pending: 3,
  processing: 2,
  resolved: 4,
  closed: 1,
  unread: 2,
}

function buildDetail(ticket: Ticket): AdminTicketDetail {
  return {
    ticket,
    messages: [
      {
        id: 1,
        ticket_id: ticket.id,
        author_role: 'user',
        content: 'I paid but my balance did not change.',
        created_time: ticket.created_time,
        author_name: 'alice',
        attachments: [],
      },
    ],
    user: {
      id: ticket.user_id,
      username: 'alice',
      display_name: 'Alice',
      email: 'alice@example.com',
      group: 'default',
    },
  }
}

/** Applies the last navigation's search updater to the current search. */
function lastNavigatedSearch() {
  const call = routeState.navigate.mock.lastCall?.[0] as {
    search: (previous: TicketManagementSearch) => TicketManagementSearch
  }
  return call.search({ ...routeState.search })
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const Wrapper = (props: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {props.children}
      <Toaster duration={60_000} />
    </QueryClientProvider>
  )
  return render(<TicketManagement />, { wrapper: Wrapper })
}

beforeEach(() => {
  routeState.search = {}
  apiMocks.getAdminTicketStats.mockResolvedValue(STATS)
})

afterEach(() => {
  toast.dismiss()
  vi.clearAllMocks()
})

describe('ticket management page', () => {
  test('requests the status, page and keyword from the URL', async () => {
    routeState.search = { status: 1, page: 2, keyword: ' refund ' }
    apiMocks.getAdminTickets.mockResolvedValue(buildPage([], 21))

    renderPage()

    expect(await screen.findByText('No tickets found')).toBeInTheDocument()
    expect(apiMocks.getAdminTickets).toHaveBeenCalledWith({
      p: 2,
      page_size: 20,
      status: 1,
      keyword: 'refund',
    })
    expect(screen.getByText('Page 2 of 2')).toBeInTheDocument()
  })

  test('lists each ticket with its owner, status and only marks unread ones', async () => {
    apiMocks.getAdminTickets.mockResolvedValue(
      buildPage([
        buildTicket({ admin_unread: true }),
        buildTicket({
          id: 13,
          user_id: 8,
          username: 'bob',
          subject: 'API key revoked',
          status: 3,
        }),
      ])
    )

    renderPage()

    const unreadRow = (
      await screen.findByRole('button', { name: 'Top-up not credited' })
    ).closest('tr')
    const readRow = screen
      .getByRole('button', { name: 'API key revoked' })
      .closest('tr')
    if (!unreadRow || !readRow) throw new Error('ticket rows not rendered')

    expect(within(unreadRow).getByText('#12')).toBeInTheDocument()
    expect(within(unreadRow).getByText('alice')).toBeInTheDocument()
    expect(within(unreadRow).getByText('#7')).toBeInTheDocument()
    expect(within(unreadRow).getByText('Awaiting support')).toBeInTheDocument()
    expect(within(unreadRow).getByText('Unread')).toBeInTheDocument()
    expect(within(readRow).getByText('Resolved')).toBeInTheDocument()
    expect(within(readRow).queryByText('Unread')).toBeNull()
  })

  test('shows each status count and the unread total from the stats', async () => {
    apiMocks.getAdminTickets.mockResolvedValue(buildPage([]))

    renderPage()

    expect(await screen.findByText('2 unread')).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'All 10' })).toHaveAttribute(
      'aria-selected',
      'true'
    )
    expect(
      screen.getByRole('tab', { name: 'Awaiting support 3' })
    ).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Closed 1' })).toBeInTheDocument()
  })

  test('choosing a status tab filters by it from page one', async () => {
    const user = userEvent.setup()
    routeState.search = { page: 3 }
    apiMocks.getAdminTickets.mockResolvedValue(buildPage([]))

    renderPage()
    await user.click(await screen.findByRole('tab', { name: /^Resolved/ }))

    expect(lastNavigatedSearch()).toEqual({ status: 3, page: 1 })
  })

  test('choosing All clears the status filter', async () => {
    const user = userEvent.setup()
    routeState.search = { status: 2 }
    apiMocks.getAdminTickets.mockResolvedValue(buildPage([]))

    renderPage()
    await user.click(await screen.findByRole('tab', { name: /^All/ }))

    expect(lastNavigatedSearch()).toEqual({ status: undefined, page: 1 })
  })

  test('searching applies the trimmed keyword from page one', async () => {
    const user = userEvent.setup()
    routeState.search = { status: 1, page: 4 }
    apiMocks.getAdminTickets.mockResolvedValue(buildPage([]))

    renderPage()
    await user.type(
      await screen.findByRole('textbox', { name: 'Search tickets' }),
      '  alice {Enter}'
    )

    expect(lastNavigatedSearch()).toEqual({
      status: 1,
      page: 1,
      keyword: 'alice',
    })
  })

  test('shows the server message inline when the list cannot be loaded', async () => {
    apiMocks.getAdminTickets.mockRejectedValue(
      new Error('Ticket list unavailable')
    )

    renderPage()

    expect(
      await screen.findByText('Ticket list unavailable')
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument()
  })

  test('opening an unread ticket shows its thread and clears the unread marker', async () => {
    const user = userEvent.setup()
    const ticket = buildTicket({ admin_unread: true })
    apiMocks.getAdminTickets
      .mockResolvedValueOnce(buildPage([ticket]))
      .mockResolvedValue(buildPage([{ ...ticket, admin_unread: false }]))
    apiMocks.getAdminTicket.mockResolvedValue(
      buildDetail({ ...ticket, admin_unread: false })
    )

    renderPage()
    await user.click(await screen.findByText('#12'))

    const sheet = await screen.findByRole('dialog')
    expect(apiMocks.getAdminTicket).toHaveBeenCalledWith(12)
    expect(
      await within(sheet).findByText('I paid but my balance did not change.')
    ).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.queryByText('Unread')).toBeNull()
    })
  })
})
