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

/** Fixed track for the lowest-ratio tag so every row lines that column up. */
export const MODEL_LIST_RATIO_COL = '3.75rem'

/** Shared row shell so the header and each summary row stay aligned. */
export const MODEL_LIST_ROW_CLASS = 'flex w-full items-center'

/** Chevron + model name. Grows on every breakpoint so long names can truncate. */
export const MODEL_LIST_PRIMARY_CLASS =
  'flex min-w-0 flex-1 items-center gap-x-2 pl-3 md:gap-x-4 md:pl-4'

export const MODEL_LIST_PRIMARY_NAME_CLASS =
  'flex min-w-0 flex-1 items-center gap-2 text-left'

/** Multiplier stays in the collapsed row on phones and desktops. */
export const MODEL_LIST_RATIO_CLASS = 'flex w-[3.75rem] shrink-0 justify-center'

/** Trailing cluster: ratio always, prices only from the md breakpoint. */
export const MODEL_LIST_META_CLASS =
  'flex shrink-0 items-center pr-3 md:min-w-0 md:flex-[1.35] md:gap-x-4 md:pr-4'

/**
 * Input / output / cache stay out of the phone summary.
 * Desktop restores the three-column price track.
 */
export const MODEL_LIST_PRICE_CLASS =
  'hidden md:grid md:flex-1 md:grid-cols-[minmax(4.75rem,1fr)_minmax(4.75rem,1fr)_minmax(5.25rem,1.2fr)] md:items-center md:gap-x-4'
