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
import { describe, expect, test } from 'vitest'

import type { ApiInfoItem } from '@/features/dashboard/types'

import { ConsoleEndpointChips } from '../components/endpoint-chips'

function endpoint(url: string, extras: Partial<ApiInfoItem> = {}): ApiInfoItem {
  return {
    url,
    route: extras.route ?? 'default',
    description: extras.description ?? '',
    color: extras.color ?? '',
  }
}

describe('next console endpoint chips', () => {
  test('renders nothing when there are no API endpoints', () => {
    const { container } = render(<ConsoleEndpointChips items={[]} />)

    expect(
      container.querySelector('[data-slot="console-endpoint-chips"]')
    ).toBeNull()
  })

  test('gives each endpoint chip its own tone and keeps labels on one line', () => {
    render(
      <ConsoleEndpointChips
        items={[
          endpoint('https://api.example.com/v1', { description: 'Default' }),
          endpoint('https://relay.example.com/v1', {
            description: 'Relay',
            color: 'purple',
          }),
          endpoint('https://openai.example.com', { description: 'OpenAI' }),
        ]}
      />
    )

    const chips = screen.getAllByText(/example.com/).map((node) => {
      const chip = node.closest('[data-slot="console-endpoint-chip"]')
      expect(chip).not.toBeNull()
      return chip as HTMLElement
    })

    expect(chips).toHaveLength(3)
    const tones = chips.map((chip) => chip.getAttribute('data-tone'))
    expect(new Set(tones).size).toBe(3)
    expect(chips[1]).toHaveAttribute('data-tone', 'violet')

    expect(screen.getByText('Default')).toHaveClass('whitespace-nowrap')
    expect(screen.getByText('Relay')).toHaveClass('whitespace-nowrap')
    expect(chips[0]?.className).not.toBe(chips[1]?.className)
    expect(chips[0]?.className).not.toBe(chips[2]?.className)
  })

  test('keeps a long endpoint URL truncatable inside the chip', () => {
    const longUrl =
      'https://very-long-api-gateway.example.com/v1/chat/completions'
    render(
      <ConsoleEndpointChips
        items={[endpoint(longUrl, { description: 'Gateway' })]}
      />
    )

    const url = screen.getByText(longUrl)
    expect(url).toHaveClass('truncate')
    expect(url).toHaveClass('min-w-0')
    expect(url.closest('[data-slot="console-endpoint-chip"]')).toHaveClass(
      'min-w-0'
    )
  })
})
