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

import { BEAT_STATUS_UP, type MonitorView } from '../types'

// `@lobehub/icons` cannot be resolved under vitest: it pulls in
// `@lobehub/fluent-emoji`, whose ESM entry uses an unsupported directory
// import. Only the external icon boundary is stubbed; the card itself is real.
vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: (iconName?: string | null) =>
    iconName ? <span data-testid={`monitor-icon-${iconName}`} /> : null,
}))

const { MonitorCard } = await import('../components/monitor-card')

function monitor(overrides: Partial<MonitorView> = {}): MonitorView {
  return {
    id: 'm1',
    name: 'ChatGPT-Pro',
    model: 'gpt-5-sol',
    icon: 'OpenAI',
    status: 'up',
    avg_ttft_ms: 3201,
    ping_ms: 23,
    uptime: 91.1543,
    beats: [{ ts: 1_700_000_000, status: BEAT_STATUS_UP, ttft_ms: 3201 }],
    ...overrides,
  }
}

function renderCard(
  overrides: Partial<MonitorView> = {},
  extras: { slots?: number; countdown?: number } = {}
) {
  return render(
    <MonitorCard
      monitor={monitor(overrides)}
      slots={extras.slots ?? 60}
      countdown={extras.countdown}
    />
  )
}

describe('MonitorCard', () => {
  test('shows the display name, model and both latency metrics', () => {
    renderCard()

    expect(screen.getByText('ChatGPT-Pro')).toBeInTheDocument()
    expect(screen.getByText('gpt-5-sol')).toBeInTheDocument()
    expect(screen.getByText('Chat latency')).toBeInTheDocument()
    expect(screen.getByText('3201')).toBeInTheDocument()
    expect(screen.getByText('Endpoint ping')).toBeInTheDocument()
    expect(screen.getByText('23')).toBeInTheDocument()
  })

  test('renders uptime with two decimals so small regressions stay visible', () => {
    renderCard({ uptime: 91.1543 })

    expect(screen.getByText('91.15%')).toBeInTheDocument()
  })

  test('labels availability separately and puts the record count on the beat bar', () => {
    renderCard({}, { slots: 30 })

    expect(screen.getByText('Availability')).toBeInTheDocument()
    expect(screen.getByText('Recent 30 records')).toBeInTheDocument()
    expect(screen.queryByText(/last 30 records|7 days/)).not.toBeInTheDocument()
  })

  test('shows the refresh countdown next to the record count when provided', () => {
    renderCard({}, { countdown: 3 })

    const caption = screen.getByText('Recent 60 records')
    const countdown = screen.getByText('Refreshing in 3s')
    expect(countdown.parentElement).toBe(caption.parentElement)
  })

  test('keeps the avatar circular and the name truncating beside a two-column metric row', () => {
    renderCard()

    const avatarIcon = screen.getAllByTestId('monitor-icon-OpenAI')[0]
    expect(avatarIcon.parentElement).toHaveClass('size-10')
    expect(avatarIcon.parentElement).toHaveClass('rounded-full')
    expect(screen.getByText('ChatGPT-Pro')).toHaveClass('truncate')

    const metricRow = screen.getByText('Chat latency').closest('.grid')
    expect(metricRow).toHaveClass('grid-cols-2')
    expect(metricRow?.childElementCount).toBe(2)

    const caption = screen.getByText('Recent 60 records')
    const bar = screen.getByRole('img')
    const follows =
      caption.compareDocumentPosition(bar) & Node.DOCUMENT_POSITION_FOLLOWING
    expect(follows).toBe(Node.DOCUMENT_POSITION_FOLLOWING)
  })

  test('falls back to a dash for metrics that have no measurement yet', () => {
    renderCard({ avg_ttft_ms: 0, ping_ms: 0, uptime: 0 })

    // Three placeholders: chat latency, endpoint ping and uptime.
    expect(screen.getAllByText('--')).toHaveLength(3)
  })

  test.each([
    ['up' as const, 'Normal'],
    ['degraded' as const, 'Degraded'],
    ['down' as const, 'Abnormal'],
    ['unknown' as const, 'No data'],
  ])('maps the %s status to the %s badge', (status, label) => {
    renderCard({ status })

    expect(screen.getByText(label)).toBeInTheDocument()
  })
})
