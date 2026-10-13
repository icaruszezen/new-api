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
import { isAxiosError } from 'axios'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'

export function NextTicketDetailSkeleton() {
  return (
    <div
      data-slot='next-ticket-detail-loading'
      aria-hidden='true'
      className='flex flex-col gap-5'
    >
      <div className='space-y-3'>
        <Skeleton className='h-9 w-2/3' />
        <Skeleton className='h-4 w-64' />
      </div>
      <Skeleton className='h-28 w-full rounded-xl' />
      <Skeleton className='h-28 w-full rounded-xl' />
    </div>
  )
}

export function NextTicketDetailError(props: {
  error: Error
  onRetry: () => void
}) {
  const { t } = useTranslation()
  // A missing or foreign ticket is a business error carrying the server
  // message; only transport failures (axios errors) are worth retrying.
  const isTransportError = isAxiosError(props.error)

  return (
    <section
      role='alert'
      data-slot='next-ticket-detail-error'
      className='bg-card flex flex-col items-center rounded-xl border px-5 py-14 text-center'
    >
      <h1 className='text-lg font-medium tracking-tight'>
        {isTransportError
          ? t('Failed to load this ticket')
          : t('Ticket not found')}
      </h1>
      <p className='text-muted-foreground mt-1 max-w-sm text-sm'>
        {isTransportError
          ? t('Check your connection and try again.')
          : t('This ticket does not exist or belongs to another account.')}
      </p>
      {isTransportError && (
        <Button
          type='button'
          variant='outline'
          onClick={props.onRetry}
          className='mt-5'
        >
          {t('Retry')}
        </Button>
      )}
    </section>
  )
}

export function NextTicketClosedNotice(props: { onNewTicket: () => void }) {
  const { t } = useTranslation()

  return (
    <div
      data-slot='next-ticket-closed'
      className='bg-muted/40 flex flex-col items-start gap-3 rounded-xl border px-5 py-4 sm:flex-row sm:items-center sm:justify-between'
    >
      <p className='text-muted-foreground text-sm'>
        {t(
          'This ticket is closed. Submit a new ticket if you still need help.'
        )}
      </p>
      <Button
        type='button'
        onClick={props.onNewTicket}
        className='h-9 shrink-0 px-3'
      >
        {t('New ticket')}
      </Button>
    </div>
  )
}
