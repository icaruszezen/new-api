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
import { ClassicHome } from './classic/home'
import { UiSkinProvider, useUiSkin } from './context'
import { NextHome } from './next/home'

function ActiveSkinHome() {
  const skin = useUiSkin()

  if (skin === 'next') {
    return <NextHome />
  }
  return <ClassicHome />
}

/**
 * Single dispatch point for the public landing page. The administrator chooses
 * the skin site-wide, the same way the authenticated console shell is chosen.
 */
export function SkinnedHome() {
  return (
    <UiSkinProvider>
      <ActiveSkinHome />
    </UiSkinProvider>
  )
}
