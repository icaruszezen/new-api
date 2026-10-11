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
  Brain,
  CalendarClock,
  Clock,
  Hash,
  Loader2,
  Maximize2,
  Repeat,
  Timer,
  TriangleAlert,
} from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toIntlLocale } from '@/i18n/languages'
import {
  formatNumber,
  formatTimestampRelative,
  formatTimestampToDate,
  formatUseTime,
} from '@/lib/format'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import { FINISH_REASON_LENGTH } from '../constants'
import type { ResultDetailState } from '../hooks/use-result-details'
import { getReasoningEffortLabels } from '../lib/request-options'
import type {
  IqCheckHistoryItem,
  IqCheckResult,
  IqCheckRunStatus,
  IqCheckView,
} from '../types'
import { PromptBlock } from './prompt-block'
import { ResultBody } from './result-body'

const RUN_STATUS_BADGE_CLASS: Record<IqCheckRunStatus, string> = {
  success:
    'border-transparent bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300',
  failed:
    'border-transparent bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300',
}

const RUN_STATUS_DOT_CLASS: Record<IqCheckRunStatus, string> = {
  success: 'bg-emerald-500',
  failed: 'bg-red-500',
}

function RunStatusDot(props: { status: IqCheckRunStatus }) {
  return (
    <span
      aria-hidden='true'
      className={cn(
        'size-2 shrink-0 rounded-full',
        RUN_STATUS_DOT_CLASS[props.status]
      )}
    />
  )
}

function MetaItem(props: {
  icon: ReactNode
  children: ReactNode
  className?: string
}) {
  return (
    <span className={cn('flex items-center gap-1', props.className)}>
      <span aria-hidden='true' className='flex items-center'>
        {props.icon}
      </span>
      {props.children}
    </span>
  )
}

type CheckCardProps = {
  check: IqCheckView
  /** Run shown on the card; undefined while the check has never finished one. */
  run?: IqCheckHistoryItem
  result?: IqCheckResult
  resultState: ResultDetailState
  onSelectRun: (runId: number) => void
  onRetryResult: () => void
}

export function CheckCard(props: CheckCardProps) {
  const { t, i18n } = useTranslation()
  const [dialogOpen, setDialogOpen] = useState(false)
  const locale = toIntlLocale(i18n.language)
  const check = props.check
  const run = props.run
  const result = props.result
  const nowSeconds = Math.floor(Date.now() / 1000)
  // A run shows only the depth it recorded, never the check's current setting;
  // that setting is shown only before the check has run.
  const reasoningEffort = run
    ? (result?.reasoning_effort ?? '')
    : check.reasoning_effort

  const runStatusLabel: Record<IqCheckRunStatus, string> = {
    success: t('Completed'),
    failed: t('Failed'),
  }
  const runLabel = (item: IqCheckHistoryItem) =>
    formatTimestampToDate(item.checked_at)
  const runItems = check.history.map((item) => ({
    value: String(item.id),
    label: runLabel(item),
  }))

  const resultBody = run ? (
    <ResultBody
      result={result}
      state={props.resultState}
      onRetry={props.onRetryResult}
    />
  ) : (
    <p className='text-muted-foreground text-sm'>
      {check.running || check.queued
        ? t('The first check is in progress.')
        : t('This check has not run yet.')}
    </p>
  )

  return (
    <Card className='rounded-2xl py-5'>
      <CardHeader className='px-5'>
        <div className='flex items-start gap-3'>
          <span
            aria-hidden='true'
            className='bg-muted flex size-10 shrink-0 items-center justify-center rounded-full'
          >
            {getLobeIcon(check.icon, 20)}
          </span>
          <div className='min-w-0 flex-1'>
            <div className='flex items-start justify-between gap-2'>
              <h2 className='truncate text-sm font-semibold' title={check.name}>
                {check.name}
              </h2>
              <div className='flex shrink-0 flex-wrap items-center justify-end gap-1.5'>
                {check.running && (
                  <Badge variant='secondary'>
                    <Loader2 className='animate-spin' aria-hidden='true' />
                    {t('Running')}
                  </Badge>
                )}
                {check.queued && !check.running && (
                  <Badge variant='outline'>{t('Queued')}</Badge>
                )}
                {run && (
                  <Badge
                    variant='outline'
                    className={RUN_STATUS_BADGE_CLASS[run.status]}
                  >
                    {runStatusLabel[run.status]}
                  </Badge>
                )}
                {!run && !check.running && !check.queued && (
                  <Badge variant='secondary'>{t('Not checked yet')}</Badge>
                )}
              </div>
            </div>
            <div
              className='text-muted-foreground mt-0.5 truncate font-mono text-xs'
              title={check.model}
            >
              {check.model}
            </div>
          </div>
        </div>
      </CardHeader>

      <CardContent className='space-y-4 px-5'>
        <div className='text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs'>
          {run && (
            <MetaItem icon={<Clock className='size-3.5' />}>
              <time
                dateTime={new Date(run.checked_at * 1000).toISOString()}
                title={formatTimestampToDate(run.checked_at)}
              >
                {t('Checked {{time}}', {
                  time: formatTimestampRelative(
                    run.checked_at,
                    'seconds',
                    locale
                  ),
                })}
              </time>
            </MetaItem>
          )}
          <MetaItem icon={<Repeat className='size-3.5' />}>
            {t('Every {{minutes}} min', { minutes: check.interval_minutes })}
          </MetaItem>
          <MetaItem icon={<CalendarClock className='size-3.5' />}>
            {check.next_check_at > nowSeconds ? (
              <span title={formatTimestampToDate(check.next_check_at)}>
                {t('Next check {{time}}', {
                  time: formatTimestampRelative(
                    check.next_check_at,
                    'seconds',
                    locale
                  ),
                })}
              </span>
            ) : (
              t('Next check soon')
            )}
          </MetaItem>
          {run && run.duration_ms > 0 && (
            <MetaItem icon={<Timer className='size-3.5' />}>
              {formatUseTime(run.duration_ms / 1000)}
            </MetaItem>
          )}
          {run && run.completion_tokens > 0 && (
            <MetaItem icon={<Hash className='size-3.5' />}>
              {t('{{tokens}} output tokens', {
                tokens: formatNumber(run.completion_tokens, locale),
              })}
              {result && result.reasoning_tokens > 0 && (
                <span>
                  {' · '}
                  {t('{{tokens}} reasoning', {
                    tokens: formatNumber(result.reasoning_tokens, locale),
                  })}
                </span>
              )}
            </MetaItem>
          )}
          {reasoningEffort && (
            <MetaItem icon={<Brain className='size-3.5' />}>
              {t('Thinking depth: {{level}}', {
                level: getReasoningEffortLabels(t)[reasoningEffort],
              })}
            </MetaItem>
          )}
          {result?.finish_reason === FINISH_REASON_LENGTH && (
            <MetaItem
              icon={<TriangleAlert className='size-3.5' />}
              className='text-amber-700 dark:text-amber-300'
            >
              {t('Truncated by max tokens')}
            </MetaItem>
          )}
        </div>

        <PromptBlock
          prompt={result?.prompt || check.prompt}
          outdated={Boolean(result && result.prompt !== check.prompt)}
        />

        <section className='space-y-1.5'>
          <div className='flex flex-wrap items-center justify-between gap-2'>
            <h3 className='text-muted-foreground text-xs font-medium'>
              {t('Output')}
            </h3>
            {run && (
              <div className='flex items-center gap-1'>
                <Select
                  items={runItems}
                  value={String(run.id)}
                  onValueChange={(value) => {
                    if (value) props.onSelectRun(Number(value))
                  }}
                >
                  <SelectTrigger size='sm' aria-label={t('Run')}>
                    <SelectValue>
                      <RunStatusDot status={run.status} />
                      <span className='tabular-nums'>{runLabel(run)}</span>
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent alignItemWithTrigger={false}>
                    {check.history.map((item) => (
                      <SelectItem key={item.id} value={String(item.id)}>
                        <RunStatusDot status={item.status} />
                        <span className='tabular-nums'>{runLabel(item)}</span>
                        <span className='sr-only'>
                          {runStatusLabel[item.status]}
                        </span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Button
                  type='button'
                  variant='ghost'
                  size='icon-sm'
                  aria-label={t('Expand')}
                  disabled={props.resultState !== 'ready'}
                  onClick={() => setDialogOpen(true)}
                >
                  <Maximize2 className='size-4' />
                </Button>
              </div>
            )}
          </div>
          <div className='border-border/60 max-h-96 overflow-y-auto rounded-lg border p-3'>
            {resultBody}
          </div>
        </section>
      </CardContent>

      {run && (
        <Dialog
          open={dialogOpen}
          onOpenChange={setDialogOpen}
          title={check.name}
          description={`${check.model} · ${runLabel(run)}`}
          contentClassName='sm:max-w-5xl'
        >
          <ResultBody
            result={result}
            state={props.resultState}
            onRetry={props.onRetryResult}
            expanded
          />
        </Dialog>
      )}
    </Card>
  )
}
