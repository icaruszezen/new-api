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
import { describe, expect, test, vi } from 'vitest'

vi.mock('../home/components/minimal-header', () => ({
  MinimalHeader: () => <header data-testid='next-header' />,
}))

vi.mock('../home/components/minimal-footer', () => ({
  MinimalFooter: () => <footer data-testid='next-footer' />,
}))

vi.mock('../components/ticket-fab', () => ({
  NextTicketFab: () => <div data-testid='ticket-fab' />,
}))

const { NextPublicShell } = await import('../public-shell')

describe('next public shell', () => {
  test('mounts the support ticket button inside the Next token scope', () => {
    render(
      <NextPublicShell>
        <p>landing</p>
      </NextPublicShell>
    )

    // Inside `data-skin="next"` the button picks up the Next palette.
    expect(
      screen.getByTestId('ticket-fab').closest('[data-skin]')
    ).toHaveAttribute('data-skin', 'next')
  })
})
