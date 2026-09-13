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

import {
  canAccessFirstTokenErrors,
  canEditFirstTokenErrorSettings,
} from '../access'

describe('first-token error access', () => {
  test('regular users cannot open the admin page', () => {
    expect(canAccessFirstTokenErrors(ROLE.USER)).toBe(false)
    expect(canAccessFirstTokenErrors(ROLE.GUEST)).toBe(false)
    expect(canAccessFirstTokenErrors(undefined)).toBe(false)
  })

  test('admins and root can open the admin page', () => {
    expect(canAccessFirstTokenErrors(ROLE.ADMIN)).toBe(true)
    expect(canAccessFirstTokenErrors(ROLE.SUPER_ADMIN)).toBe(true)
  })

  test('only root can change the four header settings', () => {
    expect(canEditFirstTokenErrorSettings(ROLE.USER)).toBe(false)
    expect(canEditFirstTokenErrorSettings(ROLE.ADMIN)).toBe(false)
    expect(canEditFirstTokenErrorSettings(ROLE.SUPER_ADMIN)).toBe(true)
  })
})
