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
import { useNavigate } from '@tanstack/react-router'

import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

import {
  isConsolePreviewActive,
  useConsolePreviewStore,
} from './console-preview-store'
import { parseUiSkin } from './registry'

/**
 * Session-level preview of the regular-user console. Does not impersonate,
 * does not change `ui_skin`, and must be imported from this module rather
 * than the skins barrel to avoid a layout import cycle.
 */
export function useUserConsolePreview() {
  const navigate = useNavigate()
  const user = useAuthStore((state) => state.auth.user)
  const siteSkin = useSystemConfigStore((state) =>
    parseUiSkin(state.config.uiSkin)
  )
  const previewUserConsole = useConsolePreviewStore(
    (state) => state.previewUserConsole
  )
  const previewUserId = useConsolePreviewStore((state) => state.previewUserId)

  const canPreview = siteSkin === 'next' && (user?.role ?? 0) >= ROLE.ADMIN
  const isPreviewing = canPreview
    ? isConsolePreviewActive({ previewUserConsole, previewUserId }, user?.id)
    : false

  const startPreview = () => {
    if (!user || !canPreview) return
    useConsolePreviewStore.getState().enterPreview(user.id)
    void navigate({
      to: '/dashboard/$section',
      params: { section: 'overview' },
    })
  }

  const stopPreview = () => {
    useConsolePreviewStore.getState().exitPreview()
  }

  return { canPreview, isPreviewing, startPreview, stopPreview }
}
