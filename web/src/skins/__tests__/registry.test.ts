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

import { ROLE } from '@/lib/roles'

import { parseUiSkin, resolveConsoleSkin } from '../registry'

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

describe('resolveConsoleSkin', () => {
  test.each([
    [
      'classic site keeps classic for a regular user',
      'classic',
      ROLE.USER,
      false,
      'classic',
    ],
    [
      'classic site keeps classic for an administrator',
      'classic',
      ROLE.ADMIN,
      false,
      'classic',
    ],
    ['classic site ignores preview', 'classic', ROLE.ADMIN, true, 'classic'],
    ['next site serves a regular user', 'next', ROLE.USER, false, 'next'],
    ['next site serves a guest-level role', 'next', ROLE.GUEST, false, 'next'],
    [
      'next site keeps administrators on classic',
      'next',
      ROLE.ADMIN,
      false,
      'classic',
    ],
    [
      'next site keeps super administrators on classic',
      'next',
      ROLE.SUPER_ADMIN,
      false,
      'classic',
    ],
    [
      'next site serves an administrator who is previewing',
      'next',
      ROLE.ADMIN,
      true,
      'next',
    ],
    ['role 9 is still a regular user on next', 'next', 9, false, 'next'],
    ['role 10 without preview stays on classic', 'next', 10, false, 'classic'],
    ['role 10 with preview uses next', 'next', 10, true, 'next'],
  ] as const)('%s', (_label, siteSkin, role, preview, expected) => {
    expect(resolveConsoleSkin(siteSkin, role, preview)).toBe(expected)
  })
})
