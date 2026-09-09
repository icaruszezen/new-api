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

vi.mock('@tanstack/react-router', () => ({
  Navigate: (props: { to: string; replace?: boolean }) => (
    <div data-testid='navigate' data-to={props.to} />
  ),
}))

vi.mock('../next/invite', () => ({
  NextInvite: () => <div data-testid='next-invite' />,
}))

const { SkinnedInvite } = await import('../invite')

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

describe('referral page skin dispatch', () => {
  afterEach(() => {
    setSiteSkin('classic')
    useConsolePreviewStore.getState().exitPreview()
    useAuthStore.getState().auth.reset()
  })

  test('renders the next referral page for a regular user', () => {
    setSiteSkin('next')
    setUser(ROLE.USER)
    render(<SkinnedInvite />)

    expect(screen.getByTestId('next-invite')).toBeInTheDocument()
    expect(screen.queryByTestId('navigate')).toBeNull()
  })

  // The referral page is a Next surface only; classic sessions must be sent to
  // the wallet, where the existing referral card lives.
  test('sends a classic session to the wallet', () => {
    setSiteSkin('classic')
    setUser(ROLE.USER)
    render(<SkinnedInvite />)

    expect(screen.getByTestId('navigate')).toHaveAttribute('data-to', '/wallet')
    expect(screen.queryByTestId('next-invite')).toBeNull()
  })

  test('sends an administrator to the wallet by default', () => {
    setSiteSkin('next')
    setUser(ROLE.ADMIN, 8)
    render(<SkinnedInvite />)

    expect(screen.getByTestId('navigate')).toHaveAttribute('data-to', '/wallet')
    expect(screen.queryByTestId('next-invite')).toBeNull()
  })

  test('renders the next referral page when an administrator previews', () => {
    setSiteSkin('next')
    setUser(ROLE.ADMIN, 8)
    useConsolePreviewStore.getState().enterPreview(8)
    render(<SkinnedInvite />)

    expect(screen.getByTestId('next-invite')).toBeInTheDocument()
    expect(screen.queryByTestId('navigate')).toBeNull()
  })
})
