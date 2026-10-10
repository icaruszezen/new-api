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

import type { IqCheckResult, IqCheckStatus, IqCheckView } from '../types'

const apiMocks = vi.hoisted(() => ({
  getIqCheckStatus: vi.fn(),
  getIqCheckResults: vi.fn(),
}))

vi.mock('../api', () => ({
  getIqCheckStatus: apiMocks.getIqCheckStatus,
  getIqCheckResults: apiMocks.getIqCheckResults,
}))

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: (iconName?: string | null) =>
    iconName ? <span data-testid={`check-icon-${iconName}`} /> : null,
}))

const { IqCheck } = await import('../index')

function checkView(
  id: string,
  name: string,
  history: IqCheckView['history']
): IqCheckView {
  return {
    id,
    name,
    model: 'gpt-5',
    icon: 'OpenAI',
    prompt: `Prompt of ${name}`,
    interval_minutes: 60,
    running: false,
    queued: false,
    last_checked_at: history[0]?.checked_at ?? 0,
    next_check_at: 0,
    history,
  }
}

function resultFor(
  id: number,
  checkId: string,
  content: string
): IqCheckResult {
  return {
    id,
    check_id: checkId,
    checked_at: 1_760_000_000,
    status: 'success',
    model: 'gpt-5',
    prompt: '',
    content,
    content_truncated: false,
    reasoning: '',
    reasoning_truncated: false,
    finish_reason: 'stop',
    error_message: '',
    duration_ms: 1000,
    prompt_tokens: 10,
    completion_tokens: 20,
    reasoning_tokens: 0,
  }
}

const RESULTS: Record<number, IqCheckResult> = {
  11: resultFor(11, 'c1', 'Newest answer of the first check'),
  10: resultFor(10, 'c1', 'Older answer of the first check'),
  21: resultFor(21, 'c2', 'Newest answer of the second check'),
}

function mockResultsApi() {
  apiMocks.getIqCheckResults.mockImplementation(async (ids: number[]) =>
    ids.flatMap((id) => (RESULTS[id] ? [RESULTS[id]] : []))
  )
}

function renderPage(status: IqCheckStatus) {
  apiMocks.getIqCheckStatus.mockResolvedValue(status)
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <IqCheck />
    </QueryClientProvider>
  )
}

describe('IqCheck page', () => {
  test('explains that the page is off when the feature is disabled', async () => {
    renderPage({ enabled: false, checks: [] })

    expect(
      await screen.findByText('The IQ check page is not enabled.')
    ).toBeInTheDocument()
    expect(apiMocks.getIqCheckResults).not.toHaveBeenCalled()
  })

  test('shows a visible title and an empty state when no checks exist', async () => {
    renderPage({ enabled: true, checks: [] })

    expect(
      await screen.findByText('No IQ checks have been configured yet.')
    ).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { level: 1, name: 'IQ Check' })
    ).not.toHaveClass('sr-only')
  })

  test('renders one card per check and loads every newest answer in one request', async () => {
    mockResultsApi()
    renderPage({
      enabled: true,
      checks: [
        checkView('c1', 'First check', [
          {
            id: 11,
            checked_at: 1_760_000_000,
            status: 'success',
            duration_ms: 1000,
            completion_tokens: 20,
          },
        ]),
        checkView('c2', 'Second check', [
          {
            id: 21,
            checked_at: 1_760_000_000,
            status: 'success',
            duration_ms: 1000,
            completion_tokens: 20,
          },
        ]),
        checkView('c3', 'Never run check', []),
      ],
    })

    expect(
      await screen.findByText('Newest answer of the first check')
    ).toBeInTheDocument()
    expect(
      screen.getByText('Newest answer of the second check')
    ).toBeInTheDocument()
    expect(screen.getByText('This check has not run yet.')).toBeInTheDocument()
    expect(apiMocks.getIqCheckResults).toHaveBeenCalledTimes(1)
    expect(apiMocks.getIqCheckResults).toHaveBeenCalledWith([11, 21])
  })

  test('loads only the picked run and reuses cached answers when switching back', async () => {
    mockResultsApi()
    const user = userEvent.setup()
    renderPage({
      enabled: true,
      checks: [
        checkView('c1', 'First check', [
          {
            id: 11,
            checked_at: 1_760_000_000,
            status: 'success',
            duration_ms: 1000,
            completion_tokens: 20,
          },
          {
            id: 10,
            checked_at: 1_759_990_000,
            status: 'failed',
            duration_ms: 1000,
            completion_tokens: 20,
          },
        ]),
      ],
    })
    expect(
      await screen.findByText('Newest answer of the first check')
    ).toBeInTheDocument()

    await user.click(screen.getByLabelText('Run'))
    await user.click(await screen.findByRole('option', { name: /Failed/ }))

    expect(
      await screen.findByText('Older answer of the first check')
    ).toBeInTheDocument()
    expect(apiMocks.getIqCheckResults).toHaveBeenLastCalledWith([10])

    await user.click(screen.getByLabelText('Run'))
    await user.click(await screen.findByRole('option', { name: /Completed/ }))

    expect(
      await screen.findByText('Newest answer of the first check')
    ).toBeInTheDocument()
    await waitFor(() => {
      expect(apiMocks.getIqCheckResults).toHaveBeenCalledTimes(2)
    })
  })

  test('tells the visitor when a run is no longer available', async () => {
    mockResultsApi()
    renderPage({
      enabled: true,
      checks: [
        checkView('c1', 'First check', [
          {
            id: 99,
            checked_at: 1_760_000_000,
            status: 'success',
            duration_ms: 1000,
            completion_tokens: 20,
          },
        ]),
      ],
    })

    expect(
      await screen.findByText('This result is no longer available.')
    ).toBeInTheDocument()
  })
})
