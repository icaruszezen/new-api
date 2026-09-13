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
  consumePreviewSearchParam,
  isAdminConsolePath,
} from '../admin-console-path'

describe('isAdminConsolePath', () => {
  test.each([
    '/channels',
    '/channels/12',
    '/channel-monitoring-settings',
    '/models',
    '/models/metadata',
    '/users',
    '/users/3',
    '/redemption-codes',
    '/subscriptions',
    '/system-info',
    '/system-settings',
    '/system-settings/site/console-ui',
    '/first-token-errors',
  ])('treats %s as an admin console route', (pathname) => {
    expect(isAdminConsolePath(pathname)).toBe(true)
  })

  test.each([
    '/dashboard',
    '/dashboard/overview',
    '/dashboard/users',
    '/usage-logs',
    '/usage-logs/common',
    '/keys',
    '/wallet',
    '/profile',
    '/playground',
    '/pricing',
    '/channel-monitoring',
    '/users-extra',
  ])('leaves shared or public path %s alone', (pathname) => {
    expect(isAdminConsolePath(pathname)).toBe(false)
  })
})

describe('consumePreviewSearchParam', () => {
  test('leaves a URL without preview=1 unchanged', () => {
    expect(consumePreviewSearchParam('?tab=overview')).toEqual({
      wantsPreview: false,
      nextSearchStr: '?tab=overview',
    })
  })

  test('strips preview=1 and keeps remaining params', () => {
    expect(consumePreviewSearchParam('?preview=1&tab=overview')).toEqual({
      wantsPreview: true,
      nextSearchStr: '?tab=overview',
    })
  })

  test('strips a lone preview=1 query', () => {
    expect(consumePreviewSearchParam('?preview=1')).toEqual({
      wantsPreview: true,
      nextSearchStr: '',
    })
  })

  test('ignores preview values other than 1', () => {
    expect(consumePreviewSearchParam('?preview=true')).toEqual({
      wantsPreview: false,
      nextSearchStr: '?preview=true',
    })
  })
})
