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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { createElement, type ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  getChannelMonitoringStatus: vi.fn(),
}))

vi.mock('../api', () => ({
  getChannelMonitoringStatus: apiMocks.getChannelMonitoringStatus,
}))

const { useMonitoringStatus } = await import('../hooks/use-monitoring-status')

function rateLimitedError(retryAfterSeconds: string) {
  return {
    response: {
      status: 429,
      headers: { 'retry-after': retryAfterSeconds },
    },
  }
}

function renderStatusHook() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return renderHook(() => useMonitoringStatus(), {
    wrapper: ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client }, children),
  })
}

describe('useMonitoringStatus Retry-After backoff', () => {
  let now = 0

  beforeEach(() => {
    now = Date.parse('2026-09-07T00:00:00.000Z')
    vi.spyOn(Date, 'now').mockImplementation(() => now)
    apiMocks.getChannelMonitoringStatus.mockReset()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  test('does not request again until Retry-After elapses, including on window focus', async () => {
    apiMocks.getChannelMonitoringStatus
      .mockRejectedValueOnce(rateLimitedError('5'))
      .mockResolvedValue({
        enabled: true,
        sample_window_seconds: 20,
        beat_limit: 60,
        monitors: [],
      })

    const { result } = renderStatusHook()

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })
    expect(apiMocks.getChannelMonitoringStatus).toHaveBeenCalledTimes(1)
    expect(result.current.backoffSeconds).toBe(5)

    await result.current.refetch()
    expect(apiMocks.getChannelMonitoringStatus).toHaveBeenCalledTimes(1)

    window.dispatchEvent(new Event('focus'))
    await waitFor(() => {
      expect(apiMocks.getChannelMonitoringStatus).toHaveBeenCalledTimes(1)
    })

    now += 6_000
    await result.current.refetch()

    await waitFor(() => {
      expect(result.current.isError).toBe(false)
    })
    expect(apiMocks.getChannelMonitoringStatus).toHaveBeenCalledTimes(2)
  })
})
