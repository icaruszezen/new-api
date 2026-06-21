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
import { ChevronDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'

import type { DebugCaptureRecord } from './debug-capture-api'

type ReasonVariant = 'default' | 'secondary' | 'destructive' | 'outline'

function reasonBadgeVariant(reason: string): ReasonVariant {
  if (reason === 'error') return 'destructive'
  if (reason === 'recent') return 'outline'
  return 'secondary'
}

function reasonLabel(reason: string): string {
  switch (reason) {
    case 'error':
      return 'Error'
    case 'billing_no_usage':
      return 'No billing info'
    case 'billing_zero_tokens':
      return 'Zero tokens (cannot charge)'
    case 'recent':
      return 'Recent'
    default:
      return reason
  }
}

function BodyBlock(props: { label: string; content: string; empty: string }) {
  return (
    <div className='min-w-0 space-y-1'>
      <div className='text-muted-foreground text-xs font-medium'>
        {props.label}
      </div>
      {props.content ? (
        <pre className='bg-muted/40 max-h-80 overflow-auto rounded-md border p-3 text-xs break-all whitespace-pre-wrap'>
          {props.content}
        </pre>
      ) : (
        <p className='text-muted-foreground text-xs italic'>{props.empty}</p>
      )}
    </div>
  )
}

type Props = {
  records: DebugCaptureRecord[]
  emptyMessage: string
}

export function DebugCaptureRecords(props: Props) {
  const { t } = useTranslation()

  if (props.records.length === 0) {
    return (
      <div className='text-muted-foreground rounded-md border border-dashed p-6 text-center text-sm'>
        {props.emptyMessage}
      </div>
    )
  }

  return (
    <div className='flex flex-col gap-2'>
      {props.records.map((record) => (
        <Collapsible
          key={record.id}
          className='min-w-0 rounded-md border'
        >
          <CollapsibleTrigger className='group/trigger hover:bg-muted/40 flex w-full min-w-0 items-center gap-2 px-3 py-2 text-left text-sm'>
            <ChevronDown
              className='text-muted-foreground size-4 shrink-0 transition-transform group-data-[panel-open]/trigger:rotate-180'
              aria-hidden='true'
            />
            <Badge variant={reasonBadgeVariant(record.reason)}>
              {t(reasonLabel(record.reason))}
            </Badge>
            <span className='text-muted-foreground shrink-0 text-xs'>
              {dayjs(record.timestamp).format('MM-DD HH:mm:ss')}
            </span>
            <span className='truncate font-medium'>
              {record.model || record.path || '-'}
            </span>
            <span className='ml-auto flex shrink-0 items-center gap-2'>
              {record.is_stream ? (
                <Badge variant='outline'>{t('Stream')}</Badge>
              ) : null}
              {record.error_code ? (
                <Badge variant='destructive'>{record.error_code}</Badge>
              ) : null}
              {record.upstream_status ? (
                <span className='text-muted-foreground text-xs'>
                  HTTP {record.upstream_status}
                </span>
              ) : null}
            </span>
          </CollapsibleTrigger>
          <CollapsibleContent className='space-y-3 border-t px-3 py-3'>
            <div className='text-muted-foreground grid grid-cols-2 gap-x-4 gap-y-1 text-xs md:grid-cols-3'>
              <div>
                {t('Request ID')}: {record.request_id || '-'}
              </div>
              <div>
                {t('Channel ID')}: {record.channel_id || '-'}
              </div>
              <div>
                {t('User ID')}: {record.user_id || '-'}
              </div>
              <div>
                {t('Method')}: {record.method || '-'}
              </div>
              <div className='col-span-2 truncate md:col-span-2'>
                {t('Path')}: {record.path || '-'}
              </div>
              {record.retry_index > 0 ? (
                <div>
                  {t('Retry')}: {record.retry_index}
                </div>
              ) : null}
            </div>

            {record.error_message ? (
              <div className='text-destructive text-xs break-all'>
                {record.error_message}
              </div>
            ) : null}

            <div
              className={cn(
                'flex flex-wrap gap-2',
                !record.truncated &&
                  !record.upstream_capture_incomplete &&
                  'hidden'
              )}
            >
              {record.truncated ? (
                <Badge variant='outline'>{t('Body truncated')}</Badge>
              ) : null}
              {record.upstream_capture_incomplete ? (
                <Badge variant='outline'>
                  {t('Upstream response not captured (SDK channel)')}
                </Badge>
              ) : null}
            </div>

            <BodyBlock
              label={t('Downstream request body')}
              content={record.downstream_request}
              empty={t('Not captured')}
            />
            <BodyBlock
              label={t('Upstream response body')}
              content={record.upstream_response}
              empty={t('Not captured')}
            />
          </CollapsibleContent>
        </Collapsible>
      ))}
    </div>
  )
}
