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
import { useQuery } from '@tanstack/react-query'
import { useCallback, useEffect, useRef, useState } from 'react'

import { getChannelMonitoringStatus } from '../api'
import {
  isRateLimitedError,
  parseRetryAfterMs,
} from '../lib/retry-after'

/** How often the status page pulls fresh beats. */
export const MONITORING_REFRESH_MS = 60_000

const BACKOFF_ERROR_NAME = 'MonitoringBackoffError'

function isMonitoringBackoffError(error: unknown): boolean {
  return error instanceof Error && error.name === BACKOFF_ERROR_NAME
}

function throwBackoffError(): never {
  const error = new Error('Channel monitoring status is backing off')
  error.name = BACKOFF_ERROR_NAME
  throw error
}

function useSecondsUntil(deadlineMs: number | undefined) {
  const [secondsLeft, setSecondsLeft] = useState(0)

  useEffect(() => {
    if (!deadlineMs) {
      setSecondsLeft(0)
      return
    }

    const tick = () => {
      setSecondsLeft(Math.max(0, Math.ceil((deadlineMs - Date.now()) / 1000)))
    }

    tick()
    const timer = window.setInterval(tick, 1000)
    return () => window.clearInterval(timer)
  }, [deadlineMs])

  return secondsLeft
}

export function useMonitoringStatus() {
  const backoffUntilRef = useRef(0)
  const [backoffUntil, setBackoffUntil] = useState(0)
  const backoffSeconds = useSecondsUntil(backoffUntil || undefined)

  const query = useQuery({
    queryKey: ['channel-monitoring-status'],
    queryFn: async () => {
      if (Date.now() < backoffUntilRef.current) {
        throwBackoffError()
      }
      try {
        const data = await getChannelMonitoringStatus()
        if (backoffUntilRef.current !== 0) {
          backoffUntilRef.current = 0
          setBackoffUntil(0)
        }
        return data
      } catch (error) {
        if (isRateLimitedError(error)) {
          const until = Date.now() + parseRetryAfterMs(error)
          backoffUntilRef.current = until
          setBackoffUntil(until)
        }
        throw error
      }
    },
    retry: (failureCount, error) => {
      if (isRateLimitedError(error) || isMonitoringBackoffError(error)) {
        return false
      }
      return failureCount < 3
    },
    refetchInterval: (current) => {
      const remaining = backoffUntilRef.current - Date.now()
      if (remaining > 0) return remaining
      if (
        current.state.status === 'error' &&
        !isRateLimitedError(current.state.error)
      ) {
        return false
      }
      return MONITORING_REFRESH_MS
    },
    refetchOnWindowFocus: false,
    staleTime: 0,
  })

  const refetch = useCallback(async () => {
    if (Date.now() < backoffUntilRef.current) {
      return query
    }
    return query.refetch()
  }, [query])

  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    dataUpdatedAt: query.dataUpdatedAt,
    backoffSeconds,
    refetch,
  }
}

/**
 * Seconds remaining until the next refetch, derived from the query's last
 * successful fetch so the countdown stays in sync with actual polling.
 */
export function useRefreshCountdown(dataUpdatedAt: number | undefined) {
  return useSecondsUntil(
    dataUpdatedAt ? dataUpdatedAt + MONITORING_REFRESH_MS : undefined
  )
}
