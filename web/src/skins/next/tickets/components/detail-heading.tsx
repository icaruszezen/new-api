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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'
import { closeSelfTicket } from '@/features/tickets/api'
import { TICKET_QUERY_KEYS, TICKET_STATUS } from '@/features/tickets/constants'
import type { Ticket } from '@/features/tickets/types'
import { formatTimestampToDate } from '@/lib/format'

import { NextTicketStatusPill } from './status-pill'

export function NextTicketDetailHeading(props: { ticket: Ticket }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const isClosed = props.ticket.status === TICKET_STATUS.CLOSED

  const closeMutation = useMutation({
    mutationFn: () => closeSelfTicket(props.ticket.id),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Request failed'))
        return
      }
      toast.success(t('Ticket closed'))
      setConfirmOpen(false)
      await queryClient.invalidateQueries({
        queryKey: [...TICKET_QUERY_KEYS.selfDetail, props.ticket.id],
      })
    },
    onError: (error) => toast.error(error.message || t('Request failed')),
  })

  return (
    <header
      data-slot='next-ticket-detail-heading'
      className='flex flex-wrap items-start justify-between gap-x-4 gap-y-3'
    >
      <div className='min-w-0 flex-1'>
        <h1 className='text-3xl font-medium tracking-tight break-words'>
          {props.ticket.subject}
        </h1>
        <div className='text-muted-foreground mt-2 flex flex-wrap items-center gap-x-3 gap-y-1.5 text-sm'>
          <span className='tabular-nums'>#{props.ticket.id}</span>
          <NextTicketStatusPill status={props.ticket.status} />
          <span className='tabular-nums'>
            {t('Created {{time}}', {
              time: formatTimestampToDate(props.ticket.created_time),
            })}
          </span>
        </div>
      </div>

      {!isClosed && (
        <>
          <Button
            type='button'
            variant='outline'
            onClick={() => setConfirmOpen(true)}
            className='h-10 shrink-0 px-4'
          >
            {t('Close ticket')}
          </Button>
          <ConfirmDialog
            open={confirmOpen}
            onOpenChange={setConfirmOpen}
            title={t('Close this ticket?')}
            desc={t(
              'Close it once your problem is solved. You will not be able to reply afterwards, but you can always submit a new ticket.'
            )}
            confirmText={t('Close ticket')}
            destructive
            isLoading={closeMutation.isPending}
            handleConfirm={() => closeMutation.mutate()}
          />
        </>
      )}
    </header>
  )
}
