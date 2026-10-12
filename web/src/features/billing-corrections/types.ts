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
  /** Experimental: look the request up in an upstream new-api's usage logs. */
  upstream_check_enabled: boolean
  /** Empty means the channel Base URL. */
  upstream_base_url: string
  upstream_user_id: number
  /** The access token itself is never sent back, only whether one is saved. */
  upstream_access_token_set: boolean
  /** The background check only looks at requests made after this time. */
  upstream_check_since: number
  /** Return the quota when the background check finds no upstream charge. */
  auto_refund: boolean
  created_time: number
  updated_time: number
  /** Empty or absent when the channel has been deleted. */
  channel_name?: string
}

export type BillingCorrectionRulePayload = {
  channel_id: number
  prompt_tokens_threshold: number
  enabled: boolean
  upstream_check_enabled: boolean
  upstream_base_url: string
  upstream_user_id: number
  /** Left out to keep the saved token. */
  upstream_access_token?: string
  auto_refund: boolean
}

export type BillingCorrectionUpstreamTestPayload = {
  /** Lets the server fall back to the saved token when none is typed. */
  rule_id?: number
  channel_id: number
  upstream_base_url: string
  upstream_user_id: number
  upstream_access_token?: string
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
 * pending: charged and not yet refunded.
 * refunded: charged, and an admin marked it as refunded elsewhere.
 * credited: charged, and the quota was returned to the user's balance.
 * corrected: a rule already waived the charge at settlement.
 */
export type BillingCorrectionRecordStatus =
  | 'pending'
  | 'refunded'
  | 'credited'
  | 'corrected'

/**
 * rule: above the threshold and billed on a local estimate.
 * zero_output: charged with zero output, but the rule does not cover it.
 */
export type BillingCorrectionRecordMatch = 'rule' | 'zero_output'

/**
 * The upstream new-api check result. unchecked and unavailable are never
 * stored: the first means not checked yet, the second that the log has no
 * upstream request id.
 */
export type BillingCorrectionUpstreamStatus =
  | 'charged'
  | 'not_charged'
  | 'not_found'
  | 'failed'
  | 'unsupported'
  | 'unchecked'
  | 'unavailable'

export type BillingCorrectionRefundKind = 'mark' | 'credit' | 'auto'

/** A zero-output text request on a channel with an enabled rule. */
export type BillingCorrectionRecord = {
  /** Empty only for legacy logs, which cannot be selected. */
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
  match: BillingCorrectionRecordMatch
  status: BillingCorrectionRecordStatus
  refund_kind?: BillingCorrectionRefundKind
  refunded_at?: number
  refund_operator_id?: number
  upstream_request_id?: string
  /** Absent when the channel has no upstream check and none was run. */
  upstream_status?: BillingCorrectionUpstreamStatus
  upstream_quota?: number
  upstream_message?: string
  upstream_checked_at?: number
}

export type BillingCorrectionRecordStats = {
  /** Every charged record: pending + refunded (which includes credited). */
  charged_count: number
  charged_quota: number
  rule_matched_count: number
  rule_matched_quota: number
  pending_count: number
  pending_quota: number
  refunded_count: number
  refunded_quota: number
  credited_count: number
  credited_quota: number
  corrected_count: number
  /** Pending records the upstream confirmed it did not charge. */
  upstream_not_charged_count: number
  upstream_not_charged_quota: number
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
  match: '' | BillingCorrectionRecordMatch
  upstream: '' | BillingCorrectionUpstreamStatus
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
  match?: BillingCorrectionRecordMatch
  upstream?: BillingCorrectionUpstreamStatus
}

/** Either the selected request ids, or `all` with the current filters. */
export type BillingCorrectionRecordsPayload = {
  request_ids?: string[]
  all?: boolean
  start_timestamp?: number
  end_timestamp?: number
  username?: string
  token_name?: string
  model_name?: string
  channel?: number
  match?: BillingCorrectionRecordMatch
  upstream?: BillingCorrectionUpstreamStatus
}

export type BillingCorrectionRefundResult = {
  marked: number
  quota: number
  skipped: number
}

export type BillingCorrectionCreditResult = {
  credited: number
  quota: number
  /** Already refunded, or no request id. */
  skipped: number
  failed: number
}

export type BillingCorrectionUpstreamCheckResult = {
  checked: number
  charged: number
  not_charged: number
  not_found: number
  failed: number
  unsupported: number
  /** Records that cannot be checked. */
  skipped: number
  /** Records left for the next run when checking every filtered record. */
  remaining: number
}
