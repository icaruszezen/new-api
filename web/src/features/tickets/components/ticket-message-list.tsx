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

import { formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import type { TicketMessage } from '../types'
import { TicketAttachmentImage } from './ticket-attachment-image'

type TicketMessageListProps = {
  messages: TicketMessage[]
  viewer: 'user' | 'admin'
  ticketId: number
  className?: string
}

/**
 * Ticket thread. Message text is user input and is rendered strictly as
 * plain text: no markdown, no HTML, no auto-linking.
 */
export function TicketMessageList(props: TicketMessageListProps) {
  const { t } = useTranslation()
  const scope = props.viewer === 'admin' ? 'admin' : 'self'

  return (
    <ol
      data-slot='ticket-message-list'
      className={cn('flex flex-col gap-3', props.className)}
    >
      {props.messages.map((message) => {
        const isSupport = message.author_role === 'admin'
        const isOwn = props.viewer === 'admin' ? isSupport : !isSupport
        const attachments = message.attachments ?? []
        let author: string
        if (props.viewer === 'user') {
          author = isSupport ? t('Support') : t('You')
        } else if (isSupport) {
          author = message.author_name || t('Admin')
        } else {
          author = message.author_name || t('User')
        }

        return (
          <li
            key={message.id}
            data-slot='ticket-message'
            data-author-role={message.author_role}
            className={cn(
              'rounded-xl border px-4 py-3',
              // Support replies get an accent edge, and the viewer's own
              // messages sit inset from the other side of the thread.
              isSupport ? 'bg-muted border-l-primary/50 border-l-4' : 'bg-card',
              isOwn ? 'ml-6 sm:ml-12' : 'mr-6 sm:mr-12'
            )}
          >
            <div className='flex flex-wrap items-center gap-x-2 gap-y-1 text-sm'>
              <span className='min-w-0 truncate font-medium'>{author}</span>
              {props.viewer === 'admin' && isSupport && (
                <span className='text-muted-foreground inline-flex items-center rounded-full border px-2 py-0.5 text-xs'>
                  {t('Support')}
                </span>
              )}
              <time
                dateTime={new Date(message.created_time * 1000).toISOString()}
                className='text-muted-foreground ml-auto text-xs tabular-nums'
              >
                {formatTimestampToDate(message.created_time)}
              </time>
            </div>
            {message.content && (
              <p className='mt-2 text-sm break-words whitespace-pre-wrap'>
                {message.content}
              </p>
            )}
            {attachments.length > 0 && (
              <div className='mt-3 flex flex-wrap gap-2'>
                {attachments.map((attachment, index) => (
                  <TicketAttachmentImage
                    key={attachment.id}
                    scope={scope}
                    ticketId={props.ticketId}
                    attachment={attachment}
                    index={index + 1}
                  />
                ))}
              </div>
            )}
          </li>
        )
      })}
    </ol>
  )
}
