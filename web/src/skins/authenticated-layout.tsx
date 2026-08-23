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
import { useAuthStore } from '@/stores/auth-store'

import { ClassicAuthenticatedLayout } from './classic/authenticated-layout'
import {
  isConsolePreviewActive,
  useConsolePreviewStore,
} from './console-preview-store'
import { UiSkinProvider, useUiSkin } from './context'
import { NextAuthenticatedLayout } from './next/authenticated-layout'
import { resolveConsoleSkin } from './registry'

function ActiveSkinLayout() {
  const siteSkin = useUiSkin()
  const role = useAuthStore((state) => state.auth.user?.role ?? 0)
  const userId = useAuthStore((state) => state.auth.user?.id)
  const previewUserConsole = useConsolePreviewStore(
    (state) => state.previewUserConsole
  )
  const previewUserId = useConsolePreviewStore((state) => state.previewUserId)
  const skin = resolveConsoleSkin(
    siteSkin,
    role,
    isConsolePreviewActive({ previewUserConsole, previewUserId }, userId)
  )

  if (skin === 'next') {
    return <NextAuthenticatedLayout />
  }
  return <ClassicAuthenticatedLayout />
}

/**
 * Single dispatch point for the authenticated console shell. The site skin is
 * still administrator-chosen; role and a session preview decide which shell
 * actually mounts. Public pages keep using `useUiSkin()` unchanged.
 */
export function SkinnedAuthenticatedLayout() {
  return (
    <UiSkinProvider>
      <ActiveSkinLayout />
    </UiSkinProvider>
  )
}
