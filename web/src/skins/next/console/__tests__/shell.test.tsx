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

vi.mock('@tanstack/react-router', () => ({
  Outlet: () => <div data-testid='console-outlet' />,
}))

vi.mock('../../home/components/minimal-header', () => ({
  MinimalHeader: () => <header>console header</header>,
}))

const { NextConsoleShell } = await import('../shell')

describe('next console shell', () => {
  afterEach(() => {
    document.body.removeAttribute('data-next-console')
  })

  test('marks the page with next tokens and offsets a preview banner', () => {
    const view = render(<NextConsoleShell previewOffset />)

    expect(document.body.hasAttribute('data-next-console')).toBe(true)
    expect(screen.getByRole('main')).toHaveAttribute('id', 'content')
    expect(screen.getByRole('main')).toHaveClass('max-w-7xl')
    expect(view.container.firstElementChild).toHaveAttribute(
      'data-skin',
      'next'
    )
    expect(view.container.firstElementChild).toHaveClass('pt-9')
    expect(view.container.firstElementChild).toHaveClass('[&_header]:top-9')

    view.unmount()
    expect(document.body.hasAttribute('data-next-console')).toBe(false)
  })
})
