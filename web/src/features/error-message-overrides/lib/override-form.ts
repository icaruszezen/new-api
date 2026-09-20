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
  ALL_CHANNELS_ID,
  type ErrorMessageOverride,
  type ErrorMessageOverridePayload,
} from '../types'

/** Mirrors the match_substring column width enforced by the backend. */
export const MATCH_SUBSTRING_MAX_LENGTH = 512

export const OVERRIDE_SCOPE_ALL = 'all'
export const OVERRIDE_SCOPE_CHANNEL = 'channel'

export function getOverrideFormSchema(t: TFunction) {
  return z
    .object({
      match_substring: z
        .string()
        .trim()
        .min(1, t('Match text cannot be empty'))
        .max(
          MATCH_SUBSTRING_MAX_LENGTH,
          t('Match text cannot exceed {{max}} characters', {
            max: MATCH_SUBSTRING_MAX_LENGTH,
          })
        ),
      replacement_message: z
        .string()
        .trim()
        .min(1, t('Replacement message cannot be empty')),
      scope: z.enum([OVERRIDE_SCOPE_ALL, OVERRIDE_SCOPE_CHANNEL]),
      channel_id: z.number(),
      priority: z
        .number()
        .int(t('Priority must be an integer'))
        .min(0, t('Priority cannot be negative')),
      enabled: z.boolean(),
    })
    .superRefine((values, ctx) => {
      if (values.scope === OVERRIDE_SCOPE_CHANNEL && values.channel_id <= 0) {
        ctx.addIssue({
          code: 'custom',
          path: ['channel_id'],
          message: t('Please select a channel'),
        })
      }
    })
}

export type OverrideFormValues = {
  match_substring: string
  replacement_message: string
  scope: typeof OVERRIDE_SCOPE_ALL | typeof OVERRIDE_SCOPE_CHANNEL
  channel_id: number
  priority: number
  enabled: boolean
}

export const OVERRIDE_FORM_DEFAULT_VALUES: OverrideFormValues = {
  match_substring: '',
  replacement_message: '',
  scope: OVERRIDE_SCOPE_ALL,
  channel_id: ALL_CHANNELS_ID,
  priority: 0,
  enabled: true,
}

export function transformOverrideToFormValues(
  override: ErrorMessageOverride
): OverrideFormValues {
  return {
    match_substring: override.match_substring,
    replacement_message: override.replacement_message,
    scope:
      override.channel_id > 0 ? OVERRIDE_SCOPE_CHANNEL : OVERRIDE_SCOPE_ALL,
    channel_id: override.channel_id,
    priority: override.priority,
    enabled: override.enabled,
  }
}

export function transformFormValuesToPayload(
  values: OverrideFormValues
): ErrorMessageOverridePayload {
  return {
    match_substring: values.match_substring.trim(),
    replacement_message: values.replacement_message.trim(),
    channel_id:
      values.scope === OVERRIDE_SCOPE_CHANNEL
        ? values.channel_id
        : ALL_CHANNELS_ID,
    priority: values.priority,
    enabled: values.enabled,
  }
}
