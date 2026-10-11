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
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import type { IqCheckConfig } from '../types'

const apiMocks = vi.hoisted(() => ({
  getIqCheckConfig: vi.fn(),
  updateIqCheckConfig: vi.fn(),
}))

vi.mock('../api', () => ({
  getIqCheckConfig: apiMocks.getIqCheckConfig,
  updateIqCheckConfig: apiMocks.updateIqCheckConfig,
}))

const { IqCheckSettings } = await import('../settings')

function renderSettings(initial: IqCheckConfig) {
  apiMocks.getIqCheckConfig.mockResolvedValue(initial)
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <IqCheckSettings />
    </QueryClientProvider>
  )
}

describe('IqCheckSettings', () => {
  beforeEach(() => {
    apiMocks.getIqCheckConfig.mockReset()
    apiMocks.updateIqCheckConfig.mockReset()
    apiMocks.updateIqCheckConfig.mockImplementation(
      async (config: IqCheckConfig) => ({
        success: true,
        message: '',
        data: config,
      })
    )
  })

  test('shows the saved entry state and address once the config loads', async () => {
    renderSettings({ enabled: true, url: 'https://iq.example.com' })

    expect(await screen.findByRole('textbox')).toHaveValue(
      'https://iq.example.com'
    )
    expect(screen.getByRole('switch')).toBeChecked()
  })

  test('saves the trimmed address together with the turned-on entry', async () => {
    const user = userEvent.setup()
    renderSettings({ enabled: false, url: '' })

    const input = await screen.findByRole('textbox')
    await user.type(input, '  https://iq.example.com/check  ')
    await user.click(screen.getByRole('switch'))
    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(apiMocks.updateIqCheckConfig).toHaveBeenCalledTimes(1)
    })
    expect(apiMocks.updateIqCheckConfig.mock.calls[0][0]).toEqual({
      enabled: true,
      url: 'https://iq.example.com/check',
    })
  })

  test('blocks an address that is not a full http(s) URL without saving', async () => {
    const user = userEvent.setup()
    renderSettings({ enabled: false, url: '' })

    const input = await screen.findByRole('textbox')
    await user.type(input, 'javascript:alert(1)')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(input).toHaveAttribute('aria-invalid', 'true')
    })
    expect(apiMocks.updateIqCheckConfig).not.toHaveBeenCalled()
  })

  test('requires an address before the entry can be turned on', async () => {
    const user = userEvent.setup()
    renderSettings({ enabled: false, url: '' })

    await user.click(await screen.findByRole('switch'))
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(
      await screen.findByText(
        'Enter the website address before turning on the IQ check entry'
      )
    ).toBeInTheDocument()
    expect(apiMocks.updateIqCheckConfig).not.toHaveBeenCalled()
  })

  test('shows a retry action instead of the form when the config fails to load', async () => {
    apiMocks.getIqCheckConfig.mockRejectedValue(new Error('network'))
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    render(
      <QueryClientProvider client={client}>
        <IqCheckSettings />
      </QueryClientProvider>
    )

    expect(
      await screen.findByRole('button', { name: 'Retry' })
    ).toBeInTheDocument()
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
  })
})
