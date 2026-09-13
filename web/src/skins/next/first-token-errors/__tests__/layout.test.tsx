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

vi.mock('@/features/first-token-errors/components/settings-bar', () => ({
  FirstTokenErrorSettingsBar: () => (
    <div data-slot='first-token-errors-settings'>settings</div>
  ),
}))

vi.mock('@/features/first-token-errors/components/table', () => ({
  FirstTokenErrorTable: () => (
    <div data-slot='first-token-errors-table'>first-token errors table</div>
  ),
}))

const { NextFirstTokenErrors } = await import('../index')

describe('next first-token errors layout', () => {
  test('renders the next page chrome without sidebar or classic layout', () => {
    render(<NextFirstTokenErrors />)

    const page = document.querySelector('[data-slot="next-first-token-errors"]')
    expect(page).toBeInTheDocument()
    expect(page).toHaveClass('flex')
    expect(page).toHaveClass('flex-col')
    expect(page).toHaveClass('min-h-0')
    expect(page).toHaveClass('flex-1')
    expect(page).toHaveClass('w-full')
    expect(page).toHaveClass('h-full')
    expect(page).not.toHaveClass('max-w-7xl')
    expect(page).not.toHaveClass('px-6')
    expect(page).toHaveClass('px-3')
    expect(page).toHaveClass('sm:px-4')

    const heading = screen.getByRole('heading', { name: 'First-token errors' })
    expect(heading.tagName).toBe('H1')
    expect(heading).toHaveClass('text-3xl')
    expect(heading).toHaveClass('font-medium')

    const card = document.querySelector(
      '[data-slot="next-first-token-errors-card"]'
    )
    expect(card).toBeInTheDocument()
    expect(card).toHaveClass('rounded-xl')
    expect(card).toHaveClass('border')
    expect(card).toContainElement(
      document.querySelector('[data-slot="first-token-errors-table"]')
    )
    expect(
      document.querySelector('[data-slot="next-first-token-errors-footer"]')
    ).toBeInTheDocument()

    expect(document.querySelector('main')).toBeNull()
    expect(document.querySelector('[data-slot="sidebar"]')).toBeNull()
    expect(document.querySelector('h2')).toBeNull()
  })
})
