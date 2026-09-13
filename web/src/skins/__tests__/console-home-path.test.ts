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
  isNextConsolePath,
  isNextStandaloneShellPath,
  isNextFirstTokenErrorsPath,
  isNextUsageLogsPath,
  resolveNextConsoleShellPathname,
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

describe('isNextFirstTokenErrorsPath', () => {
  test.each(['/first-token-errors', '/first-token-errors/'])(
    'treats %s as a first-token-errors path',
    (pathname) => {
      expect(isNextFirstTokenErrorsPath(pathname)).toBe(true)
    }
  )

  test.each(['/dashboard', '/usage-logs', '/usage-logs/common'])(
    'does not treat %s as a first-token-errors path',
    (pathname) => {
      expect(isNextFirstTokenErrorsPath(pathname)).toBe(false)
    }
  )
})

describe('isNextUsageLogsPath', () => {
  test.each([
    '/usage-logs',
    '/usage-logs/common',
    '/usage-logs/drawing',
    '/usage-logs/task',
  ])('treats %s as a usage-logs path', (pathname) => {
    expect(isNextUsageLogsPath(pathname)).toBe(true)
  })

  test.each(['/dashboard', '/profile', '/wallet', '/keys', '/'])(
    'does not treat %s as a usage-logs path',
    (pathname) => {
      expect(isNextUsageLogsPath(pathname)).toBe(false)
    }
  )
})

describe('isNextStandaloneShellPath', () => {
  test.each([
    '/dashboard',
    '/dashboard/overview',
    '/profile',
    '/wallet',
    '/invite',
    '/usage-logs',
    '/usage-logs/common',
    '/usage-logs/drawing',
    '/usage-logs/task',
    '/first-token-errors',
  ])('uses the standalone shell on %s', (pathname) => {
    expect(isNextStandaloneShellPath(pathname)).toBe(true)
  })

  test.each([
    '/dashboard/models',
    '/dashboard/flow',
    '/dashboard/users',
    '/keys',
    '/',
  ])('leaves %s on the sidebar shell', (pathname) => {
    expect(isNextStandaloneShellPath(pathname)).toBe(false)
  })
})

describe('isNextConsolePath', () => {
  test.each([
    '/dashboard',
    '/dashboard/overview',
    '/dashboard/models',
    '/profile',
    '/wallet',
    '/invite',
    '/usage-logs/common',
    '/first-token-errors',
    '/keys',
    '/playground',
    '/chat/abc',
    '/chat2link',
    '/channels',
    '/channel-monitoring-settings',
    '/models/deployments',
    '/users',
    '/redemption-codes',
    '/subscriptions',
    '/system-info',
    '/system-settings/site',
    '/errors/401',
  ])('treats %s as a console path', (pathname) => {
    expect(isNextConsolePath(pathname)).toBe(true)
  })

  test.each([
    '/',
    '/pricing',
    '/pricing/gpt-4',
    '/rankings',
    '/channel-monitoring',
    '/about',
    '/sign-in',
    '/user-agreement',
    '/privacy-policy',
    '/404',
  ])('does not treat %s as a console path', (pathname) => {
    expect(isNextConsolePath(pathname)).toBe(false)
  })
})

describe('resolveNextConsoleShellPathname', () => {
  test.each([
    {
      pending: '/pricing',
      rendered: '/dashboard/overview',
      expected: '/dashboard/overview',
    },
    {
      pending: '/dashboard/overview',
      rendered: '/pricing',
      expected: '/dashboard/overview',
    },
    {
      pending: '/',
      rendered: '/dashboard/overview',
      expected: '/dashboard/overview',
    },
    {
      pending: '/dashboard/overview',
      rendered: '/',
      expected: '/dashboard/overview',
    },
    {
      pending: '/rankings',
      rendered: '/dashboard/overview',
      expected: '/dashboard/overview',
    },
    {
      pending: '/dashboard/overview',
      rendered: '/rankings',
      expected: '/dashboard/overview',
    },
    {
      pending: '/about',
      rendered: '/dashboard/overview',
      expected: '/dashboard/overview',
    },
    {
      pending: '/pricing',
      rendered: '/keys',
      expected: '/keys',
    },
    {
      pending: '/keys',
      rendered: '/pricing',
      expected: '/keys',
    },
    {
      pending: '/keys',
      rendered: '/dashboard/overview',
      expected: '/dashboard/overview',
    },
    {
      pending: '/pricing',
      rendered: '/pricing',
      expected: '/dashboard/overview',
    },
  ])(
    'uses $expected when pending is $pending and rendered is $rendered',
    ({ pending, rendered, expected }) => {
      expect(resolveNextConsoleShellPathname(pending, rendered)).toBe(expected)
    }
  )

  test('uses the pending console path when no rendered path is available', () => {
    expect(resolveNextConsoleShellPathname('/dashboard/overview')).toBe(
      '/dashboard/overview'
    )
  })
})
