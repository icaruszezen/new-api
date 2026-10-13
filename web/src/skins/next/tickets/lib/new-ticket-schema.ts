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
  CONTENT_MAX_LENGTH,
  SUBJECT_MAX_LENGTH,
} from '@/features/tickets/lib/limits'

/**
 * New-ticket form. Values are trimmed before they are checked and sent; the
 * server sanitises and re-checks both fields.
 */
export function getNewTicketSchema(t: TFunction) {
  return z.object({
    subject: z
      .string()
      .trim()
      .min(1, t('Please enter a subject'))
      .max(
        SUBJECT_MAX_LENGTH,
        t('The subject can be at most {{max}} characters', {
          max: SUBJECT_MAX_LENGTH,
        })
      ),
    content: z
      .string()
      .trim()
      .min(1, t('Please describe your issue'))
      .max(
        CONTENT_MAX_LENGTH,
        t('The description can be at most {{max}} characters', {
          max: CONTENT_MAX_LENGTH,
        })
      ),
  })
}

export type NewTicketFormValues = z.infer<ReturnType<typeof getNewTicketSchema>>
