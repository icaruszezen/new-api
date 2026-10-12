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
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type { TicketAttachment } from '../types'

const apiMocks = vi.hoisted(() => ({
  getTicketAttachmentBlob: vi.fn(),
}))

vi.mock('../api', () => apiMocks)

const { TicketAttachmentImage } =
  await import('../components/ticket-attachment-image')

const ATTACHMENT: TicketAttachment = {
  id: 3,
  ticket_id: 7,
  message_id: 11,
  mime_type: 'image/png',
  size: 2048,
  width: 640,
  height: 480,
  created_time: 1_760_000_000,
}

// jsdom has no object URL support.
const objectUrlApi = {
  createObjectURL: Object.getOwnPropertyDescriptor(URL, 'createObjectURL'),
  revokeObjectURL: Object.getOwnPropertyDescriptor(URL, 'revokeObjectURL'),
}
const createObjectURL = vi.fn(() => 'blob:ticket-attachment-3')
const revokeObjectURL = vi.fn()

beforeEach(() => {
  Object.defineProperty(URL, 'createObjectURL', {
    configurable: true,
    writable: true,
    value: createObjectURL,
  })
  Object.defineProperty(URL, 'revokeObjectURL', {
    configurable: true,
    writable: true,
    value: revokeObjectURL,
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

function renderAttachment() {
  // Production retries failed queries by default; a permanent attachment
  // error must still settle after one request.
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: 3, retryDelay: 0 } },
  })
  const Wrapper = (props: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {props.children}
    </QueryClientProvider>
  )
  return render(
    <TicketAttachmentImage
      scope='self'
      ticketId={7}
      attachment={ATTACHMENT}
      index={1}
    />,
    { wrapper: Wrapper }
  )
}

describe('TicketAttachmentImage', () => {
  test('shows the fetched image through an object URL and revokes it on unmount', async () => {
    const blob = new Blob(['png'], { type: 'image/png' })
    apiMocks.getTicketAttachmentBlob.mockResolvedValue(blob)

    const view = renderAttachment()

    const image = await screen.findByAltText('Attachment 1')
    expect(image).toHaveAttribute('src', 'blob:ticket-attachment-3')
    expect(apiMocks.getTicketAttachmentBlob).toHaveBeenCalledWith('self', 7, 3)
    expect(createObjectURL).toHaveBeenCalledWith(blob)
    expect(revokeObjectURL).not.toHaveBeenCalled()

    view.unmount()

    expect(revokeObjectURL).toHaveBeenCalledWith('blob:ticket-attachment-3')
  })

  test('opens the full image in a dialog instead of a new tab', async () => {
    const user = userEvent.setup()
    apiMocks.getTicketAttachmentBlob.mockResolvedValue(
      new Blob(['png'], { type: 'image/png' })
    )
    renderAttachment()

    await user.click(
      await screen.findByRole('button', { name: 'View attachment 1' })
    )

    const dialog = await screen.findByRole('dialog')
    expect(dialog.querySelector('img')).toHaveAttribute(
      'src',
      'blob:ticket-attachment-3'
    )
    expect(document.querySelector('a[target="_blank"]')).toBeNull()
  })

  test('keeps the dialog close button above the full image instead of on it', async () => {
    const user = userEvent.setup()
    apiMocks.getTicketAttachmentBlob.mockResolvedValue(
      new Blob(['png'], { type: 'image/png' })
    )
    renderAttachment()

    await user.click(
      await screen.findByRole('button', { name: 'View attachment 1' })
    )

    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveClass('pt-10')
    expect(
      within(dialog).getByRole('button', { name: 'Close' })
    ).toBeInTheDocument()
  })

  test('shows a placeholder after a single request when the attachment cannot be loaded', async () => {
    apiMocks.getTicketAttachmentBlob.mockRejectedValue(
      new Error('Unexpected attachment type')
    )

    renderAttachment()

    expect(
      await screen.findByText('Attachment 1 could not be loaded')
    ).toBeInTheDocument()
    expect(apiMocks.getTicketAttachmentBlob).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('img')).toBeNull()
    expect(createObjectURL).not.toHaveBeenCalled()
  })
})
