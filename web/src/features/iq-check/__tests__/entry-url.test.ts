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
import { describe, expect, test } from 'vitest'

import { MAX_IQ_CHECK_URL_LENGTH, isIqCheckEntryUrl } from '../lib/entry-url'

describe('isIqCheckEntryUrl', () => {
  test.each([
    'https://iq.example.com',
    'http://iq.example.com:8080/check?from=new-api',
  ])('accepts the full web address %s', (url) => {
    expect(isIqCheckEntryUrl(url)).toBe(true)
  })

  test.each([
    '',
    'javascript:alert(1)',
    'data:text/html,<script>alert(1)</script>',
    'ftp://iq.example.com',
    '/iq-check',
    '//iq.example.com',
    'https://',
  ])('rejects %s because it is not a full http(s) address', (url) => {
    expect(isIqCheckEntryUrl(url)).toBe(false)
  })

  test('accepts an address at the length limit and rejects one past it', () => {
    const prefix = 'https://iq.example.com/'
    const atLimit = `${prefix}${'a'.repeat(MAX_IQ_CHECK_URL_LENGTH - prefix.length)}`

    expect(isIqCheckEntryUrl(atLimit)).toBe(true)
    expect(isIqCheckEntryUrl(`${atLimit}a`)).toBe(false)
  })
})
