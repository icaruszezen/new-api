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
import { ArrowLeft01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { Link, getRouteApi } from '@tanstack/react-router'
import { useEffect, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  getSelfTicket,
  getSelfTickets,
  replySelfTicket,
} from '@/features/tickets/api'
import { TicketMessageList } from '@/features/tickets/components/ticket-message-list'
import { TicketReplyComposer } from '@/features/tickets/components/ticket-reply-composer'
import { TICKET_QUERY_KEYS, TICKET_STATUS } from '@/features/tickets/constants'
import { shouldRetryTicketQuery } from '@/features/tickets/lib/query-retry'
import type { TicketReplyValues } from '@/features/tickets/types'

import { NextTicketDetailHeading } from './components/detail-heading'
import {
  NextTicketClosedNotice,
  NextTicketDetailError,
  NextTicketDetailSkeleton,
} from './components/detail-states'
import { NextNewTicketDialog } from './components/new-ticket-dialog'
import { NextTicketsHeading } from './components/page-heading'
import { NextTicketStatusTabs } from './components/status-tabs'
import { NextTicketList } from './components/ticket-list'

const listRoute = getRouteApi('/_authenticated/tickets/')

const TICKETS_PAGE_SIZE = 10

export function NextTickets() {
  const search = listRoute.useSearch()
  const navigate = listRoute.useNavigate()
  const [newTicketOpen, setNewTicketOpen] = useState(false)
  const page = search.page ?? 1

  const query = useQuery({
    queryKey: [
      ...TICKET_QUERY_KEYS.selfList,
      { p: page, page_size: TICKETS_PAGE_SIZE, status: search.status },
    ],
    queryFn: () =>
      getSelfTickets({
        p: page,
        page_size: TICKETS_PAGE_SIZE,
        status: search.status,
      }),
    placeholderData: keepPreviousData,
  })

  return (
    <div data-slot='next-tickets' className='flex flex-col gap-5 pb-10'>
      <NextTicketsHeading onNewTicket={() => setNewTicketOpen(true)} />

      <NextTicketStatusTabs
        status={search.status}
        onStatusChange={(status) =>
          void navigate({
            search: (prev) => ({ ...prev, status, page: undefined }),
          })
        }
      />

      <NextTicketList
        data={query.data}
        page={page}
        pageSize={TICKETS_PAGE_SIZE}
        status={search.status}
        isLoading={query.isPending}
        error={query.error}
        onRetry={() => void query.refetch()}
        onPageChange={(nextPage) =>
          void navigate({
            search: (prev) => ({
              ...prev,
              page: nextPage > 1 ? nextPage : undefined,
            }),
          })
        }
        onNewTicket={() => setNewTicketOpen(true)}
      />

      <NextNewTicketDialog
        open={newTicketOpen}
        onOpenChange={setNewTicketOpen}
      />
    </div>
  )
}

export function NextTicketDetail(props: { ticketId: number }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [newTicketOpen, setNewTicketOpen] = useState(false)
  const detailKey = [...TICKET_QUERY_KEYS.selfDetail, props.ticketId]

  const query = useQuery({
    queryKey: detailKey,
    queryFn: () => getSelfTicket(props.ticketId),
    retry: shouldRetryTicketQuery,
  })

  // Each successful load clears the ticket's unread flag on the server, so
  // the floating button badge and the list markers are stale afterwards.
  useEffect(() => {
    if (query.dataUpdatedAt === 0) return
    void queryClient.invalidateQueries({
      queryKey: TICKET_QUERY_KEYS.selfUnread,
    })
    void queryClient.invalidateQueries({ queryKey: TICKET_QUERY_KEYS.selfList })
  }, [query.dataUpdatedAt, queryClient])

  const replyMutation = useMutation({
    mutationFn: (values: TicketReplyValues) =>
      replySelfTicket(props.ticketId, values),
    onError: (error) => toast.error(error.message || t('Request failed')),
  })

  const handleReply = async (values: TicketReplyValues) => {
    const response = await replyMutation.mutateAsync(values)
    if (!response.success) {
      toast.error(response.message || t('Request failed'))
      // Support may have closed the ticket meanwhile; show its current state.
      void queryClient.invalidateQueries({ queryKey: detailKey })
      return false
    }
    await queryClient.invalidateQueries({ queryKey: detailKey })
    return true
  }

  let body: ReactNode
  if (query.data) {
    const isClosed = query.data.ticket.status === TICKET_STATUS.CLOSED
    body = (
      <>
        <NextTicketDetailHeading ticket={query.data.ticket} />
        <section aria-label={t('Conversation')} className='flex flex-col gap-5'>
          <TicketMessageList
            messages={query.data.messages}
            viewer='user'
            ticketId={props.ticketId}
          />
          {isClosed ? (
            <NextTicketClosedNotice
              onNewTicket={() => setNewTicketOpen(true)}
            />
          ) : (
            <TicketReplyComposer onSubmit={handleReply} />
          )}
        </section>
      </>
    )
  } else if (query.error) {
    body = (
      <NextTicketDetailError
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    )
  } else {
    body = <NextTicketDetailSkeleton />
  }

  return (
    <div
      data-slot='next-ticket-detail'
      className='mx-auto flex w-full max-w-4xl flex-col gap-5 pb-10'
    >
      <Link
        to='/tickets'
        className='text-muted-foreground hover:text-foreground inline-flex w-fit items-center gap-1.5 text-sm transition-colors'
      >
        <HugeiconsIcon
          icon={ArrowLeft01Icon}
          strokeWidth={2}
          aria-hidden='true'
          className='size-4'
        />
        {t('All tickets')}
      </Link>

      {body}

      <NextNewTicketDialog
        open={newTicketOpen}
        onOpenChange={setNewTicketOpen}
      />
    </div>
  )
}
