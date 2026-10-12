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

import type { Ticket, TicketPage, TicketStatus } from '@/features/tickets/types'

type TicketsSearch = { status?: TicketStatus; page?: number }

const routeState = vi.hoisted(() => ({
  search: {} as TicketsSearch,
  navigate: vi.fn(),
}))

const apiMocks = vi.hoisted(() => ({
  getSelfTickets: vi.fn(),
  createSelfTicket: vi.fn(),
}))

vi.mock('@/features/tickets/api', () => apiMocks)

vi.mock('@tanstack/react-router', () => ({
  getRouteApi: () => ({
    useSearch: () => routeState.search,
    useNavigate: () => routeState.navigate,
  }),
  useNavigate: () => routeState.navigate,
  Link: (props: {
    to: string
    params?: Record<string, string>
    className?: string
    children: ReactNode
  }) => {
    let href = props.to
    for (const [name, value] of Object.entries(props.params ?? {})) {
      href = href.replace(`$${name}`, value)
    }
    return (
      <a href={href} className={props.className}>
        {props.children}
      </a>
    )
  },
}))

const { NextTickets } = await import('../index')

function buildTicket(overrides: Partial<Ticket> = {}): Ticket {
  const now = Math.floor(Date.now() / 1000)
  return {
    id: 7,
    user_id: 1,
    subject: 'Cannot top up with card',
    status: 1,
    last_reply_role: 'user',
    message_count: 1,
    user_unread: false,
    admin_unread: true,
    created_time: now - 3600,
    updated_time: now - 600,
    closed_time: 0,
    ...overrides,
  }
}

function buildPage(items: Ticket[], total = items.length): TicketPage {
  return { page: 1, page_size: 10, total, items }
}

function renderTickets() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const Wrapper = (props: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {props.children}
    </QueryClientProvider>
  )
  return render(<NextTickets />, { wrapper: Wrapper })
}

/** Applies the last `navigate({ search })` updater to the current search. */
function lastNavigatedSearch(): TicketsSearch {
  const call = routeState.navigate.mock.calls.at(-1)?.[0] as {
    search: (prev: TicketsSearch) => TicketsSearch
  }
  return call.search({ ...routeState.search })
}

describe('next support ticket list', () => {
  beforeEach(() => {
    routeState.search = {}
  })

  afterEach(() => {
    routeState.navigate.mockReset()
    vi.clearAllMocks()
  })

  test('lists tickets as links with their status and marks unread updates', async () => {
    apiMocks.getSelfTickets.mockResolvedValue(
      buildPage([
        buildTicket({ id: 7, status: 2, user_unread: true }),
        buildTicket({ id: 5, subject: 'Invoice request', status: 3 }),
      ])
    )
    renderTickets()

    const unreadRow = await screen.findByRole('link', {
      name: /Cannot top up with card/,
    })
    expect(unreadRow).toHaveAttribute('href', '/tickets/7')
    expect(within(unreadRow).getByText('#7')).toBeInTheDocument()
    expect(within(unreadRow).getByText('In progress')).toBeInTheDocument()
    expect(within(unreadRow).getByText('New update')).toBeInTheDocument()

    const readRow = screen.getByRole('link', { name: /Invoice request/ })
    expect(readRow).toHaveAttribute('href', '/tickets/5')
    expect(within(readRow).getByText('Resolved')).toBeInTheDocument()
    expect(within(readRow).queryByText('New update')).toBeNull()
  })

  test('requests the page and status from the URL search', async () => {
    routeState.search = { status: 4, page: 2 }
    apiMocks.getSelfTickets.mockResolvedValue(
      buildPage([buildTicket({ status: 4 })], 25)
    )
    renderTickets()

    await screen.findByRole('link', { name: /Cannot top up with card/ })
    expect(apiMocks.getSelfTickets).toHaveBeenCalledWith({
      p: 2,
      page_size: 10,
      status: 4,
    })
    expect(screen.getByRole('tab', { name: 'Closed' })).toHaveAttribute(
      'aria-selected',
      'true'
    )
  })

  test('invites a first ticket from the empty state', async () => {
    const user = userEvent.setup()
    apiMocks.getSelfTickets.mockResolvedValue(buildPage([]))
    renderTickets()

    const emptyState = await screen.findByRole('heading', {
      name: 'No tickets yet',
    })
    const container = emptyState.parentElement
    if (!container) throw new Error('empty state not rendered')
    await user.click(
      within(container).getByRole('button', { name: 'New ticket' })
    )

    expect(
      await screen.findByRole('dialog', { name: 'New ticket' })
    ).toBeInTheDocument()
  })

  test('explains an empty status filter without the first-ticket prompt', async () => {
    routeState.search = { status: 3 }
    apiMocks.getSelfTickets.mockResolvedValue(buildPage([]))
    renderTickets()

    expect(
      await screen.findByText('No tickets with this status.')
    ).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'No tickets yet' })).toBeNull()
  })

  test('an empty page past the end of a status filter says the page is empty and clamps the pager', async () => {
    routeState.search = { status: 3, page: 5 }
    apiMocks.getSelfTickets.mockResolvedValue(buildPage([], 5))
    renderTickets()

    expect(
      await screen.findByText('No tickets on this page.')
    ).toBeInTheDocument()
    expect(screen.queryByText('No tickets with this status.')).toBeNull()
    expect(screen.getByText('Page 1 of 1')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Previous page' })).toBeEnabled()
  })

  test('keeps the status filter on one sideways-scrolling row instead of wrapping', async () => {
    apiMocks.getSelfTickets.mockResolvedValue(buildPage([buildTicket()]))
    renderTickets()

    const tabList = await screen.findByRole('tablist', {
      name: 'Filter by status',
    })
    expect(tabList).toHaveClass('overflow-x-auto')
    expect(tabList).not.toHaveClass('flex-wrap')
    for (const tab of within(tabList).getAllByRole('tab')) {
      expect(tab).toHaveClass('flex-none')
    }
  })

  test('selecting a status tab filters through the URL and resets the page', async () => {
    const user = userEvent.setup()
    routeState.search = { page: 3 }
    apiMocks.getSelfTickets.mockResolvedValue(buildPage([buildTicket()], 30))
    renderTickets()

    await user.click(await screen.findByRole('tab', { name: 'Resolved' }))

    expect(lastNavigatedSearch()).toEqual({ status: 3, page: undefined })
  })

  test('selecting All clears the status filter', async () => {
    const user = userEvent.setup()
    routeState.search = { status: 1 }
    apiMocks.getSelfTickets.mockResolvedValue(buildPage([buildTicket()]))
    renderTickets()

    await user.click(await screen.findByRole('tab', { name: 'All' }))

    expect(lastNavigatedSearch()).toEqual({
      status: undefined,
      page: undefined,
    })
  })

  test('pages through results with Previous and Next', async () => {
    const user = userEvent.setup()
    routeState.search = { page: 2 }
    apiMocks.getSelfTickets.mockResolvedValue(buildPage([buildTicket()], 25))
    renderTickets()

    expect(await screen.findByText('Page 2 of 3')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Next page' }))
    expect(lastNavigatedSearch()).toEqual({ page: 3 })

    await user.click(screen.getByRole('button', { name: 'Previous page' }))
    // Page 1 is the default and stays out of the URL.
    expect(lastNavigatedSearch()).toEqual({ page: undefined })
  })

  test('hides the pager when every ticket fits on one page', async () => {
    apiMocks.getSelfTickets.mockResolvedValue(buildPage([buildTicket()]))
    renderTickets()

    await screen.findByRole('link', { name: /Cannot top up with card/ })
    expect(screen.queryByRole('navigation', { name: 'Pagination' })).toBeNull()
  })

  test('shows the load error inline and retries on request', async () => {
    const user = userEvent.setup()
    apiMocks.getSelfTickets
      .mockRejectedValueOnce(new Error('Database is unavailable'))
      .mockResolvedValueOnce(buildPage([buildTicket()]))
    renderTickets()

    const alert = await screen.findByRole('alert')
    expect(within(alert).getByText('Database is unavailable')).toBeVisible()

    await user.click(within(alert).getByRole('button', { name: 'Retry' }))

    await waitFor(() =>
      expect(
        screen.getByRole('link', { name: /Cannot top up with card/ })
      ).toBeInTheDocument()
    )
    expect(apiMocks.getSelfTickets).toHaveBeenCalledTimes(2)
  })
})
