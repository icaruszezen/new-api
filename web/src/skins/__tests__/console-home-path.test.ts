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
  isNextConsoleHomePath,
  isNextStandaloneShellPath,
} from '../console-home-path'

describe('isNextConsoleHomePath', () => {
  test.each(['/dashboard', '/dashboard/overview'])(
    'treats %s as the next console homepage',
    (pathname) => {
      expect(isNextConsoleHomePath(pathname)).toBe(true)
    }
  )

  test.each([
    '/dashboard/models',
    '/dashboard/flow',
    '/dashboard/users',
    '/keys',
    '/wallet',
    '/profile',
    '/',
  ])('does not treat %s as the console homepage', (pathname) => {
    expect(isNextConsoleHomePath(pathname)).toBe(false)
  })
})

describe('isNextStandaloneShellPath', () => {
  test.each(['/dashboard', '/dashboard/overview', '/profile'])(
    'uses the standalone shell on %s',
    (pathname) => {
      expect(isNextStandaloneShellPath(pathname)).toBe(true)
    }
  )

  test.each([
    '/dashboard/models',
    '/dashboard/flow',
    '/dashboard/users',
    '/keys',
    '/wallet',
    '/',
  ])('leaves %s on the sidebar shell', (pathname) => {
    expect(isNextStandaloneShellPath(pathname)).toBe(false)
  })
})
