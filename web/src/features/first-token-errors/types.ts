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
export type FirstTokenErrorLog = {
  id: number
  created_at: number
  request_id?: string
  upstream_request_id?: string
  user_id: number
  username: string
  token_id: number
  token_name: string
  channel_id: number
  channel_name: string
  group: string
  model_name: string
  prompt_tokens: number
  completion_tokens: number
  cache_tokens: number
  would_be_quota: number
  use_time: number
  is_stream: boolean
  frt_ms: number
  error_message: string
  dirty_usage_json: string
  stream_status_json: string
  request_path: string
  other_json: string
  has_request_body: boolean
}

export type FirstTokenErrorStat = {
  count: number
  would_be_quota: number
  body_count: number
  body_capture_enabled: boolean
  correction_enabled: boolean
  treat_all_output_one_enabled: boolean
  log_enabled: boolean
  log_max_keep: number
}

export type FirstTokenErrorBody = {
  id: number
  log_id: number
  created_at: number
  body: string
  truncated: boolean
  byte_size: number
}

export type FirstTokenErrorQuery = {
  p?: number
  page_size?: number
  username?: string
  token_name?: string
  model_name?: string
  channel?: string
  group?: string
  request_id?: string
  start_timestamp?: number
  end_timestamp?: number
}

export type PageResponse<T> = {
  page: number
  page_size: number
  total: number
  items: T[]
}

export type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}
