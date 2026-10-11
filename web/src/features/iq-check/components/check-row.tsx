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
import { ArrowDown, ArrowUp, Pencil, Play, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { toIntlLocale } from '@/i18n/languages'
import { formatTimestampRelative, formatTimestampToDate } from '@/lib/format'
import { getLobeIcon } from '@/lib/lobe-icon'

import { RUNNING_GRACE_SECONDS } from '../constants'
import type { AdminIqCheck } from '../types'

type CheckRowProps = {
  check: AdminIqCheck
  isFirst: boolean
  isLast: boolean
  /** Whether scheduled checks run at all; manual runs need it too. */
  featureEnabled: boolean
  requestTimeoutSeconds: number
  /** A manual run request for this row is in flight. */
  runPending: boolean
  onRun: () => void
  onEdit: () => void
  onRemove: () => void
  onMove: (direction: -1 | 1) => void
  disabled?: boolean
}

function CheckRunState(props: {
  check: AdminIqCheck
  requestTimeoutSeconds: number
}) {
  const { t, i18n } = useTranslation()
  const check = props.check
  const nowSeconds = Math.floor(Date.now() / 1000)
  // A start without a recorded finish is abandoned once the run could no
  // longer be alive, e.g. after the node running it restarted.
  const running =
    check.last_started_at > check.last_finished_at &&
    nowSeconds - check.last_started_at <
      props.requestTimeoutSeconds + RUNNING_GRACE_SECONDS
  const queued = check.run_requested_at > check.last_started_at

  if (running) {
    return <span>{t('Running')}</span>
  }
  if (queued) {
    return <span>{t('Queued')}</span>
  }
  if (check.last_checked_at <= 0) {
    return <span>{t('Not checked yet')}</span>
  }

  const failed = check.last_status === 'failed'
  return (
    <span title={formatTimestampToDate(check.last_checked_at)}>
      {t('Last checked {{time}}', {
        time: formatTimestampRelative(
          check.last_checked_at,
          'seconds',
          toIntlLocale(i18n.language)
        ),
      })}
      {' · '}
      <span
        className={
          failed ? 'text-destructive' : 'text-emerald-700 dark:text-emerald-300'
        }
      >
        {failed ? t('Failed') : t('Completed')}
      </span>
    </span>
  )
}

export function CheckRow(props: CheckRowProps) {
  const { t } = useTranslation()
  const check = props.check

  let runBlockedReason: string | undefined
  if (!props.featureEnabled) {
    runBlockedReason = t('Enable the IQ check page first.')
  } else if (!check.enabled) {
    runBlockedReason = t('Enable this check first.')
  }

  return (
    <li className='border-border/60 bg-card flex flex-wrap items-center gap-3 rounded-lg border px-3 py-2.5'>
      <span aria-hidden='true' className='flex shrink-0 items-center'>
        {getLobeIcon(check.resolved_icon, 20)}
      </span>

      <div className='min-w-0 flex-1 space-y-0.5'>
        <div className='flex items-center gap-2'>
          <span className='truncate text-sm font-medium'>{check.name}</span>
          {!check.enabled && <Badge variant='secondary'>{t('Disabled')}</Badge>}
        </div>
        <div className='text-muted-foreground truncate text-xs'>
          {check.group} · {check.model} ·{' '}
          {t('every {{minutes}} min', { minutes: check.interval_minutes })}
        </div>
        <div className='text-muted-foreground text-xs'>
          <CheckRunState
            check={check}
            requestTimeoutSeconds={props.requestTimeoutSeconds}
          />
        </div>
        {check.last_status === 'failed' && check.last_error && (
          <div
            className='text-destructive truncate text-xs'
            title={check.last_error}
          >
            {check.last_error}
          </div>
        )}
      </div>

      <div className='flex shrink-0 items-center gap-1'>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          aria-label={t('Run now')}
          title={runBlockedReason ?? t('Run now')}
          disabled={
            props.disabled ||
            props.runPending ||
            !check.id ||
            runBlockedReason !== undefined
          }
          onClick={props.onRun}
        >
          <Play className='size-4' />
        </Button>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          aria-label={t('Move up')}
          disabled={props.disabled || props.isFirst}
          onClick={() => props.onMove(-1)}
        >
          <ArrowUp className='size-4' />
        </Button>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          aria-label={t('Move down')}
          disabled={props.disabled || props.isLast}
          onClick={() => props.onMove(1)}
        >
          <ArrowDown className='size-4' />
        </Button>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          aria-label={t('Edit')}
          disabled={props.disabled}
          onClick={props.onEdit}
        >
          <Pencil className='size-4' />
        </Button>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          aria-label={t('Delete')}
          disabled={props.disabled}
          onClick={props.onRemove}
        >
          <Trash2 className='text-destructive size-4' />
        </Button>
      </div>
    </li>
  )
}
