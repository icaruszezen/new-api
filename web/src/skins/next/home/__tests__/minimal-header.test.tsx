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
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import type { TopNavLink } from '@/hooks/use-top-nav-links'

import { MinimalHeader } from '../components/minimal-header'

const routerStateRef = vi.hoisted(() => ({ pathname: '/' }))

const NAV_LINKS: TopNavLink[] = [
  { title: 'Home', href: '/' },
  { title: 'Model Square', href: '/pricing' },
  { title: 'Docs', href: '/docs' },
]

vi.mock('@tanstack/react-router', () => ({
  Link: (props: {
    to: string
    children: ReactNode
    className?: string
    onClick?: (event: { preventDefault: () => void }) => void
  }) => (
    <a href={props.to} className={props.className} onClick={props.onClick}>
      {props.children}
    </a>
  ),
  useNavigate: () => vi.fn(),
  useRouterState: () => ({ location: { pathname: routerStateRef.pathname } }),
}))

vi.mock('@/stores/auth-store', () => ({
  useAuthStore: () => ({ auth: { user: null } }),
}))

vi.mock('@/hooks/use-system-config', () => ({
  useSystemConfig: () => ({
    systemName: 'AIGC Pro',
    logo: '',
    loading: false,
    logoLoaded: true,
    footerHtml: undefined,
  }),
}))

vi.mock('@/hooks/use-top-nav-links', () => ({
  useTopNavLinks: () => NAV_LINKS,
}))

vi.mock('@/hooks/use-notifications', () => ({
  useNotifications: () => ({
    popoverOpen: false,
    setPopoverOpen: () => undefined,
    unreadCount: 0,
    activeTab: 'notice',
    setActiveTab: () => undefined,
    notice: '',
    announcements: [],
    loading: false,
  }),
}))

vi.mock('@/components/language-switcher', () => ({
  LanguageSwitcher: () => null,
}))

vi.mock('@/components/theme-switch', () => ({
  ThemeSwitch: () => null,
}))

vi.mock('@/components/notification-popover', () => ({
  NotificationPopover: () => null,
}))

vi.mock('@/components/profile-dropdown', () => ({
  ProfileDropdown: () => null,
}))

describe('next public header', () => {
  beforeEach(() => {
    routerStateRef.pathname = '/'
  })

  test('places the logo on the left, nav links in the center, and sign-in on the right', () => {
    render(<MinimalHeader />)

    const nav = screen.getByRole('navigation')
    expect(nav).toHaveClass('grid-cols-[1fr_auto_1fr]')

    const columns = [...nav.children]
    expect(columns).toHaveLength(3)
    expect(columns[0]).toHaveTextContent('AIGC Pro')
    expect(columns[0]).toHaveClass('justify-self-start')
    expect(columns[1]).toHaveTextContent('Home')
    expect(columns[1]).toHaveTextContent('Model Square')
    expect(columns[1]).toHaveTextContent('Docs')
    expect(columns[2]).toHaveTextContent('Sign in')
    expect(columns[2]).toHaveClass('justify-end')
  })

  test('underlines the current public route', () => {
    routerStateRef.pathname = '/pricing'
    render(<MinimalHeader />)

    const modelSquare = screen.getByRole('link', { name: 'Model Square' })
    expect(modelSquare).toHaveClass('underline')
    expect(modelSquare).toHaveClass('text-foreground')

    expect(screen.getByRole('link', { name: 'Home' })).not.toHaveClass(
      'underline'
    )
  })

  test('opens the same nav links from the mobile menu', async () => {
    const user = userEvent.setup()
    render(<MinimalHeader />)

    await user.click(screen.getByRole('button', { name: 'Open menu' }))

    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByRole('link', { name: 'Home' })).toHaveAttribute(
      'href',
      '/'
    )
    expect(
      within(dialog).getByRole('link', { name: 'Model Square' })
    ).toHaveAttribute('href', '/pricing')
    expect(within(dialog).getByRole('link', { name: 'Docs' })).toHaveAttribute(
      'href',
      '/docs'
    )
  })
})
