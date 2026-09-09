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
import { CheckCircle2, Users, Wallet, type LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { formatQuotaWithCurrency } from '@/lib/currency'

type NextInviteStatsRowProps = {
  inviteeCount: number
  rechargedCount: number
  rebateQuota: number
  loading: boolean
}

type StatCardProps = {
  title: string
  value: string
  description: string
  icon: LucideIcon
  loading: boolean
}

function StatCard(props: StatCardProps) {
  const Icon = props.icon

  return (
    <article
      data-slot='next-invite-stat-card'
      className='bg-card min-w-0 rounded-xl border px-4 py-4 sm:px-5'
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

export function NextInviteStatsRow(props: NextInviteStatsRowProps) {
  const { t } = useTranslation()

  return (
    <section
      aria-label={t('Referral summary')}
      data-slot='next-invite-stats'
      className='grid grid-cols-1 gap-3 sm:grid-cols-3'
    >
      <StatCard
        title={t('Successful referrals')}
        value={String(props.inviteeCount)}
        description={t('Signed up through your link')}
        icon={Users}
        loading={props.loading}
      />
      <StatCard
        title={t('Rebate earned')}
        value={formatQuotaWithCurrency(props.rebateQuota)}
        description={t('Already credited to your balance')}
        icon={Wallet}
        loading={props.loading}
      />
      <StatCard
        title={t('Topped up')}
        value={String(props.rechargedCount)}
        description={t('Friends who reached the top-up threshold')}
        icon={CheckCircle2}
        loading={props.loading}
      />
    </section>
  )
}
