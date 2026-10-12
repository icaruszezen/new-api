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
import {
  ACCEPTED_IMAGE_TYPES,
  MAX_IMAGE_BYTES,
  MAX_IMAGES_PER_MESSAGE,
} from './limits'

type TicketImageRejectionReason = 'type' | 'size' | 'count'

export type TicketImageSelection = {
  /** Incoming files to append to `current`, in their original order. */
  accepted: File[]
  rejected: { file: File; reason: TicketImageRejectionReason }[]
}

/**
 * Client-side pre-check for images added to a ticket message. The type comes
 * from the browser and is only a UX hint; the server sniffs and re-encodes
 * every upload regardless.
 */
export function selectTicketImages(
  current: File[],
  incoming: File[]
): TicketImageSelection {
  const selection: TicketImageSelection = { accepted: [], rejected: [] }
  const slots = MAX_IMAGES_PER_MESSAGE - current.length

  for (const file of incoming) {
    if (!ACCEPTED_IMAGE_TYPES.includes(file.type)) {
      selection.rejected.push({ file, reason: 'type' })
    } else if (file.size > MAX_IMAGE_BYTES) {
      selection.rejected.push({ file, reason: 'size' })
    } else if (selection.accepted.length >= slots) {
      selection.rejected.push({ file, reason: 'count' })
    } else {
      selection.accepted.push(file)
    }
  }

  return selection
}

/**
 * Files carried by a paste or drop event. Text in the same clipboard payload
 * is left to the browser's default handling.
 */
export function getTransferFiles(data: DataTransfer | null): File[] {
  if (!data) return []

  const files: File[] = []
  for (const item of data.items) {
    if (item.kind !== 'file') continue
    const file = item.getAsFile()
    if (file) files.push(file)
  }
  if (files.length > 0) return files

  return [...data.files]
}
