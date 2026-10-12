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
  TicketStatus,
} from '@/features/tickets/types'

const apiMocks = vi.hoisted(() => ({
  getAdminTicket: vi.fn(),
  replyAdminTicket: vi.fn(),
  updateAdminTicketStatus: vi.fn(),
  getTicketAttachmentBlob: vi.fn(),
}))

vi.mock('@/features/tickets/api', () => apiMocks)

const { TicketDetailSheet } = await import('../components/ticket-detail-sheet')

const TICKET: Ticket = {
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
}

function buildDetail(status: TicketStatus = 1): AdminTicketDetail {
  return {
    ticket: { ...TICKET, status },
    messages: [
      {
        id: 1,
        ticket_id: TICKET.id,
        author_role: 'user',
        content: 'I paid but my balance did not change.',
        created_time: TICKET.created_time,
        author_name: 'alice',
        attachments: [],
      },
    ],
    user: {
      id: 7,
      username: 'alice',
      display_name: 'Alice Liddell',
      email: 'alice@example.com',
      group: 'vip',
    },
  }
}

function renderSheet() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const Wrapper = (props: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {props.children}
      <Toaster duration={60_000} />
    </QueryClientProvider>
  )
  return render(
    <TicketDetailSheet ticketId={TICKET.id} open onOpenChange={vi.fn()} />,
    { wrapper: Wrapper }
  )
}

async function findSheet() {
  const sheet = await screen.findByRole('dialog')
  await within(sheet).findByText('I paid but my balance did not change.')
  return sheet
}

// The composer's image previews use object URLs, which jsdom lacks.
const objectUrlApi = {
  createObjectURL: Object.getOwnPropertyDescriptor(URL, 'createObjectURL'),
  revokeObjectURL: Object.getOwnPropertyDescriptor(URL, 'revokeObjectURL'),
}

beforeEach(() => {
  Object.defineProperty(URL, 'createObjectURL', {
    configurable: true,
    writable: true,
    value: vi.fn(() => 'blob:preview'),
  })
  Object.defineProperty(URL, 'revokeObjectURL', {
    configurable: true,
    writable: true,
    value: vi.fn(),
  })
  apiMocks.getAdminTicket.mockResolvedValue(buildDetail())
})

afterEach(() => {
  for (const [name, descriptor] of Object.entries(objectUrlApi)) {
    if (descriptor) {
      Object.defineProperty(URL, name, descriptor)
    } else {
      Reflect.deleteProperty(URL, name)
    }
  }
  toast.dismiss()
  vi.clearAllMocks()
})

describe('ticket detail sheet', () => {
  test('shows the subject and the owner account details', async () => {
    renderSheet()
    const sheet = await findSheet()

    expect(
      within(sheet).getByRole('heading', { name: 'Top-up not credited' })
    ).toBeInTheDocument()
    expect(within(sheet).getByText('Alice Liddell')).toBeInTheDocument()
    expect(within(sheet).getByText('alice@example.com')).toBeInTheDocument()
    expect(within(sheet).getByText('vip')).toBeInTheDocument()
    expect(apiMocks.getAdminTicket).toHaveBeenCalledWith(12)
  })

  test('sends the chosen status with the reply and then resets the choice', async () => {
    const user = userEvent.setup()
    apiMocks.replyAdminTicket.mockResolvedValue({
      success: true,
      message: '',
      data: null,
    })
    renderSheet()
    const sheet = await findSheet()
    const replyStatus = within(sheet).getByRole('combobox', {
      name: 'Status after reply',
    })

    await user.selectOptions(replyStatus, 'Resolved')
    await user.type(
      within(sheet).getByRole('textbox', { name: 'Reply' }),
      'Credited manually.'
    )
    await user.click(within(sheet).getByRole('button', { name: 'Send reply' }))

    await waitFor(() => {
      expect(apiMocks.replyAdminTicket).toHaveBeenCalledWith(12, {
        content: 'Credited manually.',
        files: [],
        status: 3,
      })
    })
    expect(await screen.findByText('Reply sent')).toBeInTheDocument()
    await waitFor(() => expect(replyStatus).toHaveValue(''))
  })

  test('leaves the status out of the reply when Keep automatic is selected', async () => {
    const user = userEvent.setup()
    apiMocks.replyAdminTicket.mockResolvedValue({
      success: true,
      message: '',
      data: null,
    })
    renderSheet()
    const sheet = await findSheet()

    expect(
      within(sheet).getByRole('combobox', { name: 'Status after reply' })
    ).toHaveDisplayValue('Keep automatic')
    await user.type(
      within(sheet).getByRole('textbox', { name: 'Reply' }),
      'Looking into it.'
    )
    await user.click(within(sheet).getByRole('button', { name: 'Send reply' }))

    await waitFor(() => {
      expect(apiMocks.replyAdminTicket).toHaveBeenCalledTimes(1)
    })
    const [ticketId, values] = apiMocks.replyAdminTicket.mock.calls[0]
    expect(ticketId).toBe(12)
    expect(values).toMatchObject({ content: 'Looking into it.', files: [] })
    expect(values.status).toBeUndefined()
  })

  test('keeps the reply draft and shows the server message when the reply is rejected', async () => {
    const user = userEvent.setup()
    apiMocks.replyAdminTicket.mockResolvedValue({
      success: false,
      message: 'This ticket has reached its message limit',
      data: null,
    })
    renderSheet()
    const sheet = await findSheet()
    const reply = within(sheet).getByRole('textbox', { name: 'Reply' })

    await user.type(reply, 'One more thing')
    await user.click(within(sheet).getByRole('button', { name: 'Send reply' }))

    expect(
      await screen.findByText('This ticket has reached its message limit')
    ).toBeInTheDocument()
    expect(reply).toHaveValue('One more thing')
  })

  test('changing the status updates the ticket and reloads its detail', async () => {
    const user = userEvent.setup()
    apiMocks.updateAdminTicketStatus.mockResolvedValue({
      success: true,
      message: '',
      data: { ...TICKET, status: 4 },
    })
    renderSheet()
    const sheet = await findSheet()
    const status = within(sheet).getByRole('combobox', { name: 'Status' })
    expect(status).toHaveDisplayValue('Awaiting support')

    apiMocks.getAdminTicket.mockResolvedValue(buildDetail(4))
    await user.selectOptions(status, 'Closed')

    expect(apiMocks.updateAdminTicketStatus).toHaveBeenCalledWith(12, 4)
    expect(await screen.findByText('Ticket status updated')).toBeInTheDocument()
    await waitFor(() => {
      expect(apiMocks.getAdminTicket).toHaveBeenCalledTimes(2)
    })
    await waitFor(() => expect(status).toHaveDisplayValue('Closed'))
  })

  test('keeps the thread and the reply draft when reloading the detail fails', async () => {
    const user = userEvent.setup()
    apiMocks.updateAdminTicketStatus.mockResolvedValue({
      success: true,
      message: '',
      data: { ...TICKET, status: 4 },
    })
    renderSheet()
    const sheet = await findSheet()
    const reply = within(sheet).getByRole('textbox', { name: 'Reply' })
    await user.type(reply, 'Long draft reply')

    apiMocks.getAdminTicket.mockRejectedValue(new Error('Database error'))
    await user.selectOptions(
      within(sheet).getByRole('combobox', { name: 'Status' }),
      'Closed'
    )

    const alert = await within(sheet).findByRole('alert')
    expect(alert).toHaveTextContent('Database error')
    expect(
      within(alert).getByRole('button', { name: 'Retry' })
    ).toBeInTheDocument()
    expect(within(sheet).getByRole('textbox', { name: 'Reply' })).toBe(reply)
    expect(reply).toHaveValue('Long draft reply')
    expect(
      within(sheet).getByText('I paid but my balance did not change.')
    ).toBeInTheDocument()
  })

  test('a rejected status change shows the server message and keeps the current status', async () => {
    const user = userEvent.setup()
    apiMocks.updateAdminTicketStatus.mockResolvedValue({
      success: false,
      message: 'Invalid ticket status',
      data: null,
    })
    renderSheet()
    const sheet = await findSheet()
    const status = within(sheet).getByRole('combobox', { name: 'Status' })

    await user.selectOptions(status, 'Resolved')

    expect(await screen.findByText('Invalid ticket status')).toBeInTheDocument()
    expect(status).toHaveDisplayValue('Awaiting support')
    expect(status).toBeEnabled()
  })
})
