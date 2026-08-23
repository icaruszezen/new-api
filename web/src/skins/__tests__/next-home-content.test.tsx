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

import { NextHome } from '../next/home'

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

vi.mock('../next/home/components/minimal-footer', () => ({
  MinimalFooter: () => <footer data-testid='footer' />,
}))

vi.mock('../next/home/components/minimal-header', () => ({
  MinimalHeader: () => <header data-testid='next-header' />,
}))

vi.mock('../next/home/components/hero', () => ({
  Hero: () => <div data-testid='next-hero' />,
}))

vi.mock('../next/home/components/provider-strip', () => ({
  ProviderStrip: () => <div data-testid='next-provider-strip' />,
}))

describe('next landing page and custom home page content', () => {
  beforeEach(() => {
    homePageContent.current = { content: '', isLoaded: true, isUrl: false }
  })

  test('renders the next landing page when no custom home page is configured', () => {
    render(<NextHome />)

    expect(screen.getByTestId('next-header')).toBeInTheDocument()
    expect(screen.getByTestId('next-hero')).toBeInTheDocument()
    expect(screen.getByTestId('next-provider-strip')).toBeInTheDocument()
    expect(screen.getByTestId('footer')).toBeInTheDocument()
    expect(screen.queryByTestId('classic-home')).toBeNull()
  })

  test('defers to the classic rendering when the administrator configured a custom home page', () => {
    homePageContent.current = {
      content: '# Welcome',
      isLoaded: true,
      isUrl: false,
    }

    render(<NextHome />)

    expect(screen.getByTestId('classic-home')).toBeInTheDocument()
    expect(screen.queryByTestId('next-hero')).toBeNull()
  })

  test('defers to the classic rendering when the custom home page is an external URL', () => {
    homePageContent.current = {
      content: 'https://example.com/landing',
      isLoaded: true,
      isUrl: true,
    }

    render(<NextHome />)

    expect(screen.getByTestId('classic-home')).toBeInTheDocument()
    expect(screen.queryByTestId('next-hero')).toBeNull()
  })

  test('shows a loading state until the home page content request settles', () => {
    homePageContent.current = { content: '', isLoaded: false, isUrl: false }

    render(<NextHome />)

    expect(screen.getByText('Loading...')).toBeInTheDocument()
    expect(screen.queryByTestId('next-hero')).toBeNull()
    expect(screen.queryByTestId('classic-home')).toBeNull()
  })
})
