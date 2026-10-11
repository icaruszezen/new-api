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
/** Outcome of one finished check run. */
export type IqCheckRunStatus = 'success' | 'failed'

/** Run metadata listed on the public page; the output is fetched separately. */
export type IqCheckHistoryItem = {
  id: number
  checked_at: number
  status: IqCheckRunStatus
  duration_ms: number
  completion_tokens: number
}

/**
 * Public card payload. The backend deliberately omits group and channel
 * identity so the page can stay reachable without authentication.
 */
export type IqCheckView = {
  id: string
  name: string
  model: string
  icon: string
  /** Prompt currently configured for the check. */
  prompt: string
  interval_minutes: number
  running: boolean
  queued: boolean
  /** 0 means the check has never finished a run. */
  last_checked_at: number
  /** 0 means the check has never started a run. */
  next_check_at: number
  /** Newest first. */
  history: IqCheckHistoryItem[]
}

export type IqCheckStatus = {
  enabled: boolean
  checks: IqCheckView[]
}

export type IqCheckErrorType =
  | 'no_channel'
  | 'unsupported_model'
  | 'timeout'
  | 'upstream_error'
  | 'empty_response'

/** Full output of one run. Results are immutable once written. */
export type IqCheckResult = {
  id: number
  check_id: string
  checked_at: number
  status: IqCheckRunStatus
  /** Model name at the time of the run. */
  model: string
  /** Prompt that was actually sent for this run. */
  prompt: string
  content: string
  content_truncated: boolean
  reasoning: string
  reasoning_truncated: boolean
  finish_reason: string
  /** Category of a failed run; raw upstream errors are only shown to admins. */
  error_type: IqCheckErrorType | ''
  duration_ms: number
  prompt_tokens: number
  completion_tokens: number
  reasoning_tokens: number
}

/** Admin-side check configuration. */
export type IqCheck = {
  id: string
  name: string
  group: string
  model: string
  prompt: string
  interval_minutes: number
  /** 0 means no max-tokens field is sent upstream. */
  max_tokens: number
  icon?: string
  enabled: boolean
  sort: number
}

export type AdminIqCheck = IqCheck & {
  resolved_icon: string
  last_started_at: number
  last_finished_at: number
  run_requested_at: number
  /** Empty when the check has never finished a run. */
  last_status: IqCheckRunStatus | ''
  last_checked_at: number
  /** Raw upstream error of the latest run; admin only. */
  last_error: string
}

export type IqCheckConfig = {
  enabled: boolean
  history_limit: number
  request_timeout_seconds: number
  checks: AdminIqCheck[]
}

/** Partial update: omitted fields are left untouched by the backend. */
export type IqCheckConfigUpdate = {
  enabled?: boolean
  checks?: IqCheck[]
  history_limit?: number
  request_timeout_seconds?: number
}
