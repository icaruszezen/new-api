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
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, test, vi } from 'vitest'

import type { AdminMonitor } from '../types'

const apiMocks = vi.hoisted(() => ({
  getChannelMonitoringConfig: vi.fn(),
  resetChannelMonitor: vi.fn(),
}))

vi.mock('../api', () => ({
  getChannelMonitoringConfig: apiMocks.getChannelMonitoringConfig,
  resetChannelMonitor: apiMocks.resetChannelMonitor,
  updateChannelMonitoringConfig: vi.fn(),
}))

vi.mock('@/features/users/api', () => ({
  getGroups: vi.fn().mockResolvedValue({ success: true, data: ['default'] }),
}))

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: (iconName?: string | null) =>
    iconName ? <span data-testid={`monitor-icon-${iconName}`} /> : null,
}))

const { ChannelMonitoringSettings } = await import('../settings')

function monitor(overrides: Partial<AdminMonitor> = {}): AdminMonitor {
  return {
    id: 'm1',
    name: 'Pro',
    group: 'default',
    model: 'gpt-5',
    enabled: true,
    sort: 0,
    resolved_icon: 'OpenAI',
    ...overrides,
  }
}

function renderSettings() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <ChannelMonitoringSettings />
    </QueryClientProvider>
  )
}

describe('ChannelMonitoringSettings load error', () => {
  test('shows an error instead of an empty monitor list when config fails', async () => {
    apiMocks.getChannelMonitoringConfig.mockRejectedValue(new Error('network'))

    renderSettings()

    expect(
      await screen.findByText('Failed to load channel monitoring settings.')
    ).toBeInTheDocument()
    expect(
      screen.queryByText('No monitors have been configured yet.')
    ).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument()
  })
})

describe('ChannelMonitoringSettings reset', () => {
  test('asks for confirmation then resets the selected monitor history', async () => {
    apiMocks.getChannelMonitoringConfig.mockResolvedValue({
      enabled: true,
      monitors: [monitor()],
    })
    apiMocks.resetChannelMonitor.mockResolvedValue({
      success: true,
      message: '',
      data: null,
    })

    const user = userEvent.setup()
    renderSettings()

    expect(await screen.findByText('Pro')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Reset history' }))

    const dialog = await screen.findByRole('alertdialog')
    expect(
      within(dialog).getByText(
        'This will delete all recorded samples, uptime stats, and ping data for Pro. Monitoring will start over.'
      )
    ).toBeInTheDocument()

    await user.click(
      within(dialog).getByRole('button', { name: 'Reset history' })
    )

    await waitFor(() => {
      expect(apiMocks.resetChannelMonitor).toHaveBeenCalledWith('m1')
    })
  })

  test('hides reset when the monitor has not been saved yet', async () => {
    apiMocks.getChannelMonitoringConfig.mockResolvedValue({
      enabled: true,
      monitors: [monitor({ id: '' })],
    })

    renderSettings()

    expect(await screen.findByText('Pro')).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Reset history' })
    ).not.toBeInTheDocument()
  })
})
