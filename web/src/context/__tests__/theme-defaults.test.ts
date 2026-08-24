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
import { afterEach, describe, expect, test } from 'vitest'

import {
  SYSTEM_CONFIG_STORAGE_KEY,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import { defaultThemeForUiSkin, getSiteDefaultTheme } from '../theme-defaults'

function setSiteSkin(uiSkin: 'classic' | 'next') {
  useSystemConfigStore.setState({
    config: { ...useSystemConfigStore.getState().config, uiSkin },
  })
}

function writePersistedSkin(uiSkin: 'classic' | 'next') {
  localStorage.setItem(
    SYSTEM_CONFIG_STORAGE_KEY,
    JSON.stringify({ state: { config: { uiSkin } } })
  )
}

describe('defaultThemeForUiSkin', () => {
  test.each([
    ['next', 'light'],
    ['classic', 'system'],
  ] as const)('maps %s to %s', (skin, expected) => {
    expect(defaultThemeForUiSkin(skin)).toBe(expected)
  })
})

describe('getSiteDefaultTheme', () => {
  afterEach(() => {
    setSiteSkin('classic')
    localStorage.removeItem(SYSTEM_CONFIG_STORAGE_KEY)
  })

  test('uses light when the live store already has next enabled', () => {
    setSiteSkin('next')
    writePersistedSkin('classic')

    expect(getSiteDefaultTheme()).toBe('light')
  })

  test('uses persisted next when the live store is still the classic default', () => {
    setSiteSkin('classic')
    writePersistedSkin('next')

    expect(getSiteDefaultTheme()).toBe('light')
  })

  test('uses system when both the live store and persist are classic', () => {
    setSiteSkin('classic')
    writePersistedSkin('classic')

    expect(getSiteDefaultTheme()).toBe('system')
  })

  test('keeps an explicit defaultTheme override', () => {
    setSiteSkin('next')

    expect(getSiteDefaultTheme('dark')).toBe('dark')
  })
})
