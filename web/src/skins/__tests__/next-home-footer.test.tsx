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
import { act, fireEvent, render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type { SystemStatus } from '@/features/auth/types'

import { MinimalFooter } from '../next/home/components/minimal-footer'

const statusRef = vi.hoisted(() => ({ current: null as SystemStatus | null }))
const systemConfigRef = vi.hoisted(() => ({
  current: { systemName: 'AIGC Pro', footerHtml: undefined } as {
    systemName: string
    footerHtml?: string
  },
}))

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { to: string; children: ReactNode; className?: string }) => (
    <a href={props.to} className={props.className}>
      {props.children}
    </a>
  ),
}))

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: statusRef.current, loading: false, error: null }),
}))

vi.mock('@/hooks/use-system-config', () => ({
  useSystemConfig: () => systemConfigRef.current,
}))

// jsdom has no IntersectionObserver, and the footer's scroll-reveal wrapper
// needs one. Reporting every element as visible mirrors a footer in view.
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

function setReducedMotion(reduce: boolean) {
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: reduce,
    media: query,
    onchange: null,
    addListener: () => undefined,
    removeListener: () => undefined,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    dispatchEvent: () => false,
  }))
}

const SITE_NOTICE = 'footer-notice-site'
const PROJECT_NOTICE = 'footer-notice-project'

describe('next landing footer notices', () => {
  beforeEach(() => {
    statusRef.current = null
    systemConfigRef.current = { systemName: 'AIGC Pro', footerHtml: undefined }
    setReducedMotion(false)
    vi.stubGlobal('IntersectionObserver', IntersectionObserverStub)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  test('keeps both the site copyright and the project attribution in the document', () => {
    render(<MinimalFooter />)

    expect(screen.getByTestId(SITE_NOTICE)).toHaveTextContent('AIGC Pro')
    expect(screen.getByTestId(PROJECT_NOTICE)).toHaveTextContent('New API')
    expect(screen.getByRole('link', { name: 'New API' })).toHaveAttribute(
      'href',
      'https://github.com/QuantumNous/new-api'
    )
  })

  test('hands the slot over from the site copyright to the project attribution', () => {
    vi.useFakeTimers()
    render(<MinimalFooter />)

    const site = screen.getByTestId(SITE_NOTICE)
    const project = screen.getByTestId(PROJECT_NOTICE)

    // The waiting notice sits below its resting place and rises into it.
    expect(site).toHaveClass('opacity-100', 'translate-y-0')
    expect(project).toHaveClass('opacity-0', 'translate-y-1.5')

    act(() => {
      vi.advanceTimersByTime(6000)
    })

    expect(site).toHaveClass('opacity-0', 'translate-y-1.5')
    expect(project).toHaveClass('opacity-100', 'translate-y-0')

    act(() => {
      vi.advanceTimersByTime(6000)
    })

    expect(site).toHaveClass('opacity-100', 'translate-y-0')
    expect(project).toHaveClass('opacity-0', 'translate-y-1.5')
  })

  test('holds the current notice while the pointer rests on the slot', () => {
    vi.useFakeTimers()
    render(<MinimalFooter />)

    const site = screen.getByTestId(SITE_NOTICE)
    const slot = site.parentElement as HTMLElement

    fireEvent.mouseEnter(slot)
    act(() => {
      vi.advanceTimersByTime(18000)
    })
    expect(site).toHaveClass('opacity-100')

    fireEvent.mouseLeave(slot)
    act(() => {
      vi.advanceTimersByTime(6000)
    })
    expect(site).toHaveClass('opacity-0')
  })

  test('keeps rotating for a visitor who asked for reduced motion', () => {
    setReducedMotion(true)
    vi.useFakeTimers()
    render(<MinimalFooter />)

    const site = screen.getByTestId(SITE_NOTICE)
    const project = screen.getByTestId(PROJECT_NOTICE)

    expect(site).toHaveClass('opacity-100')
    expect(project).toHaveClass('opacity-0')

    act(() => {
      vi.advanceTimersByTime(6000)
    })

    expect(site).toHaveClass('opacity-0')
    expect(project).toHaveClass('opacity-100')
  })

  test('renders the legal links only for the pages the administrator enabled', () => {
    statusRef.current = { user_agreement_enabled: true }

    render(<MinimalFooter />)

    expect(
      screen.getByRole('link', { name: 'User Agreement' })
    ).toHaveAttribute('href', '/user-agreement')
    expect(screen.queryByRole('link', { name: 'Privacy Policy' })).toBeNull()
  })
})
