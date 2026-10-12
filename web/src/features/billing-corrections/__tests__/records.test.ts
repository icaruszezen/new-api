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
  buildAllRecordsPayload,
  buildRecordQuery,
  getDefaultRecordFilters,
  hasUpstreamCheck,
  isRecordSelectable,
  isRecordUpstreamCheckable,
} from '../lib/records'
import type {
  BillingCorrectionRecord,
  BillingCorrectionRecordFilters,
  BillingCorrectionRule,
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
    match: '',
    upstream: '',
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
    match: 'rule',
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
    expect(filters.match).toBe('')
    expect(filters.upstream).toBe('')
  })

  test('the query sends the range in seconds and trims text filters', () => {
    const query = buildRecordQuery(
      buildFilters({
        username: ' alice ',
        token: 'default',
        model: ' gpt-4o',
        channel: '7',
        status: 'pending',
        match: 'zero_output',
        upstream: 'not_charged',
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
      match: 'zero_output',
      upstream: 'not_charged',
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

  test('acting on every record keeps the condition and upstream filters but drops the status filter', () => {
    const payload = buildAllRecordsPayload(
      buildFilters({
        username: 'alice',
        channel: '7',
        status: 'refunded',
        match: 'zero_output',
        upstream: 'not_charged',
      })
    )

    expect(payload).toEqual({
      all: true,
      start_timestamp: START.getTime() / 1000,
      end_timestamp: END.getTime() / 1000,
      username: 'alice',
      channel: 7,
      match: 'zero_output',
      upstream: 'not_charged',
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

describe('billing correction upstream check', () => {
  test.each([
    {
      name: 'a record not checked yet',
      record: buildRecord({ upstream_status: 'unchecked' }),
      checkable: true,
    },
    {
      name: 'a record already checked',
      record: buildRecord({ status: 'credited', upstream_status: 'charged' }),
      checkable: true,
    },
    {
      name: 'a record without an upstream request id',
      record: buildRecord({ upstream_status: 'unavailable' }),
      checkable: false,
    },
    {
      name: 'a record on a channel without an upstream check',
      record: buildRecord(),
      checkable: false,
    },
    {
      name: 'a corrected record',
      record: buildRecord({
        status: 'corrected',
        upstream_status: 'unchecked',
      }),
      checkable: false,
    },
  ])('$name checkable: $checkable', (row) => {
    expect(isRecordUpstreamCheckable(row.record)).toBe(row.checkable)
  })

  test('only an enabled rule with the upstream check turns the upstream tools on', () => {
    const rule: BillingCorrectionRule = {
      id: 1,
      channel_id: 7,
      prompt_tokens_threshold: 1_000_000,
      enabled: true,
      upstream_check_enabled: false,
      upstream_base_url: '',
      upstream_user_id: 0,
      upstream_access_token_set: false,
      upstream_check_since: 0,
      auto_refund: false,
      created_time: 0,
      updated_time: 0,
    }

    expect(hasUpstreamCheck([rule])).toBe(false)
    expect(
      hasUpstreamCheck([
        { ...rule, enabled: false, upstream_check_enabled: true },
      ])
    ).toBe(false)
    expect(hasUpstreamCheck([{ ...rule, upstream_check_enabled: true }])).toBe(
      true
    )
  })
})
