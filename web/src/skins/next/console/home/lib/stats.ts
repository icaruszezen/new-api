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
import type { QuotaDataItem } from '@/features/dashboard/types'
import { dateToUnixTimestamp, getStartOfDay } from '@/lib/time'

export const MISSING_METRIC = '--'

export type QuotaSumField = 'count' | 'token_used' | 'quota'

export type RunwayEstimate =
  | { kind: 'days'; days: number }
  | { kind: 'less-than-one-day' }
  | { kind: 'depleted' }
  | { kind: 'no-recent-usage' }

const MAX_RUNWAY_DAYS = 999

/**
 * Estimate how long the remaining wallet quota lasts at the recent
 * daily spend rate. Zero balance and zero spend are distinct states.
 */
export function getRunwayEstimate(
  remainQuota: number,
  recentUsage: number
): RunwayEstimate {
  if (remainQuota <= 0) return { kind: 'depleted' }
  if (recentUsage <= 0) return { kind: 'no-recent-usage' }
  const days = remainQuota / recentUsage
  if (!Number.isFinite(days) || days <= 0) return { kind: 'no-recent-usage' }
  if (days < 1) return { kind: 'less-than-one-day' }
  return { kind: 'days', days: Math.min(MAX_RUNWAY_DAYS, Math.floor(days)) }
}

export function getCalendarDayRange(now: Date = new Date()): {
  start_timestamp: number
  end_timestamp: number
} {
  return {
    start_timestamp: dateToUnixTimestamp(getStartOfDay(now)),
    end_timestamp: dateToUnixTimestamp(now),
  }
}

export function sumQuotaField(
  items: QuotaDataItem[],
  field: QuotaSumField
): number {
  let total = 0
  for (const item of items) {
    total += Number(item[field]) || 0
  }
  return total
}

/**
 * Present a metric that the homepage can actually compute. Zero is a real
 * value and must stay visible; only an absent series becomes `--`.
 */
export function formatKnownMetric(
  value: number,
  formatNumber: (amount: number) => string
): string {
  return formatNumber(value)
}

export function formatMissingMetric(): string {
  return MISSING_METRIC
}
