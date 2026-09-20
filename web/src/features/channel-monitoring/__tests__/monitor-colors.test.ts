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
along with this program. If you did not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { describe, expect, test } from 'vitest'

import {
  BEAT_HEIGHT_PCT,
  HEALTH_SCORE_COLORS,
  beatBarColor,
  beatHeightPct,
  beatWindowSuccessRate,
  hslForPct,
  scoreToBand,
} from '../lib/monitor-colors'
import { BEAT_STATUS_DOWN, BEAT_STATUS_SLOW, BEAT_STATUS_UP } from '../types'

describe('monitor colors', () => {
  test('maps 0 / 50 / 100 percent onto the red-yellow-green HSL arc', () => {
    expect(hslForPct(0)).toBe('hsl(0 72% 42%)')
    expect(hslForPct(50)).toBe('hsl(60 72% 42%)')
    expect(hslForPct(100)).toBe('hsl(120 72% 42%)')
    expect(hslForPct(null)).toBeUndefined()
  })

  test('maps window success rate onto the eleven V2 score bands', () => {
    expect(scoreToBand(100)).toBe('score10')
    expect(scoreToBand(80)).toBe('score8')
    expect(scoreToBand(50)).toBe('score5')
    expect(scoreToBand(0)).toBe('score0')
    expect(scoreToBand(null)).toBe('unknown')
  })

  test('uses request counts for the window rate and falls back to status', () => {
    expect(
      beatWindowSuccessRate({
        ts: 1,
        status: BEAT_STATUS_UP,
        ttft_ms: 0,
        request_total: 10,
        request_up: 1,
        request_down: 9,
      })
    ).toBe(10)
    expect(
      beatWindowSuccessRate({ ts: 1, status: BEAT_STATUS_UP, ttft_ms: 0 })
    ).toBe(100)
    expect(
      beatWindowSuccessRate({ ts: 1, status: BEAT_STATUS_DOWN, ttft_ms: 0 })
    ).toBe(0)
  })

  test('encodes beat height by representative status', () => {
    expect(beatHeightPct(BEAT_STATUS_UP)).toBe(BEAT_HEIGHT_PCT.up)
    expect(beatHeightPct(BEAT_STATUS_SLOW)).toBe(BEAT_HEIGHT_PCT.slow)
    expect(beatHeightPct(BEAT_STATUS_DOWN)).toBe(BEAT_HEIGHT_PCT.down)
    expect(beatHeightPct(undefined)).toBe(BEAT_HEIGHT_PCT.empty)
  })

  test('paints a mixed window with the score-band colour, not the representative green', () => {
    expect(
      beatBarColor({
        ts: 1,
        status: BEAT_STATUS_UP,
        ttft_ms: 120,
        request_total: 10,
        request_up: 1,
        request_down: 9,
      })
    ).toBe(HEALTH_SCORE_COLORS.score1)
  })
})
