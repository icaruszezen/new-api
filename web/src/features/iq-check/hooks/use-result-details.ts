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
import { useQueries, useQuery, useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'

import { isRateLimitedError } from '@/features/channel-monitoring/lib/retry-after'

import { getIqCheckResults } from '../api'
import { MAX_RESULT_IDS_PER_REQUEST, iqCheckResultQueryKey } from '../constants'
import type { IqCheckResult } from '../types'

/**
 * - ready: the output is loaded.
 * - loading: the output is being fetched.
 * - error: fetching failed; `retry` tries again.
 * - unavailable: the server no longer has this run (pruned or check disabled).
 */
export type ResultDetailState = 'ready' | 'loading' | 'error' | 'unavailable'

/**
 * Loads the outputs of the given runs. Results never change once written, so
 * every run is cached under its own key and only ids missing from the cache
 * are requested, all in one batch instead of one request per card.
 */
export function useResultDetails(ids: number[]) {
  const queryClient = useQueryClient()
  const uniqueIds = [...new Set(ids)].sort((a, b) => a - b)

  // Read-only subscriptions to the per-run cache entries the batch seeds.
  const cachedQueries = useQueries({
    queries: uniqueIds.map((id) => ({
      queryKey: iqCheckResultQueryKey(id),
      queryFn: async () => (await getIqCheckResults([id]))[0] ?? null,
      enabled: false,
      staleTime: Infinity,
    })),
  })

  const results = new Map<number, IqCheckResult>()
  const missingIds: number[] = []
  uniqueIds.forEach((id, index) => {
    const result = cachedQueries[index]?.data
    if (result) {
      results.set(id, result)
    } else {
      missingIds.push(id)
    }
  })

  const batchQuery = useQuery({
    queryKey: ['iq-check-results', missingIds],
    queryFn: async () => {
      const chunks: number[][] = []
      for (let i = 0; i < missingIds.length; i += MAX_RESULT_IDS_PER_REQUEST) {
        chunks.push(missingIds.slice(i, i + MAX_RESULT_IDS_PER_REQUEST))
      }
      const loaded = (
        await Promise.all(chunks.map((chunk) => getIqCheckResults(chunk)))
      ).flat()
      for (const result of loaded) {
        queryClient.setQueryData(iqCheckResultQueryKey(result.id), result)
      }
      return loaded.map((result) => result.id)
    },
    enabled: missingIds.length > 0,
    staleTime: Infinity,
    retry: (failureCount, error) =>
      !isRateLimitedError(error) && failureCount < 2,
    refetchOnWindowFocus: false,
  })

  const getState = (id: number): ResultDetailState => {
    if (results.has(id)) return 'ready'
    if (batchQuery.isError) return 'error'
    // A finished batch that still leaves the id uncached did not return it.
    if (batchQuery.isSuccess) return 'unavailable'
    return 'loading'
  }

  const { refetch } = batchQuery
  const retry = useCallback(() => {
    void refetch()
  }, [refetch])

  return { results, getState, retry }
}
