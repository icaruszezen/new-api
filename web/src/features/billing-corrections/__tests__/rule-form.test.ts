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
  DEFAULT_PROMPT_TOKENS_THRESHOLD,
  RULE_FORM_DEFAULT_VALUES,
  UNSELECTED_CHANNEL_ID,
  getRuleFormSchema,
  transformFormValuesToPayload,
  transformRuleToFormValues,
  type RuleFormValues,
} from '../lib/rule-form'
import type { BillingCorrectionRule } from '../types'

const identityT = ((key: string) => key) as never

function buildRule(
  overrides: Partial<BillingCorrectionRule> = {}
): BillingCorrectionRule {
  return {
    id: 7,
    channel_id: 42,
    prompt_tokens_threshold: 2_000_000,
    enabled: false,
    upstream_check_enabled: false,
    upstream_base_url: '',
    upstream_user_id: 0,
    upstream_access_token_set: false,
    upstream_check_since: 0,
    auto_refund: false,
    created_time: 1_700_000_000,
    updated_time: 1_700_000_100,
    channel_name: 'azure-east',
    ...overrides,
  }
}

function buildFormValues(
  overrides: Partial<RuleFormValues> = {}
): RuleFormValues {
  return {
    channel_id: 42,
    prompt_tokens_threshold: DEFAULT_PROMPT_TOKENS_THRESHOLD,
    enabled: true,
    upstream_check_enabled: false,
    upstream_base_url: '',
    upstream_user_id: '',
    upstream_access_token: '',
    auto_refund: false,
    ...overrides,
  }
}

describe('billing correction rule form defaults', () => {
  test('a new rule starts with no channel, a 1,000,000 token threshold, enabled, and no upstream check', () => {
    expect(RULE_FORM_DEFAULT_VALUES).toEqual({
      channel_id: UNSELECTED_CHANNEL_ID,
      prompt_tokens_threshold: 1_000_000,
      enabled: true,
      upstream_check_enabled: false,
      upstream_base_url: '',
      upstream_user_id: '',
      upstream_access_token: '',
      auto_refund: false,
    })
  })
})

describe('billing correction rule form validation', () => {
  const schema = getRuleFormSchema(identityT)

  test('accepts the default threshold once a channel is picked', () => {
    expect(schema.safeParse(buildFormValues()).success).toBe(true)
  })

  test.each([
    { threshold: 1, label: 'the minimum of 1' },
    { threshold: 1_000_000_000, label: 'the maximum of 1,000,000,000' },
  ])('accepts a threshold at $label', (row) => {
    const result = schema.safeParse(
      buildFormValues({ prompt_tokens_threshold: row.threshold })
    )

    expect(result.success).toBe(true)
  })

  test.each([
    { threshold: 0, label: 'zero' },
    { threshold: -5, label: 'a negative value' },
    { threshold: 1_000_000_001, label: 'one above the maximum' },
  ])('rejects $label as the threshold', (row) => {
    const result = schema.safeParse(
      buildFormValues({ prompt_tokens_threshold: row.threshold })
    )

    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.path).toEqual(['prompt_tokens_threshold'])
    expect(result.error?.issues[0]?.message).toBe(
      'Threshold must be between {{min}} and {{max}} tokens'
    )
  })

  test('rejects a fractional threshold as not a whole number', () => {
    const result = schema.safeParse(
      buildFormValues({ prompt_tokens_threshold: 1500.5 })
    )

    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.message).toBe(
      'Threshold must be a whole number'
    )
  })

  test('rejects an empty threshold input (NaN) with a prompt to enter one', () => {
    const result = schema.safeParse(
      buildFormValues({ prompt_tokens_threshold: Number.NaN })
    )

    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.message).toBe(
      'Please enter a token threshold'
    )
  })

  test('rejects a rule without a channel', () => {
    const result = schema.safeParse(
      buildFormValues({ channel_id: UNSELECTED_CHANNEL_ID })
    )

    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.path).toEqual(['channel_id'])
    expect(result.error?.issues[0]?.message).toBe('Please select a channel')
  })
})

describe('billing correction upstream check validation', () => {
  const upstream = {
    upstream_check_enabled: true,
    upstream_user_id: '9',
    upstream_access_token: 'token',
  }

  test('hidden upstream fields are not validated while the check is off', () => {
    const result = getRuleFormSchema(identityT).safeParse(
      buildFormValues({ upstream_base_url: 'not a url', upstream_user_id: 'x' })
    )

    expect(result.success).toBe(true)
  })

  test.each([
    {
      name: 'a missing user id',
      values: { ...upstream, upstream_user_id: '' },
      path: 'upstream_user_id',
      message: 'Enter the upstream user ID',
    },
    {
      name: 'a user id of zero',
      values: { ...upstream, upstream_user_id: '0' },
      path: 'upstream_user_id',
      message: 'Enter the upstream user ID',
    },
    {
      name: 'a missing access token',
      values: { ...upstream, upstream_access_token: ' ' },
      path: 'upstream_access_token',
      message: 'Enter the upstream access token',
    },
    {
      name: 'an address without a scheme',
      values: { ...upstream, upstream_base_url: 'relay.example.com' },
      path: 'upstream_base_url',
      message: 'Enter a full URL starting with http:// or https://',
    },
  ])('rejects $name once the check is on', (row) => {
    const result = getRuleFormSchema(identityT).safeParse(
      buildFormValues(row.values)
    )

    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.path).toEqual([row.path])
    expect(result.error?.issues[0]?.message).toBe(row.message)
  })

  test('a saved access token lets the field stay empty', () => {
    const result = getRuleFormSchema(identityT, {
      upstreamTokenSaved: true,
    }).safeParse(buildFormValues({ ...upstream, upstream_access_token: '' }))

    expect(result.success).toBe(true)
  })
})

describe('billing correction rule transforms', () => {
  test('editing a rule loads its settings but never a saved access token', () => {
    expect(
      transformRuleToFormValues(
        buildRule({
          upstream_check_enabled: true,
          upstream_base_url: 'https://relay.example.com',
          upstream_user_id: 9,
          upstream_access_token_set: true,
          auto_refund: true,
        })
      )
    ).toEqual({
      channel_id: 42,
      prompt_tokens_threshold: 2_000_000,
      enabled: false,
      upstream_check_enabled: true,
      upstream_base_url: 'https://relay.example.com',
      upstream_user_id: '9',
      upstream_access_token: '',
      auto_refund: true,
    })
  })

  test('the payload carries only the fields the API accepts and leaves an empty token out', () => {
    const payload = transformFormValuesToPayload(
      buildFormValues({ channel_id: 9, prompt_tokens_threshold: 5_000_000 })
    )

    expect(payload).toEqual({
      channel_id: 9,
      prompt_tokens_threshold: 5_000_000,
      enabled: true,
      upstream_check_enabled: false,
      upstream_base_url: '',
      upstream_user_id: 0,
      auto_refund: false,
    })
  })

  test('the payload trims the upstream settings and sends a typed token', () => {
    const payload = transformFormValuesToPayload(
      buildFormValues({
        upstream_check_enabled: true,
        upstream_base_url: ' https://relay.example.com ',
        upstream_user_id: ' 9 ',
        upstream_access_token: ' token ',
        auto_refund: true,
      })
    )

    expect(payload).toMatchObject({
      upstream_check_enabled: true,
      upstream_base_url: 'https://relay.example.com',
      upstream_user_id: 9,
      upstream_access_token: 'token',
      auto_refund: true,
    })
  })

  test('turning the check off also turns automatic returns off', () => {
    const payload = transformFormValuesToPayload(
      buildFormValues({ upstream_check_enabled: false, auto_refund: true })
    )

    expect(payload.auto_refund).toBe(false)
  })
})
