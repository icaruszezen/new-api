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
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import type { SystemStatus } from '@/features/auth/types'

import { Hero } from '../next/home/components/hero'

const statusRef = vi.hoisted(() => ({ current: null as SystemStatus | null }))

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

describe('next landing hero', () => {
  beforeEach(() => {
    statusRef.current = null
  })

  test.each([
    ['https://api.example.com', 'https://api.example.com'],
    ['https://api.example.com/', 'https://api.example.com'],
    ['  https://api.example.com///  ', 'https://api.example.com'],
  ])(
    'builds the endpoint from the configured server address %s',
    (serverAddress, expectedBase) => {
      statusRef.current = { server_address: serverAddress }

      render(<Hero isAuthenticated={false} />)

      expect(
        screen.getByLabelText('Chat completions endpoint')
      ).toHaveTextContent(`${expectedBase}/v1/chat/completions`)
      expect(screen.getByText('POST')).toBeInTheDocument()
    }
  )

  test.each([
    ['no status response', null],
    ['an unset server address', {} as SystemStatus],
    ['a blank server address', { server_address: '   ' } as SystemStatus],
  ])('falls back to the current origin with %s', (_label, status) => {
    statusRef.current = status

    render(<Hero isAuthenticated={false} />)

    expect(
      screen.getByLabelText('Chat completions endpoint')
    ).toHaveTextContent(`${window.location.origin}/v1/chat/completions`)
  })

  test('renders the headline in plain type without the accent sheen', () => {
    render(<Hero isAuthenticated={false} />)

    expect(screen.getByText('One API.')).not.toHaveClass(
      'landing-animate-headline'
    )
    expect(screen.getByText('Every model.')).toHaveClass(
      'text-muted-foreground'
    )
  })

  test('sends signed-out visitors to sign-up and signed-in visitors to the dashboard', () => {
    const { unmount } = render(<Hero isAuthenticated={false} />)
    expect(screen.getByRole('link', { name: 'Get Started' })).toHaveAttribute(
      'href',
      '/sign-up'
    )
    unmount()

    render(<Hero isAuthenticated />)
    expect(
      screen.getByRole('link', { name: 'Go to Dashboard' })
    ).toHaveAttribute('href', '/dashboard')
  })
})
