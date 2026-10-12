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
import { useTranslation } from 'react-i18next'

import {
  TICKET_STATUS,
  TICKET_STATUS_CONFIG,
} from '@/features/tickets/constants'
import type { TicketStatus } from '@/features/tickets/types'
import { cn } from '@/lib/utils'

// The Next palette is monochrome; only the small dot carries a status hue.
const STATUS_DOT_CLASS: Record<TicketStatus, string> = {
  [TICKET_STATUS.PENDING]: 'bg-amber-500',
  [TICKET_STATUS.PROCESSING]: 'bg-sky-500',
  [TICKET_STATUS.RESOLVED]: 'bg-emerald-500',
  [TICKET_STATUS.CLOSED]: 'bg-muted-foreground/50',
}

export function NextTicketStatusPill(props: {
  status: TicketStatus
  className?: string
}) {
  const { t } = useTranslation()
  const isOpen =
    props.status === TICKET_STATUS.PENDING ||
    props.status === TICKET_STATUS.PROCESSING

  return (
    <span
      data-slot='next-ticket-status'
      data-status={props.status}
      className={cn(
        'border-border inline-flex shrink-0 items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs',
        isOpen ? 'text-foreground' : 'text-muted-foreground',
        props.className
      )}
    >
      <span
        aria-hidden='true'
        className={cn('size-1.5 rounded-full', STATUS_DOT_CLASS[props.status])}
      />
      {t(TICKET_STATUS_CONFIG[props.status].labelKey)}
    </span>
  )
}
