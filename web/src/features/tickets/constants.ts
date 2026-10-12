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
import type { StatusVariant } from '@/components/status-badge'

import type { TicketStatus } from './types'

export const TICKET_STATUS = {
  PENDING: 1,
  PROCESSING: 2,
  RESOLVED: 3,
  CLOSED: 4,
} as const satisfies Record<string, TicketStatus>

/** Display order for filters and selects. */
export const TICKET_STATUS_VALUES: readonly TicketStatus[] = [
  TICKET_STATUS.PENDING,
  TICKET_STATUS.PROCESSING,
  TICKET_STATUS.RESOLVED,
  TICKET_STATUS.CLOSED,
]

/** `labelKey` is an i18n key; render it with `t(config.labelKey)`. */
export const TICKET_STATUS_CONFIG: Record<
  TicketStatus,
  { labelKey: string; variant: StatusVariant }
> = {
  [TICKET_STATUS.PENDING]: { labelKey: 'Awaiting support', variant: 'warning' },
  [TICKET_STATUS.PROCESSING]: { labelKey: 'In progress', variant: 'info' },
  [TICKET_STATUS.RESOLVED]: { labelKey: 'Resolved', variant: 'success' },
  [TICKET_STATUS.CLOSED]: { labelKey: 'Closed', variant: 'neutral' },
}

/**
 * React Query key roots. Append the query params or ids, and invalidate with
 * the root to refresh every variant (e.g. all pages of the list).
 * Attachment blobs sit outside the self/admin roots so that invalidating a
 * ticket never refetches images that cannot change.
 */
export const TICKET_QUERY_KEYS = {
  selfList: ['tickets', 'self', 'list'],
  selfDetail: ['tickets', 'self', 'detail'],
  selfUnread: ['tickets', 'self', 'unread'],
  adminList: ['tickets', 'admin', 'list'],
  adminStats: ['tickets', 'admin', 'stats'],
  adminDetail: ['tickets', 'admin', 'detail'],
  attachment: ['tickets', 'attachment'],
} as const
