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
import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { selectTicketImages } from '../lib/attachments'
import { MAX_IMAGE_MB, MAX_IMAGES_PER_MESSAGE } from '../lib/limits'

/**
 * Appends images to a controlled `files` list from any source (file input,
 * paste, drop) and toasts once per rejection reason.
 */
export function useAddTicketImages(
  files: File[],
  onChange: (files: File[]) => void
) {
  const { t } = useTranslation()

  return useCallback(
    (incoming: File[]) => {
      if (incoming.length === 0) return

      const selection = selectTicketImages(files, incoming)
      if (selection.accepted.length > 0) {
        onChange([...files, ...selection.accepted])
      }

      const reasons = new Set(selection.rejected.map((entry) => entry.reason))
      if (reasons.has('type')) {
        toast.error(t('Only PNG, JPEG, GIF and WebP images are supported'))
      }
      if (reasons.has('size')) {
        toast.error(
          t('Each image must be {{max}} MB or smaller', { max: MAX_IMAGE_MB })
        )
      }
      if (reasons.has('count')) {
        toast.error(
          t('You can attach up to {{max}} images', {
            max: MAX_IMAGES_PER_MESSAGE,
          })
        )
      }
    },
    [files, onChange, t]
  )
}
