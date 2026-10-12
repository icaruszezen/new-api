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
import {
  Add01Icon,
  ArrowRight01Icon,
  CustomerService01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import type { Ticket, TicketPage, TicketStatus } from '@/features/tickets/types'
import { toIntlLocale } from '@/i18n/languages'
import { formatTimestampRelative, formatTimestampToDate } from '@/lib/format'

import { NextTicketStatusPill } from './status-pill'

const SKELETON_ROWS = ['first', 'second', 'third']

type NextTicketListProps = {
  data?: TicketPage
  page: number
  pageSize: number
  status?: TicketStatus
  isLoading: boolean
  error: Error | null
  onRetry: () => void
  onPageChange: (page: number) => void
  onNewTicket: () => void
}

function NextTicketRow(props: { ticket: Ticket }) {
  const { t, i18n } = useTranslation()
  const updatedAt = props.ticket.updated_time

  return (
    <li data-slot='next-ticket-row' data-unread={props.ticket.user_unread}>
      <Link
        to='/tickets/$ticketId'
        params={{ ticketId: String(props.ticket.id) }}
        className='hover:bg-muted/40 focus-visible:bg-muted/40 focus-visible:ring-ring/50 flex items-center gap-4 px-5 py-4 transition-colors outline-none focus-visible:ring-3 focus-visible:ring-inset'
      >
        <div className='min-w-0 flex-1'>
          <p className='truncate text-sm font-medium'>{props.ticket.subject}</p>
          <p className='text-muted-foreground mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs'>
            <span className='tabular-nums'>#{props.ticket.id}</span>
            <span aria-hidden='true'>·</span>
            <time
              dateTime={new Date(updatedAt * 1000).toISOString()}
              title={formatTimestampToDate(updatedAt)}
            >
              {t('Updated {{time}}', {
                time: formatTimestampRelative(
                  updatedAt,
                  'seconds',
                  toIntlLocale(i18n.language)
                ),
              })}
            </time>
          </p>
        </div>
        <div className='flex shrink-0 flex-col items-end gap-1.5 sm:flex-row sm:items-center sm:gap-3'>
          {props.ticket.user_unread && (
            <span
              data-slot='next-ticket-unread'
              className='inline-flex items-center gap-1.5 text-xs font-medium'
            >
              <span
                aria-hidden='true'
                className='bg-foreground size-1.5 rounded-full'
              />
              {t('New update')}
            </span>
          )}
          <NextTicketStatusPill status={props.ticket.status} />
        </div>
        <HugeiconsIcon
          icon={ArrowRight01Icon}
          strokeWidth={2}
          aria-hidden='true'
          className='text-muted-foreground hidden size-4 shrink-0 sm:block'
        />
      </Link>
    </li>
  )
}

function NextTicketListBody(props: NextTicketListProps) {
  const { t } = useTranslation()

  if (!props.data && props.isLoading) {
    return (
      <ul
        data-slot='next-tickets-loading'
        aria-hidden='true'
        className='divide-y'
      >
        {SKELETON_ROWS.map((row) => (
          <li key={row} className='flex items-center gap-4 px-5 py-4'>
            <div className='flex-1 space-y-2'>
              <Skeleton className='h-4 w-2/3 max-w-sm' />
              <Skeleton className='h-3 w-40' />
            </div>
            <Skeleton className='h-5 w-16 rounded-full' />
          </li>
        ))}
      </ul>
    )
  }

  if (!props.data) {
    return (
      <div
        role='alert'
        data-slot='next-tickets-error'
        className='flex flex-col items-center gap-3 px-5 py-12 text-center'
      >
        <div>
          <p className='text-sm font-medium'>
            {t('Failed to load your tickets')}
          </p>
          {props.error?.message ? (
            <p className='text-muted-foreground mt-1 text-sm'>
              {props.error.message}
            </p>
          ) : null}
        </div>
        <Button type='button' variant='outline' onClick={props.onRetry}>
          {t('Retry')}
        </Button>
      </div>
    )
  }

  if (props.data.items.length > 0) {
    return (
      <ul data-slot='next-tickets-rows' className='divide-y'>
        {props.data.items.map((ticket) => (
          <NextTicketRow key={ticket.id} ticket={ticket} />
        ))}
      </ul>
    )
  }

  if (props.status !== undefined || props.page > 1) {
    return (
      <p
        data-slot='next-tickets-empty'
        className='text-muted-foreground px-5 py-12 text-center text-sm'
      >
        {props.page > 1
          ? t('No tickets on this page.')
          : t('No tickets with this status.')}
      </p>
    )
  }

  return (
    <div
      data-slot='next-tickets-empty'
      className='flex flex-col items-center px-5 py-14 text-center'
    >
      <span className='bg-muted text-muted-foreground flex size-10 items-center justify-center rounded-lg'>
        <HugeiconsIcon
          icon={CustomerService01Icon}
          strokeWidth={1.75}
          aria-hidden='true'
          className='size-5'
        />
      </span>
      <h2 className='mt-4 text-lg font-medium tracking-tight'>
        {t('No tickets yet')}
      </h2>
      <p className='text-muted-foreground mt-1 max-w-sm text-sm'>
        {t(
          'Describe your problem and attach screenshots if they help. We will reply on this page.'
        )}
      </p>
      <Button
        type='button'
        onClick={props.onNewTicket}
        className='mt-5 h-10 px-4'
      >
        <HugeiconsIcon icon={Add01Icon} strokeWidth={2} aria-hidden='true' />
        {t('New ticket')}
      </Button>
    </div>
  )
}

export function NextTicketList(props: NextTicketListProps) {
  const { t } = useTranslation()
  const total = props.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / props.pageSize))
  // Keep the pager on an out-of-range page so the user can step back.
  const showPager = Boolean(props.data) && (totalPages > 1 || props.page > 1)

  return (
    <section
      data-slot='next-tickets-list'
      aria-busy={props.isLoading}
      className='bg-card overflow-hidden rounded-xl border'
    >
      <NextTicketListBody {...props} />
      {showPager && (
        <nav
          aria-label={t('Pagination')}
          data-slot='next-tickets-pager'
          className='flex items-center justify-between gap-3 border-t px-5 py-3'
        >
          <span className='text-muted-foreground text-xs tabular-nums'>
            {t('Page {{current}} of {{total}}', {
              current: Math.min(props.page, totalPages),
              total: totalPages,
            })}
          </span>
          <div className='flex items-center gap-2'>
            <Button
              type='button'
              variant='outline'
              size='sm'
              disabled={props.page <= 1}
              onClick={() =>
                props.onPageChange(Math.min(props.page - 1, totalPages))
              }
            >
              {t('Previous page')}
            </Button>
            <Button
              type='button'
              variant='outline'
              size='sm'
              disabled={props.page >= totalPages}
              onClick={() => props.onPageChange(props.page + 1)}
            >
              {t('Next page')}
            </Button>
          </div>
        </nav>
      )}
    </section>
  )
}
