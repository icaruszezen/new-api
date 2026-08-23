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
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { useConsolePreviewStore } from '../console-preview-store'
import { NextConsolePreviewBanner } from '../next/preview-banner'

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => vi.fn(),
}))

function setAdminOnNextSite(userId = 8) {
  useSystemConfigStore.setState({
    config: {
      ...useSystemConfigStore.getState().config,
      uiSkin: 'next',
    },
  })
  useAuthStore.getState().auth.setUser({
    id: userId,
    username: 'admin',
    role: ROLE.ADMIN,
  })
}

describe('next console preview banner', () => {
  afterEach(() => {
    useConsolePreviewStore.getState().exitPreview()
    useAuthStore.getState().auth.reset()
    useSystemConfigStore.setState({
      config: {
        ...useSystemConfigStore.getState().config,
        uiSkin: 'classic',
      },
    })
    sessionStorage.clear()
  })

  test('stays hidden for an administrator who is not previewing', () => {
    setAdminOnNextSite()

    render(<NextConsolePreviewBanner />)

    expect(screen.queryByRole('status')).toBeNull()
  })

  test('stays hidden for a regular user on the next console', () => {
    useSystemConfigStore.setState({
      config: {
        ...useSystemConfigStore.getState().config,
        uiSkin: 'next',
      },
    })
    useAuthStore.getState().auth.setUser({
      id: 2,
      username: 'member',
      role: ROLE.USER,
    })

    render(<NextConsolePreviewBanner />)

    expect(screen.queryByRole('status')).toBeNull()
  })

  test('shows the preview status and exits without changing the site skin', async () => {
    setAdminOnNextSite(8)
    useConsolePreviewStore.getState().enterPreview(8)

    render(<NextConsolePreviewBanner />)

    expect(screen.getByRole('status')).toHaveTextContent(
      'You are previewing the user console.'
    )

    await userEvent.click(screen.getByRole('button', { name: 'Exit preview' }))

    expect(screen.queryByRole('status')).toBeNull()
    expect(useConsolePreviewStore.getState().previewUserConsole).toBe(false)
    expect(useSystemConfigStore.getState().config.uiSkin).toBe('next')
  })
})
