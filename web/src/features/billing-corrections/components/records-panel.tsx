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
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'

import { getBillingCorrectionRecords } from '../api'
import {
  buildRecordQuery,
  getDefaultRecordFilters,
  hasUpstreamCheck,
  RECORDS_QUERY_KEY,
} from '../lib/records'
import type {
  BillingCorrectionRecord,
  BillingCorrectionRecordFilters,
  BillingCorrectionRule,
} from '../types'
import { RecordsFilterBar } from './records-filter-bar'
import { RecordsRefundActions } from './records-refund-actions'
import { RecordsStats } from './records-stats'
import { RecordsTable } from './records-table'

const PAGE_SIZE = 20

type RecordsPanelProps = {
  rules: BillingCorrectionRule[]
}

/**
 * Lists zero-output charged requests on monitored channels so an admin can
 * return the quota, mark an outside refund, or check the upstream charge.
 */
export function RecordsPanel(props: RecordsPanelProps) {
  const { t } = useTranslation()
  const upstreamEnabled = hasUpstreamCheck(props.rules)
  const [filters, setFilters] = useState(getDefaultRecordFilters)
  const [page, setPage] = useState(1)
  const [selected, setSelected] = useState<
    Map<string, BillingCorrectionRecord>
  >(() => new Map())

  const query = useMemo(
    () => buildRecordQuery(filters, page, PAGE_SIZE),
    [filters, page]
  )
  const recordsQuery = useQuery({
    queryKey: [RECORDS_QUERY_KEY, query],
    queryFn: () => getBillingCorrectionRecords(query),
    placeholderData: keepPreviousData,
  })
  const response = recordsQuery.data
  const recordPage = response?.success ? response.data : undefined

  const applyFilters = (next: BillingCorrectionRecordFilters) => {
    setFilters(next)
    setPage(1)
    setSelected(new Map())
  }

  const selectRecords = (
    records: BillingCorrectionRecord[],
    checked: boolean
  ) => {
    setSelected((previous) => {
      const next = new Map(previous)
      for (const record of records) {
        if (checked) next.set(record.request_id, record)
        else next.delete(record.request_id)
      }
      return next
    })
  }

  return (
    <div className='space-y-4'>
      <Alert>
        <AlertDescription className='text-xs'>
          <span className='block'>
            {t(
              'Lists every charged text request with zero output tokens on channels with an enabled rule, including requests made before the rule was enabled. Records above the threshold that were billed on a local estimate match the rule; the rest only had zero output. Listing a record never changes its charge.'
            )}
          </span>
          <span className='block'>
            {t(
              "Returning quota credits the charged amount back to the user's wallet. Marking records as refunded only records a refund made elsewhere."
            )}
          </span>
        </AlertDescription>
      </Alert>

      <RecordsFilterBar
        rules={props.rules}
        filters={filters}
        onApply={applyFilters}
        upstreamEnabled={upstreamEnabled}
      />

      {response && !response.success && (
        <Alert variant='destructive'>
          <AlertDescription>
            {response.message || t('Failed to load records')}
          </AlertDescription>
        </Alert>
      )}
      {recordsQuery.isError && (
        <Alert variant='destructive'>
          <AlertDescription>{t('Failed to load records')}</AlertDescription>
        </Alert>
      )}

      {recordsQuery.isLoading && (
        <div className='space-y-2'>
          <Skeleton className='h-20 w-full' />
          <Skeleton className='h-10 w-full' />
          <Skeleton className='h-10 w-full' />
        </div>
      )}

      {recordPage && (
        <>
          <RecordsStats
            stats={recordPage.stats}
            upstreamEnabled={upstreamEnabled}
          />
          <RecordsRefundActions
            filters={filters}
            stats={recordPage.stats}
            selected={selected}
            onClearSelection={() => setSelected(new Map())}
            statsOutdated={recordsQuery.isPlaceholderData}
            upstreamEnabled={upstreamEnabled}
          />
          <RecordsTable
            upstreamEnabled={upstreamEnabled}
            records={recordPage.items}
            selected={selected}
            onSelect={selectRecords}
            page={recordPage.page}
            pageSize={recordPage.page_size}
            total={recordPage.total}
            onPageChange={setPage}
          />
        </>
      )}
    </div>
  )
}
