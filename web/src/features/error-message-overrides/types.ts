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
/** channel_id === 0 means the rule applies to every channel. */
export const ALL_CHANNELS_ID = 0

export type ErrorMessageOverride = {
  id: number
  match_substring: string
  replacement_message: string
  channel_id: number
  priority: number
  enabled: boolean
  created_time: number
  updated_time: number
  channel_name?: string
}

export type ErrorMessageOverridePayload = {
  match_substring: string
  replacement_message: string
  channel_id: number
  priority: number
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
