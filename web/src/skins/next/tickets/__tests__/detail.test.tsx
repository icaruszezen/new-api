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
import { AxiosError } from 'axios'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { TICKET_QUERY_KEYS } from '@/features/tickets/constants'
import type {
  Ticket,
  TicketDetail,
  TicketStatus,
} from '@/features/tickets/types'
import { handleServerError } from '@/lib/handle-server-error'

const apiMocks = vi.hoisted(() => ({
  getSelfTicket: vi.fn(),
  replySelfTicket: vi.fn(),
  closeSelfTicket: vi.fn(),
  createSelfTicket: vi.fn(),
  getSelfTickets: vi.fn(),
  getTicketAttachmentBlob: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
}))

vi.mock('@/features/tickets/api', () => apiMocks)

vi.mock('sonner', () => ({
  toast: {
    success: (...args: unknown[]) => toastMocks.success(...args),
    error: (...args: unknown[]) => toastMocks.error(...args),
  },
}))

vi.mock('@tanstack/react-router', () => ({
  getRouteApi: () => ({
    useSearch: () => ({}),
    useNavigate: () => vi.fn(),
  }),
  useNavigate: () => vi.fn(),
  Link: (props: { to: string; className?: string; children: ReactNode }) => (
    <a href={props.to} className={props.className}>
      {props.children}
    </a>
  ),
}))

const { NextTicketDetail } = await import('../index')

const TICKET_ID = 42

function buildDetail(status: TicketStatus): TicketDetail {
  const ticket: Ticket = {
    id: TICKET_ID,
    user_id: 1,
    subject: 'API returns 502',
    status,
    last_reply_role: 'admin',
    message_count: 2,
    user_unread: false,
    admin_unread: false,
    created_time: 1760000000,
    updated_time: 1760003600,
    closed_time: 0,
  }
  return {
    ticket,
    messages: [
      {
        id: 1,
        ticket_id: TICKET_ID,
        author_role: 'user',
        content: 'Every request to /v1/chat fails.',
        created_time: 1760000000,
        attachments: [],
      },
      {
        id: 2,
        ticket_id: TICKET_ID,
        author_role: 'admin',
        content: 'We are looking into it.',
        created_time: 1760003600,
        attachments: [],
      },
    ],
  }
}

function renderDetail() {
  // Mirrors the production defaults in main.tsx: failed queries are retried
  // and failed mutations fall back to the global server error toast.
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: 3, retryDelay: 0 },
      mutations: { onError: handleServerError },
    },
  })
  const Wrapper = (props: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {props.children}
    </QueryClientProvider>
  )
  render(<NextTicketDetail ticketId={TICKET_ID} />, { wrapper: Wrapper })
  return { queryClient }
}

describe('next support ticket detail', () => {
  afterEach(() => {
    vi.resetAllMocks()
  })

  test('shows the ticket heading and the thread labelled from the user side', async () => {
    apiMocks.getSelfTicket.mockResolvedValue(buildDetail(2))
    renderDetail()

    expect(
      await screen.findByRole('heading', { level: 1, name: 'API returns 502' })
    ).toBeInTheDocument()
    expect(screen.getByText('#42')).toBeInTheDocument()
    expect(screen.getByText('In progress')).toBeInTheDocument()
    expect(screen.getByText('You')).toBeInTheDocument()
    expect(screen.getByText('Support')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'All tickets' })).toHaveAttribute(
      'href',
      '/tickets'
    )
    expect(apiMocks.getSelfTicket).toHaveBeenCalledWith(TICKET_ID)
  })

  test('sends a reply and reloads the thread', async () => {
    const user = userEvent.setup()
    apiMocks.getSelfTicket.mockResolvedValue(buildDetail(3))
    apiMocks.replySelfTicket.mockResolvedValue({
      success: true,
      message: '',
      data: {},
    })
    renderDetail()

    await user.type(
      await screen.findByRole('textbox', { name: 'Reply' }),
      'It is failing again.'
    )
    await user.click(screen.getByRole('button', { name: 'Send reply' }))

    await waitFor(() =>
      expect(apiMocks.replySelfTicket).toHaveBeenCalledWith(TICKET_ID, {
        content: 'It is failing again.',
        files: [],
      })
    )
    await waitFor(() => expect(apiMocks.getSelfTicket).toHaveBeenCalledTimes(2))
    expect(screen.getByRole('textbox', { name: 'Reply' })).toHaveValue('')
  })

  test('reports a failed reply request in one toast and keeps the draft', async () => {
    const user = userEvent.setup()
    apiMocks.getSelfTicket.mockResolvedValue(buildDetail(2))
    apiMocks.replySelfTicket.mockRejectedValue(
      new AxiosError('Request failed with status code 502')
    )
    renderDetail()

    const reply = await screen.findByRole('textbox', { name: 'Reply' })
    await user.type(reply, 'Still broken')
    await user.click(screen.getByRole('button', { name: 'Send reply' }))

    await waitFor(() =>
      expect(toastMocks.error).toHaveBeenCalledWith(
        'Request failed with status code 502'
      )
    )
    expect(toastMocks.error).toHaveBeenCalledTimes(1)
    expect(reply).toHaveValue('Still broken')
  })

  test('closes the ticket only after the user confirms', async () => {
    const user = userEvent.setup()
    apiMocks.getSelfTicket
      .mockResolvedValueOnce(buildDetail(1))
      .mockResolvedValueOnce(buildDetail(4))
    apiMocks.closeSelfTicket.mockResolvedValue({
      success: true,
      message: '',
      data: buildDetail(4).ticket,
    })
    renderDetail()

    await user.click(
      await screen.findByRole('button', { name: 'Close ticket' })
    )
    const confirm = await screen.findByRole('alertdialog', {
      name: 'Close this ticket?',
    })
    expect(apiMocks.closeSelfTicket).not.toHaveBeenCalled()

    await user.click(
      within(confirm).getByRole('button', { name: 'Close ticket' })
    )

    await waitFor(() =>
      expect(apiMocks.closeSelfTicket).toHaveBeenCalledWith(TICKET_ID)
    )
    expect(toastMocks.success).toHaveBeenCalledWith('Ticket closed')
    expect(
      await screen.findByText(
        'This ticket is closed. Submit a new ticket if you still need help.'
      )
    ).toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: 'Reply' })).toBeNull()
  })

  test('replaces the reply box of a closed ticket with a new-ticket prompt', async () => {
    const user = userEvent.setup()
    apiMocks.getSelfTicket.mockResolvedValue(buildDetail(4))
    renderDetail()

    await screen.findByRole('heading', { level: 1, name: 'API returns 502' })
    expect(screen.queryByRole('textbox', { name: 'Reply' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Close ticket' })).toBeNull()

    await user.click(screen.getByRole('button', { name: 'New ticket' }))
    expect(
      await screen.findByRole('dialog', { name: 'New ticket' })
    ).toBeInTheDocument()
  })

  test('shows a not-found state after a single request for a missing or foreign ticket', async () => {
    apiMocks.getSelfTicket.mockRejectedValue(new Error('Ticket not found'))
    renderDetail()

    const alert = await screen.findByRole('alert')
    expect(
      within(alert).getByRole('heading', { name: 'Ticket not found' })
    ).toBeInTheDocument()
    expect(apiMocks.getSelfTicket).toHaveBeenCalledTimes(1)
    expect(within(alert).queryByRole('button', { name: 'Retry' })).toBeNull()
    expect(screen.getByRole('link', { name: 'All tickets' })).toHaveAttribute(
      'href',
      '/tickets'
    )
  })

  test('offers a retry when the ticket could not be fetched', async () => {
    const user = userEvent.setup()
    apiMocks.getSelfTicket.mockRejectedValue(new AxiosError('Network Error'))
    renderDetail()

    const alert = await screen.findByRole('alert')
    expect(
      within(alert).getByRole('heading', { name: 'Failed to load this ticket' })
    ).toBeInTheDocument()

    apiMocks.getSelfTicket.mockResolvedValue(buildDetail(1))
    await user.click(within(alert).getByRole('button', { name: 'Retry' }))

    expect(
      await screen.findByRole('heading', { level: 1, name: 'API returns 502' })
    ).toBeInTheDocument()
  })

  test('refreshes the unread badge and list once the ticket has loaded', async () => {
    apiMocks.getSelfTicket.mockResolvedValue(buildDetail(2))
    const { queryClient } = renderDetail()
    const unreadKey = [...TICKET_QUERY_KEYS.selfUnread, 1]
    const listKey = [...TICKET_QUERY_KEYS.selfList, { p: 1 }]
    queryClient.setQueryData(unreadKey, 1)
    queryClient.setQueryData(listKey, { items: [] })

    await screen.findByRole('heading', { level: 1, name: 'API returns 502' })

    await waitFor(() =>
      expect(queryClient.getQueryState(unreadKey)?.isInvalidated).toBe(true)
    )
    expect(queryClient.getQueryState(listKey)?.isInvalidated).toBe(true)
  })
})
