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
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { AnimateInView } from '../animate-in-view'

const observed: Element[] = []

// Records what gets observed instead of reporting it visible, so a test can tell
// "revealed because it scrolled into view" apart from "revealed immediately".
class RecordingIntersectionObserver {
  constructor(private readonly callback: IntersectionObserverCallback) {}

  observe(element: Element) {
    observed.push(element)
  }

  reveal(element: Element) {
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

describe('AnimateInView reduced-motion handling', () => {
  beforeEach(() => {
    observed.length = 0
    vi.stubGlobal('IntersectionObserver', RecordingIntersectionObserver)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  test('reveals immediately without waiting for a scroll under reduced motion', () => {
    setReducedMotion(true)

    render(<AnimateInView>revealed</AnimateInView>)

    const target = screen.getByText('revealed')
    expect(observed).toHaveLength(0)
    expect(target).not.toHaveClass('opacity-0')
    expect(target).toHaveClass('landing-animate-fade-up')
  })

  test('keeps the scroll trigger under reduced motion when opted in', () => {
    setReducedMotion(true)

    render(<AnimateInView ignoreReducedMotion>revealed</AnimateInView>)

    const target = screen.getByText('revealed')
    expect(observed).toEqual([target])
    // Still hidden: the reveal waits for the element to scroll into view, which
    // is what keeps a below-the-fold section from animating out of sight.
    expect(target).toHaveClass('opacity-0')
    expect(target).not.toHaveClass('landing-animate-fade-up')
  })
})
