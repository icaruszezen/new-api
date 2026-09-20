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
  BEAT_STATUS_DOWN,
  BEAT_STATUS_SLOW,
  BEAT_STATUS_UP,
  type Beat,
} from '../types'

/** Availability HSL hue multiplier: 0%=red(0) / 50%=yellow(60) / 100%=green(120). */
const HSL_HUE_PER_PERCENT = 1.2
const HSL_SATURATION = 72
const HSL_LIGHTNESS = 42

export type HealthScoreBand =
  | 'score0'
  | 'score1'
  | 'score2'
  | 'score3'
  | 'score4'
  | 'score5'
  | 'score6'
  | 'score7'
  | 'score8'
  | 'score9'
  | 'score10'
  | 'unknown'

/** Multi-stop green → yellow → red, matching sub2api V2 pulse cells. */
export const HEALTH_SCORE_COLORS: Record<HealthScoreBand, string> = {
  score10: '#16a34a',
  score9: '#22c55e',
  score8: '#4ade80',
  score7: '#a3e635',
  score6: '#facc15',
  score5: '#fbbf24',
  score4: '#f59e0b',
  score3: '#f97316',
  score2: '#fb7185',
  score1: '#f87171',
  score0: 'rgb(239, 67, 67)',
  unknown: '#9ca3af',
}

export const BEAT_HEIGHT_PCT = {
  up: 100,
  slow: 65,
  down: 35,
  empty: 15,
} as const

export function hslForPct(pct: number | null | undefined): string | undefined {
  if (pct === null || pct === undefined || Number.isNaN(pct)) return undefined
  const clamped = Math.max(0, Math.min(100, pct))
  const hue = clamped * HSL_HUE_PER_PERCENT
  return `hsl(${hue} ${HSL_SATURATION}% ${HSL_LIGHTNESS}%)`
}

export function scoreToBand(
  score: number | null | undefined
): HealthScoreBand {
  if (score == null || Number.isNaN(score)) return 'unknown'
  const clamped = Math.max(0, Math.min(100, score))
  const band = Math.round(clamped / 10)
  return `score${Math.max(0, Math.min(10, band))}` as HealthScoreBand
}

export function beatWindowSuccessRate(beat: Beat): number | null {
  const total = beat.request_total ?? 0
  if (total > 0) {
    return (((beat.request_up ?? 0) + (beat.request_slow ?? 0)) / total) * 100
  }
  if (beat.status === BEAT_STATUS_UP || beat.status === BEAT_STATUS_SLOW) {
    return 100
  }
  if (beat.status === BEAT_STATUS_DOWN) {
    return 0
  }
  return null
}

export function beatHeightPct(status: number | undefined): number {
  if (status === BEAT_STATUS_UP) return BEAT_HEIGHT_PCT.up
  if (status === BEAT_STATUS_SLOW) return BEAT_HEIGHT_PCT.slow
  if (status === BEAT_STATUS_DOWN) return BEAT_HEIGHT_PCT.down
  return BEAT_HEIGHT_PCT.empty
}

export function beatBarColor(beat?: Beat): string {
  if (!beat) return HEALTH_SCORE_COLORS.unknown
  return HEALTH_SCORE_COLORS[scoreToBand(beatWindowSuccessRate(beat))]
}
