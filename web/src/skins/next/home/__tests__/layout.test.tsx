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
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type { HomePageContentResult } from '@/features/home/types'

import { NextHome } from '../index'

const homePageContent = vi.hoisted(() => ({
  current: {
    content: '',
    isLoaded: true,
    isUrl: false,
  } as HomePageContentResult,
}))

vi.mock('@/features/home', () => ({
  Home: () => <div data-testid='classic-home' />,
}))

vi.mock('@/features/home/hooks', () => ({
  useHomePageContent: () => homePageContent.current,
}))

vi.mock('../components/hero', () => ({
  Hero: () => <div data-testid='next-hero' />,
}))

vi.mock('../components/provider-strip', () => ({
  ProviderStrip: () => <div data-testid='next-provider-strip' />,
}))

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { to: string; children: ReactNode; className?: string }) => (
    <a href={props.to} className={props.className}>
      {props.children}
    </a>
  ),
  useNavigate: () => vi.fn(),
  useRouterState: () => ({ location: { pathname: '/' } }),
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
  useTopNavLinks: () => [],
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

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: null, loading: false, error: null }),
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

class IntersectionObserverStub {
  constructor(private readonly callback: IntersectionObserverCallback) {}

  observe(element: Element) {
    this.callback(
      [{ isIntersecting: true, target: element } as IntersectionObserverEntry],
      this as unknown as IntersectionObserver
    )
  }

  unobserve() {}
  disconnect() {}
}

describe('next landing page measure', () => {
  beforeEach(() => {
    homePageContent.current = { content: '', isLoaded: true, isUrl: false }
    vi.stubGlobal('IntersectionObserver', IntersectionObserverStub)
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => undefined,
      removeListener: () => undefined,
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      dispatchEvent: () => false,
    }))
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  test('keeps the header, main column and footer on a 7xl measure instead of 5xl', () => {
    render(<NextHome />)

    const main = screen.getByRole('main')
    expect(main).toHaveClass('max-w-7xl')
    expect(main).not.toHaveClass('max-w-5xl')

    const nav = screen.getByRole('navigation')
    expect(nav).toHaveClass('max-w-7xl')
    expect(nav).not.toHaveClass('max-w-5xl')

    const footerMeasure = screen
      .getByRole('contentinfo')
      .querySelector('[class*="max-w-"]')
    expect(footerMeasure).toHaveClass('max-w-7xl')
    expect(footerMeasure).not.toHaveClass('max-w-5xl')
  })
})
