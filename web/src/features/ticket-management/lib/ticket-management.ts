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
import { TICKET_STATUS } from '@/features/tickets/constants'
import type {
  AdminTicketQuery,
  TicketStats,
  TicketStatus,
} from '@/features/tickets/types'

export const TICKET_PAGE_SIZE = 20

/** The admin stats field that counts tickets in each status. */
export const TICKET_STATUS_STATS_FIELD: Record<
  TicketStatus,
  Exclude<keyof TicketStats, 'unread'>
> = {
  [TICKET_STATUS.PENDING]: 'pending',
  [TICKET_STATUS.PROCESSING]: 'processing',
  [TICKET_STATUS.RESOLVED]: 'resolved',
  [TICKET_STATUS.CLOSED]: 'closed',
}

/**
 * Statuses support can set together with a reply. Leaving the choice empty
 * omits the field, so the server moves a pending ticket to in progress.
 */
export const REPLY_STATUS_OPTIONS: readonly TicketStatus[] = [
  TICKET_STATUS.PROCESSING,
  TICKET_STATUS.RESOLVED,
  TICKET_STATUS.CLOSED,
]

export type TicketManagementSearch = {
  status?: TicketStatus
  page?: number
  keyword?: string
}

/** Maps the page's URL search state to the admin list request. */
export function buildAdminTicketQuery(
  search: TicketManagementSearch
): AdminTicketQuery {
  const query: AdminTicketQuery = {
    p: search.page ?? 1,
    page_size: TICKET_PAGE_SIZE,
  }
  if (search.status !== undefined) query.status = search.status
  const keyword = search.keyword?.trim()
  if (keyword) query.keyword = keyword
  return query
}
