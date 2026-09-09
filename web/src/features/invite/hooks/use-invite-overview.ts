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
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useState } from 'react'

import { generateAffiliateLink } from '@/features/wallet/lib/affiliate'
import { useAuthStore } from '@/stores/auth-store'

import { getInviteOverview } from '../api'
import type { InviteeFilter, InviteOverview } from '../types'

export function useInviteOverview() {
  const [filter, setFilter] = useState<InviteeFilter>('all')
  const canLoad = useAuthStore((state) => Boolean(state.auth.accessToken))

  const query = useQuery({
    queryKey: ['invite', 'overview', filter],
    queryFn: () => getInviteOverview(filter),
    enabled: canLoad,
    staleTime: 60 * 1000,
    // Switching the invitee filter refetches; keeping the previous page avoids
    // collapsing the stat cards and table back to skeletons on every toggle.
    placeholderData: keepPreviousData,
  })

  const overview: InviteOverview | null = query.data?.data ?? null

  return {
    filter,
    setFilter,
    overview,
    // Only the very first load shows skeletons; filter refetches keep the
    // previous data visible.
    loading: query.isPending,
    error: query.isError,
    inviteLink: overview?.aff_code
      ? generateAffiliateLink(overview.aff_code)
      : '',
  }
}
