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
import { AxiosHeaders } from 'axios'
import { describe, expect, test } from 'vitest'

import {
  DEFAULT_RETRY_AFTER_MS,
  MAX_RETRY_AFTER_MS,
  isRateLimitedError,
  parseRetryAfterMs,
} from '../lib/retry-after'

describe('parseRetryAfterMs', () => {
  test('converts a delta-seconds Retry-After header to milliseconds', () => {
    expect(
      parseRetryAfterMs({
        response: { status: 429, headers: { 'retry-after': '5' } },
      })
    ).toBe(5_000)
  })

  test('reads Retry-After from AxiosHeaders', () => {
    const headers = new AxiosHeaders()
    headers.set('retry-after', '12')
    expect(
      parseRetryAfterMs({
        response: { status: 429, headers },
      })
    ).toBe(12_000)
  })

  test('falls back to 60s when the header is missing or invalid', () => {
    expect(parseRetryAfterMs({ response: { status: 429 } })).toBe(
      DEFAULT_RETRY_AFTER_MS
    )
    expect(
      parseRetryAfterMs({
        response: { status: 429, headers: { 'retry-after': 'soon' } },
      })
    ).toBe(DEFAULT_RETRY_AFTER_MS)
    expect(
      parseRetryAfterMs({
        response: { status: 429, headers: { 'retry-after': '0' } },
      })
    ).toBe(DEFAULT_RETRY_AFTER_MS)
    expect(
      parseRetryAfterMs({
        response: { status: 429, headers: { 'retry-after': '-3' } },
      })
    ).toBe(DEFAULT_RETRY_AFTER_MS)
    expect(parseRetryAfterMs(undefined)).toBe(DEFAULT_RETRY_AFTER_MS)
  })

  test('caps an oversized Retry-After at 20 minutes', () => {
    expect(
      parseRetryAfterMs({
        response: { status: 429, headers: { 'retry-after': '999999' } },
      })
    ).toBe(MAX_RETRY_AFTER_MS)
  })
})

describe('isRateLimitedError', () => {
  test('detects HTTP 429 responses', () => {
    expect(isRateLimitedError({ response: { status: 429 } })).toBe(true)
    expect(isRateLimitedError({ response: { status: 500 } })).toBe(false)
    expect(isRateLimitedError(new Error('network'))).toBe(false)
  })
})
