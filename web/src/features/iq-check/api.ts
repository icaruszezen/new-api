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

import type { IqCheckConfig } from './types'

type ApiResponse<T> = {
  success: boolean
  message: string
  data: T
}

export async function getIqCheckConfig(): Promise<IqCheckConfig> {
  const res = await api.get<ApiResponse<IqCheckConfig>>('/api/iq-check/config')
  return res.data.data
}

/**
 * Business errors are reported by the settings page, so the global toast is
 * suppressed to avoid showing the same message twice.
 */
export async function updateIqCheckConfig(config: IqCheckConfig) {
  const res = await api.put<ApiResponse<IqCheckConfig>>(
    '/api/iq-check/config',
    config,
    { skipBusinessError: true }
  )
  return res.data
}
