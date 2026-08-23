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

/**
 * Display-only mask for a token suffix. The list API never returns the
 * full secret; this only formats whatever fragment is already public.
 */
export function maskApiKey(key: string): string {
  const full = key.startsWith('sk-') ? key : `sk-${key}`
  if (full.length <= 14) return full
  return `${full.slice(0, 7)}***${full.slice(-4)}`
}

export function formatGroupRatio(
  ratio: number | string | undefined
): string | null {
  if (ratio == null || ratio === '') return null
  if (typeof ratio === 'number') {
    if (!Number.isFinite(ratio)) return null
    return `${ratio}x`
  }
  const numeric = Number(ratio)
  if (Number.isFinite(numeric)) return `${numeric}x`
  return ratio
}
