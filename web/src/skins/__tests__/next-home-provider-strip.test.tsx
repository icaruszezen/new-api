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

import { ProviderStrip } from '../next/home/components/provider-strip'

// `@lobehub/icons` cannot be resolved under vitest: it pulls in
// `@lobehub/fluent-emoji`, whose ESM entry uses an unsupported directory
// import. Only the external icon boundary is stubbed; the strip itself is real.
vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: () => null,
}))

// jsdom has no IntersectionObserver, and the scroll-reveal wrapper needs one.
// Reporting every observed element as visible mirrors a strip that is in view.
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

describe('next landing provider strip', () => {
  beforeEach(() => {
    vi.stubGlobal('IntersectionObserver', IntersectionObserverStub)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  test('reveals every supported provider once the strip scrolls into view', () => {
    render(<ProviderStrip />)

    const strip = screen.getByRole('region', { name: 'Supported providers' })
    expect(strip).toBeInTheDocument()

    for (const name of [
      'OpenAI',
      'Anthropic',
      'Google',
      'Azure',
      'Bedrock',
      'DeepSeek',
    ]) {
      expect(screen.getByText(name)).toBeInTheDocument()
    }

    expect(screen.getAllByRole('listitem')).toHaveLength(6)
  })

  test('drops the pre-animation hidden state so the row stays visible', () => {
    render(<ProviderStrip />)

    for (const item of screen.getAllByRole('listitem')) {
      expect(item).not.toHaveClass('opacity-0')
    }
  })
})
