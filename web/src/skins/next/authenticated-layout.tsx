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
import { useRouterState } from '@tanstack/react-router'

// Imported from the concrete module instead of the `@/components/layout`
// barrel, which reaches the sidebar config and cycles back into features.
import { AuthenticatedLayout } from '@/components/layout/components/authenticated-layout'

import {
  isNextStandaloneShellPath,
  resolveNextConsoleShellPathname,
} from '../console-home-path'
import { useUserConsolePreview } from '../use-user-console-preview'
import { NextConsoleShell } from './console/shell'
import { NextConsolePreviewBanner } from './preview-banner'

/**
 * Next console chrome. The user homepage and personal center are standalone
 * pages; every other authenticated route still uses the existing sidebar
 * layout.
 *
 * Shell choice ignores public paths. A pending `/pricing` (or `/`, `/rankings`)
 * must not look like a sidebar console route, and a stale public
 * `resolvedLocation` must not hide an incoming `/dashboard` standalone page.
 *
 * The preview banner lives on this side so administrators can tell they are
 * looking at the user console without changing classic chrome.
 */
export function NextAuthenticatedLayout() {
  const pathname = useRouterState({
    select: (state) =>
      resolveNextConsoleShellPathname(
        state.location.pathname,
        state.resolvedLocation?.pathname
      ),
  })
  const preview = useUserConsolePreview()
  const isStandaloneShell = isNextStandaloneShellPath(pathname)

  return (
    <>
      <NextConsolePreviewBanner />
      {isStandaloneShell ? (
        <NextConsoleShell previewOffset={preview.isPreviewing} />
      ) : (
        <AuthenticatedLayout />
      )}
    </>
  )
}
