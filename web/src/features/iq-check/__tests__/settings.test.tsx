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

import type { AdminIqCheck, IqCheckConfig } from '../types'

const apiMocks = vi.hoisted(() => ({
  getIqCheckConfig: vi.fn(),
  updateIqCheckConfig: vi.fn(),
  runIqCheck: vi.fn(),
}))

vi.mock('../api', () => ({
  getIqCheckConfig: apiMocks.getIqCheckConfig,
  updateIqCheckConfig: apiMocks.updateIqCheckConfig,
  runIqCheck: apiMocks.runIqCheck,
}))

vi.mock('@/features/users/api', () => ({
  getGroups: vi.fn().mockResolvedValue({ success: true, data: ['default'] }),
}))

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: (iconName?: string | null) =>
    iconName ? <span data-testid={`check-icon-${iconName}`} /> : null,
}))

const { IqCheckSettings } = await import('../settings')

function adminCheck(overrides: Partial<AdminIqCheck> = {}): AdminIqCheck {
  return {
    id: 'c1',
    name: 'Strawberry test',
    group: 'default',
    model: 'gpt-5',
    prompt: 'How many r are in strawberry?',
    interval_minutes: 60,
    max_tokens: 8192,
    icon: '',
    enabled: true,
    sort: 0,
    resolved_icon: 'OpenAI',
    last_started_at: 0,
    last_finished_at: 0,
    run_requested_at: 0,
    last_status: '',
    last_checked_at: 0,
    last_error: '',
    ...overrides,
  }
}

function config(overrides: Partial<IqCheckConfig> = {}): IqCheckConfig {
  return {
    enabled: true,
    history_limit: 10,
    request_timeout_seconds: 300,
    checks: [adminCheck()],
    ...overrides,
  }
}

function renderSettings(initial: IqCheckConfig) {
  apiMocks.getIqCheckConfig.mockResolvedValue(initial)
  apiMocks.updateIqCheckConfig.mockResolvedValue({
    success: true,
    message: '',
    data: initial,
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <IqCheckSettings />
    </QueryClientProvider>
  )
}

function rowOf(name: string) {
  const row = screen.getByText(name).closest('li')
  if (!row) throw new Error(`No row for ${name}`)
  return row
}

describe('IqCheckSettings', () => {
  test('saves the reordered list with sort rewritten from the new order', async () => {
    const user = userEvent.setup()
    renderSettings(
      config({
        checks: [
          adminCheck({ id: 'a', name: 'First', sort: 0 }),
          adminCheck({ id: 'b', name: 'Second', sort: 1 }),
        ],
      })
    )

    await screen.findByText('First')
    await user.click(
      within(rowOf('First')).getByRole('button', { name: 'Move down' })
    )

    await waitFor(() => {
      expect(apiMocks.updateIqCheckConfig).toHaveBeenCalledTimes(1)
    })
    const update = apiMocks.updateIqCheckConfig.mock.calls[0][0]
    expect(
      update.checks.map((check: AdminIqCheck) => [check.id, check.sort])
    ).toEqual([
      ['b', 0],
      ['a', 1],
    ])
    // Server-owned run state is never sent back.
    expect(update.checks[0]).not.toHaveProperty('last_started_at')
    expect(update.checks[0]).not.toHaveProperty('resolved_icon')
  })

  test('queues a manual run for the chosen check', async () => {
    const user = userEvent.setup()
    apiMocks.runIqCheck.mockResolvedValue({
      success: true,
      message: '',
      data: null,
    })
    renderSettings(config())

    await screen.findByText('Strawberry test')
    await user.click(screen.getByRole('button', { name: 'Run now' }))

    await waitFor(() => {
      expect(apiMocks.runIqCheck).toHaveBeenCalledWith('c1')
    })
  })

  test.each([
    { name: 'the page is disabled', initial: config({ enabled: false }) },
    {
      name: 'the check is disabled',
      initial: config({ checks: [adminCheck({ enabled: false })] }),
    },
  ])('disables Run now when $name', async ({ initial }) => {
    renderSettings(initial)

    await screen.findByText('Strawberry test')

    expect(screen.getByRole('button', { name: 'Run now' })).toBeDisabled()
  })

  test('shows the latest outcome with the raw upstream error', async () => {
    renderSettings(
      config({
        checks: [
          adminCheck({
            last_started_at: 1_760_000_000,
            last_finished_at: 1_760_000_100,
            last_checked_at: 1_760_000_100,
            last_status: 'failed',
            last_error: 'upstream 503: overloaded',
          }),
        ],
      })
    )

    await screen.findByText('Strawberry test')
    const row = rowOf('Strawberry test')

    expect(within(row).getByText('Failed')).toBeInTheDocument()
    expect(
      within(row).getByText('upstream 503: overloaded')
    ).toBeInTheDocument()
  })

  test('shows a requested run as queued', async () => {
    renderSettings(
      config({
        checks: [
          adminCheck({
            last_started_at: 1_760_000_000,
            last_finished_at: 1_760_000_100,
            run_requested_at: 1_760_000_200,
          }),
        ],
      })
    )

    await screen.findByText('Strawberry test')

    expect(
      within(rowOf('Strawberry test')).getByText('Queued')
    ).toBeInTheDocument()
  })

  test('asks for confirmation before deleting a check', async () => {
    const user = userEvent.setup()
    renderSettings(
      config({
        checks: [
          adminCheck({ id: 'a', name: 'First' }),
          adminCheck({ id: 'b', name: 'Second', sort: 1 }),
        ],
      })
    )

    await screen.findByText('First')
    await user.click(
      within(rowOf('First')).getByRole('button', { name: 'Delete' })
    )
    const dialog = await screen.findByRole('alertdialog')
    expect(apiMocks.updateIqCheckConfig).not.toHaveBeenCalled()

    await user.click(within(dialog).getByRole('button', { name: 'Delete' }))

    await waitFor(() => {
      expect(apiMocks.updateIqCheckConfig).toHaveBeenCalledTimes(1)
    })
    const update = apiMocks.updateIqCheckConfig.mock.calls[0][0]
    expect(update.checks.map((check: AdminIqCheck) => check.id)).toEqual(['b'])
  })

  test('saves history size and timeout together and rejects out-of-range values', async () => {
    const user = userEvent.setup()
    renderSettings(config())

    await screen.findByText('Strawberry test')
    const history = screen.getByLabelText('History size')
    const timeout = screen.getByLabelText('Request timeout (seconds)')

    await user.clear(history)
    await user.type(history, '51')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(
      await screen.findByText('Enter a whole number between 1 and 50')
    ).toBeInTheDocument()
    expect(apiMocks.updateIqCheckConfig).not.toHaveBeenCalled()

    await user.clear(history)
    await user.type(history, '20')
    await user.clear(timeout)
    await user.type(timeout, '600')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(apiMocks.updateIqCheckConfig).toHaveBeenCalledWith({
        history_limit: 20,
        request_timeout_seconds: 600,
      })
    })
  })

  // Saving replaces the whole list and deletes the history of missing checks,
  // so adding one before the real list is loaded would wipe every other check.
  test('shows an error and blocks adding checks when the config fails to load', async () => {
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
      await screen.findByText('Failed to load IQ check settings.')
    ).toBeInTheDocument()
    expect(
      screen.queryByText('No IQ checks have been configured yet.')
    ).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add check' })).toBeDisabled()
  })
})
