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
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import {
  CodeBlock,
  CodeBlockCopyButton,
} from '@/components/ai-elements/code-block'
import { Dialog } from '@/components/dialog'
import { formatLogQuota, formatTimestampToDate } from '@/lib/format'

import { getFirstTokenErrorBody } from '../api'
import type { FirstTokenErrorLog } from '../types'

type FirstTokenErrorDetailsDialogProps = {
  log: FirstTokenErrorLog | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

function DetailRow(props: { label: string; value?: string | number | boolean }) {
  return (
    <div className='grid grid-cols-[8rem_1fr] gap-2 text-sm'>
      <div className='text-muted-foreground'>{props.label}</div>
      <div className='min-w-0 break-all'>{String(props.value ?? '-')}</div>
    </div>
  )
}

function prettyJson(raw: string): string {
  if (!raw) return ''
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

export function FirstTokenErrorDetailsDialog(
  props: FirstTokenErrorDetailsDialogProps
) {
  const { t } = useTranslation()
  const log = props.log
  const bodyQuery = useQuery({
    queryKey: ['first-token-error-body', log?.id],
    queryFn: () => {
      if (!log) {
        return Promise.reject(new Error('missing log'))
      }
      return getFirstTokenErrorBody(log.id)
    },
    enabled: props.open && Boolean(log?.has_request_body),
  })

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('First-token error details')}
      description={t(
        'These values are the dirty usage before correction. The user was not charged.'
      )}
      contentClassName='sm:max-w-2xl'
      contentHeight='auto'
      bodyClassName='space-y-4'
    >
      {log ? (
        <div className='space-y-4'>
          <p className='text-muted-foreground text-sm'>
            {t(
              'The following is dirty usage from before the billing fix. No quota was deducted.'
            )}
          </p>
          <div className='space-y-2'>
            <DetailRow
              label={t('Time')}
              value={formatTimestampToDate(log.created_at)}
            />
            <DetailRow label={t('User')} value={log.username} />
            <DetailRow label={t('Token')} value={log.token_name} />
            <DetailRow label={t('Model')} value={log.model_name} />
            <DetailRow label={t('Channel')} value={log.channel_name} />
            <DetailRow label={t('Group')} value={log.group} />
            <DetailRow label={t('Input tokens')} value={log.prompt_tokens} />
            <DetailRow
              label={t('Output tokens')}
              value={log.completion_tokens}
            />
            <DetailRow
              label={t('Would-be quota')}
              value={formatLogQuota(log.would_be_quota)}
            />
            <DetailRow label={t('Duration')} value={`${log.use_time}s`} />
            <DetailRow label={t('Request ID')} value={log.request_id} />
            <DetailRow label={t('Error Message')} value={log.error_message} />
          </div>
          {log.dirty_usage_json ? (
            <CodeBlock
              code={prettyJson(log.dirty_usage_json)}
              language='json'
            >
              <CodeBlockCopyButton />
            </CodeBlock>
          ) : null}
          {log.other_json ? (
            <CodeBlock code={prettyJson(log.other_json)} language='json'>
              <CodeBlockCopyButton />
            </CodeBlock>
          ) : null}
          {log.has_request_body ? (
            <div className='space-y-2'>
              <div className='text-sm font-medium'>{t('View request body')}</div>
              {bodyQuery.data?.data?.truncated ? (
                <p className='text-amber-600 text-sm'>
                  {t('This request body was truncated to the 8MB limit.')}
                </p>
              ) : null}
              {bodyQuery.data?.data?.body ? (
                <CodeBlock
                  code={prettyJson(bodyQuery.data.data.body)}
                  language='json'
                >
                  <CodeBlockCopyButton />
                </CodeBlock>
              ) : (
                <p className='text-muted-foreground text-sm'>
                  {bodyQuery.isLoading
                    ? t('Loading...')
                    : t('Failed to load request body')}
                </p>
              )}
            </div>
          ) : (
            <p className='text-muted-foreground text-sm'>
              {t(
                'No request body is stored. Capture may be off, or this row was pushed out of the last 10 bodies.'
              )}
            </p>
          )}
        </div>
      ) : null}
    </Dialog>
  )
}
