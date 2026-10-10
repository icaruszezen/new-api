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
import { act, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { HtmlPreview } from '../components/html-preview'

const PAGE = '<html><body><script>alert(1)</script><h1>Clock</h1></body></html>'

function getFrame(container: HTMLElement = document.body) {
  return within(container).getByTitle('HTML preview')
}

describe('HtmlPreview', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  test('renders the page in a frame that may only run scripts', () => {
    render(<HtmlPreview html={PAGE} complete />)

    const frame = getFrame()
    expect(frame.tagName).toBe('IFRAME')
    // Exactly allow-scripts: no same-origin, top navigation, popups, forms or
    // modals for untrusted model output.
    expect(frame.getAttribute('sandbox')).toBe('allow-scripts')
    expect(frame.getAttribute('srcdoc')).toBe(PAGE)
    expect(frame.getAttribute('referrerpolicy')).toBe('no-referrer')
    expect(screen.queryByText('Incomplete')).not.toBeInTheDocument()
  })

  test('flags a page that the answer did not finish', () => {
    render(<HtmlPreview html='<html><body>cut' complete={false} />)

    expect(screen.getByText('Incomplete')).toBeInTheDocument()
  })

  test('switches between the rendered page and its escaped source', async () => {
    const user = userEvent.setup()
    render(<HtmlPreview html={PAGE} complete />)

    await user.click(screen.getByRole('button', { name: 'View source' }))

    expect(screen.queryByTitle('HTML preview')).not.toBeInTheDocument()
    expect(screen.getByText(PAGE).tagName).toBe('PRE')
    expect(
      screen.getByRole('button', { name: 'View preview' })
    ).toHaveAttribute('aria-pressed', 'true')

    await user.click(screen.getByRole('button', { name: 'View preview' }))

    expect(getFrame()).toHaveAttribute('srcdoc', PAGE)
  })

  test('opens the page in a larger sandboxed frame', async () => {
    const user = userEvent.setup()
    render(<HtmlPreview html={PAGE} complete />)

    await user.click(screen.getByRole('button', { name: 'Open larger' }))

    const dialog = await screen.findByRole('dialog')
    const frame = getFrame(dialog)
    expect(frame).toHaveAttribute('sandbox', 'allow-scripts')
    expect(frame).toHaveAttribute('srcdoc', PAGE)
  })

  test('hides the enlarge action when already shown at reading size', () => {
    render(<HtmlPreview html={PAGE} complete expanded />)

    expect(
      screen.queryByRole('button', { name: 'Open larger' })
    ).not.toBeInTheDocument()
    expect(getFrame()).toHaveAttribute('srcdoc', PAGE)
  })

  test('waits until the box nears the viewport before creating the frame', () => {
    let reveal: (() => void) | undefined
    vi.stubGlobal(
      'IntersectionObserver',
      class {
        constructor(callback: IntersectionObserverCallback) {
          reveal = () =>
            callback(
              [{ isIntersecting: true } as IntersectionObserverEntry],
              this as unknown as IntersectionObserver
            )
        }
        observe() {}
        unobserve() {}
        disconnect() {}
      }
    )

    render(<HtmlPreview html={PAGE} complete />)

    expect(screen.queryByTitle('HTML preview')).not.toBeInTheDocument()

    act(() => reveal?.())

    expect(getFrame()).toHaveAttribute('srcdoc', PAGE)
  })
})
