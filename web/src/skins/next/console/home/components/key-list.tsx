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
  flexRender,
  type Row,
  type Table as TanstackTable,
} from '@tanstack/react-table'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DISABLED_ROW_DESKTOP, useDataTable } from '@/components/data-table'
import { Skeleton } from '@/components/ui/skeleton'
import {
  useApiKeysColumns,
  useGroupRatios,
} from '@/features/keys/components/api-keys-columns'
import { ApiKeysMobileList } from '@/features/keys/components/api-keys-mobile-list'
import { useApiKeys } from '@/features/keys/components/api-keys-provider'
import { DataTableRowActions } from '@/features/keys/components/data-table-row-actions'
import { isDisabledApiKeyRow } from '@/features/keys/constants'
import type { ApiKey } from '@/features/keys/types'
import { useMediaQuery } from '@/hooks'
import { cn } from '@/lib/utils'

import { ConsoleKeyGroupCell } from './key-group-cell'

type ConsoleKeyListProps = {
  isLoading: boolean
  keys: ApiKey[]
}

function ConsoleKeyDesktopTable(props: {
  isLoading: boolean
  table: TanstackTable<ApiKey>
}) {
  const { t } = useTranslation()
  const columns = props.table.getAllColumns()

  return (
    <div className='overflow-x-auto' data-slot='console-key-desktop-table'>
      <table className='w-full min-w-[72rem] text-left text-sm'>
        <thead>
          {props.table.getHeaderGroups().map((headerGroup) => (
            <tr
              key={headerGroup.id}
              className='text-muted-foreground border-border/70 border-t text-xs font-medium'
            >
              {headerGroup.headers.map((header) => {
                const isActions = header.column.id === 'actions'
                return (
                  <th
                    key={header.id}
                    className={cn(
                      'px-4 py-2.5 font-medium',
                      isActions && 'bg-card sticky right-0 text-end'
                    )}
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(
                          header.column.columnDef.header,
                          header.getContext()
                        )}
                  </th>
                )
              })}
            </tr>
          ))}
        </thead>
        <tbody>
          {props.isLoading ? (
            <tr className='border-border/70 border-t'>
              <td colSpan={columns.length} className='px-4 py-6'>
                <Skeleton className='h-8 w-full' />
              </td>
            </tr>
          ) : null}
          {!props.isLoading && props.table.getRowModel().rows.length === 0 ? (
            <tr className='border-border/70 border-t'>
              <td
                colSpan={columns.length}
                className='text-muted-foreground px-4 py-8 text-center'
              >
                {t('No API keys yet')}
              </td>
            </tr>
          ) : null}
          {props.table.getRowModel().rows.map((row) => (
            <tr
              key={row.id}
              className={cn(
                'border-border/70 border-t',
                isDisabledApiKeyRow(row.original) && DISABLED_ROW_DESKTOP
              )}
            >
              {row.getVisibleCells().map((cell) => {
                const isActions = cell.column.id === 'actions'
                return (
                  <td
                    key={cell.id}
                    className={cn(
                      'px-4 py-3',
                      isActions && 'bg-card sticky right-0'
                    )}
                  >
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                )
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function ConsoleKeyList(props: ConsoleKeyListProps) {
  const { t } = useTranslation()
  const { triggerRefresh } = useApiKeys()
  const [now, setNow] = useState(() => Date.now())
  const isMobile = useMediaQuery('(max-width: 640px)')
  const allColumns = useApiKeysColumns(now)
  const groupRatios = useGroupRatios()
  const columns = useMemo(
    () =>
      allColumns
        .filter((column) => {
          const columnId =
            column.id ??
            ('accessorKey' in column ? column.accessorKey : undefined)
          return (
            columnId !== 'select' &&
            columnId !== 'created_time' &&
            columnId !== 'accessed_time'
          )
        })
        .map((column) => {
          const columnId =
            column.id ??
            ('accessorKey' in column ? column.accessorKey : undefined)
          if (columnId === 'actions') {
            return {
              ...column,
              cell: ({ row }: { row: Row<ApiKey> }) => (
                <DataTableRowActions row={row} overflow='delete' />
              ),
            }
          }
          if (columnId === 'group') {
            return {
              ...column,
              cell: ({ row }: { row: Row<ApiKey> }) => {
                const group = row.original.group ?? ''
                return (
                  <ConsoleKeyGroupCell
                    apiKey={row.original}
                    ratio={groupRatios[group]}
                    onSwitched={triggerRefresh}
                  />
                )
              },
            }
          }
          return column
        }),
    [allColumns, groupRatios, triggerRefresh]
  )

  useEffect(() => {
    const intervalId = window.setInterval(() => {
      setNow(Date.now())
    }, 30_000)

    return () => window.clearInterval(intervalId)
  }, [])

  const { table } = useDataTable({
    data: props.keys,
    columns,
    enableRowSelection: false,
    columnVisibilityStorageKey: false,
    columnSizingStorageKey: false,
    getRowId: (row) => String(row.id),
    manualPagination: true,
    pagination: { pageIndex: 0, pageSize: Math.max(props.keys.length, 1) },
  })

  if (isMobile) {
    return (
      <ApiKeysMobileList
        table={table}
        isLoading={props.isLoading}
        showGroupRatio
        className='rounded-none border-0'
        emptyTitle={t('No API keys yet')}
        emptyDescription=''
        renderRowActions={(row) => (
          <DataTableRowActions row={row} overflow='delete' />
        )}
        renderGroup={(apiKey) => (
          <div className='space-y-1'>
            <div className='text-muted-foreground text-xs'>{t('Group')}</div>
            <ConsoleKeyGroupCell
              apiKey={apiKey}
              ratio={groupRatios[apiKey.group ?? '']}
              onSwitched={triggerRefresh}
              className='w-full'
            />
          </div>
        )}
      />
    )
  }

  return <ConsoleKeyDesktopTable table={table} isLoading={props.isLoading} />
}
