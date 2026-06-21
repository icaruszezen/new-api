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

export type DebugCaptureRecord = {
  id: number
  timestamp: number
  request_id: string
  reason: string
  method: string
  path: string
  model: string
  channel_id: number
  user_id: number
  is_stream: boolean
  retry_index: number
  error_code?: string
  error_message?: string
  upstream_status?: number
  downstream_request: string
  upstream_response: string
  upstream_capture_incomplete: boolean
  truncated: boolean
}

type DebugCaptureResponse = {
  success: boolean
  message?: string
  data?: {
    records: DebugCaptureRecord[]
    total: number
  }
}

type DebugCaptureMutateResponse = {
  success: boolean
  message?: string
}

export async function getDebugCaptureErrors() {
  const res = await api.get<DebugCaptureResponse>('/api/debug/capture/errors')
  return res.data
}

export async function clearDebugCaptureErrors() {
  const res = await api.delete<DebugCaptureMutateResponse>(
    '/api/debug/capture/errors'
  )
  return res.data
}

export async function getDebugCaptureRecent() {
  const res = await api.get<DebugCaptureResponse>('/api/debug/capture/recent')
  return res.data
}
