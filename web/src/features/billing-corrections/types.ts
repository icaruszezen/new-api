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
export type BillingCorrectionRule = {
  id: number
  channel_id: number
  /** The rule fires only when prompt tokens are strictly greater than this. */
  prompt_tokens_threshold: number
  enabled: boolean
  created_time: number
  updated_time: number
  /** Empty or absent when the channel has been deleted. */
  channel_name?: string
}

export type BillingCorrectionRulePayload = {
  channel_id: number
  prompt_tokens_threshold: number
  enabled: boolean
}

export type ApiResponse<T> = {
  success: boolean
  message: string
  data: T
}

export type ChannelOption = {
  id: number
  name: string
}

/**
 * pending: charged and not yet marked as refunded.
 * refunded: charged, and an admin marked it as refunded elsewhere.
 * corrected: a rule already waived the charge at settlement.
 */
export type BillingCorrectionRecordStatus = 'pending' | 'refunded' | 'corrected'

/** A consume log that matches the large-context mis-billing conditions. */
export type BillingCorrectionRecord = {
  /** Empty only for legacy logs, which cannot be marked. */
  request_id: string
  created_at: number
  user_id: number
  username: string
  token_name: string
  model_name: string
  channel_id: number
  channel_name: string
  prompt_tokens: number
  prompt_tokens_threshold: number
  quota: number
  /** What a corrected record would have cost without the rule. */
  original_quota?: number
  use_time: number
  is_stream: boolean
  status: BillingCorrectionRecordStatus
  refunded_at?: number
  refund_operator_id?: number
}

export type BillingCorrectionRecordStats = {
  charged_count: number
  charged_quota: number
  pending_count: number
  pending_quota: number
  refunded_count: number
  refunded_quota: number
  corrected_count: number
}

export type BillingCorrectionRecordPage = {
  items: BillingCorrectionRecord[]
  total: number
  page: number
  page_size: number
  stats: BillingCorrectionRecordStats
}

/** Filters applied to the records query; empty strings mean "any". */
export type BillingCorrectionRecordFilters = {
  /** The range is open on a side left unset. */
  start?: Date
  end?: Date
  username: string
  token: string
  model: string
  /** Channel id as a string, empty for every channel with an enabled rule. */
  channel: string
  status: '' | BillingCorrectionRecordStatus
}

export type BillingCorrectionRecordQuery = {
  p: number
  page_size: number
  start_timestamp?: number
  end_timestamp?: number
  username?: string
  token_name?: string
  model_name?: string
  channel?: number
  status?: BillingCorrectionRecordStatus
}

/** Either the selected request ids, or `all` with the current filters. */
export type BillingCorrectionRefundPayload = {
  request_ids?: string[]
  all?: boolean
  start_timestamp?: number
  end_timestamp?: number
  username?: string
  token_name?: string
  model_name?: string
  channel?: number
}

export type BillingCorrectionRefundResult = {
  marked: number
  quota: number
  skipped: number
}
