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

import { parseUiSkin } from '../registry'

describe('parseUiSkin', () => {
  test.each([
    ['classic', 'classic'],
    ['next', 'next'],
  ])('keeps the supported skin %s', (input, expected) => {
    expect(parseUiSkin(input)).toBe(expected)
  })

  test.each([
    ['missing value', undefined],
    ['null value', null],
    ['empty string', ''],
    ['blank string', '   '],
    ['wrong case', 'NEXT'],
    ['unknown skin', 'legacy'],
    ['non-string value', 1],
  ])('falls back to classic for %s', (_label, input) => {
    expect(parseUiSkin(input)).toBe('classic')
  })
})
