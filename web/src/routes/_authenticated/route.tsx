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
import { createFileRoute, redirect } from '@tanstack/react-router'

import { ROLE } from '@/lib/roles'
import { SkinnedAuthenticatedLayout } from '@/skins'
import {
  consumePreviewSearchParam,
  isAdminConsolePath,
} from '@/skins/admin-console-path'
import { useConsolePreviewStore } from '@/skins/console-preview-store'
import { parseUiSkin } from '@/skins/registry'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: ({ location }) => {
    const { auth } = useAuthStore.getState()

    if (!auth.user || !auth.accessToken) {
      throw redirect({
        to: '/sign-in',
        search: { redirect: location.href },
      })
    }

    const consumed = consumePreviewSearchParam(location.searchStr)
    const siteSkin = parseUiSkin(useSystemConfigStore.getState().config.uiSkin)
    const canPreview =
      siteSkin === 'next' && (auth.user.role ?? 0) >= ROLE.ADMIN

    if (
      consumed.wantsPreview &&
      canPreview &&
      !isAdminConsolePath(location.pathname)
    ) {
      useConsolePreviewStore.getState().enterPreview(auth.user.id)
    }

    if (isAdminConsolePath(location.pathname)) {
      useConsolePreviewStore.getState().exitPreview()
    }

    if (consumed.wantsPreview) {
      const hash = location.hash ? `#${location.hash}` : ''
      throw redirect({
        href: `${location.pathname}${consumed.nextSearchStr}${hash}`,
        replace: true,
      })
    }
  },
  component: SkinnedAuthenticatedLayout,
})
