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
/** 1 pending, 2 in progress, 3 resolved, 4 closed (mirrors the backend). */
export type TicketStatus = 1 | 2 | 3 | 4

type TicketAuthorRole = 'user' | 'admin'

/** Which API surface a request goes through: the ticket owner or support. */
export type TicketScope = 'self' | 'admin'

export type ApiResponse<T> = {
  success: boolean
  message: string
  data: T
}

export type Ticket = {
  id: number
  user_id: number
  subject: string
  status: TicketStatus
  last_reply_role: TicketAuthorRole
  message_count: number
  user_unread: boolean
  admin_unread: boolean
  created_time: number
  updated_time: number
  closed_time: number
  /** Filled for admin views only. */
  username?: string
}

/** Metadata only; the image bytes are fetched separately as a blob. */
export type TicketAttachment = {
  id: number
  ticket_id: number
  message_id: number
  mime_type: string
  size: number
  width: number
  height: number
  created_time: number
}

export type TicketMessage = {
  id: number
  ticket_id: number
  author_role: TicketAuthorRole
  content: string
  created_time: number
  /** Admin views only: the author's username. Empty for the ticket owner. */
  author_name?: string
  attachments: TicketAttachment[] | null
}

export type TicketDetail = {
  ticket: Ticket
  messages: TicketMessage[]
}

export type TicketUser = {
  id: number
  username: string
  display_name: string
  email: string
  group: string
}

export type AdminTicketDetail = TicketDetail & {
  user: TicketUser
}

export type TicketPage = {
  page: number
  page_size: number
  total: number
  items: Ticket[]
}

export type TicketStats = {
  pending: number
  processing: number
  resolved: number
  closed: number
  unread: number
}

export type SelfTicketQuery = {
  p: number
  page_size: number
  status?: TicketStatus
}

export type AdminTicketQuery = SelfTicketQuery & {
  keyword?: string
}

export type CreateTicketValues = {
  subject: string
  content: string
  files: File[]
}

export type TicketReplyValues = {
  content: string
  files: File[]
}

export type AdminTicketReplyValues = TicketReplyValues & {
  /** Omitted: the server moves a pending ticket to in progress. */
  status?: TicketStatus
}
