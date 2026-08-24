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
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { getCookie, removeCookie, setCookie } from '@/lib/cookies'
import {
  SYSTEM_CONFIG_STORAGE_KEY,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import { ThemeProvider, useTheme } from '../theme-provider'

const THEME_COOKIE = 'vite-ui-theme'

function ThemeProbe() {
  const { defaultTheme, resetTheme, resolvedTheme, setTheme, theme } =
    useTheme()
  return (
    <div>
      <span>{`preference ${theme}`}</span>
      <span>{`resolved ${resolvedTheme}`}</span>
      <span>{`default ${defaultTheme}`}</span>
      <button type='button' onClick={() => setTheme('dark')}>
        Choose dark
      </button>
      <button type='button' onClick={() => setTheme('system')}>
        Choose system
      </button>
      <button type='button' onClick={resetTheme}>
        Reset theme
      </button>
    </div>
  )
}

function stubPrefersColorScheme(scheme: 'dark' | 'light') {
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: query.includes('prefers-color-scheme: dark') && scheme === 'dark',
    media: query,
    onchange: null,
    addListener: () => undefined,
    removeListener: () => undefined,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    dispatchEvent: () => false,
  }))
}

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

function renderTheme() {
  return render(
    <ThemeProvider>
      <ThemeProbe />
    </ThemeProvider>
  )
}

async function expectHtmlTheme(resolved: 'light' | 'dark') {
  await waitFor(() => {
    expect(document.documentElement.classList.contains(resolved)).toBe(true)
  })
  expect(
    document.documentElement.classList.contains(
      resolved === 'light' ? 'dark' : 'light'
    )
  ).toBe(false)
}

describe('ThemeProvider site default', () => {
  beforeEach(() => {
    setSiteSkin('classic')
    localStorage.removeItem(SYSTEM_CONFIG_STORAGE_KEY)
    removeCookie(THEME_COOKIE)
    document.documentElement.classList.remove('light', 'dark')
    stubPrefersColorScheme('dark')
  })

  afterEach(() => {
    setSiteSkin('classic')
    localStorage.removeItem(SYSTEM_CONFIG_STORAGE_KEY)
    removeCookie(THEME_COOKIE)
    document.documentElement.classList.remove('light', 'dark')
    vi.unstubAllGlobals()
  })

  test('follows the OS when the site is classic and the user never chose a theme', async () => {
    setSiteSkin('classic')

    renderTheme()

    expect(screen.getByText('preference system')).toBeInTheDocument()
    expect(screen.getByText('default system')).toBeInTheDocument()
    await expectHtmlTheme('dark')
    expect(getCookie(THEME_COOKIE)).toBeUndefined()
  })

  test('uses light when the site is next even if the OS prefers dark', async () => {
    setSiteSkin('next')

    renderTheme()

    expect(screen.getByText('preference light')).toBeInTheDocument()
    expect(screen.getByText('default light')).toBeInTheDocument()
    await expectHtmlTheme('light')
    expect(getCookie(THEME_COOKIE)).toBeUndefined()
  })

  test('keeps a stored dark preference when the site is next', async () => {
    setSiteSkin('next')
    setCookie(THEME_COOKIE, 'dark')

    renderTheme()

    expect(screen.getByText('preference dark')).toBeInTheDocument()
    await expectHtmlTheme('dark')
  })

  test('keeps a stored system preference when the site is next', async () => {
    setSiteSkin('next')
    setCookie(THEME_COOKIE, 'system')

    renderTheme()

    expect(screen.getByText('preference system')).toBeInTheDocument()
    await expectHtmlTheme('dark')
  })

  test('switches to light when the site becomes next and the user never chose a theme', async () => {
    setSiteSkin('classic')
    renderTheme()
    expect(screen.getByText('preference system')).toBeInTheDocument()
    await expectHtmlTheme('dark')

    act(() => {
      setSiteSkin('next')
    })

    expect(screen.getByText('preference light')).toBeInTheDocument()
    await expectHtmlTheme('light')
    expect(getCookie(THEME_COOKIE)).toBeUndefined()
  })

  test('does not change a stored theme when the site skin changes', async () => {
    setSiteSkin('classic')
    setCookie(THEME_COOKIE, 'dark')
    renderTheme()
    expect(screen.getByText('preference dark')).toBeInTheDocument()

    act(() => {
      setSiteSkin('next')
    })

    expect(screen.getByText('preference dark')).toBeInTheDocument()
    await expectHtmlTheme('dark')
    expect(getCookie(THEME_COOKIE)).toBe('dark')
  })

  test('uses persisted next on the first paint while the live store is still classic', async () => {
    setSiteSkin('classic')
    writePersistedSkin('next')

    renderTheme()

    expect(screen.getByText('preference light')).toBeInTheDocument()
    await expectHtmlTheme('light')
    expect(getCookie(THEME_COOKIE)).toBeUndefined()
  })

  test('reset on next returns to light and clears the theme cookie', async () => {
    const user = userEvent.setup()
    setSiteSkin('next')
    setCookie(THEME_COOKIE, 'dark')
    renderTheme()
    expect(screen.getByText('preference dark')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Reset theme' }))

    expect(screen.getByText('preference light')).toBeInTheDocument()
    await expectHtmlTheme('light')
    expect(getCookie(THEME_COOKIE)).toBeUndefined()
  })
})
