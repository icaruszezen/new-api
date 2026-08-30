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
import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, test, vi } from 'vitest'

const routerPathRef = {
  location: '/dashboard/overview',
  resolvedLocation: '/dashboard/overview',
}

vi.mock('@tanstack/react-router', () => ({
  useRouterState: (options?: {
    select?: (state: {
      location: { pathname: string }
      resolvedLocation?: { pathname: string }
    }) => unknown
  }) => {
    const state = {
      location: { pathname: routerPathRef.location },
      resolvedLocation: { pathname: routerPathRef.resolvedLocation },
    }
    return options?.select ? options.select(state) : state
  },
  useNavigate: () => vi.fn(),
}))

vi.mock('@/components/layout/components/authenticated-layout', () => ({
  AuthenticatedLayout: () => <div data-testid='sidebar-shell' />,
}))

vi.mock('../console/shell', () => ({
  NextConsoleShell: (props: { previewOffset?: boolean }) => (
    <div
      data-testid='console-home-shell'
      data-preview-offset={props.previewOffset ? 'true' : 'false'}
    />
  ),
}))

vi.mock('../preview-banner', () => ({
  NextConsolePreviewBanner: () => <div data-testid='preview-banner' />,
}))

vi.mock('../../use-user-console-preview', () => ({
  useUserConsolePreview: () => ({
    canPreview: false,
    isPreviewing: false,
    startPreview: () => undefined,
    stopPreview: () => undefined,
  }),
}))

const { NextAuthenticatedLayout } = await import('../authenticated-layout')

describe('next authenticated layout chrome', () => {
  afterEach(() => {
    routerPathRef.location = '/dashboard/overview'
    routerPathRef.resolvedLocation = '/dashboard/overview'
  })

  test('uses the standalone console shell on the user homepage', () => {
    render(<NextAuthenticatedLayout />)

    expect(screen.getByTestId('console-home-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('sidebar-shell')).toBeNull()
  })

  test('uses the standalone console shell on the personal center', () => {
    routerPathRef.location = '/profile'
    routerPathRef.resolvedLocation = '/profile'
    render(<NextAuthenticatedLayout />)

    expect(screen.getByTestId('console-home-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('sidebar-shell')).toBeNull()
  })

  test('uses the standalone console shell on the wallet', () => {
    routerPathRef.location = '/wallet'
    routerPathRef.resolvedLocation = '/wallet'
    render(<NextAuthenticatedLayout />)

    expect(screen.getByTestId('console-home-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('sidebar-shell')).toBeNull()
  })

  test.each(['/usage-logs/common', '/usage-logs/drawing', '/usage-logs/task'])(
    'uses the standalone console shell on %s',
    (pathname) => {
      routerPathRef.location = pathname
      routerPathRef.resolvedLocation = pathname
      render(<NextAuthenticatedLayout />)

      expect(screen.getByTestId('console-home-shell')).toBeInTheDocument()
      expect(screen.queryByTestId('sidebar-shell')).toBeNull()
    }
  )

  test('keeps the sidebar shell on other console routes', () => {
    routerPathRef.location = '/dashboard/models'
    routerPathRef.resolvedLocation = '/dashboard/models'
    render(<NextAuthenticatedLayout />)

    expect(screen.getByTestId('sidebar-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('console-home-shell')).toBeNull()
  })

  test('keeps the standalone shell while the model square is still pending', () => {
    routerPathRef.location = '/pricing'
    routerPathRef.resolvedLocation = '/dashboard/overview'
    render(<NextAuthenticatedLayout />)

    expect(screen.getByTestId('console-home-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('sidebar-shell')).toBeNull()
  })

  test('keeps the standalone shell while entering the console from the model square', () => {
    routerPathRef.location = '/dashboard/overview'
    routerPathRef.resolvedLocation = '/pricing'
    render(<NextAuthenticatedLayout />)

    expect(screen.getByTestId('console-home-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('sidebar-shell')).toBeNull()
  })

  test.each(['/', '/rankings', '/about'])(
    'keeps the standalone shell while entering the console from %s',
    (publicPath) => {
      routerPathRef.location = '/dashboard/overview'
      routerPathRef.resolvedLocation = publicPath
      render(<NextAuthenticatedLayout />)

      expect(screen.getByTestId('console-home-shell')).toBeInTheDocument()
      expect(screen.queryByTestId('sidebar-shell')).toBeNull()
    }
  )
})
