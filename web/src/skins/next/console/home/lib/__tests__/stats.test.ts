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
  CACHE_RATE_MIN_SAMPLE,
  formatCacheReadRate,
  formatKnownMetric,
  getCalendarDayRange,
  getRunwayEstimate,
  sumCacheSampledCalls,
  sumQuotaField,
} from '../stats'

const ENOUGH_CALLS = CACHE_RATE_MIN_SAMPLE

const asPercent = (rate: number) => `${rate}%`

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
  })

  test('reports the cache read rate as a share of input tokens', () => {
    expect(formatCacheReadRate(250, 1000, ENOUGH_CALLS, asPercent)).toBe('25%')
    expect(formatCacheReadRate(0, 1000, ENOUGH_CALLS, asPercent)).toBe('0%')
  })

  test('withholds the cache read rate until the sample reaches the threshold', () => {
    expect(
      formatCacheReadRate(250, 1000, CACHE_RATE_MIN_SAMPLE - 1, asPercent)
    ).toBeNull()
    expect(formatCacheReadRate(250, 1000, 0, asPercent)).toBeNull()
  })

  test('withholds the cache read rate when the sample carries no input tokens', () => {
    expect(formatCacheReadRate(0, 0, ENOUGH_CALLS, asPercent)).toBeNull()
    expect(formatCacheReadRate(120, 0, ENOUGH_CALLS, asPercent)).toBeNull()
    expect(
      formatCacheReadRate(10, Number.NaN, ENOUGH_CALLS, asPercent)
    ).toBeNull()
  })

  test('counts only buckets carrying input tokens toward the sample', () => {
    expect(
      sumCacheSampledCalls([
        { created_at: 1, count: 30, prompt_tokens: 900 },
        { created_at: 2, count: 25, prompt_tokens: 0 },
        { created_at: 3, count: 12 },
        { created_at: 4, prompt_tokens: 400 },
      ])
    ).toBe(30)
  })

  test('caps the cache read rate at 100 percent when cache exceeds input', () => {
    expect(formatCacheReadRate(1500, 1000, ENOUGH_CALLS, asPercent)).toBe(
      '100%'
    )
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
