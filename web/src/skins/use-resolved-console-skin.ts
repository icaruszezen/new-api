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
import { useSystemConfigStore } from '@/stores/system-config-store'

import {
  isConsolePreviewActive,
  useConsolePreviewStore,
} from './console-preview-store'
import { parseUiSkin, resolveConsoleSkin } from './registry'
import type { UiSkin } from './types'

/**
 * Console shell actually shown to this session. Site `classic` stays
 * classic. Site `next` serves regular users; administrators stay on classic
 * unless they opted into a user-console preview.
 */
export function useResolvedConsoleSkin(): UiSkin {
  const siteSkin = useSystemConfigStore((state) =>
    parseUiSkin(state.config.uiSkin)
  )
  const role = useAuthStore((state) => state.auth.user?.role ?? 0)
  const userId = useAuthStore((state) => state.auth.user?.id)
  const previewUserConsole = useConsolePreviewStore(
    (state) => state.previewUserConsole
  )
  const previewUserId = useConsolePreviewStore((state) => state.previewUserId)

  return resolveConsoleSkin(
    siteSkin,
    role,
    isConsolePreviewActive({ previewUserConsole, previewUserId }, userId)
  )
}
