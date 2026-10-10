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
