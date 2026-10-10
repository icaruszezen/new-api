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
import { afterEach, describe, expect, test, vi } from 'vitest'

import {
  buildMarkAllPayload,
  buildRecordQuery,
  getDefaultRecordFilters,
  isRecordSelectable,
} from '../lib/records'
import type {
  BillingCorrectionRecord,
  BillingCorrectionRecordFilters,
} from '../types'

const START = new Date(2026, 9, 3, 0, 0, 0)
const END = new Date(2026, 9, 10, 16, 30, 0)

function buildFilters(
  overrides: Partial<BillingCorrectionRecordFilters> = {}
): BillingCorrectionRecordFilters {
  return {
    start: START,
    end: END,
    username: '',
    token: '',
    model: '',
    channel: '',
    status: '',
    ...overrides,
  }
}

function buildRecord(
  overrides: Partial<BillingCorrectionRecord> = {}
): BillingCorrectionRecord {
  return {
    request_id: 'req-1',
    created_at: 1_760_000_000,
    user_id: 1,
    username: 'alice',
    token_name: 'default',
    model_name: 'gpt-4o',
    channel_id: 7,
    channel_name: 'heartbeat-upstream',
    prompt_tokens: 1_500_000,
    prompt_tokens_threshold: 1_000_000,
    quota: 750_000,
    use_time: 30,
    is_stream: true,
    status: 'pending',
    ...overrides,
  }
}

afterEach(() => {
  vi.useRealTimers()
})

describe('billing correction records filters', () => {
  test('the default range starts at midnight seven days ago and ends an hour from now', () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date(2026, 9, 10, 15, 30, 0))

    const filters = getDefaultRecordFilters()

    expect(filters.start).toEqual(START)
    expect(filters.end).toEqual(END)
    expect(filters.status).toBe('')
  })

  test('the query sends the range in seconds and trims text filters', () => {
    const query = buildRecordQuery(
      buildFilters({
        username: ' alice ',
        token: 'default',
        model: ' gpt-4o',
        channel: '7',
        status: 'pending',
      }),
      2,
      20
    )

    expect(query).toEqual({
      p: 2,
      page_size: 20,
      start_timestamp: START.getTime() / 1000,
      end_timestamp: END.getTime() / 1000,
      username: 'alice',
      token_name: 'default',
      model_name: 'gpt-4o',
      channel: 7,
      status: 'pending',
    })
  })

  test('empty filters and a cleared range are left out of the query', () => {
    const query = buildRecordQuery(
      buildFilters({ start: undefined, end: undefined, username: '   ' }),
      1,
      20
    )

    expect(query).toEqual({ p: 1, page_size: 20 })
  })

  test.each(['abc', '0', '-3', '1.5'])(
    'a channel value of "%s" means every channel',
    (channel) => {
      expect(
        buildRecordQuery(buildFilters({ channel }), 1, 20)
      ).not.toHaveProperty('channel', expect.anything())
    }
  )

  test('marking everything keeps the filters but drops the status filter', () => {
    const payload = buildMarkAllPayload(
      buildFilters({ username: 'alice', channel: '7', status: 'refunded' })
    )

    expect(payload).toEqual({
      all: true,
      start_timestamp: START.getTime() / 1000,
      end_timestamp: END.getTime() / 1000,
      username: 'alice',
      channel: 7,
    })
  })
})

describe('billing correction record selection', () => {
  test.each([
    { name: 'a pending record', record: buildRecord(), selectable: true },
    {
      name: 'a refunded record',
      record: buildRecord({ status: 'refunded' }),
      selectable: true,
    },
    {
      name: 'a corrected record',
      record: buildRecord({ status: 'corrected', quota: 0 }),
      selectable: false,
    },
    {
      name: 'a legacy record without a request id',
      record: buildRecord({ request_id: '' }),
      selectable: false,
    },
  ])('$name selectable: $selectable', (row) => {
    expect(isRecordSelectable(row.record)).toBe(row.selectable)
  })
})
