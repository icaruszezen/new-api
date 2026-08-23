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
import { create } from 'zustand'

export const CONSOLE_PREVIEW_STORAGE_KEY = 'new-api.console-preview'

export type ConsolePreviewState = {
  previewUserConsole: boolean
  previewUserId: number | null
  enterPreview: (userId: number) => void
  exitPreview: () => void
}

type PersistedConsolePreview = {
  previewUserConsole: boolean
  previewUserId: number | null
}

function sessionStorageOrNull(): Storage | null {
  try {
    return sessionStorage
  } catch {
    return null
  }
}

export function readConsolePreviewState(): PersistedConsolePreview {
  const storage = sessionStorageOrNull()
  if (!storage) {
    return { previewUserConsole: false, previewUserId: null }
  }

  try {
    const raw = storage.getItem(CONSOLE_PREVIEW_STORAGE_KEY)
    if (!raw) {
      return { previewUserConsole: false, previewUserId: null }
    }

    const parsed: unknown = JSON.parse(raw)
    if (typeof parsed !== 'object' || parsed == null) {
      return { previewUserConsole: false, previewUserId: null }
    }

    const record = parsed as {
      previewUserConsole?: unknown
      previewUserId?: unknown
    }
    if (record.previewUserConsole !== true) {
      return { previewUserConsole: false, previewUserId: null }
    }
    if (
      typeof record.previewUserId !== 'number' ||
      !Number.isInteger(record.previewUserId)
    ) {
      return { previewUserConsole: false, previewUserId: null }
    }

    return {
      previewUserConsole: true,
      previewUserId: record.previewUserId,
    }
  } catch {
    return { previewUserConsole: false, previewUserId: null }
  }
}

function writeConsolePreviewState(state: PersistedConsolePreview) {
  const storage = sessionStorageOrNull()
  if (!storage) return

  try {
    if (!state.previewUserConsole || state.previewUserId == null) {
      storage.removeItem(CONSOLE_PREVIEW_STORAGE_KEY)
      return
    }

    storage.setItem(
      CONSOLE_PREVIEW_STORAGE_KEY,
      JSON.stringify({
        previewUserConsole: true,
        previewUserId: state.previewUserId,
      })
    )
  } catch {
    /* privacy mode or full storage — preview still works in memory */
  }
}

export function isConsolePreviewActive(
  state: Pick<ConsolePreviewState, 'previewUserConsole' | 'previewUserId'>,
  userId: number | undefined | null
): boolean {
  return (
    state.previewUserConsole && userId != null && state.previewUserId === userId
  )
}

const initialPreview = readConsolePreviewState()

export const useConsolePreviewStore = create<ConsolePreviewState>((set) => ({
  previewUserConsole: initialPreview.previewUserConsole,
  previewUserId: initialPreview.previewUserId,
  enterPreview: (userId) => {
    const next = { previewUserConsole: true, previewUserId: userId }
    writeConsolePreviewState(next)
    set(next)
  },
  exitPreview: () => {
    const next = { previewUserConsole: false, previewUserId: null }
    writeConsolePreviewState(next)
    set(next)
  },
}))
