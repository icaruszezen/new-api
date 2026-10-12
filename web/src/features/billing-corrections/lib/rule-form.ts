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
import type { TFunction } from 'i18next'
import { z } from 'zod'

import { formatNumber } from '@/lib/format'

import type {
  BillingCorrectionRule,
  BillingCorrectionRulePayload,
} from '../types'

/** Mirrors the prompt_tokens_threshold range enforced by the backend. */
export const PROMPT_TOKENS_THRESHOLD_MIN = 1
export const PROMPT_TOKENS_THRESHOLD_MAX = 1_000_000_000
export const DEFAULT_PROMPT_TOKENS_THRESHOLD = 1_000_000

/** Form placeholder for "no channel picked yet"; real channel ids are > 0. */
export const UNSELECTED_CHANNEL_ID = 0

/** Mirrors the upstream access token column width enforced by the backend. */
export const UPSTREAM_ACCESS_TOKEN_MAX_LENGTH = 255

type RuleFormSchemaOptions = {
  /** The rule being edited already has an upstream access token saved. */
  upstreamTokenSaved?: boolean
}

/** The backend parses the URL fully; this only catches a missing scheme. */
const HTTP_URL_PATTERN = /^https?:\/\/[^\s/]+/i

export function getRuleFormSchema(
  t: TFunction,
  options: RuleFormSchemaOptions = {}
) {
  const thresholdRangeMessage = t(
    'Threshold must be between {{min}} and {{max}} tokens',
    {
      min: formatNumber(PROMPT_TOKENS_THRESHOLD_MIN),
      max: formatNumber(PROMPT_TOKENS_THRESHOLD_MAX),
    }
  )

  return z
    .object({
      channel_id: z
        .number()
        .int(t('Please select a channel'))
        .positive(t('Please select a channel')),
      prompt_tokens_threshold: z
        .number({ error: t('Please enter a token threshold') })
        .int(t('Threshold must be a whole number'))
        .min(PROMPT_TOKENS_THRESHOLD_MIN, thresholdRangeMessage)
        .max(PROMPT_TOKENS_THRESHOLD_MAX, thresholdRangeMessage),
      enabled: z.boolean(),
      upstream_check_enabled: z.boolean(),
      upstream_base_url: z.string(),
      /** Kept as text so an empty input is not NaN while the check is off. */
      upstream_user_id: z.string(),
      upstream_access_token: z
        .string()
        .max(
          UPSTREAM_ACCESS_TOKEN_MAX_LENGTH,
          t('The access token is too long')
        ),
      auto_refund: z.boolean(),
    })
    .superRefine((values, ctx) => {
      // The upstream fields are hidden while the check is off, so they are
      // only validated when it is on.
      if (!values.upstream_check_enabled) return
      const baseUrl = values.upstream_base_url.trim()
      if (baseUrl !== '' && !HTTP_URL_PATTERN.test(baseUrl)) {
        ctx.addIssue({
          code: 'custom',
          path: ['upstream_base_url'],
          message: t('Enter a full URL starting with http:// or https://'),
        })
      }
      if (!/^[1-9]\d*$/.test(values.upstream_user_id.trim())) {
        ctx.addIssue({
          code: 'custom',
          path: ['upstream_user_id'],
          message: t('Enter the upstream user ID'),
        })
      }
      if (
        values.upstream_access_token.trim() === '' &&
        !options.upstreamTokenSaved
      ) {
        ctx.addIssue({
          code: 'custom',
          path: ['upstream_access_token'],
          message: t('Enter the upstream access token'),
        })
      }
    })
}

export type RuleFormValues = z.infer<ReturnType<typeof getRuleFormSchema>>

export const RULE_FORM_DEFAULT_VALUES: RuleFormValues = {
  channel_id: UNSELECTED_CHANNEL_ID,
  prompt_tokens_threshold: DEFAULT_PROMPT_TOKENS_THRESHOLD,
  enabled: true,
  upstream_check_enabled: false,
  upstream_base_url: '',
  upstream_user_id: '',
  upstream_access_token: '',
  auto_refund: false,
}

/** The saved access token is never sent back, so its input starts empty. */
export function transformRuleToFormValues(
  rule: BillingCorrectionRule
): RuleFormValues {
  return {
    channel_id: rule.channel_id,
    prompt_tokens_threshold: rule.prompt_tokens_threshold,
    enabled: rule.enabled,
    upstream_check_enabled: rule.upstream_check_enabled,
    upstream_base_url: rule.upstream_base_url,
    upstream_user_id:
      rule.upstream_user_id > 0 ? String(rule.upstream_user_id) : '',
    upstream_access_token: '',
    auto_refund: rule.auto_refund,
  }
}

/** An empty access token is left out so the server keeps the saved one. */
export function transformFormValuesToPayload(
  values: RuleFormValues
): BillingCorrectionRulePayload {
  const userId = Number(values.upstream_user_id.trim())
  const accessToken = values.upstream_access_token.trim()
  return {
    channel_id: values.channel_id,
    prompt_tokens_threshold: values.prompt_tokens_threshold,
    enabled: values.enabled,
    upstream_check_enabled: values.upstream_check_enabled,
    upstream_base_url: values.upstream_base_url.trim(),
    upstream_user_id: Number.isInteger(userId) && userId > 0 ? userId : 0,
    ...(accessToken ? { upstream_access_token: accessToken } : {}),
    auto_refund: values.upstream_check_enabled && values.auto_refund,
  }
}
