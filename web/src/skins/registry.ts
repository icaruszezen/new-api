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
import { ROLE } from '@/lib/roles'

import type { UiSkin } from './types'

export const UI_SKIN_VALUES: ReadonlySet<UiSkin> = new Set<UiSkin>([
  'classic',
  'next',
])

export const DEFAULT_UI_SKIN: UiSkin = 'classic'

/**
 * Normalize an unknown skin value coming from `/api/status`, persisted state or
 * the admin form. Missing, empty and unknown values fall back to `classic` so
 * an unconfigured deployment keeps its current console.
 */
export function parseUiSkin(value: unknown): UiSkin {
  if (typeof value === 'string' && UI_SKIN_VALUES.has(value as UiSkin)) {
    return value as UiSkin
  }
  return DEFAULT_UI_SKIN
}

/**
 * Choose the authenticated console shell. Site `classic` stays classic for
 * everyone. Site `next` serves regular users; administrators keep classic
 * unless they opted into a session-level user-console preview.
 */
export function resolveConsoleSkin(
  siteSkin: UiSkin,
  role: number,
  previewUserConsole: boolean
): UiSkin {
  if (siteSkin !== 'next') return 'classic'
  if (role >= ROLE.ADMIN && !previewUserConsole) return 'classic'
  return 'next'
}
