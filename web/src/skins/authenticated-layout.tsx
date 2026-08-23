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
import { ClassicAuthenticatedLayout } from './classic/authenticated-layout'
import { UiSkinProvider, useUiSkin } from './context'
import { NextAuthenticatedLayout } from './next/authenticated-layout'

function ActiveSkinLayout() {
  const skin = useUiSkin()

  if (skin === 'next') {
    return <NextAuthenticatedLayout />
  }
  return <ClassicAuthenticatedLayout />
}

/**
 * Single dispatch point for the authenticated console shell. The administrator
 * chooses the skin site-wide; users have no switch of their own.
 */
export function SkinnedAuthenticatedLayout() {
  return (
    <UiSkinProvider>
      <ActiveSkinLayout />
    </UiSkinProvider>
  )
}
