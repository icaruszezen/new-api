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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'
import { formatLogQuota } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import {
  markBillingCorrectionRecordsRefunded,
  unmarkBillingCorrectionRecordsRefunded,
} from '../api'
import { buildMarkAllPayload, RECORDS_QUERY_KEY } from '../lib/records'
import type {
  BillingCorrectionRecord,
  BillingCorrectionRecordFilters,
  BillingCorrectionRecordStats,
  BillingCorrectionRefundPayload,
} from '../types'

type ConfirmAction = 'mark-all' | 'mark-selected' | 'unmark-selected'

type RecordsRefundActionsProps = {
  /** The applied filters; "mark all" covers every pending record they match. */
  filters: BillingCorrectionRecordFilters
  stats: BillingCorrectionRecordStats
  selected: ReadonlyMap<string, BillingCorrectionRecord>
  onClearSelection: () => void
  /** Set while the stats still belong to the previous filters. */
  statsOutdated: boolean
}

/**
 * Marks records as refunded, or removes that mark. Every action is confirmed
 * first, and none of them changes a balance: the admin refunds separately.
 */
export function RecordsRefundActions(props: RecordsRefundActionsProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirmAction, setConfirmAction] = useState<ConfirmAction | null>(null)

  const selectedPending = useMemo(
    () => [...props.selected.values()].filter((r) => r.status === 'pending'),
    [props.selected]
  )
  const selectedRefunded = useMemo(
    () => [...props.selected.values()].filter((r) => r.status === 'refunded'),
    [props.selected]
  )
  const selectedPendingQuota = selectedPending.reduce(
    (sum, record) => sum + record.quota,
    0
  )

  const finishWrite = () => {
    setConfirmAction(null)
    props.onClearSelection()
    void queryClient.invalidateQueries({ queryKey: [RECORDS_QUERY_KEY] })
  }

  const markMutation = useMutation({
    mutationFn: (payload: BillingCorrectionRefundPayload) =>
      markBillingCorrectionRecordsRefunded(payload),
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to mark records as refunded'))
        return
      }
      toast.success(
        t('Marked {{count}} record(s) as refunded, {{quota}} in total', {
          count: result.data.marked,
          quota: formatLogQuota(result.data.quota),
        })
      )
      if (result.data.skipped > 0) {
        toast.warning(
          t('{{count}} record(s) without a request ID were skipped', {
            count: result.data.skipped,
          })
        )
      }
      finishWrite()
    },
    onError: handleServerError,
  })

  const unmarkMutation = useMutation({
    mutationFn: (requestIds: string[]) =>
      unmarkBillingCorrectionRecordsRefunded(requestIds),
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to remove the refunded mark'))
        return
      }
      toast.success(
        t('Removed the refunded mark from {{count}} record(s)', {
          count: result.data.unmarked,
        })
      )
      finishWrite()
    },
    onError: handleServerError,
  })

  const refundOnlyNote = t(
    'This only records the refund; it does not return any quota to users.'
  )
  const confirmCopy: Record<ConfirmAction, { title: string; desc: string }> = {
    'mark-all': {
      title: t('Mark all pending records as refunded?'),
      desc: `${t(
        '{{count}} charged record(s) under the current filters, {{quota}} in total, will be marked as refunded.',
        {
          count: props.stats.pending_count,
          quota: formatLogQuota(props.stats.pending_quota),
        }
      )} ${refundOnlyNote}`,
    },
    'mark-selected': {
      title: t('Mark selected records as refunded?'),
      desc: `${t(
        '{{count}} selected record(s), {{quota}} in total, will be marked as refunded.',
        {
          count: selectedPending.length,
          quota: formatLogQuota(selectedPendingQuota),
        }
      )} ${refundOnlyNote}`,
    },
    'unmark-selected': {
      title: t('Remove the refunded mark?'),
      desc: t('{{count}} selected record(s) will go back to pending refund.', {
        count: selectedRefunded.length,
      }),
    },
  }

  const runConfirmedAction = () => {
    if (confirmAction === 'mark-all') {
      markMutation.mutate(buildMarkAllPayload(props.filters))
      return
    }
    if (confirmAction === 'mark-selected') {
      markMutation.mutate({
        request_ids: selectedPending.map((record) => record.request_id),
      })
      return
    }
    if (confirmAction === 'unmark-selected') {
      unmarkMutation.mutate(selectedRefunded.map((record) => record.request_id))
    }
  }

  return (
    <>
      <div className='flex flex-wrap items-center gap-2'>
        {props.selected.size > 0 && (
          <>
            <span className='text-muted-foreground text-sm'>
              {t('Selected {{count}}', { count: props.selected.size })}
            </span>
            <Button
              size='sm'
              disabled={selectedPending.length === 0}
              onClick={() => setConfirmAction('mark-selected')}
            >
              {t('Mark selected as refunded')}
            </Button>
            <Button
              size='sm'
              variant='outline'
              disabled={selectedRefunded.length === 0}
              onClick={() => setConfirmAction('unmark-selected')}
            >
              {t('Remove refunded mark')}
            </Button>
            <Button size='sm' variant='ghost' onClick={props.onClearSelection}>
              {t('Clear selection')}
            </Button>
          </>
        )}
        <Button
          size='sm'
          variant='outline'
          className='ml-auto'
          disabled={props.statsOutdated || props.stats.pending_count === 0}
          onClick={() => setConfirmAction('mark-all')}
        >
          {t('Mark all pending as refunded')}
        </Button>
      </div>

      <ConfirmDialog
        open={confirmAction !== null}
        onOpenChange={(open) => {
          if (!open) setConfirmAction(null)
        }}
        title={confirmAction ? confirmCopy[confirmAction].title : ''}
        desc={confirmAction ? confirmCopy[confirmAction].desc : ''}
        confirmText={t('Confirm')}
        isLoading={markMutation.isPending || unmarkMutation.isPending}
        handleConfirm={runConfirmedAction}
      />
    </>
  )
}
