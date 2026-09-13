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
import type { FirstTokenErrorQuery } from '../types'

export function getDefaultTimeRange(): { start: Date; end: Date } {
  const now = new Date()
  const start = new Date(now)
  start.setHours(0, 0, 0, 0)
  const end = new Date(now.getTime() + 3600 * 1000)
  return { start, end }
}

export function buildFirstTokenErrorQuery(input: {
  page: number
  pageSize: number
  username?: string
  token?: string
  model?: string
  channel?: string
  group?: string
  requestId?: string
  startTime?: number
  endTime?: number
}): FirstTokenErrorQuery {
  const defaults = getDefaultTimeRange()
  const start = input.startTime ?? defaults.start.getTime()
  const end = input.endTime ?? defaults.end.getTime()
  return {
    p: input.page,
    page_size: input.pageSize,
    username: input.username,
    token_name: input.token,
    model_name: input.model,
    channel: input.channel,
    group: input.group,
    request_id: input.requestId,
    start_timestamp: Math.floor(start / 1000),
    end_timestamp: Math.floor(end / 1000),
  }
}
