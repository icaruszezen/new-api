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

import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { useConsolePreviewStore } from '../console-preview-store'

const routeParams = { current: { section: 'overview' } }

vi.mock('@tanstack/react-router', () => ({
  getRouteApi: () => ({
    useParams: () => routeParams.current,
  }),
}))

vi.mock('@/features/dashboard', () => ({
  Dashboard: () => <div data-testid='classic-dashboard' />,
}))

vi.mock('../next/console/home', () => ({
  NextConsoleHome: () => <div data-testid='next-console-home' />,
}))

const { SkinnedDashboard } = await import('../dashboard')

function renderDashboard() {
  return render(<SkinnedDashboard />)
}

function setUser(role: number, userId = 1) {
  useAuthStore.getState().auth.setUser({
    id: userId,
    username: 'tester',
    role,
  })
}

function setSiteSkin(uiSkin: 'classic' | 'next') {
  useSystemConfigStore.setState({
    config: { ...useSystemConfigStore.getState().config, uiSkin },
  })
}

describe('dashboard skin dispatch', () => {
  afterEach(() => {
    routeParams.current = { section: 'overview' }
    setSiteSkin('classic')
    useConsolePreviewStore.getState().exitPreview()
    useAuthStore.getState().auth.reset()
  })

  test('renders the next homepage for a regular user on overview', () => {
    setSiteSkin('next')
    setUser(ROLE.USER)
    renderDashboard()

    expect(screen.getByTestId('next-console-home')).toBeInTheDocument()
    expect(screen.queryByTestId('classic-dashboard')).toBeNull()
  })

  test('keeps the shared dashboard on model analytics for a next user', () => {
    setSiteSkin('next')
    setUser(ROLE.USER)
    routeParams.current = { section: 'models' }
    renderDashboard()

    expect(screen.getByTestId('classic-dashboard')).toBeInTheDocument()
    expect(screen.queryByTestId('next-console-home')).toBeNull()
  })

  test('keeps the classic overview when the site skin is classic', () => {
    setSiteSkin('classic')
    setUser(ROLE.USER)
    renderDashboard()

    expect(screen.getByTestId('classic-dashboard')).toBeInTheDocument()
    expect(screen.queryByTestId('next-console-home')).toBeNull()
  })

  test('keeps an administrator on the classic overview unless they preview', () => {
    setSiteSkin('next')
    setUser(ROLE.ADMIN, 8)
    renderDashboard()

    expect(screen.getByTestId('classic-dashboard')).toBeInTheDocument()
    expect(screen.queryByTestId('next-console-home')).toBeNull()
  })

  test('renders the next homepage when an administrator previews overview', () => {
    setSiteSkin('next')
    setUser(ROLE.ADMIN, 8)
    useConsolePreviewStore.getState().enterPreview(8)
    renderDashboard()

    expect(screen.getByTestId('next-console-home')).toBeInTheDocument()
    expect(screen.queryByTestId('classic-dashboard')).toBeNull()
  })
})
