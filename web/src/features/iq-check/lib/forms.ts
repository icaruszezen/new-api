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

import {
  DEFAULT_INTERVAL_MINUTES,
  DEFAULT_MAX_TOKENS,
  DEFAULT_REASONING_SUMMARY,
  ENDPOINT_TYPES,
  MAX_CHECK_NAME_LENGTH,
  MAX_HISTORY_LIMIT,
  MAX_ICON_LENGTH,
  MAX_INTERVAL_MINUTES,
  MAX_MAX_TOKENS,
  MAX_PROMPT_LENGTH,
  MAX_REQUEST_TIMEOUT_SECONDS,
  MIN_HISTORY_LIMIT,
  MIN_INTERVAL_MINUTES,
  MIN_REQUEST_TIMEOUT_SECONDS,
  REASONING_EFFORTS,
} from '../constants'
import type { IqCheck } from '../types'

/** Whole number within [min, max]; an empty input arrives as NaN. */
function boundedInteger(min: number, max: number, message: string) {
  return z
    .number({ error: message })
    .int(message)
    .min(min, message)
    .max(max, message)
}

/**
 * Counts what the user perceives as characters, matching the rune-based
 * prompt limit on the backend for text outside the Basic Multilingual Plane.
 */
export function countCharacters(text: string): number {
  return [...text].length
}

export function getCheckFormSchema(t: TFunction) {
  return z.object({
    name: z
      .string()
      .trim()
      .min(1, t('Display name is required'))
      .max(
        MAX_CHECK_NAME_LENGTH,
        t('Display name cannot exceed {{max}} characters', {
          max: MAX_CHECK_NAME_LENGTH,
        })
      ),
    group: z.string().trim().min(1, t('Please select a group')),
    model: z.string().trim().min(1, t('Please select a model')),
    // The backend stores the trimmed prompt, so it is validated the same way.
    prompt: z
      .string()
      .trim()
      .min(1, t('Prompt is required'))
      .refine(
        (value) => countCharacters(value) <= MAX_PROMPT_LENGTH,
        t('Prompt cannot exceed {{max}} characters', {
          max: MAX_PROMPT_LENGTH,
        })
      ),
    interval_minutes: boundedInteger(
      MIN_INTERVAL_MINUTES,
      MAX_INTERVAL_MINUTES,
      t('Enter a whole number of minutes between {{min}} and {{max}}', {
        min: MIN_INTERVAL_MINUTES,
        max: MAX_INTERVAL_MINUTES,
      })
    ),
    max_tokens: boundedInteger(
      0,
      MAX_MAX_TOKENS,
      t('Enter a whole number between {{min}} and {{max}}', {
        min: 0,
        max: MAX_MAX_TOKENS,
      })
    ),
    endpoint_type: z.enum(ENDPOINT_TYPES),
    reasoning_effort: z.enum(REASONING_EFFORTS),
    reasoning_summary: z.boolean(),
    icon: z
      .string()
      .trim()
      .max(
        MAX_ICON_LENGTH,
        t('Icon cannot exceed {{max}} characters', { max: MAX_ICON_LENGTH })
      ),
    enabled: z.boolean(),
  })
}

export type CheckFormValues = z.infer<ReturnType<typeof getCheckFormSchema>>

export const CHECK_FORM_DEFAULT_VALUES: CheckFormValues = {
  name: '',
  group: '',
  model: '',
  prompt: '',
  interval_minutes: DEFAULT_INTERVAL_MINUTES,
  max_tokens: DEFAULT_MAX_TOKENS,
  endpoint_type: '',
  reasoning_effort: '',
  reasoning_summary: DEFAULT_REASONING_SUMMARY,
  icon: '',
  enabled: true,
}

export function checkToFormValues(check: IqCheck): CheckFormValues {
  return {
    name: check.name,
    group: check.group,
    model: check.model,
    prompt: check.prompt,
    interval_minutes: check.interval_minutes || DEFAULT_INTERVAL_MINUTES,
    max_tokens: check.max_tokens,
    endpoint_type: check.endpoint_type ?? '',
    reasoning_effort: check.reasoning_effort ?? '',
    reasoning_summary: check.reasoning_summary,
    icon: check.icon ?? '',
    enabled: check.enabled,
  }
}

export function getGeneralSettingsSchema(t: TFunction) {
  return z.object({
    history_limit: boundedInteger(
      MIN_HISTORY_LIMIT,
      MAX_HISTORY_LIMIT,
      t('Enter a whole number between {{min}} and {{max}}', {
        min: MIN_HISTORY_LIMIT,
        max: MAX_HISTORY_LIMIT,
      })
    ),
    request_timeout_seconds: boundedInteger(
      MIN_REQUEST_TIMEOUT_SECONDS,
      MAX_REQUEST_TIMEOUT_SECONDS,
      t('Enter a whole number between {{min}} and {{max}}', {
        min: MIN_REQUEST_TIMEOUT_SECONDS,
        max: MAX_REQUEST_TIMEOUT_SECONDS,
      })
    ),
  })
}

export type GeneralSettingsValues = z.infer<
  ReturnType<typeof getGeneralSettingsSchema>
>
