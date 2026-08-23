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
import { describe, expect, test } from 'vitest'

import {
  formatKnownMetric,
  formatMissingMetric,
  getCalendarDayRange,
  getRunwayEstimate,
  MISSING_METRIC,
  sumQuotaField,
} from '../stats'

describe('console homepage stats', () => {
  test('sums today call and token fields and treats missing numbers as zero', () => {
    const total = sumQuotaField(
      [
        { created_at: 1, count: 10, token_used: 100 },
        { created_at: 2, count: 7, token_used: 20 },
        { created_at: 3 },
      ],
      'count'
    )
    const tokens = sumQuotaField(
      [
        { created_at: 1, count: 10, token_used: 100 },
        { created_at: 2, count: 7, token_used: 20 },
        { created_at: 3 },
      ],
      'token_used'
    )

    expect(total).toBe(17)
    expect(tokens).toBe(120)
  })

  test('keeps a zero total visible instead of collapsing it to a missing mark', () => {
    expect(sumQuotaField([], 'count')).toBe(0)
    expect(formatKnownMetric(0, (value) => String(value))).toBe('0')
    expect(formatMissingMetric()).toBe(MISSING_METRIC)
    expect(formatMissingMetric()).toBe('--')
  })

  test('uses local midnight through the supplied instant for today', () => {
    const now = new Date('2026-08-24T15:30:00')
    const range = getCalendarDayRange(now)
    const start = new Date(range.start_timestamp * 1000)
    const end = new Date(range.end_timestamp * 1000)

    expect(start.getHours()).toBe(0)
    expect(start.getMinutes()).toBe(0)
    expect(end.getTime()).toBe(now.getTime())
  })

  test('estimates remaining days from wallet quota and recent spend', () => {
    expect(getRunwayEstimate(500, 100)).toEqual({ kind: 'days', days: 5 })
    expect(getRunwayEstimate(50, 100)).toEqual({ kind: 'less-than-one-day' })
    expect(getRunwayEstimate(0, 100)).toEqual({ kind: 'depleted' })
    expect(getRunwayEstimate(500, 0)).toEqual({ kind: 'no-recent-usage' })
    expect(getRunwayEstimate(1_000_000, 1)).toEqual({ kind: 'days', days: 999 })
  })
})
