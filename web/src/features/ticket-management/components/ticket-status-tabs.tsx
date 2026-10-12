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

import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  TICKET_STATUS_CONFIG,
  TICKET_STATUS_VALUES,
} from '@/features/tickets/constants'
import type { TicketStats, TicketStatus } from '@/features/tickets/types'

import { TICKET_STATUS_STATS_FIELD } from '../lib/ticket-management'

const ALL_STATUSES = 'all'

type TicketStatusTabsProps = {
  /** `undefined` selects every status. */
  value?: TicketStatus
  /** Counts appear once the stats have loaded. */
  stats?: TicketStats
  onChange: (status: TicketStatus | undefined) => void
}

export function TicketStatusTabs(props: TicketStatusTabsProps) {
  const { t } = useTranslation()
  const stats = props.stats
  const tabs: {
    value: TicketStatus | typeof ALL_STATUSES
    label: string
    count?: number
  }[] = [
    {
      value: ALL_STATUSES,
      label: t('All'),
      count: stats
        ? TICKET_STATUS_VALUES.reduce(
            (sum, status) => sum + stats[TICKET_STATUS_STATS_FIELD[status]],
            0
          )
        : undefined,
    },
    ...TICKET_STATUS_VALUES.map((status) => ({
      value: status,
      label: t(TICKET_STATUS_CONFIG[status].labelKey),
      count: stats?.[TICKET_STATUS_STATS_FIELD[status]],
    })),
  ]

  return (
    <div className='flex min-w-0 flex-wrap items-center gap-2'>
      <div className='max-w-full overflow-x-auto'>
        <Tabs
          value={props.value ?? ALL_STATUSES}
          onValueChange={(value: TicketStatus | typeof ALL_STATUSES) =>
            props.onChange(value === ALL_STATUSES ? undefined : value)
          }
        >
          <TabsList>
            {tabs.map((tab) => (
              <TabsTrigger key={tab.value} value={tab.value}>
                {tab.label}
                {tab.count !== undefined && (
                  <>
                    {' '}
                    <span className='text-muted-foreground text-xs tabular-nums'>
                      {tab.count}
                    </span>
                  </>
                )}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>
      {stats && stats.unread > 0 && (
        <span className='inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs'>
          <span
            aria-hidden='true'
            className='bg-primary size-1.5 rounded-full'
          />
          {t('{{count}} unread', { count: stats.unread })}
        </span>
      )}
    </div>
  )
}
