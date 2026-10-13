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

import { TICKET_QUERY_KEYS } from '@/features/tickets/constants'

const mocks = vi.hoisted(() => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn() },
  navigate: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}))

// The real ticket API module runs, so the multipart body is the one sent.
vi.mock('@/lib/api', () => ({ api: mocks.api }))

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => mocks.navigate,
}))

vi.mock('sonner', () => ({
  toast: {
    success: (...args: unknown[]) => mocks.toastSuccess(...args),
    error: (...args: unknown[]) => mocks.toastError(...args),
  },
}))

const { NextNewTicketDialog } = await import('../components/new-ticket-dialog')

// jsdom has no object URL support; the picker previews need it.
const objectUrlApi = {
  createObjectURL: Object.getOwnPropertyDescriptor(URL, 'createObjectURL'),
  revokeObjectURL: Object.getOwnPropertyDescriptor(URL, 'revokeObjectURL'),
}

function renderDialog() {
  const onOpenChange = vi.fn()
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const Wrapper = (props: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {props.children}
    </QueryClientProvider>
  )
  render(<NextNewTicketDialog open onOpenChange={onOpenChange} />, {
    wrapper: Wrapper,
  })
  const dialog = screen.getByRole('dialog', { name: 'New ticket' })
  const fileInput = dialog.querySelector<HTMLInputElement>('input[type="file"]')
  if (!fileInput) throw new Error('file input not rendered')
  return { dialog, fileInput, onOpenChange, queryClient }
}

function postedForm(): FormData {
  return mocks.api.post.mock.calls[0][1] as FormData
}

describe('next new ticket dialog', () => {
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
  })

  afterEach(() => {
    for (const [name, descriptor] of Object.entries(objectUrlApi)) {
      if (descriptor) {
        Object.defineProperty(URL, name, descriptor)
      } else {
        Reflect.deleteProperty(URL, name)
      }
    }
    vi.resetAllMocks()
  })

  test('posts the trimmed subject, description and images as multipart form data', async () => {
    const user = userEvent.setup()
    mocks.api.post.mockResolvedValue({
      data: { success: true, message: '', data: { id: 15 } },
    })
    const { dialog, fileInput, onOpenChange, queryClient } = renderDialog()
    const listKey = [...TICKET_QUERY_KEYS.selfList, { p: 1 }]
    const unreadKey = [...TICKET_QUERY_KEYS.selfUnread, 1]
    queryClient.setQueryData(listKey, { items: [] })
    queryClient.setQueryData(unreadKey, 0)
    const screenshot = new File(['png'], 'error.png', { type: 'image/png' })

    await user.type(
      within(dialog).getByRole('textbox', { name: 'Subject' }),
      '  Top-up failed  '
    )
    await user.type(
      within(dialog).getByRole('textbox', { name: 'Issue description' }),
      'The card payment was declined twice.'
    )
    await user.upload(fileInput, screenshot)
    await user.click(
      within(dialog).getByRole('button', { name: 'Submit ticket' })
    )

    await waitFor(() => expect(mocks.api.post).toHaveBeenCalledTimes(1))
    expect(mocks.api.post.mock.calls[0][0]).toBe('/api/ticket/self')
    const form = postedForm()
    expect(form.get('subject')).toBe('Top-up failed')
    expect(form.get('content')).toBe('The card payment was declined twice.')
    expect(form.getAll('files')).toEqual([screenshot])

    await waitFor(() =>
      expect(mocks.navigate).toHaveBeenCalledWith({
        to: '/tickets/$ticketId',
        params: { ticketId: '15' },
      })
    )
    expect(mocks.toastSuccess).toHaveBeenCalledWith('Ticket submitted')
    expect(onOpenChange).toHaveBeenCalledWith(false)
    // The list and the floating button badge refetch with the new ticket.
    expect(queryClient.getQueryState(listKey)?.isInvalidated).toBe(true)
    expect(queryClient.getQueryState(unreadKey)?.isInvalidated).toBe(true)
  })

  test('blocks a whitespace-only subject and description before any request', async () => {
    const user = userEvent.setup()
    const { dialog } = renderDialog()

    await user.type(
      within(dialog).getByRole('textbox', { name: 'Subject' }),
      '   '
    )
    await user.type(
      within(dialog).getByRole('textbox', { name: 'Issue description' }),
      '   '
    )
    await user.click(
      within(dialog).getByRole('button', { name: 'Submit ticket' })
    )

    expect(
      await within(dialog).findByText('Please enter a subject')
    ).toBeInTheDocument()
    expect(
      within(dialog).getByText('Please describe your issue')
    ).toBeInTheDocument()
    expect(
      within(dialog).getByRole('textbox', { name: 'Subject' })
    ).toHaveAttribute('aria-invalid', 'true')
    expect(mocks.api.post).not.toHaveBeenCalled()
  })

  test('keeps the dialog and draft open when the server rejects the ticket', async () => {
    const user = userEvent.setup()
    mocks.api.post.mockResolvedValue({
      data: {
        success: false,
        message: 'You can have at most 5 open tickets',
        data: null,
      },
    })
    const { dialog, onOpenChange } = renderDialog()

    await user.type(
      within(dialog).getByRole('textbox', { name: 'Subject' }),
      'Another question'
    )
    await user.type(
      within(dialog).getByRole('textbox', { name: 'Issue description' }),
      'Details'
    )
    await user.click(
      within(dialog).getByRole('button', { name: 'Submit ticket' })
    )

    await waitFor(() =>
      expect(mocks.toastError).toHaveBeenCalledWith(
        'You can have at most 5 open tickets'
      )
    )
    expect(onOpenChange).not.toHaveBeenCalled()
    expect(mocks.navigate).not.toHaveBeenCalled()
    expect(
      within(dialog).getByRole('textbox', { name: 'Subject' })
    ).toHaveValue('Another question')
  })
})
