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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

const mocks = vi.hoisted(() => ({
  getFirstTokenErrorStat: vi.fn(),
  toastError: vi.fn(),
}))

vi.mock('../../api', () => ({
  getFirstTokenErrorStat: mocks.getFirstTokenErrorStat,
}))

vi.mock('@/features/system-settings/api', () => ({
  updateSystemOption: vi.fn(),
}))

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: (...args: unknown[]) => mocks.toastError(...args),
  },
}))

const { FirstTokenErrorSettingsBar } = await import('../settings-bar')

function renderBar() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <FirstTokenErrorSettingsBar />
    </QueryClientProvider>
  )
}

function setRole(role: number) {
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'tester',
    role,
  })
}

describe('first-token error settings bar', () => {
  afterEach(() => {
    useAuthStore.getState().auth.reset()
    mocks.getFirstTokenErrorStat.mockReset()
    mocks.toastError.mockReset()
  })

  test('toasts when the stat request fails', async () => {
    setRole(ROLE.SUPER_ADMIN)
    mocks.getFirstTokenErrorStat.mockResolvedValue({
      success: false,
      message: 'stat failed',
    })

    renderBar()

    await waitFor(() => {
      expect(mocks.toastError).toHaveBeenCalledWith('stat failed')
    })
    expect(screen.getByText('Failed to load')).toBeInTheDocument()
    expect(
      screen.queryByRole('switch', {
        name: /Correct first-token billing errors/i,
      })
    ).toBeNull()
  })

  test('shows a read-only hint for admins who cannot edit', async () => {
    setRole(ROLE.ADMIN)
    mocks.getFirstTokenErrorStat.mockResolvedValue({
      success: true,
      data: {
        count: 0,
        would_be_quota: 0,
        body_count: 0,
        body_capture_enabled: false,
        correction_enabled: true,
        treat_all_output_one_enabled: false,
        log_enabled: true,
        log_max_keep: 5000,
      },
    })

    renderBar()

    expect(
      await screen.findByText('Only super admins can change these settings.')
    ).toBeInTheDocument()
    expect(
      screen.getByRole('switch', {
        name: /Correct first-token billing errors/i,
      })
    ).toHaveAttribute('aria-disabled', 'true')
  })
})
