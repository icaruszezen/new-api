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
import { AxiosError, AxiosHeaders } from 'axios'
import { afterEach, describe, expect, test, vi } from 'vitest'

const apiMock = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/lib/api', () => ({ api: apiMock }))

const {
  createSelfTicket,
  getSelfTicket,
  getSelfTickets,
  getTicketAttachmentBlob,
  replyAdminTicket,
  replySelfTicket,
  updateAdminTicketStatus,
} = await import('../api')

const SUCCESS = { success: true, message: '', data: { id: 1 } }

// Writes report HTTP failures themselves instead of the interceptor toasts.
const WRITE_CONFIG = { skipBusinessError: true, skipErrorHandler: true }

function httpError(status: number, data: unknown = '') {
  return new AxiosError(
    `Request failed with status code ${status}`,
    AxiosError.ERR_BAD_REQUEST,
    undefined,
    undefined,
    {
      status,
      statusText: '',
      headers: {},
      config: { headers: new AxiosHeaders() },
      data,
    }
  )
}

function postedForm(): FormData {
  return apiMock.post.mock.calls[0][1] as FormData
}

afterEach(() => {
  vi.resetAllMocks()
})

describe('ticket write requests', () => {
  test('create sends subject, content and one files part per image', async () => {
    apiMock.post.mockResolvedValue({ data: SUCCESS })
    const first = new File(['a'], 'a.png', { type: 'image/png' })
    const second = new File(['b'], 'b.jpg', { type: 'image/jpeg' })

    const response = await createSelfTicket({
      subject: 'Billing question',
      content: 'Charged twice',
      files: [first, second],
    })

    expect(response).toEqual(SUCCESS)
    expect(apiMock.post).toHaveBeenCalledWith(
      '/api/ticket/self',
      expect.any(FormData),
      WRITE_CONFIG
    )
    const form = postedForm()
    expect(form.get('subject')).toBe('Billing question')
    expect(form.get('content')).toBe('Charged twice')
    expect(form.getAll('files')).toEqual([first, second])
  })

  test('user reply posts content and images to the ticket reply endpoint', async () => {
    apiMock.post.mockResolvedValue({ data: SUCCESS })
    const image = new File(['a'], 'a.webp', { type: 'image/webp' })

    await replySelfTicket(7, { content: 'More details', files: [image] })

    expect(apiMock.post).toHaveBeenCalledWith(
      '/api/ticket/self/7/reply',
      expect.any(FormData),
      WRITE_CONFIG
    )
    const form = postedForm()
    expect([...form.keys()]).toEqual(['content', 'files'])
    expect(form.get('content')).toBe('More details')
    expect(form.getAll('files')).toEqual([image])
  })

  test('admin reply omits status when it is left automatic', async () => {
    apiMock.post.mockResolvedValue({ data: SUCCESS })

    await replyAdminTicket(7, { content: 'Looking into it', files: [] })

    expect(apiMock.post).toHaveBeenCalledWith(
      '/api/ticket/admin/7/reply',
      expect.any(FormData),
      WRITE_CONFIG
    )
    const form = postedForm()
    expect(form.has('status')).toBe(false)
    expect(form.getAll('files')).toEqual([])
  })

  test('admin reply sends the chosen status', async () => {
    apiMock.post.mockResolvedValue({ data: SUCCESS })

    await replyAdminTicket(7, { content: 'Fixed', files: [], status: 3 })

    expect(postedForm().get('status')).toBe('3')
  })

  test('admin status change sends JSON to the status endpoint', async () => {
    apiMock.put.mockResolvedValue({ data: SUCCESS })

    await updateAdminTicketStatus(7, 4)

    expect(apiMock.put).toHaveBeenCalledWith(
      '/api/ticket/admin/7/status',
      { status: 4 },
      WRITE_CONFIG
    )
  })
})

describe('ticket write rejections before the ticket handler', () => {
  test('a rate-limited reply resolves as a failed envelope with a translated message', async () => {
    apiMock.post.mockRejectedValue(httpError(429))

    const response = await replySelfTicket(7, { content: 'Hi', files: [] })

    expect(response).toMatchObject({
      success: false,
      message: 'Too many requests. Please try again later.',
    })
  })

  test('an oversized new ticket resolves as a failed envelope with a translated message', async () => {
    apiMock.post.mockRejectedValue(httpError(413))

    const response = await createSelfTicket({
      subject: 'Screenshots',
      content: 'See attached',
      files: [new File(['a'], 'a.png', { type: 'image/png' })],
    })

    expect(response).toMatchObject({
      success: false,
      message: 'The upload is too large. Try fewer or smaller images.',
    })
  })

  test('a rate-limited admin status change resolves as a failed envelope', async () => {
    apiMock.put.mockRejectedValue(httpError(429))

    const response = await updateAdminTicketStatus(7, 3)

    expect(response).toMatchObject({
      success: false,
      message: 'Too many requests. Please try again later.',
    })
  })

  test('a forbidden write resolves with the server message instead of the axios text', async () => {
    apiMock.post.mockRejectedValue(
      httpError(403, { success: false, message: 'No permission' })
    )

    const response = await replyAdminTicket(7, { content: 'Fixed', files: [] })

    expect(response).toMatchObject({ success: false, message: 'No permission' })
  })

  test('other HTTP failures still reject with the original error', async () => {
    const error = httpError(502)
    apiMock.post.mockRejectedValue(error)

    await expect(
      replyAdminTicket(7, { content: 'Fixed', files: [] })
    ).rejects.toBe(error)
  })
})

describe('ticket read requests', () => {
  test('list passes paging and filter params and normalizes null items', async () => {
    apiMock.get.mockResolvedValue({
      data: {
        success: true,
        message: '',
        data: { page: 1, page_size: 10, total: 0, items: null },
      },
    })

    const page = await getSelfTickets({ p: 1, page_size: 10, status: 2 })

    expect(page.items).toEqual([])
    expect(apiMock.get).toHaveBeenCalledWith('/api/ticket/self', {
      skipBusinessError: true,
      params: { p: 1, page_size: 10, status: 2 },
    })
  })

  test('detail rejects with the server message when the ticket is not found', async () => {
    apiMock.get.mockResolvedValue({
      data: { success: false, message: 'Ticket not found', data: null },
    })

    await expect(getSelfTicket(99)).rejects.toThrow('Ticket not found')
  })
})

describe('getTicketAttachmentBlob', () => {
  test('fetches the attachment as an undeduplicated blob without the global error toast', async () => {
    const blob = new Blob(['png'], { type: 'image/png' })
    apiMock.get.mockResolvedValue({ data: blob })

    await expect(getTicketAttachmentBlob('admin', 7, 3)).resolves.toBe(blob)
    expect(apiMock.get).toHaveBeenCalledWith(
      '/api/ticket/admin/7/attachments/3',
      { responseType: 'blob', disableDuplicate: true, skipErrorHandler: true }
    )
  })

  test('uses the owner endpoint for the self scope', async () => {
    apiMock.get.mockResolvedValue({
      data: new Blob(['jpg'], { type: 'image/jpeg' }),
    })

    await getTicketAttachmentBlob('self', 7, 3)

    expect(apiMock.get.mock.calls[0][0]).toBe(
      '/api/ticket/self/7/attachments/3'
    )
  })

  test('rejects a response that is not an allowed image type', async () => {
    apiMock.get.mockResolvedValue({
      data: new Blob(['<script>'], { type: 'text/html' }),
    })

    await expect(getTicketAttachmentBlob('self', 7, 3)).rejects.toThrow()
  })
})
