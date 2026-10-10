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
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, test, vi } from 'vitest'

import type { ResultDetailState } from '../hooks/use-result-details'
import type { IqCheckHistoryItem, IqCheckResult, IqCheckView } from '../types'

// `@lobehub/icons` cannot be resolved under vitest; only the external icon
// boundary is stubbed, the card itself is real.
vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: (iconName?: string | null) =>
    iconName ? <span data-testid={`check-icon-${iconName}`} /> : null,
}))

const { CheckCard } = await import('../components/check-card')

const NEWEST: IqCheckHistoryItem = {
  id: 12,
  checked_at: 1_760_000_000,
  status: 'success',
  duration_ms: 4200,
  completion_tokens: 1234,
}

const OLDER: IqCheckHistoryItem = {
  id: 7,
  checked_at: 1_759_990_000,
  status: 'failed',
  duration_ms: 300_000,
  completion_tokens: 0,
}

function check(overrides: Partial<IqCheckView> = {}): IqCheckView {
  return {
    id: 'c1',
    name: 'Strawberry test',
    model: 'gpt-5',
    icon: 'OpenAI',
    prompt: 'How many r are in strawberry?',
    interval_minutes: 60,
    running: false,
    queued: false,
    last_checked_at: NEWEST.checked_at,
    next_check_at: NEWEST.checked_at + 3600,
    history: [NEWEST, OLDER],
    ...overrides,
  }
}

function result(overrides: Partial<IqCheckResult> = {}): IqCheckResult {
  return {
    id: NEWEST.id,
    check_id: 'c1',
    checked_at: NEWEST.checked_at,
    status: 'success',
    model: 'gpt-5',
    prompt: 'How many r are in strawberry?',
    content: 'There are 3 r letters.',
    content_truncated: false,
    reasoning: '',
    reasoning_truncated: false,
    finish_reason: 'stop',
    error_message: '',
    duration_ms: NEWEST.duration_ms,
    prompt_tokens: 20,
    completion_tokens: NEWEST.completion_tokens,
    reasoning_tokens: 0,
    ...overrides,
  }
}

function renderCard(options: {
  check?: IqCheckView
  run?: IqCheckHistoryItem
  result?: IqCheckResult
  resultState?: ResultDetailState
  onSelectRun?: (runId: number) => void
}) {
  const cardCheck = options.check ?? check()
  const onSelectRun = options.onSelectRun ?? vi.fn()
  render(
    <CheckCard
      check={cardCheck}
      run={'run' in options ? options.run : cardCheck.history[0]}
      result={options.result}
      resultState={options.resultState ?? 'ready'}
      onSelectRun={onSelectRun}
      onRetryResult={vi.fn()}
    />
  )
  return { onSelectRun }
}

describe('CheckCard', () => {
  test('shows a completed text answer with its prompt and model', () => {
    renderCard({ result: result() })

    expect(
      screen.getByRole('heading', { name: 'Strawberry test' })
    ).toBeInTheDocument()
    expect(screen.getByText('gpt-5')).toBeInTheDocument()
    expect(screen.getByText('Completed')).toBeInTheDocument()
    expect(
      screen.getByText('How many r are in strawberry?')
    ).toBeInTheDocument()
    expect(screen.getByText('There are 3 r letters.')).toBeInTheDocument()
    expect(screen.getByText('Every 60 min')).toBeInTheDocument()
    expect(screen.getByText('1,234 output tokens')).toBeInTheDocument()
    expect(screen.queryByTitle('HTML preview')).not.toBeInTheDocument()
  })

  test('renders HTML embedded in the answer as a sandboxed preview', () => {
    renderCard({
      result: result({
        content: 'Here you go:\n```html\n<h1>Clock</h1>\n```\nEnjoy.',
      }),
    })

    expect(screen.getByText('Here you go:')).toBeInTheDocument()
    expect(screen.getByText('Enjoy.')).toBeInTheDocument()
    expect(screen.getByTitle('HTML preview')).toHaveAttribute(
      'srcdoc',
      '<h1>Clock</h1>'
    )
  })

  test('shows the error of a failed run and keeps its partial answer', () => {
    renderCard({
      check: check({ history: [OLDER] }),
      result: result({
        id: OLDER.id,
        status: 'failed',
        content: 'Partial answer',
        error_message: 'timed out after 300 seconds',
      }),
    })

    const alert = screen.getByRole('alert')
    expect(within(alert).getByText('Check failed')).toBeInTheDocument()
    expect(
      within(alert).getByText('timed out after 300 seconds')
    ).toBeInTheDocument()
    expect(screen.getByText('Partial answer')).toBeInTheDocument()
    expect(screen.getByText('Failed')).toBeInTheDocument()
  })

  test('explains a check that has never run', () => {
    renderCard({
      check: check({ history: [], last_checked_at: 0, next_check_at: 0 }),
      run: undefined,
    })

    expect(screen.getByText('Not checked yet')).toBeInTheDocument()
    expect(screen.getByText('This check has not run yet.')).toBeInTheDocument()
    expect(screen.getByText('Next check soon')).toBeInTheDocument()
    expect(screen.queryByLabelText('Run')).not.toBeInTheDocument()
  })

  test('reports a queued first run instead of the never-run badge', () => {
    renderCard({
      check: check({ history: [], queued: true }),
      run: undefined,
    })

    expect(screen.getByText('Queued')).toBeInTheDocument()
    expect(screen.queryByText('Not checked yet')).not.toBeInTheDocument()
    expect(
      screen.getByText('The first check is in progress.')
    ).toBeInTheDocument()
  })

  test('asks for another run when one is picked from the history', async () => {
    const user = userEvent.setup()
    const { onSelectRun } = renderCard({ result: result() })

    await user.click(screen.getByLabelText('Run'))
    await user.click(await screen.findByRole('option', { name: /Failed/ }))

    expect(onSelectRun).toHaveBeenCalledWith(OLDER.id)
  })

  test('shows the prompt the run used and notes that it has since changed', () => {
    renderCard({
      check: check({ prompt: 'New prompt' }),
      result: result({ prompt: 'Old prompt' }),
    })

    expect(screen.getByText('Old prompt')).toBeInTheDocument()
    expect(screen.queryByText('New prompt')).not.toBeInTheDocument()
    expect(
      screen.getByText('This run used an earlier version of the prompt.')
    ).toBeInTheDocument()
  })

  test('warns when max tokens cut the answer short', () => {
    renderCard({ result: result({ finish_reason: 'length' }) })

    expect(screen.getByText('Truncated by max tokens')).toBeInTheDocument()
  })

  test('keeps the thinking collapsed until it is opened', async () => {
    const user = userEvent.setup()
    renderCard({
      result: result({ reasoning: 'Counting letters one by one' }),
    })

    const toggle = screen.getByRole('button', { name: 'Thinking' })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(
      screen.queryByText('Counting letters one by one')
    ).not.toBeInTheDocument()

    await user.click(toggle)

    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Counting letters one by one')).toBeInTheDocument()
  })

  test('opens the full answer in a dialog', async () => {
    const user = userEvent.setup()
    renderCard({ result: result() })

    await user.click(screen.getByRole('button', { name: 'Expand' }))

    const dialog = await screen.findByRole('dialog')
    expect(
      within(dialog).getByText('There are 3 r letters.')
    ).toBeInTheDocument()
  })

  test('shows a loading placeholder until the output arrives', () => {
    renderCard({ resultState: 'loading' })

    expect(screen.getByRole('button', { name: 'Expand' })).toBeDisabled()
    expect(screen.queryByText('There are 3 r letters.')).not.toBeInTheDocument()
  })
})
