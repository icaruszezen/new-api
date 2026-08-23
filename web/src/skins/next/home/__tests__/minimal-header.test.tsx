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
const authUserRef = vi.hoisted(() => ({
  current: null as { username: string } | null,
}))

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
  useAuthStore: () => ({ auth: { user: authUserRef.current } }),
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
  LanguageSwitcher: () => <button type='button'>Change language</button>,
}))

vi.mock('@/components/theme-switch', () => ({
  ThemeSwitch: () => <button type='button'>Toggle theme</button>,
}))

vi.mock('@/components/notification-popover', () => ({
  NotificationPopover: () => <button type='button'>Notifications</button>,
}))

vi.mock('@/components/profile-dropdown', () => ({
  ProfileDropdown: () => <button type='button'>Account</button>,
}))

describe('next public header', () => {
  beforeEach(() => {
    routerStateRef.pathname = '/'
    authUserRef.current = null
  })

  test('places the logo on the left, nav links in the center, and sign-in on the right', () => {
    render(<MinimalHeader />)

    const nav = screen.getByRole('navigation')
    expect(nav).toHaveClass('flex')
    expect(nav).toHaveClass('justify-between')
    expect(nav).toHaveClass('sm:grid')
    expect(nav).toHaveClass('sm:grid-cols-[1fr_auto_1fr]')

    const columns = [...nav.children]
    expect(columns).toHaveLength(3)
    expect(columns[0]).toHaveTextContent('AIGC Pro')
    expect(columns[0]).toHaveClass('justify-self-start')
    expect(columns[1]).toHaveTextContent('Home')
    expect(columns[1]).toHaveTextContent('Model Square')
    expect(columns[1]).toHaveTextContent('Docs')
    expect(columns[2]).toHaveTextContent('Sign in')
    expect(columns[2]).toHaveClass('justify-end')
    expect(columns[2]).toHaveClass('gap-2')

    const brandName = within(columns[0] as HTMLElement).getByText('AIGC Pro')
    expect(brandName).toHaveClass('truncate')
    expect(brandName).toHaveClass('max-w-[40vw]')
  })

  test('keeps notifications and sign-in on the bar and puts the menu last on narrow viewports', () => {
    render(<MinimalHeader />)

    const actions = screen.getByRole('navigation').lastElementChild
    expect(actions).not.toBeNull()

    expect(
      within(actions as HTMLElement).getByRole('button', {
        name: 'Notifications',
      })
    ).toBeInTheDocument()
    expect(
      within(actions as HTMLElement).getByRole('link', { name: 'Sign in' })
    ).toBeInTheDocument()

    const menu = within(actions as HTMLElement).getByRole('button', {
      name: 'Open menu',
    })
    expect(actions?.lastElementChild).toBe(menu)
    expect(menu).toHaveClass('size-11')
    expect(menu).toHaveClass('sm:hidden')
    expect(menu).toHaveAttribute('aria-expanded', 'false')
  })

  test('hides language and theme from the bar on narrow viewports and keeps them in the menu', async () => {
    const user = userEvent.setup()
    render(<MinimalHeader />)

    const nav = screen.getByRole('navigation')
    const languageInBar = within(nav).getByRole('button', {
      name: 'Change language',
    })
    const themeInBar = within(nav).getByRole('button', {
      name: 'Toggle theme',
    })

    expect(languageInBar.parentElement).toHaveClass('hidden')
    expect(languageInBar.parentElement).toHaveClass('sm:flex')
    expect(themeInBar.parentElement).toBe(languageInBar.parentElement)

    await user.click(screen.getByRole('button', { name: 'Open menu' }))

    const dialog = await screen.findByRole('dialog')
    expect(
      within(dialog).getByRole('button', { name: 'Change language' })
    ).toBeInTheDocument()
    expect(
      within(dialog).getByRole('button', { name: 'Toggle theme' })
    ).toBeInTheDocument()
    expect(within(dialog).getByText('Preferences')).toBeInTheDocument()
  })

  test('keeps the signed-in account control on the bar instead of moving it into the menu', () => {
    authUserRef.current = { username: 'z' }
    render(<MinimalHeader />)

    const actions = screen.getByRole('navigation').lastElementChild
    expect(
      within(actions as HTMLElement).getByRole('button', { name: 'Account' })
    ).toBeInTheDocument()
    expect(
      within(actions as HTMLElement).queryByRole('link', { name: 'Sign in' })
    ).toBeNull()
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
