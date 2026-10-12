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
import type { TicketStatus } from '@/features/tickets/types'

const ALL_TAB = 'all'

type NextTicketStatusTabsProps = {
  status?: TicketStatus
  onStatusChange: (status: TicketStatus | undefined) => void
}

export function NextTicketStatusTabs(props: NextTicketStatusTabsProps) {
  const { t } = useTranslation()

  return (
    <Tabs
      data-slot='next-tickets-status-tabs'
      value={props.status === undefined ? ALL_TAB : String(props.status)}
      onValueChange={(value) =>
        props.onStatusChange(
          TICKET_STATUS_VALUES.find((status) => String(status) === value)
        )
      }
    >
      {/* One row that scrolls sideways on narrow screens instead of wrapping. */}
      <TabsList
        aria-label={t('Filter by status')}
        className='no-scrollbar h-9 max-w-full justify-start overflow-x-auto rounded-full p-1 group-data-horizontal/tabs:h-auto'
      >
        <TabsTrigger value={ALL_TAB} className='flex-none rounded-full px-3.5'>
          {t('All')}
        </TabsTrigger>
        {TICKET_STATUS_VALUES.map((status) => (
          <TabsTrigger
            key={status}
            value={String(status)}
            className='flex-none rounded-full px-3.5'
          >
            {t(TICKET_STATUS_CONFIG[status].labelKey)}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  )
}
