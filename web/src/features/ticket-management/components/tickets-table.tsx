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

import { StaticDataTable } from '@/components/data-table'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { TICKET_STATUS_CONFIG } from '@/features/tickets/constants'
import type { Ticket } from '@/features/tickets/types'
import { formatNumber, formatTimestampToDate } from '@/lib/format'

type TicketsTableProps = {
  tickets: Ticket[]
  page: number
  pageSize: number
  total: number
  onPageChange: (page: number) => void
  onOpen: (ticket: Ticket) => void
}

/**
 * Server-paginated ticket list. The whole row opens the ticket on click; the
 * subject is a button so the same action is reachable from the keyboard.
 */
export function TicketsTable(props: TicketsTableProps) {
  const { t } = useTranslation()
  const pageCount = Math.max(1, Math.ceil(props.total / props.pageSize))

  return (
    <div className='space-y-2'>
      <StaticDataTable tableClassName='min-w-max'>
        <TableHeader>
          <TableRow>
            <TableHead>{t('ID')}</TableHead>
            <TableHead>{t('Subject')}</TableHead>
            <TableHead>{t('User')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead className='text-right'>{t('Messages')}</TableHead>
            <TableHead>{t('Updated')}</TableHead>
            <TableHead>{t('Created')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {props.tickets.length === 0 && (
            <TableRow>
              <TableCell
                colSpan={7}
                className='text-muted-foreground h-24 py-8 text-center'
              >
                {t('No tickets found')}
              </TableCell>
            </TableRow>
          )}
          {props.tickets.map((ticket) => {
            const status = TICKET_STATUS_CONFIG[ticket.status]
            return (
              <TableRow
                key={ticket.id}
                className='cursor-pointer'
                onClick={() => props.onOpen(ticket)}
              >
                <TableCell className='text-muted-foreground font-mono'>
                  #{ticket.id}
                </TableCell>
                <TableCell>
                  <div className='flex max-w-[360px] items-center gap-2'>
                    {ticket.admin_unread && (
                      <>
                        <span
                          aria-hidden='true'
                          className='bg-primary size-2 shrink-0 rounded-full'
                        />
                        <span className='sr-only'>{t('Unread')}</span>
                      </>
                    )}
                    <button
                      type='button'
                      onClick={(event) => {
                        event.stopPropagation()
                        props.onOpen(ticket)
                      }}
                      className='focus-visible:ring-ring/50 min-w-0 truncate rounded-sm text-left font-medium outline-none hover:underline focus-visible:ring-3'
                    >
                      {ticket.subject}
                    </button>
                  </div>
                </TableCell>
                <TableCell>
                  <span className='font-medium'>{ticket.username || '-'}</span>
                  <span className='text-muted-foreground ml-1.5 font-mono text-xs'>
                    #{ticket.user_id}
                  </span>
                </TableCell>
                <TableCell>
                  <StatusBadge
                    label={t(status.labelKey)}
                    variant={status.variant}
                    copyable={false}
                  />
                </TableCell>
                <TableCell className='text-right font-mono tabular-nums'>
                  {formatNumber(ticket.message_count)}
                </TableCell>
                <TableCell className='font-mono'>
                  {formatTimestampToDate(ticket.updated_time)}
                </TableCell>
                <TableCell className='text-muted-foreground font-mono'>
                  {formatTimestampToDate(ticket.created_time)}
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </StaticDataTable>
      <div className='flex flex-wrap items-center justify-end gap-2 text-sm'>
        <span className='text-muted-foreground'>
          {t('{{count}} ticket(s)', { count: props.total })}
        </span>
        <span>
          {t('Page {{current}} of {{total}}', {
            current: props.page,
            total: pageCount,
          })}
        </span>
        <Button
          size='sm'
          variant='outline'
          disabled={props.page <= 1}
          onClick={() =>
            props.onPageChange(Math.min(props.page - 1, pageCount))
          }
        >
          {t('Previous page')}
        </Button>
        <Button
          size='sm'
          variant='outline'
          disabled={props.page >= pageCount}
          onClick={() => props.onPageChange(props.page + 1)}
        >
          {t('Next page')}
        </Button>
      </div>
    </div>
  )
}
