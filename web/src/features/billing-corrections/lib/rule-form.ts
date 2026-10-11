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

export function getRuleFormSchema(t: TFunction) {
  const thresholdRangeMessage = t(
    'Threshold must be between {{min}} and {{max}} tokens',
    {
      min: formatNumber(PROMPT_TOKENS_THRESHOLD_MIN),
      max: formatNumber(PROMPT_TOKENS_THRESHOLD_MAX),
    }
  )

  return z.object({
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
  })
}

export type RuleFormValues = z.infer<ReturnType<typeof getRuleFormSchema>>

export const RULE_FORM_DEFAULT_VALUES: RuleFormValues = {
  channel_id: UNSELECTED_CHANNEL_ID,
  prompt_tokens_threshold: DEFAULT_PROMPT_TOKENS_THRESHOLD,
  enabled: true,
}

export function transformRuleToFormValues(
  rule: BillingCorrectionRule
): RuleFormValues {
  return {
    channel_id: rule.channel_id,
    prompt_tokens_threshold: rule.prompt_tokens_threshold,
    enabled: rule.enabled,
  }
}

export function transformFormValuesToPayload(
  values: RuleFormValues
): BillingCorrectionRulePayload {
  return {
    channel_id: values.channel_id,
    prompt_tokens_threshold: values.prompt_tokens_threshold,
    enabled: values.enabled,
  }
}
