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

import type { IqCheck } from '../types'

const apiMocks = vi.hoisted(() => ({
  getGroupModels: vi.fn(),
}))

vi.mock('@/features/channel-monitoring/api', () => ({
  getGroupModels: apiMocks.getGroupModels,
}))

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: (iconName?: string | null) =>
    iconName ? <span data-testid={`check-icon-${iconName}`} /> : null,
}))

const { CheckDialog } = await import('../components/check-dialog')

function savedCheck(overrides: Partial<IqCheck> = {}): IqCheck {
  return {
    id: 'c1',
    name: 'Strawberry test',
    group: 'vip',
    model: 'gpt-5',
    prompt: 'How many r are in strawberry?',
    interval_minutes: 60,
    max_tokens: 8192,
    reasoning_summary: false,
    icon: '',
    enabled: true,
    sort: 0,
    ...overrides,
  }
}

function renderDialog(editData?: IqCheck, saveAccepted = true) {
  apiMocks.getGroupModels.mockResolvedValue(['gpt-5'])
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const onSave = vi.fn().mockResolvedValue(saveAccepted)
  const onOpenChange = vi.fn()
  const ui = (data?: IqCheck) => (
    <QueryClientProvider client={client}>
      <CheckDialog
        open
        onOpenChange={onOpenChange}
        editData={data}
        groups={['default', 'vip']}
        onSave={onSave}
      />
    </QueryClientProvider>
  )

  const { rerender } = render(ui(editData))

  return {
    onSave,
    onOpenChange,
    rerenderWith: (data?: IqCheck) => rerender(ui(data)),
  }
}

describe('CheckDialog', () => {
  test('starts a new check with the default interval and token budget', () => {
    renderDialog()

    expect(screen.getByLabelText('Check interval (minutes)')).toHaveValue(60)
    expect(screen.getByLabelText('Max output tokens')).toHaveValue(8192)
    expect(screen.getByRole('switch', { name: 'Enabled' })).toBeChecked()
    expect(
      screen.getByRole('combobox', { name: 'Request format' })
    ).toHaveTextContent('Auto detect (default)')
    expect(
      screen.getByRole('combobox', { name: 'Thinking depth' })
    ).toHaveTextContent('Model default')
    expect(
      screen.getByRole('switch', { name: 'Request thinking summary' })
    ).toBeChecked()
  })

  test('submits the chosen request format and thinking depth', async () => {
    const user = userEvent.setup()
    const { onSave } = renderDialog(
      savedCheck({ model: 'claude-opus-4-8', reasoning_effort: 'low' })
    )

    expect(
      screen.getByRole('combobox', { name: 'Thinking depth' })
    ).toHaveTextContent('Low')
    await user.click(screen.getByRole('combobox', { name: 'Request format' }))
    await user.click(
      await screen.findByRole('option', { name: 'Claude Messages' })
    )
    await user.click(screen.getByRole('combobox', { name: 'Thinking depth' }))
    await user.click(await screen.findByRole('option', { name: 'Extra high' }))
    await user.click(
      screen.getByRole('switch', { name: 'Request thinking summary' })
    )
    await user.click(screen.getByRole('button', { name: 'Update' }))

    await waitFor(() => {
      expect(onSave).toHaveBeenCalledWith(
        expect.objectContaining({
          endpoint_type: 'anthropic',
          reasoning_effort: 'xhigh',
          reasoning_summary: true,
        })
      )
    })
  })

  test('loads models scoped to the selected group', async () => {
    renderDialog(savedCheck())

    await waitFor(() => {
      expect(apiMocks.getGroupModels).toHaveBeenCalledWith('vip')
    })
  })

  test('rejects a prompt that is only whitespace', async () => {
    const user = userEvent.setup()
    const { onSave } = renderDialog(savedCheck({ prompt: '   ' }))

    await user.click(screen.getByRole('button', { name: 'Update' }))

    expect(await screen.findByText('Prompt is required')).toBeInTheDocument()
    expect(onSave).not.toHaveBeenCalled()
  })

  test('rejects an interval shorter than five minutes', async () => {
    const user = userEvent.setup()
    const { onSave } = renderDialog(savedCheck())

    const interval = screen.getByLabelText('Check interval (minutes)')
    await user.clear(interval)
    await user.type(interval, '4')
    await user.click(screen.getByRole('button', { name: 'Update' }))

    expect(
      await screen.findByText(
        'Enter a whole number of minutes between 5 and 10080'
      )
    ).toBeInTheDocument()
    expect(interval).toHaveAttribute('aria-invalid', 'true')
    expect(onSave).not.toHaveBeenCalled()
  })

  test('rejects an empty max tokens field instead of sending zero', async () => {
    const user = userEvent.setup()
    const { onSave } = renderDialog(savedCheck())

    await user.clear(screen.getByLabelText('Max output tokens'))
    await user.click(screen.getByRole('button', { name: 'Update' }))

    expect(
      await screen.findByText('Enter a whole number between 0 and 200000')
    ).toBeInTheDocument()
    expect(onSave).not.toHaveBeenCalled()
  })

  test('submits the edited check with a trimmed prompt and numeric fields', async () => {
    const user = userEvent.setup()
    const { onSave } = renderDialog(
      savedCheck({ prompt: '  Draw a clock as HTML.  ' })
    )

    const maxTokens = screen.getByLabelText('Max output tokens')
    await user.clear(maxTokens)
    await user.type(maxTokens, '0')
    await user.click(screen.getByRole('button', { name: 'Update' }))

    await waitFor(() => {
      expect(onSave).toHaveBeenCalledWith({
        name: 'Strawberry test',
        group: 'vip',
        model: 'gpt-5',
        prompt: 'Draw a clock as HTML.',
        interval_minutes: 60,
        max_tokens: 0,
        endpoint_type: '',
        reasoning_effort: '',
        reasoning_summary: false,
        icon: '',
        enabled: true,
      })
    })
  })

  test('closes once the server accepts the save', async () => {
    const user = userEvent.setup()
    const { onOpenChange } = renderDialog(savedCheck())

    await user.click(screen.getByRole('button', { name: 'Update' }))

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false)
    })
  })

  test('stays open with the typed prompt when the server rejects the save', async () => {
    const user = userEvent.setup()
    const { onSave, onOpenChange } = renderDialog(savedCheck(), false)

    const prompt = screen.getByLabelText('Prompt')
    await user.clear(prompt)
    await user.type(prompt, 'Write a page with a working calculator.')
    await user.click(screen.getByRole('button', { name: 'Update' }))

    await waitFor(() => {
      expect(onSave).toHaveBeenCalled()
    })
    expect(onOpenChange).not.toHaveBeenCalled()
    expect(prompt).toHaveValue('Write a page with a working calculator.')
  })

  test('keeps typed edits when the edited row is refreshed while open', async () => {
    const user = userEvent.setup()
    const { rerenderWith } = renderDialog(savedCheck())

    const prompt = screen.getByLabelText('Prompt')
    await user.clear(prompt)
    await user.type(prompt, 'Unsaved prompt')
    // A refetch after the check ran replaces the row with a new object.
    rerenderWith(savedCheck({ prompt: 'How many r are in strawberry?' }))

    expect(screen.getByLabelText('Prompt')).toHaveValue('Unsaved prompt')
  })

  test('counts prompt characters against the limit', () => {
    renderDialog(savedCheck({ prompt: 'abc' }))

    expect(screen.getByText('3 / 20000')).toBeInTheDocument()
  })
})
