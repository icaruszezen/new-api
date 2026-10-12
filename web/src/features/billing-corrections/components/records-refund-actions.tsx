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
  checkBillingCorrectionRecordsUpstream,
  creditBillingCorrectionRecords,
  markBillingCorrectionRecordsRefunded,
  unmarkBillingCorrectionRecordsRefunded,
} from '../api'
import {
  buildAllRecordsPayload,
  isRecordUpstreamCheckable,
  RECORDS_QUERY_KEY,
} from '../lib/records'
import type {
  BillingCorrectionRecord,
  BillingCorrectionRecordFilters,
  BillingCorrectionRecordStats,
  BillingCorrectionRecordsPayload,
} from '../types'

type ConfirmAction =
  | 'credit-all'
  | 'credit-selected'
  | 'mark-all'
  | 'mark-selected'
  | 'unmark-selected'

type RecordsRefundActionsProps = {
  /** The applied filters; the "all" actions cover every record they match. */
  filters: BillingCorrectionRecordFilters
  stats: BillingCorrectionRecordStats
  selected: ReadonlyMap<string, BillingCorrectionRecord>
  onClearSelection: () => void
  /** Set while the stats still belong to the previous filters. */
  statsOutdated: boolean
  /** Shows the upstream re-check actions when any rule checks the upstream. */
  upstreamEnabled: boolean
}

/**
 * Returns quota to users, marks refunds made elsewhere (or removes that mark),
 * and re-checks the upstream charge. Every write is confirmed first; the
 * upstream re-check only records results and never refunds.
 */
export function RecordsRefundActions(props: RecordsRefundActionsProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirmAction, setConfirmAction] = useState<ConfirmAction | null>(null)

  const selectedRecords = useMemo(
    () => [...props.selected.values()],
    [props.selected]
  )
  const selectedPending = selectedRecords.filter(
    (record) => record.status === 'pending'
  )
  const selectedRefunded = selectedRecords.filter(
    (record) => record.status === 'refunded'
  )
  const selectedCheckable = selectedRecords.filter(isRecordUpstreamCheckable)
  const selectedPendingQuota = selectedPending.reduce(
    (sum, record) => sum + record.quota,
    0
  )

  const refreshRecords = () => {
    void queryClient.invalidateQueries({ queryKey: [RECORDS_QUERY_KEY] })
  }

  const finishWrite = () => {
    setConfirmAction(null)
    props.onClearSelection()
    refreshRecords()
  }

  const creditMutation = useMutation({
    mutationFn: (payload: BillingCorrectionRecordsPayload) =>
      creditBillingCorrectionRecords(payload),
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to return quota'))
        return
      }
      toast.success(
        t('Returned {{quota}} to users for {{count}} record(s)', {
          count: result.data.credited,
          quota: formatLogQuota(result.data.quota),
        })
      )
      if (result.data.skipped > 0) {
        toast.warning(
          t(
            '{{count}} record(s) were skipped because they were already refunded or have no request ID',
            { count: result.data.skipped }
          )
        )
      }
      if (result.data.failed > 0) {
        toast.error(
          t(
            '{{count}} record(s) could not be returned; see the system log for details',
            { count: result.data.failed }
          )
        )
      }
      finishWrite()
    },
    onError: handleServerError,
  })

  const markMutation = useMutation({
    mutationFn: (payload: BillingCorrectionRecordsPayload) =>
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

  const checkMutation = useMutation({
    mutationFn: (payload: BillingCorrectionRecordsPayload) =>
      checkBillingCorrectionRecordsUpstream(payload),
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to check the upstream'))
        return
      }
      const summary = result.data
      toast.success(
        t(
          'Checked {{count}} record(s): {{charged}} charged upstream, {{notCharged}} not charged, {{notFound}} not found, {{failed}} failed',
          {
            count: summary.checked,
            charged: summary.charged,
            notCharged: summary.not_charged,
            notFound: summary.not_found,
            failed: summary.failed + summary.unsupported,
          }
        )
      )
      if (summary.remaining > 0) {
        toast.info(
          t(
            '{{count}} record(s) are still waiting; run the check again to continue',
            { count: summary.remaining }
          )
        )
      }
      refreshRecords()
    },
    onError: handleServerError,
  })

  const quotaReturnNote = t(
    "The quota goes back to each user's wallet balance, and the user gets a refund log. Each record can be returned only once."
  )
  const refundOnlyNote = t(
    'This only records the refund; it does not return any quota to users.'
  )
  const confirmCopy: Record<
    ConfirmAction,
    { title: string; desc: React.JSX.Element | string }
  > = {
    'credit-all': {
      title: t('Return quota for all pending records?'),
      desc: (
        <>
          <span className='block'>
            {t(
              '{{count}} pending record(s) under the current filters, {{quota}} in total, will be returned to users.',
              {
                count: props.stats.pending_count,
                quota: formatLogQuota(props.stats.pending_quota),
              }
            )}
          </span>
          <span className='block'>{quotaReturnNote}</span>
        </>
      ),
    },
    'credit-selected': {
      title: t('Return quota for the selected records?'),
      desc: (
        <>
          <span className='block'>
            {t(
              '{{count}} selected pending record(s), {{quota}} in total, will be returned to users.',
              {
                count: selectedPending.length,
                quota: formatLogQuota(selectedPendingQuota),
              }
            )}
          </span>
          <span className='block'>{quotaReturnNote}</span>
        </>
      ),
    },
    'mark-all': {
      title: t('Mark all pending records as refunded?'),
      desc: (
        <>
          <span className='block'>
            {t(
              '{{count}} charged record(s) under the current filters, {{quota}} in total, will be marked as refunded.',
              {
                count: props.stats.pending_count,
                quota: formatLogQuota(props.stats.pending_quota),
              }
            )}
          </span>
          <span className='block'>{refundOnlyNote}</span>
        </>
      ),
    },
    'mark-selected': {
      title: t('Mark selected records as refunded?'),
      desc: (
        <>
          <span className='block'>
            {t(
              '{{count}} selected record(s), {{quota}} in total, will be marked as refunded.',
              {
                count: selectedPending.length,
                quota: formatLogQuota(selectedPendingQuota),
              }
            )}
          </span>
          <span className='block'>{refundOnlyNote}</span>
        </>
      ),
    },
    'unmark-selected': {
      title: t('Remove the refunded mark?'),
      desc: t('{{count}} selected record(s) will go back to pending refund.', {
        count: selectedRefunded.length,
      }),
    },
  }

  const selectedPendingIds = selectedPending.map((record) => record.request_id)
  const runConfirmedAction = () => {
    switch (confirmAction) {
      case 'credit-all':
        creditMutation.mutate(buildAllRecordsPayload(props.filters))
        return
      case 'credit-selected':
        creditMutation.mutate({ request_ids: selectedPendingIds })
        return
      case 'mark-all':
        markMutation.mutate(buildAllRecordsPayload(props.filters))
        return
      case 'mark-selected':
        markMutation.mutate({ request_ids: selectedPendingIds })
        return
      case 'unmark-selected':
        unmarkMutation.mutate(
          selectedRefunded.map((record) => record.request_id)
        )
    }
  }

  const allActionsDisabled =
    props.statsOutdated || props.stats.pending_count === 0

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
              onClick={() => setConfirmAction('credit-selected')}
            >
              {t('Return quota')}
            </Button>
            <Button
              size='sm'
              variant='outline'
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
            {props.upstreamEnabled && (
              <Button
                size='sm'
                variant='outline'
                disabled={
                  selectedCheckable.length === 0 || checkMutation.isPending
                }
                onClick={() =>
                  checkMutation.mutate({
                    request_ids: selectedCheckable.map(
                      (record) => record.request_id
                    ),
                  })
                }
              >
                {t('Re-check upstream')}
              </Button>
            )}
            <Button size='sm' variant='ghost' onClick={props.onClearSelection}>
              {t('Clear selection')}
            </Button>
          </>
        )}
        <div className='ml-auto flex flex-wrap items-center gap-2'>
          {props.upstreamEnabled && (
            <Button
              size='sm'
              variant='outline'
              disabled={props.statsOutdated || checkMutation.isPending}
              onClick={() =>
                checkMutation.mutate(buildAllRecordsPayload(props.filters))
              }
            >
              {checkMutation.isPending
                ? t('Checking upstream...')
                : t('Check upstream for filtered records')}
            </Button>
          )}
          <Button
            size='sm'
            variant='outline'
            disabled={allActionsDisabled}
            onClick={() => setConfirmAction('mark-all')}
          >
            {t('Mark all pending as refunded')}
          </Button>
          <Button
            size='sm'
            disabled={allActionsDisabled}
            onClick={() => setConfirmAction('credit-all')}
          >
            {t('Return quota for all pending')}
          </Button>
        </div>
      </div>

      <ConfirmDialog
        open={confirmAction !== null}
        onOpenChange={(open) => {
          if (!open) setConfirmAction(null)
        }}
        title={confirmAction ? confirmCopy[confirmAction].title : ''}
        desc={confirmAction ? confirmCopy[confirmAction].desc : ''}
        confirmText={t('Confirm')}
        isLoading={
          creditMutation.isPending ||
          markMutation.isPending ||
          unmarkMutation.isPending
        }
        handleConfirm={runConfirmedAction}
      />
    </>
  )
}
