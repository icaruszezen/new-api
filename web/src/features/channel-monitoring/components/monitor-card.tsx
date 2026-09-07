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
import { Gauge, Timer } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { TooltipProvider } from '@/components/ui/tooltip'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import type { MonitorStatus, MonitorView } from '../types'
import { BeatBar } from './beat-bar'

type MonitorCardProps = {
  monitor: MonitorView
  /** Sample count behind both the status bar and the availability percentage. */
  slots: number
  /** Seconds until the status page refetches; omitted when the parent has no clock. */
  countdown?: number
}

type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'warning'

function statusBadgeVariant(status: MonitorStatus): BadgeVariant {
  switch (status) {
    case 'up':
      return 'default'
    case 'degraded':
      return 'warning'
    case 'down':
      return 'destructive'
    default:
      return 'secondary'
  }
}

/** Uptime colouring mirrors the status bar so a red bar never sits under a green number. */
function uptimeToneClass(uptime: number): string {
  if (uptime >= 95) return 'text-emerald-600 dark:text-emerald-400'
  if (uptime >= 90) return 'text-amber-600 dark:text-amber-400'
  return 'text-rose-600 dark:text-rose-400'
}

function MetricTile(props: {
  icon: ReactNode
  label: string
  value: number
  unit: string
}) {
  return (
    <div className='border-border/60 bg-muted/30 rounded-xl border px-3 py-2.5'>
      <div className='text-muted-foreground flex items-center gap-1.5 text-xs'>
        <span aria-hidden='true' className='flex items-center'>
          {props.icon}
        </span>
        <span className='truncate'>{props.label}</span>
      </div>
      <div className='mt-1 flex items-baseline gap-1'>
        <span className='text-xl leading-none font-semibold tabular-nums'>
          {props.value > 0 ? props.value : '--'}
        </span>
        {props.value > 0 && (
          <span className='text-muted-foreground text-xs'>{props.unit}</span>
        )}
      </div>
    </div>
  )
}

export function MonitorCard(props: MonitorCardProps) {
  const { t } = useTranslation()
  const vendorName = props.monitor.icon.split('.')[0]

  const statusLabel: Record<MonitorStatus, string> = {
    up: t('Normal'),
    degraded: t('Degraded'),
    down: t('Abnormal'),
    unknown: t('No data'),
  }

  return (
    <Card className='rounded-2xl py-5'>
      <CardHeader className='px-5'>
        <div className='flex items-center gap-3'>
          <span
            aria-hidden='true'
            className='bg-muted flex size-10 shrink-0 items-center justify-center rounded-full'
          >
            {getLobeIcon(props.monitor.icon, 22)}
          </span>
          <div className='min-w-0 flex-1'>
            <div className='flex items-start justify-between gap-2'>
              <span
                className='truncate text-sm font-semibold'
                title={props.monitor.name}
              >
                {props.monitor.name}
              </span>
              <Badge variant={statusBadgeVariant(props.monitor.status)}>
                {statusLabel[props.monitor.status]}
              </Badge>
            </div>
            <div className='text-muted-foreground mt-0.5 flex min-w-0 items-center gap-1.5 text-xs'>
              {vendorName ? (
                <>
                  <span
                    aria-hidden='true'
                    className='flex shrink-0 items-center'
                  >
                    {getLobeIcon(props.monitor.icon, 12)}
                  </span>
                  <span className='shrink-0'>{vendorName}</span>
                </>
              ) : null}
              <span
                className='min-w-0 truncate font-mono'
                title={props.monitor.model}
              >
                {props.monitor.model}
              </span>
            </div>
          </div>
        </div>
      </CardHeader>

      <CardContent className='space-y-4 px-5'>
        <div className='grid grid-cols-2 gap-2'>
          <MetricTile
            icon={<Timer className='size-3.5' />}
            label={t('Chat latency')}
            value={props.monitor.avg_ttft_ms}
            unit='ms'
          />
          <MetricTile
            icon={<Gauge className='size-3.5' />}
            label={t('Endpoint ping')}
            value={props.monitor.ping_ms}
            unit='ms'
          />
        </div>

        <div className='flex items-baseline justify-between gap-2'>
          <span className='text-muted-foreground text-xs'>
            {t('Availability')}
          </span>
          <span
            className={cn(
              'text-2xl font-semibold tabular-nums',
              uptimeToneClass(props.monitor.uptime)
            )}
          >
            {props.monitor.uptime > 0
              ? `${props.monitor.uptime.toFixed(2)}%`
              : '--'}
          </span>
        </div>

        <div className='space-y-1.5'>
          <div className='text-muted-foreground flex items-center justify-between gap-2 text-xs'>
            <span>{t('Recent {{count}} records', { count: props.slots })}</span>
            {props.countdown !== undefined && (
              <span className='tabular-nums'>
                {t('Refreshing in {{seconds}}s', { seconds: props.countdown })}
              </span>
            )}
          </div>
          <TooltipProvider delay={80}>
            <BeatBar beats={props.monitor.beats} slots={props.slots} />
          </TooltipProvider>
        </div>
      </CardContent>
    </Card>
  )
}
