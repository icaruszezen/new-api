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

/** Beat status codes shared with the backend status bar colours. */
export const BEAT_STATUS_UP = 1
export const BEAT_STATUS_SLOW = 2
export const BEAT_STATUS_DOWN = 3

export type MonitorStatus = 'up' | 'degraded' | 'down' | 'unknown'

export type Beat = {
  ts: number
  status: number
  ttft_ms: number
}

/**
 * Public status card payload. The backend deliberately omits group and channel
 * identity so the page can stay reachable without authentication.
 */
export type MonitorView = {
  id: string
  name: string
  model: string
  icon: string
  status: MonitorStatus
  avg_ttft_ms: number
  ping_ms: number
  /** Null means no samples yet; 0 means every sample failed. */
  uptime: number | null
  beats: Beat[]
}

export type ChannelMonitoringStatus = {
  enabled: boolean
  sample_window_seconds: number
  /** Sample count behind the status bar. */
  beat_limit: number
  monitors: MonitorView[]
}

/** Admin-side monitor configuration. */
export type ChannelMonitor = {
  id: string
  name: string
  group: string
  model: string
  icon?: string
  enabled: boolean
  sort: number
  /** recent = status-bar window; all = retained hourly totals. */
  uptime_scope?: string
}

export type AdminMonitor = ChannelMonitor & {
  resolved_icon: string
}

export type ChannelMonitorsConfig = {
  enabled: boolean
  monitors: AdminMonitor[]
}
