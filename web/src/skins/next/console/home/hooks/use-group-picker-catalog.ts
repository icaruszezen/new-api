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
import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'

import { getPricing } from '@/features/pricing/api'

import { resolvePricingModels } from '../lib/group-picker'

export function useGroupPickerCatalog(enabled: boolean) {
  const query = useQuery({
    queryKey: ['pricing'],
    queryFn: getPricing,
    enabled,
    staleTime: 5 * 60 * 1000,
  })

  const models = useMemo(() => {
    if (!query.data?.data) return []
    return resolvePricingModels(query.data.data, query.data.vendors ?? [])
  }, [query.data])

  return {
    models,
    isLoading: query.isLoading,
    isError: query.isError,
  }
}
