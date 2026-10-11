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

import { formatLogQuota } from '@/lib/format'

import type { BillingCorrectionRecordStats } from '../types'

type RecordsStatsProps = {
  stats: BillingCorrectionRecordStats
}

type StatTileProps = {
  label: string
  amount: string
  count: number
}

function StatTile(props: StatTileProps) {
  const { t } = useTranslation()
  return (
    <div
      className='rounded-lg border p-3'
      role='group'
      aria-label={props.label}
    >
      <div className='text-muted-foreground text-xs'>{props.label}</div>
      <div className='mt-1 text-lg font-semibold tabular-nums'>
        {props.amount}
      </div>
      <div className='text-muted-foreground text-xs'>
        {t('{{count}} record(s)', { count: props.count })}
      </div>
    </div>
  )
}

/**
 * Charged = pending + refunded. Corrected records were waived at settlement,
 * so they are only counted, never added to the charged amount.
 */
export function RecordsStats(props: RecordsStatsProps) {
  const { t } = useTranslation()
  return (
    <div className='grid grid-cols-2 gap-2 lg:grid-cols-4'>
      <StatTile
        label={t('Wrongly charged')}
        amount={formatLogQuota(props.stats.charged_quota)}
        count={props.stats.charged_count}
      />
      <StatTile
        label={t('Pending refund')}
        amount={formatLogQuota(props.stats.pending_quota)}
        count={props.stats.pending_count}
      />
      <StatTile
        label={t('Refunded')}
        amount={formatLogQuota(props.stats.refunded_quota)}
        count={props.stats.refunded_count}
      />
      <StatTile
        label={t('Corrected')}
        amount={t('Not charged')}
        count={props.stats.corrected_count}
      />
    </div>
  )
}
