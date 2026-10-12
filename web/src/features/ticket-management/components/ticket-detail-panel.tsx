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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import {
  getAdminTicket,
  replyAdminTicket,
  updateAdminTicketStatus,
} from '@/features/tickets/api'
import { TicketMessageList } from '@/features/tickets/components/ticket-message-list'
import { TicketReplyComposer } from '@/features/tickets/components/ticket-reply-composer'
import {
  TICKET_QUERY_KEYS,
  TICKET_STATUS_CONFIG,
  TICKET_STATUS_VALUES,
} from '@/features/tickets/constants'
import { shouldRetryTicketQuery } from '@/features/tickets/lib/query-retry'
import type {
  AdminTicketReplyValues,
  TicketReplyValues,
  TicketStatus,
} from '@/features/tickets/types'
import { formatTimestampToDate } from '@/lib/format'

import { REPLY_STATUS_OPTIONS } from '../lib/ticket-management'
import { TicketUserSummary } from './ticket-user-summary'

type TicketDetailPanelProps = {
  ticketId: number
}

/** Support view of one ticket: owner, status, thread and reply box. */
export function TicketDetailPanel(props: TicketDetailPanelProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const statusSelectId = useId()
  const replyStatusSelectId = useId()
  const [replyStatus, setReplyStatus] = useState<TicketStatus | undefined>()
  const detailQuery = useQuery({
    queryKey: [...TICKET_QUERY_KEYS.adminDetail, props.ticketId],
    queryFn: () => getAdminTicket(props.ticketId),
    retry: shouldRetryTicketQuery,
  })
  const fetchedAfterOpen = detailQuery.isFetchedAfterMount

  useEffect(() => {
    if (!fetchedAfterOpen) return
    // 打开详情时服务端已清除管理员未读标记，同步列表和计数
    void queryClient.invalidateQueries({
      queryKey: TICKET_QUERY_KEYS.adminList,
    })
    void queryClient.invalidateQueries({
      queryKey: TICKET_QUERY_KEYS.adminStats,
    })
  }, [fetchedAfterOpen, queryClient])

  const refreshTickets = () =>
    Promise.all(
      [
        TICKET_QUERY_KEYS.adminList,
        TICKET_QUERY_KEYS.adminStats,
        TICKET_QUERY_KEYS.adminDetail,
      ].map((queryKey) => queryClient.invalidateQueries({ queryKey }))
    )

  const statusMutation = useMutation({
    mutationFn: (status: TicketStatus) =>
      updateAdminTicketStatus(props.ticketId, status),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to update ticket status'))
        return
      }
      toast.success(t('Ticket status updated'))
      await refreshTickets()
    },
    onError: (error) =>
      toast.error(error.message || t('Failed to update ticket status')),
  })

  const replyMutation = useMutation({
    mutationFn: (values: AdminTicketReplyValues) =>
      replyAdminTicket(props.ticketId, values),
    onError: (error) => toast.error(error.message || t('Failed to send reply')),
  })

  const submitReply = async (values: TicketReplyValues) => {
    const response = await replyMutation.mutateAsync({
      ...values,
      status: replyStatus,
    })
    if (!response.success) {
      toast.error(response.message || t('Failed to send reply'))
      return false
    }
    toast.success(t('Reply sent'))
    setReplyStatus(undefined)
    await refreshTickets()
    return true
  }

  if (detailQuery.isPending) {
    return (
      <>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle className='pr-10'>#{props.ticketId}</SheetTitle>
        </SheetHeader>
        <div className={sideDrawerFormClassName('gap-4')}>
          <Skeleton className='h-28 w-full' />
          <Skeleton className='h-24 w-full' />
          <Skeleton className='h-24 w-full' />
        </div>
      </>
    )
  }

  // A failed background refetch keeps the loaded thread on screen so the
  // composer and its unsent draft stay mounted.
  const detail = detailQuery.data
  if (!detail) {
    return (
      <>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle className='pr-10'>#{props.ticketId}</SheetTitle>
        </SheetHeader>
        <div className={sideDrawerFormClassName('gap-4')}>
          <Alert variant='destructive'>
            <AlertDescription>{detailQuery.error?.message}</AlertDescription>
          </Alert>
          <Button
            variant='outline'
            className='self-start'
            onClick={() => void detailQuery.refetch()}
          >
            {t('Retry')}
          </Button>
        </div>
      </>
    )
  }

  const ticket = detail.ticket
  const displayedStatus = statusMutation.isPending
    ? statusMutation.variables
    : ticket.status

  return (
    <>
      <SheetHeader className={sideDrawerHeaderClassName()}>
        <SheetTitle className='pr-10 break-words'>{ticket.subject}</SheetTitle>
        <SheetDescription className='pr-10'>
          <span className='font-mono'>#{ticket.id}</span>
          <span aria-hidden='true'> · </span>
          {t('Created {{time}}', {
            time: formatTimestampToDate(ticket.created_time),
          })}
        </SheetDescription>
      </SheetHeader>
      <div className={sideDrawerFormClassName('gap-5')}>
        {detailQuery.isError && (
          <Alert variant='destructive'>
            <AlertDescription>{detailQuery.error.message}</AlertDescription>
            <Button
              type='button'
              variant='outline'
              size='xs'
              className='mt-1 justify-self-start'
              onClick={() => void detailQuery.refetch()}
            >
              {t('Retry')}
            </Button>
          </Alert>
        )}
        <TicketUserSummary user={detail.user} />
        <div className='flex flex-wrap items-center gap-3'>
          <label htmlFor={statusSelectId} className='text-sm font-medium'>
            {t('Status')}
          </label>
          <NativeSelect
            id={statusSelectId}
            value={String(displayedStatus)}
            disabled={statusMutation.isPending}
            onChange={(event) => {
              const next = TICKET_STATUS_VALUES.find(
                (status) => String(status) === event.target.value
              )
              if (next !== undefined && next !== ticket.status) {
                statusMutation.mutate(next)
              }
            }}
          >
            {TICKET_STATUS_VALUES.map((status) => (
              <NativeSelectOption key={status} value={String(status)}>
                {t(TICKET_STATUS_CONFIG[status].labelKey)}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </div>
        <TicketMessageList
          messages={detail.messages}
          viewer='admin'
          ticketId={ticket.id}
        />
        <TicketReplyComposer
          onSubmit={submitReply}
          extraActions={
            <div className='flex items-center gap-2'>
              <label
                htmlFor={replyStatusSelectId}
                className='text-muted-foreground text-xs'
              >
                {t('Status after reply')}
              </label>
              <NativeSelect
                id={replyStatusSelectId}
                size='sm'
                value={replyStatus === undefined ? '' : String(replyStatus)}
                onChange={(event) =>
                  setReplyStatus(
                    REPLY_STATUS_OPTIONS.find(
                      (status) => String(status) === event.target.value
                    )
                  )
                }
              >
                <NativeSelectOption value=''>
                  {t('Keep automatic')}
                </NativeSelectOption>
                {REPLY_STATUS_OPTIONS.map((status) => (
                  <NativeSelectOption key={status} value={String(status)}>
                    {t(TICKET_STATUS_CONFIG[status].labelKey)}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </div>
          }
        />
      </div>
    </>
  )
}
