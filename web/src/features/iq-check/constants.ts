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
// Bounds mirror pkg/modeliqcheck and setting/model_iq_check_setting so the
// forms reject what the backend would reject before a round trip.

/** Mirrors modeliqcheck.MaxChecks. */
export const MAX_CHECKS = 50

/** Mirrors modeliqcheck.MaxNameLength (runes). */
export const MAX_CHECK_NAME_LENGTH = 64

/** Mirrors modeliqcheck.MaxIconLength. */
export const MAX_ICON_LENGTH = 128

/** Mirrors modeliqcheck.MaxPromptLength (runes). */
export const MAX_PROMPT_LENGTH = 20000

/** Mirrors modeliqcheck interval bounds, in minutes. */
export const DEFAULT_INTERVAL_MINUTES = 60
export const MIN_INTERVAL_MINUTES = 5
export const MAX_INTERVAL_MINUTES = 10080

/** Mirrors modeliqcheck.DefaultMaxTokens / MaxMaxTokens; 0 sends no limit. */
export const DEFAULT_MAX_TOKENS = 8192
export const MAX_MAX_TOKENS = 200000

/** Mirrors model_iq_check_setting history bounds (results kept per check). */
export const DEFAULT_HISTORY_LIMIT = 10
export const MIN_HISTORY_LIMIT = 1
export const MAX_HISTORY_LIMIT = 50

/** Mirrors model_iq_check_setting request timeout bounds, in seconds. */
export const DEFAULT_REQUEST_TIMEOUT_SECONDS = 600
export const MIN_REQUEST_TIMEOUT_SECONDS = 600
export const MAX_REQUEST_TIMEOUT_SECONDS = 3600

/**
 * A run whose finish was never recorded is treated as abandoned once this much
 * time passed beyond the request timeout, matching the public status view.
 */
export const RUNNING_GRACE_SECONDS = 120

/** Most ids accepted by one GET /api/iq-check/results call. */
export const MAX_RESULT_IDS_PER_REQUEST = 50

/** Upstream finish reason reported when max tokens cut the answer short. */
export const FINISH_REASON_LENGTH = 'length'

export const IQ_CHECK_STATUS_QUERY_KEY = ['iq-check-status'] as const
export const IQ_CHECK_CONFIG_QUERY_KEY = ['iq-check-config'] as const

export function iqCheckResultQueryKey(id: number) {
  return ['iq-check-result', id] as const
}
