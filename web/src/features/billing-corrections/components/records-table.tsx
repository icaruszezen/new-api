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

import { StaticDataTable } from '@/components/data-table'
import { StatusBadge, type StatusVariant } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  formatLogQuota,
  formatNumber,
  formatTimestampToDate,
} from '@/lib/format'

import { getChannelLabel, isRecordSelectable } from '../lib/records'
import type {
  BillingCorrectionRecord,
  BillingCorrectionRecordStatus,
  BillingCorrectionUpstreamStatus,
} from '../types'

type RecordsTableProps = {
  /** Shows the upstream result column when any rule checks the upstream. */
  upstreamEnabled: boolean
  records: BillingCorrectionRecord[]
  selected: ReadonlyMap<string, BillingCorrectionRecord>
  onSelect: (records: BillingCorrectionRecord[], checked: boolean) => void
  page: number
  pageSize: number
  total: number
  onPageChange: (page: number) => void
}

const STATUS_BADGE_VARIANTS: Record<
  BillingCorrectionRecordStatus,
  StatusVariant
> = {
  pending: 'warning',
  refunded: 'success',
  credited: 'success',
  corrected: 'neutral',
}

const UPSTREAM_BADGE_VARIANTS: Record<
  BillingCorrectionUpstreamStatus,
  StatusVariant
> = {
  charged: 'neutral',
  not_charged: 'warning',
  not_found: 'neutral',
  failed: 'danger',
  unsupported: 'neutral',
  unchecked: 'neutral',
  unavailable: 'neutral',
}

export function RecordsTable(props: RecordsTableProps) {
  const { t } = useTranslation()
  const statusLabels: Record<BillingCorrectionRecordStatus, string> = {
    pending: t('Pending refund'),
    refunded: t('Marked as refunded'),
    credited: t('Quota returned'),
    corrected: t('Corrected'),
  }
  const upstreamLabels: Record<BillingCorrectionUpstreamStatus, string> = {
    charged: t('Upstream charged'),
    not_charged: t('Upstream not charged'),
    not_found: t('Not found upstream'),
    failed: t('Upstream check failed'),
    unsupported: t('Upstream unsupported'),
    unchecked: t('Not checked'),
    unavailable: t('Cannot check'),
  }
  const selectableRecords = props.records.filter(isRecordSelectable)
  const selectedOnPage = selectableRecords.filter((record) =>
    props.selected.has(record.request_id)
  ).length
  const pageCount = Math.max(1, Math.ceil(props.total / props.pageSize))

  return (
    <div className='space-y-2'>
      <StaticDataTable
        tableClassName='min-w-max'
        data={props.records}
        getRowKey={(record, index) => record.request_id || `row-${index}`}
        emptyClassName='text-muted-foreground py-8'
        emptyContent={t('No matching records')}
        columns={[
          {
            id: 'select',
            header: (
              <Checkbox
                aria-label={t('Select all on this page')}
                checked={
                  selectedOnPage > 0 &&
                  selectedOnPage === selectableRecords.length
                }
                indeterminate={
                  selectedOnPage > 0 &&
                  selectedOnPage < selectableRecords.length
                }
                disabled={selectableRecords.length === 0}
                onCheckedChange={(checked) =>
                  props.onSelect(selectableRecords, checked)
                }
              />
            ),
            cell: (record) => (
              <Checkbox
                aria-label={t('Select record {{id}}', {
                  id: record.request_id || '-',
                })}
                checked={props.selected.has(record.request_id)}
                disabled={!isRecordSelectable(record)}
                onCheckedChange={(checked) => props.onSelect([record], checked)}
              />
            ),
          },
          {
            id: 'time',
            header: t('Time'),
            cellClassName: 'font-mono whitespace-nowrap',
            cell: (record) => formatTimestampToDate(record.created_at),
          },
          {
            id: 'user',
            header: t('User'),
            cell: (record) => record.username,
          },
          {
            id: 'token',
            header: t('Token'),
            cell: (record) => record.token_name || '-',
          },
          {
            id: 'channel',
            header: t('Channel'),
            cell: (record) =>
              getChannelLabel(record.channel_id, record.channel_name),
          },
          {
            id: 'model',
            header: t('Model'),
            cell: (record) => record.model_name,
          },
          {
            id: 'prompt',
            header: t('Input tokens'),
            cellClassName: 'font-mono tabular-nums',
            cell: (record) => formatNumber(record.prompt_tokens),
          },
          {
            id: 'quota',
            header: t('Cost'),
            cellClassName: 'font-mono tabular-nums',
            cell: (record) => (
              <div>
                <div>{formatLogQuota(record.quota)}</div>
                {record.original_quota !== undefined && (
                  <div className='text-muted-foreground text-xs'>
                    {t('Original cost')}:{' '}
                    {formatLogQuota(record.original_quota)}
                  </div>
                )}
              </div>
            ),
          },
          {
            id: 'match',
            header: t('Condition'),
            cell: (record) =>
              record.match === 'rule'
                ? t('Matches rule')
                : t('Zero output only'),
          },
          {
            id: 'status',
            header: t('Status'),
            cell: (record) => (
              <StatusBadge
                label={
                  record.refund_kind === 'auto'
                    ? t('Returned automatically')
                    : statusLabels[record.status]
                }
                variant={STATUS_BADGE_VARIANTS[record.status]}
                copyable={false}
              />
            ),
          },
          ...(props.upstreamEnabled
            ? [
                {
                  id: 'upstream',
                  header: t('Upstream result'),
                  cell: (record: BillingCorrectionRecord) => {
                    if (!record.upstream_status) return '-'
                    return (
                      <div title={record.upstream_message || undefined}>
                        <StatusBadge
                          label={upstreamLabels[record.upstream_status]}
                          variant={
                            UPSTREAM_BADGE_VARIANTS[record.upstream_status]
                          }
                          copyable={false}
                        />
                        {record.upstream_status === 'charged' && (
                          <div className='text-muted-foreground mt-1 text-xs tabular-nums'>
                            {formatLogQuota(record.upstream_quota ?? 0)}
                          </div>
                        )}
                      </div>
                    )
                  },
                },
              ]
            : []),
          {
            id: 'refunded_at',
            header: t('Refunded at'),
            cellClassName: 'font-mono whitespace-nowrap',
            cell: (record) => formatTimestampToDate(record.refunded_at),
          },
          {
            id: 'request_id',
            header: t('Request ID'),
            cellClassName: 'font-mono text-xs',
            cell: (record) => record.request_id || '-',
          },
        ]}
      />
      <div className='flex items-center justify-end gap-2 text-sm'>
        <span className='text-muted-foreground'>
          {t('{{count}} record(s)', { count: props.total })}
        </span>
        <span>
          {t('Page {{current}} of {{total}}', {
            current: props.page,
            total: pageCount,
          })}
        </span>
        <Button
          size='sm'
          variant='outline'
          disabled={props.page <= 1}
          onClick={() => props.onPageChange(props.page - 1)}
        >
          {t('Previous page')}
        </Button>
        <Button
          size='sm'
          variant='outline'
          disabled={props.page >= pageCount}
          onClick={() => props.onPageChange(props.page + 1)}
        >
          {t('Next page')}
        </Button>
      </div>
    </div>
  )
}
