/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
    30|but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, test, vi } from 'vitest'

vi.mock('../../hooks/use-update-option', () => ({
  useUpdateOption: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
}))

vi.mock('../../api', () => ({
  getCurrentLogCleanupTask: vi.fn().mockResolvedValue({ success: true }),
  getSystemTask: vi.fn(),
  startLogCleanupTask: vi.fn(),
}))

vi.mock('@/lib/api', () => ({
  api: {
    get: vi.fn().mockResolvedValue({ data: { success: false } }),
    delete: vi.fn(),
  },
}))

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { to: string; className?: string; children?: ReactNode }) => (
    <a href={props.to} className={props.className}>
      {props.children}
    </a>
  ),
}))

const { LogSettingsSection } = await import('../log-settings-section')

function renderSection() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <LogSettingsSection defaultEnabled={true} />
    </QueryClientProvider>
  )
}

describe('log settings first-token entry', () => {
  test('links to the first-token errors page instead of embedding billing switches', () => {
    renderSection()

    const link = screen.getByRole('link', {
      name: 'Open first-token error settings',
    })
    expect(link).toHaveAttribute('href', '/first-token-errors')
    expect(
      screen.queryByRole('switch', {
        name: /Treat every 1-token output as an upstream error/i,
      })
    ).toBeNull()
  })
})
