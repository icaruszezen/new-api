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
import { beforeEach, describe, expect, test, vi } from 'vitest'

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

vi.mock('../components/minimal-footer', () => ({
  MinimalFooter: () => <footer data-testid='footer' />,
}))

vi.mock('../components/minimal-header', () => ({
  MinimalHeader: () => <header data-testid='next-header' />,
}))

vi.mock('../components/hero', () => ({
  Hero: () => <div data-testid='next-hero' />,
}))

vi.mock('../components/provider-strip', () => ({
  ProviderStrip: () => <div data-testid='next-provider-strip' />,
}))

describe('next landing atmosphere', () => {
  beforeEach(() => {
    homePageContent.current = { content: '', isLoaded: true, isUrl: false }
  })

  test('paints a decorative colour wash behind the default landing page', () => {
    render(<NextHome />)

    const atmosphere = screen.getByTestId('landing-atmosphere')
    expect(atmosphere).toBeInTheDocument()
    expect(atmosphere).toHaveAttribute('aria-hidden')
  })

  test('omits the wash when the administrator configured a custom home page', () => {
    homePageContent.current = {
      content: '# Welcome',
      isLoaded: true,
      isUrl: false,
    }

    render(<NextHome />)

    expect(screen.queryByTestId('landing-atmosphere')).toBeNull()
  })

  test('omits the wash while the home page content request is still loading', () => {
    homePageContent.current = { content: '', isLoaded: false, isUrl: false }

    render(<NextHome />)

    expect(screen.queryByTestId('landing-atmosphere')).toBeNull()
  })
})
