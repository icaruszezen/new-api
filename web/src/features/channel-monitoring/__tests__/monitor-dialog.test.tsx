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
import { describe, expect, test, vi } from 'vitest'

import type { ChannelMonitor } from '../types'
import { monitorPairKey } from '../constants'

const apiMocks = vi.hoisted(() => ({
  getGroupModels: vi.fn(),
}))

vi.mock('../api', () => ({
  getGroupModels: apiMocks.getGroupModels,
}))

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: (iconName?: string | null) =>
    iconName ? <span data-testid={`monitor-icon-${iconName}`} /> : null,
}))

const { MonitorDialog } = await import('../components/monitor-dialog')

function renderDialog(options: {
  editData?: ChannelMonitor
  takenPairs?: string[]
  onSave?: (values: unknown) => void
}) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const onSave = options.onSave ?? vi.fn()

  render(
    <QueryClientProvider client={client}>
      <MonitorDialog
        open
        onOpenChange={vi.fn()}
        editData={options.editData}
        groups={['default', 'vip']}
        takenPairs={options.takenPairs ?? []}
        onSave={onSave}
      />
    </QueryClientProvider>
  )

  return { onSave }
}

describe('MonitorDialog', () => {
  test('loads models scoped to the selected group', async () => {
    apiMocks.getGroupModels.mockResolvedValue(['gpt-5', 'claude-4'])

    renderDialog({
      editData: {
        id: 'm1',
        name: 'Pro',
        group: 'vip',
        model: 'gpt-5',
        enabled: true,
        sort: 0,
      },
    })

    await waitFor(() => {
      expect(apiMocks.getGroupModels).toHaveBeenCalledWith('vip')
    })
  })

  test('does not fetch models before a group is chosen', async () => {
    apiMocks.getGroupModels.mockResolvedValue([])

    renderDialog({})

    // A group-less request would return every model in the deployment, which is
    // not routable for the monitor being created.
    await waitFor(() => {
      expect(apiMocks.getGroupModels).not.toHaveBeenCalled()
    })
    expect(
      screen.getByPlaceholderText('Select a group first')
    ).toBeInTheDocument()
  })

  test('blocks submission and reports the clash for an already monitored pair', async () => {
    apiMocks.getGroupModels.mockResolvedValue(['gpt-5'])
    const user = userEvent.setup()

    const { onSave } = renderDialog({
      editData: {
        id: 'm1',
        name: 'Pro',
        group: 'vip',
        model: 'gpt-5',
        enabled: true,
        sort: 0,
      },
      takenPairs: [monitorPairKey('vip', 'gpt-5')],
    })

    await user.click(screen.getByRole('button', { name: 'Update' }))

    await waitFor(() => {
      expect(
        screen.getByText('This group and model pair is already monitored.')
      ).toBeInTheDocument()
    })
    expect(onSave).not.toHaveBeenCalled()
  })

  test('submits the edited monitor when the pair is free', async () => {
    apiMocks.getGroupModels.mockResolvedValue(['gpt-5'])
    const user = userEvent.setup()

    const { onSave } = renderDialog({
      editData: {
        id: 'm1',
        name: 'Pro',
        group: 'vip',
        model: 'gpt-5',
        icon: 'OpenAI',
        enabled: true,
        sort: 0,
      },
    })

    await user.click(screen.getByRole('button', { name: 'Update' }))

    await waitFor(() => {
      expect(onSave).toHaveBeenCalledWith({
        name: 'Pro',
        group: 'vip',
        model: 'gpt-5',
        icon: 'OpenAI',
        enabled: true,
      })
    })
  })

  test('rejects an empty display name', async () => {
    apiMocks.getGroupModels.mockResolvedValue(['gpt-5'])
    const user = userEvent.setup()

    const { onSave } = renderDialog({
      editData: {
        id: 'm1',
        name: '',
        group: 'vip',
        model: 'gpt-5',
        enabled: true,
        sort: 0,
      },
    })

    await user.click(screen.getByRole('button', { name: 'Update' }))

    await waitFor(() => {
      expect(onSave).not.toHaveBeenCalled()
    })
  })
})
