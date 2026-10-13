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
import { render, screen, within } from '@testing-library/react'
import { describe, expect, test, vi } from 'vitest'

import type { TicketMessage } from '../types'

vi.mock('../components/ticket-attachment-image', () => ({
  TicketAttachmentImage: (props: { scope: string; index: number }) => (
    <div data-testid='attachment' data-scope={props.scope}>
      {props.index}
    </div>
  ),
}))

const { TicketMessageList } = await import('../components/ticket-message-list')

function message(overrides: Partial<TicketMessage>): TicketMessage {
  return {
    id: 1,
    ticket_id: 7,
    author_role: 'user',
    content: 'Hello',
    created_time: 1_760_000_000,
    attachments: [],
    ...overrides,
  }
}

const THREAD: TicketMessage[] = [
  message({ id: 1, author_role: 'user', author_name: 'alice' }),
  message({
    id: 2,
    author_role: 'admin',
    author_name: 'root',
    content: 'We are on it',
  }),
]

describe('TicketMessageList', () => {
  test('labels the owner as You and support as Support without exposing staff names', () => {
    render(<TicketMessageList messages={THREAD} viewer='user' ticketId={7} />)

    const [own, support] = screen.getAllByRole('listitem')
    expect(within(own).getByText('You')).toBeInTheDocument()
    expect(within(support).getByText('Support')).toBeInTheDocument()
    expect(screen.queryByText('root')).toBeNull()
    expect(screen.queryByText('alice')).toBeNull()
  })

  test('shows author names and a Support tag to administrators', () => {
    render(<TicketMessageList messages={THREAD} viewer='admin' ticketId={7} />)

    const [user, support] = screen.getAllByRole('listitem')
    expect(within(user).getByText('alice')).toBeInTheDocument()
    expect(within(support).getByText('root')).toBeInTheDocument()
    expect(within(support).getByText('Support')).toBeInTheDocument()
  })

  test('sets support replies apart from the owner messages in the user view', () => {
    render(<TicketMessageList messages={THREAD} viewer='user' ticketId={7} />)

    const [own, support] = screen.getAllByRole('listitem')
    expect(support).toHaveClass('bg-muted', 'border-l-4', 'mr-6')
    expect(own).toHaveClass('bg-card', 'ml-6')
    expect(own).not.toHaveClass('border-l-4')
  })

  test('insets support replies as the own side in the admin view', () => {
    render(<TicketMessageList messages={THREAD} viewer='admin' ticketId={7} />)

    const [user, support] = screen.getAllByRole('listitem')
    expect(support).toHaveClass('bg-muted', 'border-l-4', 'ml-6')
    expect(user).toHaveClass('bg-card', 'mr-6')
  })

  test('falls back to User when an administrator sees no author name', () => {
    render(
      <TicketMessageList
        messages={[message({ author_name: '' })]}
        viewer='admin'
        ticketId={7}
      />
    )

    expect(screen.getByText('User')).toBeInTheDocument()
  })

  test('renders markup in message content as literal text', () => {
    const markup = '<img src=x onerror=alert(1)> **bold** https://evil.test'
    render(
      <TicketMessageList
        messages={[message({ content: markup })]}
        viewer='user'
        ticketId={7}
      />
    )

    expect(screen.getByText(markup)).toBeInTheDocument()
    expect(screen.queryByRole('img')).toBeNull()
    expect(screen.queryByRole('link')).toBeNull()
  })

  test('loads attachments through the scope that matches the viewer', () => {
    const attachments = [
      {
        id: 5,
        ticket_id: 7,
        message_id: 1,
        mime_type: 'image/png',
        size: 10,
        width: 1,
        height: 1,
        created_time: 1_760_000_000,
      },
    ]
    const { rerender } = render(
      <TicketMessageList
        messages={[message({ attachments })]}
        viewer='user'
        ticketId={7}
      />
    )
    expect(screen.getByTestId('attachment')).toHaveAttribute(
      'data-scope',
      'self'
    )

    rerender(
      <TicketMessageList
        messages={[message({ attachments })]}
        viewer='admin'
        ticketId={7}
      />
    )
    expect(screen.getByTestId('attachment')).toHaveAttribute(
      'data-scope',
      'admin'
    )
  })
})
