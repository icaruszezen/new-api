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
import { getRouteApi } from '@tanstack/react-router'

import { Dashboard } from '@/features/dashboard'
import {
  DASHBOARD_DEFAULT_SECTION,
  type DashboardSectionId,
} from '@/features/dashboard/section-registry'

import { NextConsoleHome } from './next/console/home'
import { useResolvedConsoleSkin } from './use-resolved-console-skin'

const route = getRouteApi('/_authenticated/dashboard/$section')

/**
 * Dashboard entry for the authenticated console. Next users (and
 * administrators previewing the user console) get the standalone homepage
 * on overview; classic and the analytics sections stay on the shared page.
 */
export function SkinnedDashboard() {
  const params = route.useParams()
  const skin = useResolvedConsoleSkin()
  const section = (params.section ??
    DASHBOARD_DEFAULT_SECTION) as DashboardSectionId

  if (skin === 'next' && section === 'overview') {
    return <NextConsoleHome />
  }

  return <Dashboard />
}
