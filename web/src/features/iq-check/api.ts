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
  IqCheckConfig,
  IqCheckConfigUpdate,
  IqCheckResult,
  IqCheckStatus,
} from './types'

type ApiResponse<T> = {
  success: boolean
  message: string
  data: T
}

export async function getIqCheckStatus(): Promise<IqCheckStatus> {
  const res = await api.get<ApiResponse<IqCheckStatus>>(
    '/api/iq-check/status',
    { skipErrorHandler: true }
  )
  return res.data.data
}

/**
 * Batch-loads run outputs. The backend omits ids that no longer exist or that
 * belong to a disabled check, so callers must not assume one result per id.
 */
export async function getIqCheckResults(
  ids: number[]
): Promise<IqCheckResult[]> {
  const res = await api.get<ApiResponse<IqCheckResult[] | null>>(
    '/api/iq-check/results',
    {
      params: { ids: ids.join(',') },
      skipBusinessError: true,
      skipErrorHandler: true,
    }
  )
  if (!res.data.success) {
    throw new Error(res.data.message || 'Failed to load IQ check results')
  }
  return res.data.data ?? []
}

export async function getIqCheckConfig(): Promise<IqCheckConfig> {
  const res = await api.get<ApiResponse<IqCheckConfig>>('/api/iq-check/config')
  return res.data.data
}

/**
 * Business errors are reported by the calling hook, so the global toast is
 * suppressed to avoid showing the same message twice.
 */
export async function updateIqCheckConfig(update: IqCheckConfigUpdate) {
  const res = await api.put<ApiResponse<IqCheckConfig>>(
    '/api/iq-check/config',
    update,
    { skipBusinessError: true }
  )
  return res.data
}

export async function runIqCheck(id: string) {
  const res = await api.post<ApiResponse<null>>(
    `/api/iq-check/${encodeURIComponent(id)}/run`,
    undefined,
    { skipBusinessError: true }
  )
  return res.data
}
