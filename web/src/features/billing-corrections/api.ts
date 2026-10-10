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
import { getChannels } from '@/features/channels/api'
import { api, type ApiRequestConfig } from '@/lib/api'

import type {
  ApiResponse,
  BillingCorrectionRule,
  BillingCorrectionRulePayload,
  ChannelOption,
} from './types'

const BASE_URL = '/api/billing-correction'

/**
 * The page toasts the server message itself when a write is rejected
 * (success=false), so the global interceptor must not toast it a second time.
 */
const WRITE_REQUEST_CONFIG: ApiRequestConfig = { skipBusinessError: true }

/** The channel list endpoint caps page_size at 100, so the picker walks pages. */
const CHANNEL_PAGE_SIZE = 100
const MAX_CHANNEL_PAGES = 20

export async function getBillingCorrectionRules(): Promise<
  BillingCorrectionRule[]
> {
  const res = await api.get<ApiResponse<BillingCorrectionRule[] | null>>(
    `${BASE_URL}/`
  )
  return res.data.data ?? []
}

export async function createBillingCorrectionRule(
  payload: BillingCorrectionRulePayload
) {
  const res = await api.post<ApiResponse<BillingCorrectionRule>>(
    `${BASE_URL}/`,
    payload,
    WRITE_REQUEST_CONFIG
  )
  return res.data
}

export async function updateBillingCorrectionRule(
  id: number,
  payload: BillingCorrectionRulePayload
) {
  const res = await api.put<ApiResponse<BillingCorrectionRule>>(
    `${BASE_URL}/${id}`,
    payload,
    WRITE_REQUEST_CONFIG
  )
  return res.data
}

export async function deleteBillingCorrectionRule(id: number) {
  const res = await api.delete<ApiResponse<null>>(
    `${BASE_URL}/${id}`,
    WRITE_REQUEST_CONFIG
  )
  return res.data
}

export async function getChannelOptions(): Promise<ChannelOption[]> {
  const options: ChannelOption[] = []
  for (let page = 1; page <= MAX_CHANNEL_PAGES; page++) {
    const res = await getChannels({
      p: page,
      page_size: CHANNEL_PAGE_SIZE,
      id_sort: true,
    })
    const items = res.data?.items ?? []
    options.push(...items.map((item) => ({ id: item.id, name: item.name })))
    if (items.length < CHANNEL_PAGE_SIZE) break
  }
  return options
}
