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

const pathnameRef = { current: '/dashboard/overview' }

vi.mock('@tanstack/react-router', () => ({
  useRouterState: (options?: {
    select?: (state: { location: { pathname: string } }) => unknown
  }) => {
    const state = { location: { pathname: pathnameRef.current } }
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
    pathnameRef.current = '/dashboard/overview'
  })

  test('uses the standalone console shell on the user homepage', () => {
    render(<NextAuthenticatedLayout />)

    expect(screen.getByTestId('console-home-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('sidebar-shell')).toBeNull()
  })

  test('keeps the sidebar shell on other console routes', () => {
    pathnameRef.current = '/dashboard/models'
    render(<NextAuthenticatedLayout />)

    expect(screen.getByTestId('sidebar-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('console-home-shell')).toBeNull()
  })
})
