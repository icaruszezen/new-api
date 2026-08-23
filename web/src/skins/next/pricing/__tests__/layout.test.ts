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

import { MODEL_LIST_GRID_CLASS, MODEL_LIST_RATIO_COL } from '../layout'

describe('next model square list layout', () => {
  test('reserves a fixed-width ratio column immediately after the model name', () => {
    expect(MODEL_LIST_RATIO_COL).toBe('3.75rem')
    expect(MODEL_LIST_GRID_CLASS).toContain(
      `minmax(0,1.5fr)_${MODEL_LIST_RATIO_COL}_minmax(4.75rem,1fr)`
    )
  })
})
