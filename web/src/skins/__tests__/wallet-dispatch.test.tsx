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

vi.mock('@/features/wallet', () => ({
  Wallet: () => <div data-testid='classic-wallet' />,
}))

vi.mock('../next/wallet', () => ({
  NextWallet: () => <div data-testid='next-wallet' />,
}))

const { SkinnedWallet } = await import('../wallet')

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

describe('wallet skin dispatch', () => {
  afterEach(() => {
    setSiteSkin('classic')
    useConsolePreviewStore.getState().exitPreview()
    useAuthStore.getState().auth.reset()
  })

  test('renders the next wallet for a regular user', () => {
    setSiteSkin('next')
    setUser(ROLE.USER)
    render(<SkinnedWallet />)

    expect(screen.getByTestId('next-wallet')).toBeInTheDocument()
    expect(screen.queryByTestId('classic-wallet')).toBeNull()
  })

  test('keeps the classic wallet when the site skin is classic', () => {
    setSiteSkin('classic')
    setUser(ROLE.USER)
    render(<SkinnedWallet />)

    expect(screen.getByTestId('classic-wallet')).toBeInTheDocument()
    expect(screen.queryByTestId('next-wallet')).toBeNull()
  })

  test('keeps an administrator on the classic wallet unless they preview', () => {
    setSiteSkin('next')
    setUser(ROLE.ADMIN, 8)
    render(<SkinnedWallet />)

    expect(screen.getByTestId('classic-wallet')).toBeInTheDocument()
    expect(screen.queryByTestId('next-wallet')).toBeNull()
  })

  test('renders the next wallet when an administrator previews', () => {
    setSiteSkin('next')
    setUser(ROLE.ADMIN, 8)
    useConsolePreviewStore.getState().enterPreview(8)
    render(<SkinnedWallet />)

    expect(screen.getByTestId('next-wallet')).toBeInTheDocument()
    expect(screen.queryByTestId('classic-wallet')).toBeNull()
  })
})
