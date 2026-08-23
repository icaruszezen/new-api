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

import { SkinnedAuthenticatedLayout } from '../authenticated-layout'
import { useConsolePreviewStore } from '../console-preview-store'

vi.mock('../classic/authenticated-layout', () => ({
  ClassicAuthenticatedLayout: () => <div data-testid='classic-shell' />,
}))

vi.mock('../next/authenticated-layout', () => ({
  NextAuthenticatedLayout: () => <div data-testid='next-shell' />,
}))

function renderConsole(uiSkin: unknown) {
  const store = useSystemConfigStore.getState()
  useSystemConfigStore.setState({
    config: { ...store.config, uiSkin: uiSkin as never },
  })

  return render(<SkinnedAuthenticatedLayout />)
}

function setUserRole(role: number, userId = 1) {
  useAuthStore.getState().auth.setUser({
    id: userId,
    username: 'tester',
    role,
  })
}

describe('authenticated console skin dispatch', () => {
  afterEach(() => {
    useSystemConfigStore.setState({
      config: { ...useSystemConfigStore.getState().config, uiSkin: 'classic' },
    })
    useConsolePreviewStore.getState().exitPreview()
    useAuthStore.getState().auth.reset()
    sessionStorage.clear()
    document.body.removeAttribute('data-ui-skin')
  })

  test('renders the classic shell when the administrator chose classic', () => {
    renderConsole('classic')

    expect(screen.getByTestId('classic-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('next-shell')).toBeNull()
    expect(document.body.getAttribute('data-ui-skin')).toBe('classic')
  })

  test('renders the next shell when the administrator chose next', () => {
    renderConsole('next')

    expect(screen.getByTestId('next-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('classic-shell')).toBeNull()
    expect(document.body.getAttribute('data-ui-skin')).toBe('next')
  })

  test('renders the next shell for a regular user when the site skin is next', () => {
    setUserRole(ROLE.USER)
    renderConsole('next')

    expect(screen.getByTestId('next-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('classic-shell')).toBeNull()
  })

  test('keeps an administrator on the classic shell when the site skin is next', () => {
    setUserRole(ROLE.ADMIN)
    renderConsole('next')

    expect(screen.getByTestId('classic-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('next-shell')).toBeNull()
    expect(document.body.getAttribute('data-ui-skin')).toBe('next')
  })

  test('renders the next shell when an administrator previews the user console', () => {
    setUserRole(ROLE.ADMIN, 8)
    useConsolePreviewStore.getState().enterPreview(8)
    renderConsole('next')

    expect(screen.getByTestId('next-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('classic-shell')).toBeNull()
    expect(document.body.getAttribute('data-ui-skin')).toBe('next')
  })

  test('ignores a preview saved for a different user', () => {
    setUserRole(ROLE.ADMIN, 8)
    useConsolePreviewStore.getState().enterPreview(99)
    renderConsole('next')

    expect(screen.getByTestId('classic-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('next-shell')).toBeNull()
  })

  test('keeps classic when the site skin is classic even if preview is on', () => {
    setUserRole(ROLE.ADMIN, 8)
    useConsolePreviewStore.getState().enterPreview(8)
    renderConsole('classic')

    expect(screen.getByTestId('classic-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('next-shell')).toBeNull()
  })

  test('renders the classic shell while the status response is unavailable', () => {
    renderConsole(undefined)

    expect(screen.getByTestId('classic-shell')).toBeInTheDocument()
    expect(screen.queryByTestId('next-shell')).toBeNull()
    expect(document.body.getAttribute('data-ui-skin')).toBe('classic')
  })

  test('clears the body marker once the console unmounts', () => {
    const view = renderConsole('next')
    expect(document.body.getAttribute('data-ui-skin')).toBe('next')

    view.unmount()

    expect(document.body.hasAttribute('data-ui-skin')).toBe(false)
  })
})
