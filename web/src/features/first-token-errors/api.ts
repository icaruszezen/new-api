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
  ApiResponse,
  FirstTokenErrorBody,
  FirstTokenErrorLog,
  FirstTokenErrorQuery,
  FirstTokenErrorStat,
  PageResponse,
} from './types'

function buildQuery(params: FirstTokenErrorQuery): string {
  const search = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      search.append(key, String(value))
    }
  })
  return search.toString()
}

export async function getFirstTokenErrorLogs(
  params: FirstTokenErrorQuery
): Promise<ApiResponse<PageResponse<FirstTokenErrorLog>>> {
  const query = buildQuery(params)
  const res = await api.get(`/api/log/first-token-error?${query}`)
  return res.data
}

export async function getFirstTokenErrorStat(
  params: FirstTokenErrorQuery = {}
): Promise<ApiResponse<FirstTokenErrorStat>> {
  const query = buildQuery(params)
  const res = await api.get(`/api/log/first-token-error/stat?${query}`)
  return res.data
}

export async function getFirstTokenErrorLog(
  id: number
): Promise<ApiResponse<FirstTokenErrorLog>> {
  const res = await api.get(`/api/log/first-token-error/${id}`)
  return res.data
}

export async function getFirstTokenErrorBody(
  id: number
): Promise<ApiResponse<FirstTokenErrorBody>> {
  const res = await api.get(`/api/log/first-token-error/${id}/body`)
  return res.data
}
