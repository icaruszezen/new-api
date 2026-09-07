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
import { useEffect, useState } from 'react'

import { getChannelMonitoringStatus } from '../api'

/** How often the status page pulls fresh beats. */
export const MONITORING_REFRESH_MS = 60_000

export function useMonitoringStatus() {
  return useQuery({
    queryKey: ['channel-monitoring-status'],
    queryFn: getChannelMonitoringStatus,
    refetchInterval: MONITORING_REFRESH_MS,
    refetchOnWindowFocus: true,
    staleTime: 0,
  })
}

/**
 * Seconds remaining until the next refetch, derived from the query's last
 * successful fetch so the countdown stays in sync with actual polling.
 */
export function useRefreshCountdown(dataUpdatedAt: number | undefined) {
  const [secondsLeft, setSecondsLeft] = useState(
    MONITORING_REFRESH_MS / 1000
  )

  useEffect(() => {
    if (!dataUpdatedAt) return

    const tick = () => {
      const elapsed = Date.now() - dataUpdatedAt
      const remaining = Math.ceil((MONITORING_REFRESH_MS - elapsed) / 1000)
      setSecondsLeft(Math.max(0, Math.min(MONITORING_REFRESH_MS / 1000, remaining)))
    }

    tick()
    const timer = window.setInterval(tick, 1000)
    return () => window.clearInterval(timer)
  }, [dataUpdatedAt])

  return secondsLeft
}
