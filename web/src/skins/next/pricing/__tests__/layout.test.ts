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
  MODEL_LIST_META_CLASS,
  MODEL_LIST_PRICE_CLASS,
  MODEL_LIST_PRIMARY_CLASS,
  MODEL_LIST_RATIO_CLASS,
  MODEL_LIST_RATIO_COL,
} from '../layout'

describe('next model square list layout', () => {
  test('reserves a fixed-width ratio column immediately after the model name', () => {
    expect(MODEL_LIST_RATIO_COL).toBe('3.75rem')
    expect(MODEL_LIST_RATIO_CLASS).toContain(`w-[${MODEL_LIST_RATIO_COL}]`)
    expect(MODEL_LIST_META_CLASS).toContain('flex')
    expect(MODEL_LIST_PRIMARY_CLASS).toContain('flex-1')
    expect(MODEL_LIST_PRIMARY_CLASS).toContain('min-w-0')
  })

  test('keeps prices out of the phone summary and restores them from the md breakpoint', () => {
    expect(MODEL_LIST_PRICE_CLASS).toContain('hidden')
    expect(MODEL_LIST_PRICE_CLASS).toContain('md:grid')
    expect(MODEL_LIST_PRICE_CLASS).not.toContain('overflow-x-auto')
    expect(MODEL_LIST_PRIMARY_CLASS).not.toContain('sticky')
    expect(MODEL_LIST_PRICE_CLASS).toContain(
      'md:grid-cols-[minmax(4.75rem,1fr)_minmax(4.75rem,1fr)_minmax(5.25rem,1.2fr)]'
    )
  })
})
