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

/** Fallback when Retry-After is missing or invalid. Matches the 60s poll. */
export const DEFAULT_RETRY_AFTER_MS = 60_000

/** Cap a bad or huge Retry-After so the status page cannot freeze. */
export const MAX_RETRY_AFTER_MS = 20 * 60 * 1000

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object'
}

export function isRateLimitedError(error: unknown): boolean {
  if (!isRecord(error)) return false
  const response = error.response
  return isRecord(response) && response.status === 429
}

function readHeaderValue(headers: unknown, name: string): string | undefined {
  if (!headers) return undefined
  if (typeof (headers as { get?: unknown }).get === 'function') {
    const value = (headers as { get: (headerName: string) => unknown }).get(
      name
    )
    return value == null ? undefined : String(value)
  }
  if (!isRecord(headers)) return undefined
  const value = headers[name] ?? headers['Retry-After']
  if (Array.isArray(value)) {
    return value[0] == null ? undefined : String(value[0])
  }
  return value == null ? undefined : String(value)
}

/**
 * Reads a delta-seconds Retry-After from an axios-like 429 error.
 * The backend only emits seconds; HTTP-date values fall back to 60s.
 */
export function parseRetryAfterMs(error: unknown): number {
  if (!isRecord(error) || !isRecord(error.response)) {
    return DEFAULT_RETRY_AFTER_MS
  }
  const header = readHeaderValue(error.response.headers, 'retry-after')
  if (header == null || header === '') {
    return DEFAULT_RETRY_AFTER_MS
  }
  const seconds = Number(header)
  if (!Number.isFinite(seconds) || seconds <= 0) {
    return DEFAULT_RETRY_AFTER_MS
  }
  return Math.min(Math.round(seconds * 1000), MAX_RETRY_AFTER_MS)
}
