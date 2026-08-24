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
import type { Row, Table as TanstackTable } from '@tanstack/react-table'
import { Database } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { DISABLED_ROW_MOBILE } from '@/components/data-table'
import { GroupRatioPill } from '@/components/group-badge'
import { StatusBadge } from '@/components/status-badge'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { useMediaQuery } from '@/hooks'
import { formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

import { API_KEY_STATUSES, isDisabledApiKeyRow } from '../constants'
import type { ApiKey } from '../types'
import { ApiKeyCell, UnlimitedQuotaBadge } from './api-keys-cells'
import { useGroupRatios } from './api-keys-columns'
import { GroupRatioBadge } from './auto-group-visuals'
import { DataTableRowActions } from './data-table-row-actions'

const API_KEYS_MOBILE_SKELETON_IDS = Array.from(
  { length: 5 },
  (_, index) => `api-key-mobile-skeleton-${index + 1}`
)

function ApiKeyMobileGroupRatio(props: {
  group: string
  ratio?: number | string
  shouldReduceMotion: boolean
  renderRatio?: (ratio: number) => ReactNode
}) {
  const { t } = useTranslation()
  let ratioValue = (
    <span
      data-slot='api-key-mobile-group-ratio'
      className='text-muted-foreground'
    >
      --
    </span>
  )
  if (typeof props.ratio === 'number') {
    ratioValue = (
      <span data-slot='api-key-mobile-group-ratio'>
        {props.renderRatio ? (
          props.renderRatio(props.ratio)
        ) : (
          <GroupRatioPill ratio={props.ratio} />
        )}
      </span>
    )
  } else if (
    props.group === 'auto' &&
    props.ratio !== undefined &&
    props.ratio !== null &&
    props.ratio !== ''
  ) {
    ratioValue = (
      <span data-slot='api-key-mobile-group-ratio'>
        <GroupRatioBadge
          ratio={props.ratio}
          isAuto
          shouldReduceMotion={props.shouldReduceMotion}
        />
      </span>
    )
  }

  return (
    <div className='space-y-1 text-xs'>
      <div className='flex items-center justify-between gap-2'>
        <span className='text-muted-foreground shrink-0'>{t('Group')}</span>
        {ratioValue}
      </div>
      {props.group ? (
        <div
          data-slot='api-key-mobile-group-name'
          className='min-w-0 text-end font-medium break-all'
        >
          {props.group === 'auto' ? t('Cross-group') : props.group}
        </div>
      ) : null}
    </div>
  )
}

function ApiKeysMobileSkeleton(props: { className?: string }) {
  return (
    <div
      className={cn(
        'divide-border overflow-hidden rounded-lg border',
        props.className
      )}
    >
      {API_KEYS_MOBILE_SKELETON_IDS.map((id) => (
        <div
          key={id}
          className='space-y-2 border-b px-3 py-2.5 last:border-b-0'
        >
          <div className='flex items-center justify-between'>
            <Skeleton className='h-4 w-32' />
            <Skeleton className='h-5 w-16 rounded-md' />
          </div>
          <div className='flex items-center justify-between gap-3'>
            <Skeleton className='h-7 w-44' />
            <Skeleton className='h-8 w-16' />
          </div>
          <Skeleton className='h-3 w-28' />
        </div>
      ))}
    </div>
  )
}

type ApiKeysMobileListProps = {
  table: TanstackTable<ApiKey>
  isLoading: boolean
  className?: string
  emptyTitle?: string
  emptyDescription?: string
  showGroupRatio?: boolean
  renderRowActions?: (row: Row<ApiKey>) => ReactNode
  renderRatio?: (ratio: number) => ReactNode
}

export function ApiKeysMobileList(props: ApiKeysMobileListProps) {
  const { t } = useTranslation()
  const rows = props.table.getRowModel().rows
  const groupRatios = useGroupRatios()
  const shouldReduceMotion = useMediaQuery('(prefers-reduced-motion: reduce)')
  const emptyTitle = props.emptyTitle ?? t('No API Keys Found')
  const emptyDescription =
    props.emptyDescription ??
    t('No API keys available. Create your first API key to get started.')

  if (props.isLoading) {
    return <ApiKeysMobileSkeleton className={props.className} />
  }

  if (!rows.length) {
    return (
      <div className={cn('rounded-lg border p-8', props.className)}>
        <Empty className='border-none p-0'>
          <EmptyHeader>
            <EmptyMedia variant='icon'>
              <Database className='size-6' />
            </EmptyMedia>
            <EmptyTitle>{emptyTitle}</EmptyTitle>
            {emptyDescription ? (
              <EmptyDescription>{emptyDescription}</EmptyDescription>
            ) : null}
          </EmptyHeader>
        </Empty>
      </div>
    )
  }

  return (
    <div
      data-slot='api-key-mobile-list'
      className={cn(
        'divide-border overflow-hidden rounded-lg border',
        props.className
      )}
    >
      {rows.map((row) => {
        const apiKey = row.original
        const statusConfig = API_KEY_STATUSES[apiKey.status]
        const total = apiKey.used_quota + apiKey.remain_quota
        const group = apiKey.group ?? ''

        return (
          <div
            key={row.id}
            className={cn(
              'bg-card space-y-2.5 border-b px-3 py-2.5 last:border-b-0',
              isDisabledApiKeyRow(apiKey) && DISABLED_ROW_MOBILE
            )}
          >
            <div className='flex items-start justify-between gap-3'>
              <div className='min-w-0'>
                <div className='truncate text-sm font-semibold'>
                  {apiKey.name}
                </div>
                <div className='text-muted-foreground text-[11px]'>
                  {t('API Key')}
                </div>
              </div>
              {statusConfig && (
                <StatusBadge
                  label={t(statusConfig.label)}
                  variant={statusConfig.variant}
                  copyable={false}
                />
              )}
            </div>

            <div className='flex min-w-0 items-center justify-between gap-2'>
              <div className='min-w-0 flex-1 [&_button:first-child]:max-w-full [&_button:first-child]:truncate [&_button:first-child]:px-0'>
                <ApiKeyCell apiKey={apiKey} />
              </div>
              {props.renderRowActions ? (
                props.renderRowActions(row)
              ) : (
                <DataTableRowActions row={row} />
              )}
            </div>

            <div className='flex items-center justify-between gap-2 text-xs'>
              <span className='text-muted-foreground'>{t('Quota')}</span>
              {apiKey.unlimited_quota ? (
                <UnlimitedQuotaBadge used={apiKey.used_quota} />
              ) : (
                <span className='font-medium tabular-nums'>
                  {formatQuota(apiKey.remain_quota)}
                  <span className='text-muted-foreground font-normal'>
                    {' / '}
                    {formatQuota(total)}
                  </span>
                </span>
              )}
            </div>

            {props.showGroupRatio ? (
              <ApiKeyMobileGroupRatio
                group={group}
                ratio={groupRatios[group]}
                shouldReduceMotion={shouldReduceMotion}
                renderRatio={props.renderRatio}
              />
            ) : null}
          </div>
        )
      })}
    </div>
  )
}
