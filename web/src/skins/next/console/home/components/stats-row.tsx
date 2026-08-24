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
import { Braces, Database, Send, Wallet, type LucideIcon } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import {
  getUserQuotaDates,
  getUserQuotaSummary,
} from '@/features/dashboard/api'
import { toIntlLocale } from '@/i18n/languages'
import { formatCompactNumber, formatPercent, formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import {
  formatCacheReadRate,
  formatKnownMetric,
  getCalendarDayRange,
  getRunwayEstimate,
  sumCacheSampledCalls,
  sumQuotaField,
} from '../lib/stats'

type StatCardProps = {
  title: string
  value: string
  description: string
  icon: LucideIcon
  loading: boolean
  className?: string
}

function StatCard(props: StatCardProps) {
  const Icon = props.icon

  return (
    <article
      data-slot='console-stat-cell'
      className={cn('min-w-0 px-4 py-4 sm:px-5', props.className)}
    >
      <div className='flex items-start justify-between gap-3'>
        <h2 className='text-muted-foreground text-sm font-medium'>
          {props.title}
        </h2>
        <span className='bg-muted text-muted-foreground flex size-8 shrink-0 items-center justify-center rounded-lg'>
          <Icon className='size-4' aria-hidden='true' />
        </span>
      </div>
      {props.loading ? (
        <Skeleton className='mt-3 h-8 w-24' />
      ) : (
        <p className='mt-3 text-2xl font-medium tracking-tight tabular-nums'>
          {props.value}
        </p>
      )}
      <p className='text-muted-foreground mt-1.5 truncate text-xs'>
        {props.description}
      </p>
    </article>
  )
}

export function ConsoleStatsRow() {
  const { t, i18n } = useTranslation()
  const requestCount = useAuthStore((state) =>
    Number(state.auth.user?.request_count ?? 0)
  )
  const remainQuota = useAuthStore((state) =>
    Number(state.auth.user?.quota ?? 0)
  )
  const canLoadStats = useAuthStore((state) => Boolean(state.auth.accessToken))
  const todayRange = useMemo(() => getCalendarDayRange(), [])
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)

  const todayQuery = useQuery({
    queryKey: [
      'console',
      'home',
      'today-stats',
      todayRange.start_timestamp,
      todayRange.end_timestamp,
    ],
    queryFn: () =>
      getUserQuotaDates({
        start_timestamp: todayRange.start_timestamp,
        end_timestamp: todayRange.end_timestamp,
        default_time: 'hour',
      }),
    enabled: canLoadStats,
    staleTime: 60 * 1000,
  })

  const lifetimeQuery = useQuery({
    queryKey: ['console', 'home', 'lifetime-stats'],
    queryFn: getUserQuotaSummary,
    enabled: canLoadStats,
    staleTime: 60 * 1000,
  })

  const todayItems = todayQuery.data?.data ?? []
  const todayCalls = sumQuotaField(todayItems, 'count')
  const todayTokens = sumQuotaField(todayItems, 'token_used')
  const todayQuota = sumQuotaField(todayItems, 'quota')
  const todayCacheRate = formatCacheReadRate(
    sumQuotaField(todayItems, 'cache_tokens'),
    sumQuotaField(todayItems, 'prompt_tokens'),
    sumCacheSampledCalls(todayItems),
    formatPercent
  )
  const lifetime = lifetimeQuery.data?.data
  const lifetimeTokens = Number(lifetime?.token_used ?? 0)
  const lifetimeCacheRate = formatCacheReadRate(
    Number(lifetime?.cache_tokens ?? 0),
    Number(lifetime?.prompt_tokens ?? 0),
    Number(lifetime?.cache_sampled_count ?? 0),
    formatPercent
  )
  let todayCacheDescription = t('Cache read rate pending')
  if (todayCacheRate !== null) {
    todayCacheDescription = t('Cache read rate {{rate}}', {
      rate: todayCacheRate,
    })
  }
  let lifetimeCacheDescription = t('Cache read rate pending')
  if (lifetimeCacheRate !== null) {
    lifetimeCacheDescription = t('Cache read rate {{rate}}', {
      rate: lifetimeCacheRate,
    })
  }
  const formatNumber = (value: number) => formatCompactNumber(value, locale)
  const runway = getRunwayEstimate(remainQuota, todayQuota)
  let runwayDescription = t('No recent usage')
  if (runway.kind === 'days') {
    runwayDescription = t('About {{count}} days left', { count: runway.days })
  } else if (runway.kind === 'less-than-one-day') {
    runwayDescription = t('Less than 1 day left')
  } else if (runway.kind === 'depleted') {
    runwayDescription = t('Balance depleted')
  }

  return (
    <section
      aria-label={t("Today's usage")}
      data-slot='console-stat-panel'
      className='bg-card grid h-full grid-cols-2 overflow-hidden rounded-xl border'
    >
      <StatCard
        title={t("Today's calls")}
        value={formatKnownMetric(todayCalls, formatNumber)}
        description={t('Historical requests {{count}}', {
          count: formatNumber(requestCount),
        })}
        icon={Send}
        loading={todayQuery.isLoading}
        className='border-border border-r border-b'
      />
      <StatCard
        title={t("Today's tokens")}
        value={formatKnownMetric(todayTokens, formatNumber)}
        description={todayCacheDescription}
        icon={Braces}
        loading={todayQuery.isLoading}
        className='border-border border-b'
      />
      <StatCard
        title={t('Lifetime tokens')}
        value={formatKnownMetric(lifetimeTokens, formatNumber)}
        description={lifetimeCacheDescription}
        icon={Database}
        loading={lifetimeQuery.isLoading}
        className='border-border border-r'
      />
      <StatCard
        title={t('Wallet balance')}
        value={formatQuota(remainQuota)}
        description={runwayDescription}
        icon={Wallet}
        loading={todayQuery.isLoading}
      />
    </section>
  )
}
