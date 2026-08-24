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
import type { LogsViewScope } from '@/features/usage-logs/components/usage-logs-provider'

type NextUsageLogsHeadingProps = {
  title: string
  canManageScope: boolean
  viewScope: LogsViewScope
  onViewScopeChange: (scope: string) => void
}

export function NextUsageLogsHeading(props: NextUsageLogsHeadingProps) {
  const { t } = useTranslation()

  return (
    <header
      data-slot='next-usage-logs-heading'
      className='flex flex-wrap items-center justify-between gap-x-3 gap-y-2'
    >
      <h1 className='text-3xl font-medium tracking-tight'>{props.title}</h1>
      {props.canManageScope && (
        <Tabs value={props.viewScope} onValueChange={props.onViewScopeChange}>
          <TabsList className='h-9 rounded-full p-1'>
            <TabsTrigger value='all' className='rounded-full px-3.5'>
              {t('All')}
            </TabsTrigger>
            <TabsTrigger value='self' className='rounded-full px-3.5'>
              {t('Only Mine')}
            </TabsTrigger>
          </TabsList>
        </Tabs>
      )}
    </header>
  )
}
