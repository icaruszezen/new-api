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
import { isAxiosError } from 'axios'

const MAX_TRANSPORT_RETRIES = 3

/**
 * Retry policy for ticket reads. Only transport failures (network errors,
 * timeouts, 5xx) are retried; a business error thrown by the API unwrap
 * (missing or foreign ticket) and a 4xx answer are final, so their error
 * state shows after the first response instead of after the backoff.
 */
export function shouldRetryTicketQuery(
  failureCount: number,
  error: Error
): boolean {
  if (!isAxiosError(error)) return false
  const status = error.response?.status ?? 0
  if (status >= 400 && status < 500) return false
  return failureCount < MAX_TRANSPORT_RETRIES
}
