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
import { RefreshCw } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import { CheckCard } from './components/check-card'
import { CheckCardSkeleton } from './components/check-card-skeleton'
import {
  useIqCheckStatus,
  useRefreshCountdown,
} from './hooks/use-iq-check-status'
import { useResultDetails } from './hooks/use-result-details'

const SKELETON_KEYS = ['a', 'b', 'c', 'd']

/**
 * Ticks every second on its own so the cards do not re-render with it; their
 * relative times only need refreshing when new data arrives.
 */
function RefreshCountdown(props: { lastFetchedAt: number }) {
  const { t } = useTranslation()
  const seconds = useRefreshCountdown(props.lastFetchedAt)

  return (
    <p className='text-muted-foreground text-xs tabular-nums'>
      {t('Refreshing in {{seconds}}s', { seconds })}
    </p>
  )
}

function PageShell(props: { children: ReactNode; lastFetchedAt?: number }) {
  const { t } = useTranslation()

  return (
    <div className='pt-16 pb-20'>
      <div className='mb-8 flex flex-wrap items-end justify-between gap-x-6 gap-y-2'>
        <div className='space-y-1.5'>
          <h1 className='text-2xl font-semibold tracking-tight'>
            {t('IQ Check')}
          </h1>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Models answer the same fixed prompts on a schedule, so you can see whether their answers get worse over time.'
            )}
          </p>
        </div>
        {props.lastFetchedAt !== undefined && (
          <RefreshCountdown lastFetchedAt={props.lastFetchedAt} />
        )}
      </div>
      {props.children}
    </div>
  )
}

export function IqCheck() {
  const { t } = useTranslation()
  const { data, isLoading, isError, refetch, lastFetchedAt, backoffSeconds } =
    useIqCheckStatus()
  // Run picked per check; checks without an entry follow their newest run.
  const [selectedRuns, setSelectedRuns] = useState<Record<string, number>>({})

  const checks = data?.enabled ? data.checks : []
  // A picked run that has since been pruned falls back to the newest one.
  const displayedRuns = checks.map(
    (check) =>
      check.history.find((item) => item.id === selectedRuns[check.id]) ??
      check.history.at(0)
  )
  const resultDetails = useResultDetails(
    displayedRuns.flatMap((run) => (run ? [run.id] : []))
  )

  if (isLoading) {
    return (
      <PageShell>
        <div className='grid gap-4 lg:grid-cols-2'>
          {SKELETON_KEYS.map((key) => (
            <CheckCardSkeleton key={key} />
          ))}
        </div>
      </PageShell>
    )
  }

  if (isError && !data) {
    const retrying = backoffSeconds > 0
    return (
      <PageShell>
        <div className='mx-auto max-w-md space-y-4 text-center'>
          <p className='text-muted-foreground text-sm'>
            {t('Failed to load IQ check data.')}
          </p>
          <Button
            variant='outline'
            disabled={retrying}
            onClick={() => void refetch()}
          >
            <RefreshCw className='size-4' />
            {retrying
              ? t('Retry in {{seconds}}s', { seconds: backoffSeconds })
              : t('Retry')}
          </Button>
        </div>
      </PageShell>
    )
  }

  if (!data?.enabled) {
    return (
      <PageShell>
        <p className='text-muted-foreground text-center text-sm'>
          {t('The IQ check page is not enabled.')}
        </p>
      </PageShell>
    )
  }

  if (checks.length === 0) {
    return (
      <PageShell lastFetchedAt={lastFetchedAt}>
        <p className='text-muted-foreground text-center text-sm'>
          {t('No IQ checks have been configured yet.')}
        </p>
      </PageShell>
    )
  }

  return (
    <PageShell lastFetchedAt={lastFetchedAt}>
      <div className='grid gap-4 lg:grid-cols-2'>
        {checks.map((check, index) => {
          const run = displayedRuns[index]
          return (
            <CheckCard
              key={check.id}
              check={check}
              run={run}
              result={run ? resultDetails.results.get(run.id) : undefined}
              resultState={run ? resultDetails.getState(run.id) : 'loading'}
              onSelectRun={(runId) =>
                setSelectedRuns((current) => ({
                  ...current,
                  [check.id]: runId,
                }))
              }
              onRetryResult={resultDetails.retry}
            />
          )
        })}
      </div>
    </PageShell>
  )
}
