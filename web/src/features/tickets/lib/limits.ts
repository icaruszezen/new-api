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
// User-visible limits mirrored from the backend. The server stays
// authoritative; these only give earlier feedback.

export const SUBJECT_MAX_LENGTH = 100
export const CONTENT_MAX_LENGTH = 5000
export const MAX_IMAGES_PER_MESSAGE = 4
export const MAX_IMAGE_MB = 8
export const MAX_IMAGE_BYTES = MAX_IMAGE_MB * 1024 * 1024

/** SVG and other active formats are deliberately excluded. */
export const ACCEPTED_IMAGE_TYPES: readonly string[] = [
  'image/png',
  'image/jpeg',
  'image/gif',
  'image/webp',
]
