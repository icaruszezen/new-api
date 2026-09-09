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
import { Navigate } from '@tanstack/react-router'

import { NextInvite } from './next/invite'
import { useResolvedConsoleSkin } from './use-resolved-console-skin'

/**
 * Referral page entry. The page is a Next surface; classic sessions (including
 * administrators, who stay on classic) reach the referral card from the wallet
 * page instead of seeing this layout inside the sidebar shell.
 */
export function SkinnedInvite() {
  const skin = useResolvedConsoleSkin()

  if (skin === 'next') {
    return <NextInvite />
  }

  return <Navigate to='/wallet' replace />
}
