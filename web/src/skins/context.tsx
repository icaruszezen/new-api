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
import { createContext, useContext, useEffect } from 'react'

import { useSystemConfigStore } from '@/stores/system-config-store'

import { DEFAULT_UI_SKIN, parseUiSkin } from './registry'
import type { UiSkin } from './types'

const UiSkinContext = createContext<UiSkin>(DEFAULT_UI_SKIN)

export function UiSkinProvider(props: { children: React.ReactNode }) {
  const configuredSkin = useSystemConfigStore((state) => state.config.uiSkin)
  // The store is persisted, so the last known skin is available on first paint
  // and the console never renders one shell before switching to the other.
  const skin = parseUiSkin(configuredSkin)

  // Mirror the active shell to <body> so it can be inspected and styled later.
  // Only `data-ui-skin` is touched; theme customization owns `data-theme-*`.
  useEffect(() => {
    if (typeof document === 'undefined') return
    const body = document.body
    if (!body) return

    body.setAttribute('data-ui-skin', skin)
    return () => body.removeAttribute('data-ui-skin')
  }, [skin])

  return (
    <UiSkinContext.Provider value={skin}>
      {props.children}
    </UiSkinContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useUiSkin(): UiSkin {
  return useContext(UiSkinContext)
}
