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
import type {
  BillingCorrectionRecord,
  BillingCorrectionRecordFilters,
  BillingCorrectionRecordQuery,
  BillingCorrectionRefundPayload,
} from '../types'

const DEFAULT_RANGE_DAYS = 7

/** Prefix of every records query key, so a refund mark refreshes all pages. */
export const RECORDS_QUERY_KEY = 'billing-correction-records'

/** From midnight seven days ago until an hour from now. */
export function getDefaultRecordFilters(): BillingCorrectionRecordFilters {
  const now = new Date()
  const start = new Date(now)
  start.setDate(start.getDate() - DEFAULT_RANGE_DAYS)
  start.setHours(0, 0, 0, 0)
  return {
    start,
    end: new Date(now.getTime() + 3600 * 1000),
    username: '',
    token: '',
    model: '',
    channel: '',
    status: '',
  }
}

/** The filter part shared by the list query and "mark all as refunded". */
function buildFilterParams(filters: BillingCorrectionRecordFilters) {
  const channel = Number(filters.channel)
  return {
    start_timestamp: filters.start
      ? Math.floor(filters.start.getTime() / 1000)
      : undefined,
    end_timestamp: filters.end
      ? Math.floor(filters.end.getTime() / 1000)
      : undefined,
    username: filters.username.trim() || undefined,
    token_name: filters.token.trim() || undefined,
    model_name: filters.model.trim() || undefined,
    channel: Number.isInteger(channel) && channel > 0 ? channel : undefined,
  }
}

export function buildRecordQuery(
  filters: BillingCorrectionRecordFilters,
  page: number,
  pageSize: number
): BillingCorrectionRecordQuery {
  return {
    p: page,
    page_size: pageSize,
    ...buildFilterParams(filters),
    status: filters.status || undefined,
  }
}

/**
 * Marks every charged, unmarked record under the current filters. The status
 * filter is left out on purpose: only pending records can be marked anyway.
 */
export function buildMarkAllPayload(
  filters: BillingCorrectionRecordFilters
): BillingCorrectionRefundPayload {
  return { all: true, ...buildFilterParams(filters) }
}

/** Deleted channels come back without a name, so fall back to the id. */
export function getChannelLabel(channelId: number, channelName?: string) {
  return channelName || `#${channelId}`
}

/** Corrected records were never charged, and legacy logs have no request id. */
export function isRecordSelectable(record: BillingCorrectionRecord): boolean {
  return record.status !== 'corrected' && record.request_id !== ''
}
