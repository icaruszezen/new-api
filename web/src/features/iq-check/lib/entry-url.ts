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

/** Matches the backend limit on the IQ check website address. */
export const MAX_IQ_CHECK_URL_LENGTH = 2048

/**
 * The address is rendered as a link for every visitor, so only a full http(s)
 * address is accepted; javascript: and relative paths are not.
 */
export function isIqCheckEntryUrl(value: string): boolean {
  if (value.length > MAX_IQ_CHECK_URL_LENGTH) return false
  try {
    const parsed = new URL(value)
    return (
      (parsed.protocol === 'http:' || parsed.protocol === 'https:') &&
      parsed.host !== ''
    )
  } catch {
    return false
  }
}

export function getIqCheckEntrySchema(t: TFunction) {
  return z
    .object({
      enabled: z.boolean(),
      // The backend stores the trimmed address, so it is validated the same way.
      url: z.string().trim(),
    })
    .superRefine((values, ctx) => {
      if (values.url === '') {
        if (values.enabled) {
          ctx.addIssue({
            code: 'custom',
            path: ['url'],
            message: t(
              'Enter the website address before turning on the IQ check entry'
            ),
          })
        }
        return
      }
      if (!isIqCheckEntryUrl(values.url)) {
        ctx.addIssue({
          code: 'custom',
          path: ['url'],
          message: t(
            'Enter a full http:// or https:// website address of at most {{max}} characters',
            { max: MAX_IQ_CHECK_URL_LENGTH }
          ),
        })
      }
    })
}

export type IqCheckEntryValues = z.infer<
  ReturnType<typeof getIqCheckEntrySchema>
>
