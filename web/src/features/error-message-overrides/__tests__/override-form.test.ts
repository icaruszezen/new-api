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

import {
  MATCH_SUBSTRING_MAX_LENGTH,
  OVERRIDE_SCOPE_ALL,
  OVERRIDE_SCOPE_CHANNEL,
  getOverrideFormSchema,
  transformFormValuesToPayload,
  transformOverrideToFormValues,
  type OverrideFormValues,
} from '../lib/override-form'
import { ALL_CHANNELS_ID, type ErrorMessageOverride } from '../types'

const identityT = ((key: string) => key) as never

function buildOverride(
  overrides: Partial<ErrorMessageOverride> = {}
): ErrorMessageOverride {
  return {
    id: 1,
    match_substring: 'insufficient_quota',
    replacement_message: 'Service is busy',
    channel_id: ALL_CHANNELS_ID,
    priority: 0,
    enabled: true,
    created_time: 0,
    updated_time: 0,
    ...overrides,
  }
}

function buildFormValues(
  overrides: Partial<OverrideFormValues> = {}
): OverrideFormValues {
  return {
    match_substring: 'insufficient_quota',
    replacement_message: 'Service is busy',
    scope: OVERRIDE_SCOPE_ALL,
    channel_id: ALL_CHANNELS_ID,
    priority: 0,
    enabled: true,
    ...overrides,
  }
}

describe('override scope round trip', () => {
  test('channel_id 0 loads as the all-channels scope', () => {
    const values = transformOverrideToFormValues(buildOverride())

    expect(values.scope).toBe(OVERRIDE_SCOPE_ALL)
    expect(values.channel_id).toBe(ALL_CHANNELS_ID)
  })

  test('a positive channel_id loads as the single-channel scope', () => {
    const values = transformOverrideToFormValues(
      buildOverride({ channel_id: 42 })
    )

    expect(values.scope).toBe(OVERRIDE_SCOPE_CHANNEL)
    expect(values.channel_id).toBe(42)
  })

  test('switching back to all channels clears a previously picked channel', () => {
    const payload = transformFormValuesToPayload(
      buildFormValues({ scope: OVERRIDE_SCOPE_ALL, channel_id: 42 })
    )

    expect(payload.channel_id).toBe(ALL_CHANNELS_ID)
  })

  test('the payload trims surrounding whitespace', () => {
    const payload = transformFormValuesToPayload(
      buildFormValues({
        match_substring: '  insufficient_quota  ',
        replacement_message: '  Service is busy  ',
      })
    )

    expect(payload.match_substring).toBe('insufficient_quota')
    expect(payload.replacement_message).toBe('Service is busy')
  })
})

describe('override form validation', () => {
  const schema = getOverrideFormSchema(identityT)

  test('accepts a complete all-channels rule', () => {
    expect(schema.safeParse(buildFormValues()).success).toBe(true)
  })

  test('rejects a whitespace-only match text', () => {
    const result = schema.safeParse(buildFormValues({ match_substring: '   ' }))

    expect(result.success).toBe(false)
  })

  test('rejects a match text longer than the column allows', () => {
    const result = schema.safeParse(
      buildFormValues({
        match_substring: 'a'.repeat(MATCH_SUBSTRING_MAX_LENGTH + 1),
      })
    )

    expect(result.success).toBe(false)
  })

  test('rejects an empty replacement message', () => {
    const result = schema.safeParse(
      buildFormValues({ replacement_message: '  ' })
    )

    expect(result.success).toBe(false)
  })

  test('rejects the single-channel scope without a channel', () => {
    const result = schema.safeParse(
      buildFormValues({
        scope: OVERRIDE_SCOPE_CHANNEL,
        channel_id: ALL_CHANNELS_ID,
      })
    )

    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.path).toEqual(['channel_id'])
  })

  test('rejects a negative priority', () => {
    const result = schema.safeParse(buildFormValues({ priority: -1 }))

    expect(result.success).toBe(false)
  })
})
