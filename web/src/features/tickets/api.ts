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
import { isAxiosError, type AxiosResponse } from 'axios'
import { t } from 'i18next'

import { api, type ApiRequestConfig } from '@/lib/api'

import { ACCEPTED_IMAGE_TYPES } from './lib/limits'
import type {
  AdminTicketDetail,
  AdminTicketQuery,
  AdminTicketReplyValues,
  ApiResponse,
  CreateTicketValues,
  SelfTicketQuery,
  Ticket,
  TicketDetail,
  TicketMessage,
  TicketPage,
  TicketReplyValues,
  TicketScope,
  TicketStats,
  TicketStatus,
} from './types'

const SELF_URL = '/api/ticket/self'
const ADMIN_URL = '/api/ticket/admin'

/**
 * Callers show rejected requests themselves: writes toast `response.message`
 * and reads surface the thrown error inline, so the global interceptor must
 * not toast business errors a second time.
 */
const SKIP_BUSINESS_ERROR: ApiRequestConfig = { skipBusinessError: true }

/**
 * Writes also skip the interceptor's HTTP error toast: a 429 or 413 becomes
 * a failed envelope (see `sendTicketWrite`), and the calling mutation's
 * `onError` reports any other failure once.
 */
const TICKET_WRITE: ApiRequestConfig = {
  skipBusinessError: true,
  skipErrorHandler: true,
}

/**
 * Sends a ticket write. The per-user write limit (429) and a request body
 * limit (413, often set by a reverse proxy) reject the request before the
 * ticket handler runs, with an empty or non-JSON body, so they resolve as a
 * failed envelope carrying a translated message. Other failures reject.
 */
async function sendTicketWrite<T>(
  request: Promise<AxiosResponse<ApiResponse<T>>>
): Promise<ApiResponse<T>> {
  try {
    const res = await request
    return res.data
  } catch (error) {
    if (!isAxiosError(error)) throw error
    // Auth failures (401/403) still carry the server's translated envelope.
    const body: unknown = error.response?.data
    if (isFailedEnvelope(body)) return body as ApiResponse<T>
    const status = error.response?.status
    if (status === 429) {
      return {
        success: false,
        message: t('Too many requests. Please try again later.'),
        data: undefined as T,
      }
    }
    if (status === 413) {
      return {
        success: false,
        message: t('The upload is too large. Try fewer or smaller images.'),
        data: undefined as T,
      }
    }
    throw error
  }
}

function isFailedEnvelope(body: unknown): boolean {
  return (
    typeof body === 'object' &&
    body !== null &&
    'success' in body &&
    body.success === false &&
    'message' in body &&
    typeof body.message === 'string' &&
    body.message !== ''
  )
}

/** Reads resolve with `data` and reject with the server message. */
function unwrap<T>(response: ApiResponse<T>): T {
  if (!response.success) {
    throw new Error(response.message || t('Request failed'))
  }
  return response.data
}

/**
 * Multipart body for create/reply. One `files` part per image; the browser
 * sets the multipart Content-Type with its boundary.
 */
function buildTicketFormData(fields: Record<string, string>, files: File[]) {
  const formData = new FormData()
  for (const [name, value] of Object.entries(fields)) {
    formData.append(name, value)
  }
  for (const file of files) {
    formData.append('files', file)
  }
  return formData
}

// ---------------------------------------------------------------------------
// Ticket owner
// ---------------------------------------------------------------------------

export async function getSelfTickets(
  query: SelfTicketQuery
): Promise<TicketPage> {
  const res = await api.get<ApiResponse<TicketPage>>(SELF_URL, {
    ...SKIP_BUSINESS_ERROR,
    params: query,
  })
  const page = unwrap(res.data)
  return { ...page, items: page.items ?? [] }
}

/** Polled in the background, so failures never toast. */
export async function getSelfTicketUnreadCount(): Promise<number> {
  const res = await api.get<ApiResponse<{ count: number }>>(
    `${SELF_URL}/unread`,
    { skipBusinessError: true, skipErrorHandler: true }
  )
  return unwrap(res.data).count
}

/** Opening the detail also clears the ticket's unread flag on the server. */
export async function getSelfTicket(ticketId: number): Promise<TicketDetail> {
  const res = await api.get<ApiResponse<TicketDetail>>(
    `${SELF_URL}/${ticketId}`,
    SKIP_BUSINESS_ERROR
  )
  const detail = unwrap(res.data)
  return { ...detail, messages: detail.messages ?? [] }
}

export async function createSelfTicket(values: CreateTicketValues) {
  const formData = buildTicketFormData(
    { subject: values.subject, content: values.content },
    values.files
  )
  return sendTicketWrite(
    api.post<ApiResponse<Ticket>>(SELF_URL, formData, TICKET_WRITE)
  )
}

export async function replySelfTicket(
  ticketId: number,
  values: TicketReplyValues
) {
  const formData = buildTicketFormData(
    { content: values.content },
    values.files
  )
  return sendTicketWrite(
    api.post<ApiResponse<TicketMessage>>(
      `${SELF_URL}/${ticketId}/reply`,
      formData,
      TICKET_WRITE
    )
  )
}

export async function closeSelfTicket(ticketId: number) {
  return sendTicketWrite(
    api.post<ApiResponse<Ticket>>(
      `${SELF_URL}/${ticketId}/close`,
      undefined,
      TICKET_WRITE
    )
  )
}

// ---------------------------------------------------------------------------
// Support (admin)
// ---------------------------------------------------------------------------

export async function getAdminTickets(
  query: AdminTicketQuery
): Promise<TicketPage> {
  const res = await api.get<ApiResponse<TicketPage>>(ADMIN_URL, {
    ...SKIP_BUSINESS_ERROR,
    params: query,
  })
  const page = unwrap(res.data)
  return { ...page, items: page.items ?? [] }
}

export async function getAdminTicketStats(): Promise<TicketStats> {
  const res = await api.get<ApiResponse<TicketStats>>(
    `${ADMIN_URL}/stats`,
    SKIP_BUSINESS_ERROR
  )
  return unwrap(res.data)
}

/** Opening the detail also clears the ticket's admin unread flag. */
export async function getAdminTicket(
  ticketId: number
): Promise<AdminTicketDetail> {
  const res = await api.get<ApiResponse<AdminTicketDetail>>(
    `${ADMIN_URL}/${ticketId}`,
    SKIP_BUSINESS_ERROR
  )
  const detail = unwrap(res.data)
  return { ...detail, messages: detail.messages ?? [] }
}

export async function replyAdminTicket(
  ticketId: number,
  values: AdminTicketReplyValues
) {
  const fields: Record<string, string> = { content: values.content }
  if (values.status !== undefined) {
    fields.status = String(values.status)
  }
  return sendTicketWrite(
    api.post<ApiResponse<TicketMessage>>(
      `${ADMIN_URL}/${ticketId}/reply`,
      buildTicketFormData(fields, values.files),
      TICKET_WRITE
    )
  )
}

export async function updateAdminTicketStatus(
  ticketId: number,
  status: TicketStatus
) {
  return sendTicketWrite(
    api.put<ApiResponse<Ticket>>(
      `${ADMIN_URL}/${ticketId}/status`,
      { status },
      TICKET_WRITE
    )
  )
}

// ---------------------------------------------------------------------------
// Attachments
// ---------------------------------------------------------------------------

/**
 * Attachment bytes need the Authorization header, so they are fetched as a
 * blob and shown through an object URL; tokens never go into image URLs.
 * The blob must still be an allowlisted image type, so an object URL can
 * never carry markup into this origin.
 */
export async function getTicketAttachmentBlob(
  scope: TicketScope,
  ticketId: number,
  attachmentId: number
): Promise<Blob> {
  const baseUrl = scope === 'admin' ? ADMIN_URL : SELF_URL
  const res = await api.get<Blob>(
    `${baseUrl}/${ticketId}/attachments/${attachmentId}`,
    { responseType: 'blob', disableDuplicate: true, skipErrorHandler: true }
  )
  const mimeType = res.data.type.split(';')[0].trim().toLowerCase()
  if (!ACCEPTED_IMAGE_TYPES.includes(mimeType)) {
    throw new Error('Unexpected attachment type')
  }
  return res.data
}
