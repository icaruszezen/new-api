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
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import { MonitorCard } from './components/monitor-card'
import { MonitorCardSkeleton } from './components/monitor-card-skeleton'
import {
  useMonitoringStatus,
  useRefreshCountdown,
} from './hooks/use-monitoring-status'

const SKELETON_KEYS = ['a', 'b', 'c', 'd']
const DEFAULT_BEAT_SLOTS = 60

function PageShell(props: { children: ReactNode; countdown?: number }) {
  const { t } = useTranslation()

  return (
    <div className='pt-16 pb-20'>
      <h1 className='sr-only'>{t('Channel Monitoring')}</h1>
      {props.countdown !== undefined && (
        <div className='mb-6 flex justify-end'>
          <p className='text-muted-foreground text-xs tabular-nums'>
            {t('Refreshing in {{seconds}}s', { seconds: props.countdown })}
          </p>
        </div>
      )}
      {props.children}
    </div>
  )
}

export function ChannelMonitoring() {
  const { t } = useTranslation()
  const { data, isLoading, isError, refetch, dataUpdatedAt } =
    useMonitoringStatus()
  const countdown = useRefreshCountdown(dataUpdatedAt)

  if (isLoading) {
    return (
      <PageShell>
        <div className='grid gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4'>
          {SKELETON_KEYS.map((key) => (
            <MonitorCardSkeleton key={key} />
          ))}
        </div>
      </PageShell>
    )
  }

  if (isError) {
    return (
      <PageShell>
        <div className='mx-auto max-w-md space-y-4 text-center'>
          <p className='text-muted-foreground text-sm'>
            {t('Failed to load channel monitoring data.')}
          </p>
          <Button variant='outline' onClick={() => void refetch()}>
            <RefreshCw className='size-4' />
            {t('Retry')}
          </Button>
        </div>
      </PageShell>
    )
  }

  if (!data?.enabled) {
    return (
      <PageShell>
        <p className='text-muted-foreground text-center text-sm'>
          {t('Channel monitoring is not enabled.')}
        </p>
      </PageShell>
    )
  }

  if (data.monitors.length === 0) {
    return (
      <PageShell countdown={countdown}>
        <p className='text-muted-foreground text-center text-sm'>
          {t('No monitors have been configured yet.')}
        </p>
      </PageShell>
    )
  }

  return (
    <PageShell>
      <div className='grid gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4'>
        {data.monitors.map((monitor) => (
          <MonitorCard
            key={monitor.id}
            monitor={monitor}
            slots={data.beat_limit || DEFAULT_BEAT_SLOTS}
            countdown={countdown}
          />
        ))}
      </div>
    </PageShell>
  )
}
