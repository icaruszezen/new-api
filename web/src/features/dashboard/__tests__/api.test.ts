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
import { beforeEach, describe, expect, test, vi } from 'vitest'

const getMock = vi.hoisted(() => vi.fn())

vi.mock('@/lib/api', () => ({
  api: {
    get: getMock,
  },
}))

const { getUserQuotaSummary } = await import('../api')

describe('getUserQuotaSummary', () => {
  beforeEach(() => {
    getMock.mockReset()
    getMock.mockResolvedValue({
      data: {
        success: true,
        data: { token_used: 42, prompt_tokens: 10, cache_tokens: 4 },
      },
    })
  })

  test('loads lifetime totals from /api/data/self instead of /api/data/summary', async () => {
    await getUserQuotaSummary()

    expect(getMock).toHaveBeenCalledTimes(1)
    expect(getMock).toHaveBeenCalledWith(
      '/api/data/self',
      expect.objectContaining({
        params: { lifetime: 1 },
        skipErrorHandler: true,
        skipAuthRefresh: true,
      })
    )
    expect(getMock.mock.calls[0]?.[0]).not.toBe('/api/data/summary')
  })
})
