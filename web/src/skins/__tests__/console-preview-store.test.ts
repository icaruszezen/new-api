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
  CONSOLE_PREVIEW_STORAGE_KEY,
  isConsolePreviewActive,
  readConsolePreviewState,
  useConsolePreviewStore,
} from '../console-preview-store'

afterEach(() => {
  useConsolePreviewStore.getState().exitPreview()
  sessionStorage.clear()
})

describe('console preview store', () => {
  test('writes the preview flag to sessionStorage immediately', () => {
    useConsolePreviewStore.getState().enterPreview(7)

    expect(useConsolePreviewStore.getState()).toMatchObject({
      previewUserConsole: true,
      previewUserId: 7,
    })
    expect(sessionStorage.getItem(CONSOLE_PREVIEW_STORAGE_KEY)).toBe(
      JSON.stringify({ previewUserConsole: true, previewUserId: 7 })
    )
  })

  test('clears sessionStorage when preview ends', () => {
    useConsolePreviewStore.getState().enterPreview(7)
    useConsolePreviewStore.getState().exitPreview()

    expect(useConsolePreviewStore.getState()).toMatchObject({
      previewUserConsole: false,
      previewUserId: null,
    })
    expect(sessionStorage.getItem(CONSOLE_PREVIEW_STORAGE_KEY)).toBeNull()
  })

  test('treats a persisted preview as inactive for a different user', () => {
    useConsolePreviewStore.getState().enterPreview(1)
    const state = useConsolePreviewStore.getState()

    expect(isConsolePreviewActive(state, 1)).toBe(true)
    expect(isConsolePreviewActive(state, 2)).toBe(false)
    expect(isConsolePreviewActive(state, undefined)).toBe(false)
  })

  test('reads a well-formed session snapshot', () => {
    sessionStorage.setItem(
      CONSOLE_PREVIEW_STORAGE_KEY,
      JSON.stringify({ previewUserConsole: true, previewUserId: 4 })
    )

    expect(readConsolePreviewState()).toEqual({
      previewUserConsole: true,
      previewUserId: 4,
    })
  })

  test.each([
    ['missing key', null],
    ['empty string', ''],
    ['invalid json', '{'],
    ['non-object', '"yes"'],
    [
      'inactive flag',
      JSON.stringify({ previewUserConsole: false, previewUserId: 1 }),
    ],
    [
      'non-integer user id',
      JSON.stringify({ previewUserConsole: true, previewUserId: 1.5 }),
    ],
  ])('falls back to an inactive preview for %s', (_label, raw) => {
    if (raw == null) {
      sessionStorage.removeItem(CONSOLE_PREVIEW_STORAGE_KEY)
    } else {
      sessionStorage.setItem(CONSOLE_PREVIEW_STORAGE_KEY, raw)
    }

    expect(readConsolePreviewState()).toEqual({
      previewUserConsole: false,
      previewUserId: null,
    })
  })
})
