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

vi.mock('@/features/profile', () => ({
  Profile: () => <div data-testid='classic-profile' />,
}))

vi.mock('../next/profile', () => ({
  NextProfile: () => <div data-testid='next-profile' />,
}))

const { SkinnedProfile } = await import('../profile')

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

describe('profile skin dispatch', () => {
  afterEach(() => {
    setSiteSkin('classic')
    useConsolePreviewStore.getState().exitPreview()
    useAuthStore.getState().auth.reset()
  })

  test('renders the next personal center for a regular user', () => {
    setSiteSkin('next')
    setUser(ROLE.USER)
    render(<SkinnedProfile />)

    expect(screen.getByTestId('next-profile')).toBeInTheDocument()
    expect(screen.queryByTestId('classic-profile')).toBeNull()
  })

  test('keeps the classic personal center when the site skin is classic', () => {
    setSiteSkin('classic')
    setUser(ROLE.USER)
    render(<SkinnedProfile />)

    expect(screen.getByTestId('classic-profile')).toBeInTheDocument()
    expect(screen.queryByTestId('next-profile')).toBeNull()
  })

  test('keeps an administrator on the classic personal center unless they preview', () => {
    setSiteSkin('next')
    setUser(ROLE.ADMIN, 8)
    render(<SkinnedProfile />)

    expect(screen.getByTestId('classic-profile')).toBeInTheDocument()
    expect(screen.queryByTestId('next-profile')).toBeNull()
  })

  test('renders the next personal center when an administrator previews', () => {
    setSiteSkin('next')
    setUser(ROLE.ADMIN, 8)
    useConsolePreviewStore.getState().enterPreview(8)
    render(<SkinnedProfile />)

    expect(screen.getByTestId('next-profile')).toBeInTheDocument()
    expect(screen.queryByTestId('classic-profile')).toBeNull()
  })
})
