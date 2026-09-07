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

/** Mirrors pkg/channelmonitor.MaxMonitorNameLength. */
export const MAX_MONITOR_NAME_LENGTH = 64

/** Mirrors pkg/channelmonitor.MaxMonitors. */
export const MAX_MONITORS = 50

/** Composite key used to reject duplicated group + model pairs. */
export function monitorPairKey(group: string, model: string): string {
  return `${group}\u0000${model}`
}

/** Mirrors pkg/channelmonitor.UptimeScopeRecent / UptimeScopeAll. */
export const UPTIME_SCOPE_RECENT = 'recent'
export const UPTIME_SCOPE_ALL = 'all'

export type UptimeScope = typeof UPTIME_SCOPE_RECENT | typeof UPTIME_SCOPE_ALL

export function normalizeUptimeScope(scope?: string): UptimeScope {
  return scope === UPTIME_SCOPE_ALL ? UPTIME_SCOPE_ALL : UPTIME_SCOPE_RECENT
}
