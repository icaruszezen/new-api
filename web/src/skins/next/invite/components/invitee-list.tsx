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

import { Skeleton } from '@/components/ui/skeleton'
import type { InviteeFilter, InviteeSummary } from '@/features/invite/types'
import { formatQuotaWithCurrency } from '@/lib/currency'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

type NextInviteeListProps = {
  invitees: InviteeSummary[]
  filter: InviteeFilter
  onFilterChange: (filter: InviteeFilter) => void
  hasInvitees: boolean
  loading: boolean
}

const FILTERS: { value: InviteeFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'registered', label: 'Signed up' },
  { value: 'recharged', label: 'Topped up' },
]

function InviteeStatusPill(props: { status: InviteeSummary['status'] }) {
  const { t } = useTranslation()
  const isRecharged = props.status === 'recharged'

  return (
    <span
      data-slot='next-invitee-status'
      data-status={props.status}
      className={cn(
        'inline-flex items-center rounded-md border px-2 py-0.5 text-xs',
        isRecharged
          ? 'bg-foreground text-background border-transparent'
          : 'border-border text-muted-foreground'
      )}
    >
      {isRecharged ? t('Topped up') : t('Signed up')}
    </span>
  )
}

export function NextInviteeList(props: NextInviteeListProps) {
  const { t } = useTranslation()

  let emptyMessage = t(
    'No invited friends yet. Send the link above to a friend.'
  )
  if (props.hasInvitees) {
    emptyMessage = t('No friends match this filter.')
  }

  return (
    <section
      data-slot='next-invitee-list'
      className='bg-card overflow-hidden rounded-xl border'
    >
      <div className='flex flex-wrap items-start justify-between gap-3 px-5 py-5'>
        <div className='min-w-0'>
          <h2 className='text-lg font-medium tracking-tight'>
            {t('Invited friends')}
          </h2>
          <p className='text-muted-foreground mt-1 text-sm'>
            {props.hasInvitees
              ? t(
                  'Only masked usernames, sign-up time, status and rebate amount are shown.'
                )
              : t('Friends will show up here once they sign up.')}
          </p>
        </div>
        <div
          role='group'
          aria-label={t('Filter invited friends')}
          className='bg-muted/40 flex shrink-0 gap-1 rounded-lg p-1'
        >
          {FILTERS.map((filter) => (
            <button
              key={filter.value}
              type='button'
              aria-pressed={props.filter === filter.value}
              onClick={() => props.onFilterChange(filter.value)}
              className={cn(
                'rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
                props.filter === filter.value
                  ? 'bg-card text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              {t(filter.label)}
            </button>
          ))}
        </div>
      </div>

      <div className='overflow-x-auto border-t'>
        <table className='w-full min-w-[36rem] text-left text-sm'>
          <thead>
            <tr className='text-muted-foreground text-xs font-medium'>
              <th className='px-5 py-2.5 font-medium'>{t('Username')}</th>
              <th className='px-5 py-2.5 font-medium'>{t('Signed up at')}</th>
              <th className='px-5 py-2.5 font-medium'>{t('Status')}</th>
              <th className='px-5 py-2.5 text-end font-medium'>
                {t('Rebate amount')}
              </th>
            </tr>
          </thead>
          <tbody>
            {props.loading ? (
              <tr className='border-border/70 border-t'>
                <td colSpan={4} className='px-5 py-6'>
                  <Skeleton className='h-8 w-full' />
                </td>
              </tr>
            ) : null}
            {!props.loading && props.invitees.length === 0 ? (
              <tr className='border-border/70 border-t'>
                <td
                  colSpan={4}
                  className='text-muted-foreground px-5 py-10 text-center'
                >
                  {emptyMessage}
                </td>
              </tr>
            ) : null}
            {!props.loading
              ? props.invitees.map((invitee) => (
                  <tr
                    key={`${invitee.username}-${invitee.registered_at}`}
                    className='border-border/70 border-t'
                  >
                    <td className='px-5 py-3 font-medium'>
                      {invitee.username}
                    </td>
                    <td className='text-muted-foreground px-5 py-3 tabular-nums'>
                      {dayjs(invitee.registered_at * 1000).format(
                        'YYYY-MM-DD HH:mm'
                      )}
                    </td>
                    <td className='px-5 py-3'>
                      <InviteeStatusPill status={invitee.status} />
                    </td>
                    <td className='px-5 py-3 text-end tabular-nums'>
                      {formatQuotaWithCurrency(invitee.rebate_quota)}
                    </td>
                  </tr>
                ))
              : null}
          </tbody>
        </table>
      </div>
    </section>
  )
}
