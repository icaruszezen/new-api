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

import { BEAT_STATUS_UP } from '../types'

const statusMocks = vi.hoisted(() => ({
  useMonitoringStatus: vi.fn(),
  useRefreshCountdown: vi.fn(),
}))

vi.mock('../hooks/use-monitoring-status', () => ({
  useMonitoringStatus: statusMocks.useMonitoringStatus,
  useRefreshCountdown: statusMocks.useRefreshCountdown,
}))

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: (iconName?: string | null) =>
    iconName ? <span data-testid={`monitor-icon-${iconName}`} /> : null,
}))

const { ChannelMonitoring } = await import('../index')

describe('ChannelMonitoring page', () => {
  test('hides the page title and shows the refresh countdown on each card', () => {
    statusMocks.useRefreshCountdown.mockReturnValue(42)
    statusMocks.useMonitoringStatus.mockReturnValue({
      data: {
        enabled: true,
        sample_window_seconds: 20,
        beat_limit: 60,
        monitors: [
          {
            id: 'm1',
            name: 'ChatGPT-Pro',
            model: 'gpt-5-sol',
            icon: 'OpenAI',
            status: 'up',
            avg_ttft_ms: 3201,
            ping_ms: 196,
            uptime: 100,
            beats: [
              { ts: 1_700_000_000, status: BEAT_STATUS_UP, ttft_ms: 3201 },
            ],
          },
        ],
      },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
      dataUpdatedAt: Date.now(),
    })

    render(<ChannelMonitoring />)

    const title = screen.getByRole('heading', { name: 'Channel Monitoring' })
    expect(title).toHaveClass('sr-only')
    expect(screen.getByText('ChatGPT-Pro')).toBeInTheDocument()

    const caption = screen.getByText('Recent 60 records')
    const countdown = screen.getByText('Refreshing in 42s')
    expect(countdown.parentElement).toBe(caption.parentElement)
    expect(
      screen.queryByText(
        'Live availability and latency of upstream model endpoints, sampled from real traffic.'
      )
    ).not.toBeInTheDocument()
  })

  test('pins the refresh countdown to the top right when no monitors exist', () => {
    statusMocks.useRefreshCountdown.mockReturnValue(42)
    statusMocks.useMonitoringStatus.mockReturnValue({
      data: {
        enabled: true,
        sample_window_seconds: 20,
        beat_limit: 60,
        monitors: [],
      },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
      dataUpdatedAt: Date.now(),
    })

    render(<ChannelMonitoring />)

    const countdown = screen.getByText('Refreshing in 42s')
    expect(countdown.parentElement).toHaveClass('justify-end')
    expect(
      screen.getByText('No monitors have been configured yet.')
    ).toBeInTheDocument()
  })

  test('keeps the last successful cards when a later refetch fails', () => {
    statusMocks.useRefreshCountdown.mockReturnValue(12)
    statusMocks.useMonitoringStatus.mockReturnValue({
      data: {
        enabled: true,
        sample_window_seconds: 20,
        beat_limit: 60,
        monitors: [
          {
            id: 'm1',
            name: 'ChatGPT-Pro',
            model: 'gpt-5-sol',
            icon: 'OpenAI',
            status: 'up',
            avg_ttft_ms: 3201,
            ping_ms: 196,
            uptime: 100,
            beats: [
              { ts: 1_700_000_000, status: BEAT_STATUS_UP, ttft_ms: 3201 },
            ],
          },
        ],
      },
      isLoading: false,
      isError: true,
      refetch: vi.fn(),
      dataUpdatedAt: Date.now(),
      backoffSeconds: 12,
    })

    render(<ChannelMonitoring />)

    expect(screen.getByText('ChatGPT-Pro')).toBeInTheDocument()
    expect(
      screen.queryByText('Failed to load channel monitoring data.')
    ).not.toBeInTheDocument()
  })

  test('disables retry while a Retry-After backoff is active', () => {
    statusMocks.useRefreshCountdown.mockReturnValue(0)
    statusMocks.useMonitoringStatus.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      refetch: vi.fn(),
      dataUpdatedAt: 0,
      backoffSeconds: 12,
    })

    render(<ChannelMonitoring />)

    expect(
      screen.getByRole('button', { name: 'Retry in 12s' })
    ).toBeDisabled()
  })
})
