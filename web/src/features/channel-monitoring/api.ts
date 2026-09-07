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
import { api } from '@/lib/api'

import type {
  ChannelMonitor,
  ChannelMonitoringStatus,
  ChannelMonitorsConfig,
} from './types'

type ApiResponse<T> = {
  success: boolean
  message: string
  data: T
}

export async function getChannelMonitoringStatus(): Promise<ChannelMonitoringStatus> {
  const res =
    await api.get<ApiResponse<ChannelMonitoringStatus>>(
      '/api/channel-monitoring/status'
    )
  return res.data.data
}

export async function getChannelMonitoringConfig(): Promise<ChannelMonitorsConfig> {
  const res = await api.get<ApiResponse<ChannelMonitorsConfig>>(
    '/api/channel-monitoring/config'
  )
  return res.data.data
}

/**
 * Partial update: omitted fields are left untouched, so toggling the switch
 * does not have to resend the whole monitor list.
 */
export async function updateChannelMonitoringConfig(update: {
  enabled?: boolean
  monitors?: ChannelMonitor[]
}) {
  const res = await api.put<ApiResponse<ChannelMonitorsConfig>>(
    '/api/channel-monitoring/config',
    update
  )
  return res.data
}

export async function getGroupModels(group: string): Promise<string[]> {
  const res = await api.get<ApiResponse<string[]>>('/api/group/models', {
    params: { group },
  })
  return res.data.data ?? []
}
