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
import { getRouteApi } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { DataTablePage, useDataTable } from '@/components/data-table'
import { useTableUrlState } from '@/hooks/use-table-url-state'

import { getFirstTokenErrorLogs } from '../api'
import { buildFirstTokenErrorQuery } from '../lib/utils'
import { useFirstTokenErrorColumns } from './columns'
import { FirstTokenErrorFilterBar } from './filter-bar'

const route = getRouteApi('/_authenticated/first-token-errors/')

export function FirstTokenErrorTable() {
  const { t } = useTranslation()
  const search = route.useSearch()
  const columns = useFirstTokenErrorColumns()
  const {
    pagination,
    onPaginationChange,
    ensurePageInRange,
  } = useTableUrlState({
    search,
    navigate: route.useNavigate(),
    pagination: { defaultPage: 1, defaultPageSize: 20 },
    globalFilter: { enabled: false },
    columnFilters: [],
  })

  const query = buildFirstTokenErrorQuery({
    page: pagination.pageIndex + 1,
    pageSize: pagination.pageSize,
    username: search.username,
    token: search.token,
    model: search.model,
    channel: search.channel,
    group: search.group,
    requestId: search.requestId,
    startTime: search.startTime,
    endTime: search.endTime,
  })

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['first-token-errors', query],
    queryFn: async () => {
      const result = await getFirstTokenErrorLogs(query)
      if (!result.success) {
        toast.error(result.message || t('Failed to load logs'))
        return { items: [], total: 0, page: 1, page_size: query.page_size ?? 20 }
      }
      return (
        result.data ?? {
          items: [],
          total: 0,
          page: 1,
          page_size: query.page_size ?? 20,
        }
      )
    },
  })

  const { table } = useDataTable({
    data: data?.items ?? [],
    columns,
    pagination,
    enableRowSelection: false,
    onPaginationChange,
    manualPagination: true,
    totalCount: data?.total ?? 0,
    ensurePageInRange,
    columnVisibilityStorageKey: 'first-token-errors:column-visibility',
  })

  return (
    <div data-slot='first-token-errors-table' className='min-h-0 flex-1'>
      <DataTablePage
        table={table}
        columns={columns}
        isLoading={isLoading}
        isFetching={isFetching}
        emptyTitle={t('No first-token errors')}
        emptyDescription={t(
          'Corrected first-token upstream errors will appear here.'
        )}
        toolbar={<FirstTokenErrorFilterBar />}
      />
    </div>
  )
}
