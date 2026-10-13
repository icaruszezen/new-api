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
import { ImageNotFound01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

import { getTicketAttachmentBlob } from '../api'
import { TICKET_QUERY_KEYS } from '../constants'
import { shouldRetryTicketQuery } from '../lib/query-retry'
import type { TicketAttachment, TicketScope } from '../types'

type TicketAttachmentImageProps = {
  scope: TicketScope
  ticketId: number
  attachment: TicketAttachment
  /** 1-based position within the message, used for the alt text. */
  index: number
  className?: string
}

/**
 * Authenticated attachment thumbnail. The bytes are fetched with the API
 * client and shown through an object URL that is revoked on unmount; the
 * full image opens in a dialog, never in a new tab.
 */
export function TicketAttachmentImage(props: TicketAttachmentImageProps) {
  const { t } = useTranslation()
  const [objectUrl, setObjectUrl] = useState<string | null>(null)
  const query = useQuery({
    queryKey: [
      ...TICKET_QUERY_KEYS.attachment,
      props.scope,
      props.ticketId,
      props.attachment.id,
    ],
    queryFn: () =>
      getTicketAttachmentBlob(props.scope, props.ticketId, props.attachment.id),
    staleTime: Infinity,
    retry: shouldRetryTicketQuery,
  })

  useEffect(() => {
    if (!query.data) return
    const url = URL.createObjectURL(query.data)
    setObjectUrl(url)
    return () => {
      URL.revokeObjectURL(url)
      setObjectUrl(null)
    }
  }, [query.data])

  const alt = t('Attachment {{index}}', { index: props.index })
  const frameClassName = cn(
    'size-24 shrink-0 overflow-hidden rounded-lg border',
    props.className
  )

  if (query.isError) {
    return (
      <div
        data-slot='ticket-attachment-error'
        className={cn(
          frameClassName,
          'bg-muted text-muted-foreground flex items-center justify-center'
        )}
      >
        <HugeiconsIcon
          icon={ImageNotFound01Icon}
          strokeWidth={1.5}
          className='size-6'
          aria-hidden='true'
        />
        <span className='sr-only'>
          {t('Attachment {{index}} could not be loaded', {
            index: props.index,
          })}
        </span>
      </div>
    )
  }

  if (!objectUrl) {
    return (
      <Skeleton
        data-slot='ticket-attachment-loading'
        className={cn(frameClassName, 'rounded-lg')}
      />
    )
  }

  return (
    <Dialog>
      <DialogTrigger
        render={
          <button
            type='button'
            aria-label={t('View attachment {{index}}', { index: props.index })}
            className={cn(
              frameClassName,
              'bg-muted focus-visible:ring-ring/50 block cursor-zoom-in outline-none focus-visible:ring-3'
            )}
          />
        }
      >
        <img
          src={objectUrl}
          alt={alt}
          draggable={false}
          className='size-full object-cover'
        />
      </DialogTrigger>
      {/* Top padding keeps the shared close button off the image. */}
      <DialogContent className='w-max max-w-[calc(100%-2rem)] p-2 pt-10 sm:max-w-[min(90vw,1200px)]'>
        <DialogTitle className='sr-only'>{alt}</DialogTitle>
        <img
          src={objectUrl}
          alt={alt}
          draggable={false}
          className='mx-auto max-h-[min(80vh,calc(100dvh-5rem))] w-auto max-w-full rounded-lg object-contain'
        />
      </DialogContent>
    </Dialog>
  )
}
