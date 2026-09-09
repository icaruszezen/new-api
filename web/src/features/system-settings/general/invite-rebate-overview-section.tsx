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
  CheckCircle2,
  UserPlus,
  Users,
  Wallet,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import { useInviteAdminStats } from '@/features/invite/hooks/use-invite-admin-stats'
import type {
  InviteAdminInvitee,
  InviteAdminSummary,
  InviteCountRank,
  InviteRebateRank,
} from '@/features/invite/types'
import { formatQuotaWithCurrency } from '@/lib/currency'
import dayjs from '@/lib/dayjs'

import { SettingsSection } from '../components/settings-section'

export function InviteRebateOverviewSection() {
  const { t } = useTranslation()
  const query = useInviteAdminStats()

  return (
    <SettingsSection title={t('Referral overview')}>
      <p className='text-muted-foreground text-sm'>
        {t('Site-wide referral activity and rebate payouts.')}
      </p>
      {query.error ? (
        <Alert variant='destructive'>
          <AlertDescription>
            {t('Failed to load referral overview. Please try again later.')}
          </AlertDescription>
        </Alert>
      ) : (
        <div
          className='flex flex-col gap-4'
          data-slot='invite-admin-overview'
        >
          <OverviewStatsRow
            summary={query.stats?.summary ?? null}
            loading={query.loading}
          />
          <div className='grid grid-cols-1 gap-4 lg:grid-cols-2'>
            <InviteCountRankingTable
              rows={query.stats?.invite_rankings ?? []}
              loading={query.loading}
            />
            <RebateRankingTable
              rows={query.stats?.rebate_rankings ?? []}
              loading={query.loading}
            />
          </div>
          <RecentInviteesTable
            rows={query.stats?.recent_invitees ?? []}
            loading={query.loading}
          />
        </div>
      )}
    </SettingsSection>
  )
}

type OverviewStatsRowProps = {
  summary: InviteAdminSummary | null
  loading: boolean
}

function OverviewStatsRow(props: OverviewStatsRowProps) {
  const { t } = useTranslation()
  const summary = props.summary

  return (
    <div className='grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4'>
      <OverviewStatCard
        title={t('Inviters')}
        value={summary ? String(summary.inviter_count) : '0'}
        description={t('Users who invited at least one friend')}
        icon={Users}
        loading={props.loading}
      />
      <OverviewStatCard
        title={t('Invitees')}
        value={summary ? String(summary.invitee_count) : '0'}
        description={t('Users who signed up with a referral')}
        icon={UserPlus}
        loading={props.loading}
      />
      <OverviewStatCard
        title={t('Topped-up invitees')}
        value={summary ? String(summary.recharged_count) : '0'}
        description={t('Invitees with a successful online top-up')}
        icon={CheckCircle2}
        loading={props.loading}
      />
      <OverviewStatCard
        title={t('Rebate issued')}
        value={formatQuotaWithCurrency(summary?.rebate_quota ?? 0)}
        description={t('Sign-up {{signUp}} · Top-up {{topUp}}', {
          signUp: formatQuotaWithCurrency(summary?.register_rebate_quota ?? 0),
          topUp: formatQuotaWithCurrency(summary?.topup_rebate_quota ?? 0),
        })}
        icon={Wallet}
        loading={props.loading}
      />
    </div>
  )
}

type OverviewStatCardProps = {
  title: string
  value: string
  description: string
  icon: LucideIcon
  loading: boolean
}

function OverviewStatCard(props: OverviewStatCardProps) {
  const Icon = props.icon

  return (
    <article
      data-slot='invite-admin-stat-card'
      className='bg-card min-w-0 rounded-xl border px-4 py-4'
    >
      <div className='flex items-start justify-between gap-3'>
        <h4 className='text-muted-foreground text-sm font-medium'>
          {props.title}
        </h4>
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

type InviteCountRankingTableProps = {
  rows: InviteCountRank[]
  loading: boolean
}

function InviteCountRankingTable(props: InviteCountRankingTableProps) {
  const { t } = useTranslation()

  return (
    <div data-slot='invite-admin-invite-rankings' className='min-w-0'>
      <h4 className='mb-2 text-sm font-medium'>{t('Referral rankings')}</h4>
      {props.loading ? (
        <Skeleton className='h-40 w-full' />
      ) : (
        <StaticDataTable
          data={props.rows}
          getRowKey={(row) => row.user_id}
          emptyContent={t('No referral rankings yet.')}
          columns={[
            {
              id: 'rank',
              header: t('Rank'),
              className: 'w-14',
              cell: (row) => row.rank,
            },
            {
              id: 'username',
              header: t('User'),
              cell: (row) => row.username || '—',
            },
            {
              id: 'user_id',
              header: t('User ID'),
              cell: (row) => row.user_id,
            },
            {
              id: 'invitee_count',
              header: t('Invites'),
              cell: (row) => row.invitee_count,
            },
            {
              id: 'rebate_quota',
              header: t('Rebate'),
              cell: (row) => formatQuotaWithCurrency(row.rebate_quota),
            },
          ]}
        />
      )}
    </div>
  )
}

type RebateRankingTableProps = {
  rows: InviteRebateRank[]
  loading: boolean
}

function RebateRankingTable(props: RebateRankingTableProps) {
  const { t } = useTranslation()

  return (
    <div data-slot='invite-admin-rebate-rankings' className='min-w-0'>
      <h4 className='mb-2 text-sm font-medium'>{t('Rebate rankings')}</h4>
      {props.loading ? (
        <Skeleton className='h-40 w-full' />
      ) : (
        <StaticDataTable
          data={props.rows}
          getRowKey={(row) => row.user_id}
          emptyContent={t('No rebate rankings yet.')}
          columns={[
            {
              id: 'rank',
              header: t('Rank'),
              className: 'w-14',
              cell: (row) => row.rank,
            },
            {
              id: 'username',
              header: t('User'),
              cell: (row) => row.username || '—',
            },
            {
              id: 'user_id',
              header: t('User ID'),
              cell: (row) => row.user_id,
            },
            {
              id: 'rebate_quota',
              header: t('Rebate'),
              cell: (row) => formatQuotaWithCurrency(row.rebate_quota),
            },
            {
              id: 'payout_count',
              header: t('Payouts'),
              cell: (row) => row.payout_count,
            },
          ]}
        />
      )}
    </div>
  )
}

type RecentInviteesTableProps = {
  rows: InviteAdminInvitee[]
  loading: boolean
}

function RecentInviteesTable(props: RecentInviteesTableProps) {
  const { t } = useTranslation()

  return (
    <div data-slot='invite-admin-recent-invitees' className='min-w-0'>
      <h4 className='mb-2 text-sm font-medium'>{t('Recent referrals')}</h4>
      {props.loading ? (
        <Skeleton className='h-40 w-full' />
      ) : (
        <StaticDataTable
          data={props.rows}
          getRowKey={(row) => row.invitee_id}
          emptyContent={t('No referrals yet.')}
          columns={[
            {
              id: 'invitee',
              header: t('Invitee'),
              cell: (row) => row.invitee_username || '—',
            },
            {
              id: 'inviter',
              header: t('Inviter'),
              cell: (row) => row.inviter_username || '—',
            },
            {
              id: 'registered_at',
              header: t('Signed up at'),
              cell: (row) =>
                dayjs(row.registered_at * 1000).format('YYYY-MM-DD HH:mm'),
            },
            {
              id: 'status',
              header: t('Status'),
              cell: (row) =>
                row.status === 'recharged' ? t('Topped up') : t('Signed up'),
            },
            {
              id: 'rebate_quota',
              header: t('Rebate'),
              cell: (row) => formatQuotaWithCurrency(row.rebate_quota),
            },
          ]}
        />
      )}
    </div>
  )
}
