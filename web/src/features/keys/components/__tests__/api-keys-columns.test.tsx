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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook } from '@testing-library/react'
import type { ColumnDef } from '@tanstack/react-table'
import type { ReactNode } from 'react'
import { describe, expect, test, vi } from 'vitest'

import type { ApiKey } from '../../types'

vi.mock('@/lib/api', () => ({
  getUserGroups: vi.fn().mockResolvedValue({ success: true, data: {} }),
}))

const { useApiKeysColumns } = await import('../api-keys-columns')

function getColumnId(column: ColumnDef<ApiKey>): string | undefined {
  if (column.id) return column.id
  if ('accessorKey' in column && typeof column.accessorKey === 'string') {
    return column.accessorKey
  }
  return undefined
}

describe('API key list columns', () => {
  test('omits Models and IP Restriction from the visible table columns', () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const { result } = renderHook(() => useApiKeysColumns(Date.now()), {
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      ),
    })

    const ids = result.current.map((column) => getColumnId(column))
    expect(ids).toContain('name')
    expect(ids).toContain('quota')
    expect(ids).toContain('group')
    expect(ids).not.toContain('model_limits')
    expect(ids).not.toContain('allow_ips')
  })
})
