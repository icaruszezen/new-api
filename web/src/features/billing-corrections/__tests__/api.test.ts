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
import type { AxiosAdapter, InternalAxiosRequestConfig } from 'axios'
import { toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import {
  checkBillingCorrectionRecordsUpstream,
  createBillingCorrectionRule,
  creditBillingCorrectionRecords,
  deleteBillingCorrectionRule,
  getBillingCorrectionRecords,
  getBillingCorrectionRules,
  markBillingCorrectionRecordsRefunded,
  testBillingCorrectionUpstream,
  unmarkBillingCorrectionRecordsRefunded,
  updateBillingCorrectionRule,
} from '../api'

const REJECTION = {
  success: false,
  message: 'This channel already has a billing correction rule',
  data: null,
}

const PAYLOAD = {
  channel_id: 5,
  prompt_tokens_threshold: 1_000_000,
  enabled: true,
  upstream_check_enabled: false,
  upstream_base_url: '',
  upstream_user_id: 0,
  auto_refund: false,
}

const originalAdapter = api.defaults.adapter
const requests: InternalAxiosRequestConfig[] = []

function respondWith(body: unknown): AxiosAdapter {
  return async (config) => {
    requests.push(config)
    return {
      data: body,
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    }
  }
}

beforeEach(() => {
  vi.spyOn(toast, 'error')
})

afterEach(() => {
  api.defaults.adapter = originalAdapter
  requests.length = 0
  vi.restoreAllMocks()
})

describe('billing correction api', () => {
  test('an empty rule list returned as null becomes an empty array', async () => {
    api.defaults.adapter = respondWith({
      success: true,
      message: '',
      data: null,
    })

    expect(await getBillingCorrectionRules()).toEqual([])
    expect(requests[0]).toMatchObject({
      method: 'get',
      url: '/api/billing-correction/',
    })
  })

  test.each([
    {
      action: 'create',
      send: () => createBillingCorrectionRule(PAYLOAD),
      method: 'post',
      url: '/api/billing-correction/',
    },
    {
      action: 'update',
      send: () => updateBillingCorrectionRule(3, PAYLOAD),
      method: 'put',
      url: '/api/billing-correction/3',
    },
    {
      action: 'delete',
      send: () => deleteBillingCorrectionRule(3),
      method: 'delete',
      url: '/api/billing-correction/3',
    },
    {
      action: 'records query',
      send: () => getBillingCorrectionRecords({ p: 1, page_size: 20 }),
      method: 'get',
      url: '/api/billing-correction/records',
    },
    {
      action: 'refund mark',
      send: () => markBillingCorrectionRecordsRefunded({ request_ids: ['r'] }),
      method: 'post',
      url: '/api/billing-correction/records/refund',
    },
    {
      action: 'refund unmark',
      send: () => unmarkBillingCorrectionRecordsRefunded(['r']),
      method: 'post',
      url: '/api/billing-correction/records/unrefund',
    },
    {
      action: 'quota return',
      send: () => creditBillingCorrectionRecords({ request_ids: ['r'] }),
      method: 'post',
      url: '/api/billing-correction/records/credit',
    },
    {
      action: 'upstream re-check',
      send: () => checkBillingCorrectionRecordsUpstream({ all: true }),
      method: 'post',
      url: '/api/billing-correction/records/upstream-check',
    },
    {
      action: 'upstream connection test',
      send: () =>
        testBillingCorrectionUpstream({
          channel_id: 5,
          upstream_base_url: '',
          upstream_user_id: 9,
        }),
      method: 'post',
      url: '/api/billing-correction/upstream/test',
    },
  ])(
    'a rejected $action hands the server message to the page without a second global toast',
    async (row) => {
      api.defaults.adapter = respondWith(REJECTION)

      const response = await row.send()

      expect(response).toEqual(REJECTION)
      expect(requests[0]).toMatchObject({ method: row.method, url: row.url })
      expect(toast.error).not.toHaveBeenCalled()
    }
  )
})
