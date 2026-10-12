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
import { AxiosError, AxiosHeaders } from 'axios'
import { describe, expect, test } from 'vitest'

import { shouldRetryTicketQuery } from '../lib/query-retry'

function httpError(status: number) {
  return new AxiosError(
    `Request failed with status code ${status}`,
    undefined,
    undefined,
    undefined,
    {
      status,
      statusText: '',
      headers: {},
      config: { headers: new AxiosHeaders() },
      data: null,
    }
  )
}

describe('ticket query retry policy', () => {
  test.each([
    ['a business error from the API envelope', new Error('Ticket not found')],
    ['an unexpected attachment type', new Error('Unexpected attachment type')],
    ['an HTTP 404', httpError(404)],
    ['an HTTP 403', httpError(403)],
  ])('never retries %s', (_name, error) => {
    expect(shouldRetryTicketQuery(0, error)).toBe(false)
  })

  test.each([
    ['a network failure', new AxiosError('Network Error')],
    ['an HTTP 502', httpError(502)],
  ])('retries %s up to three times', (_name, error) => {
    expect(shouldRetryTicketQuery(0, error)).toBe(true)
    expect(shouldRetryTicketQuery(2, error)).toBe(true)
    expect(shouldRetryTicketQuery(3, error)).toBe(false)
  })
})
