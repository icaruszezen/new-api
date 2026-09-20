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
import i18next from 'i18next'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import zh from '@/i18n/locales/zh.json'

import type { ErrorMessageOverride } from '../types'

const apiMocks = vi.hoisted(() => ({
  getErrorMessageOverrides: vi.fn(),
  createErrorMessageOverride: vi.fn(),
  updateErrorMessageOverride: vi.fn(),
  deleteErrorMessageOverride: vi.fn(),
  getChannelOptions: vi.fn(),
}))

vi.mock('../api', () => apiMocks)

const { ErrorMessageOverrides } = await import('..')

function buildOverride(
  overrides: Partial<ErrorMessageOverride> = {}
): ErrorMessageOverride {
  return {
    id: 1,
    match_substring: 'insufficient_quota',
    replacement_message: 'Service is busy, please retry later',
    channel_id: 0,
    priority: 0,
    enabled: true,
    created_time: 0,
    updated_time: 0,
    ...overrides,
  }
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const Wrapper = (props: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {props.children}
    </QueryClientProvider>
  )
  return render(<ErrorMessageOverrides />, { wrapper: Wrapper })
}

afterEach(async () => {
  vi.clearAllMocks()
  i18next.options.keySeparator = '.'
  i18next.options.nsSeparator = ':'
  if (i18next.language !== 'en') {
    await i18next.changeLanguage('en')
  }
})

describe('error message override page', () => {
  test('shows an empty state when no rules are configured', async () => {
    apiMocks.getErrorMessageOverrides.mockResolvedValue([])

    renderPage()

    expect(await screen.findByText('No override rules yet')).toBeInTheDocument()
  })

  test('labels a global rule as applying to all channels', async () => {
    apiMocks.getErrorMessageOverrides.mockResolvedValue([buildOverride()])

    renderPage()

    expect(await screen.findByText('insufficient_quota')).toBeInTheDocument()
    expect(screen.getByText('All channels')).toBeInTheDocument()
    expect(screen.getByText('Enabled')).toBeInTheDocument()
  })

  test('shows the channel name for a channel-scoped rule', async () => {
    apiMocks.getErrorMessageOverrides.mockResolvedValue([
      buildOverride({ channel_id: 42, channel_name: 'azure-east' }),
    ])

    renderPage()

    expect(await screen.findByText('azure-east')).toBeInTheDocument()
    expect(screen.queryByText('All channels')).not.toBeInTheDocument()
  })

  test('falls back to the channel id when the channel was deleted', async () => {
    apiMocks.getErrorMessageOverrides.mockResolvedValue([
      buildOverride({ channel_id: 42, channel_name: '' }),
    ])

    renderPage()

    expect(await screen.findByText('#42')).toBeInTheDocument()
  })

  test('marks a disabled rule so admins can tell it is inactive', async () => {
    apiMocks.getErrorMessageOverrides.mockResolvedValue([
      buildOverride({ enabled: false }),
    ])

    renderPage()

    expect(await screen.findByText('Disabled')).toBeInTheDocument()
  })

  test('creates a global rule from the add dialog', async () => {
    const user = userEvent.setup()
    apiMocks.getErrorMessageOverrides.mockResolvedValue([])
    apiMocks.createErrorMessageOverride.mockResolvedValue({
      success: true,
      message: '',
      data: buildOverride(),
    })

    renderPage()
    await screen.findByText('No override rules yet')

    await user.click(screen.getByRole('button', { name: 'Add Override Rule' }))
    await user.type(
      await screen.findByLabelText('Match text'),
      'insufficient_quota'
    )
    await user.type(
      screen.getByLabelText('Replacement message'),
      'Service is busy'
    )
    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(apiMocks.createErrorMessageOverride).toHaveBeenCalledWith({
        match_substring: 'insufficient_quota',
        replacement_message: 'Service is busy',
        channel_id: 0,
        priority: 0,
        enabled: true,
      })
    })
  })

  test('blocks saving when the replacement message is missing', async () => {
    const user = userEvent.setup()
    apiMocks.getErrorMessageOverrides.mockResolvedValue([])

    renderPage()
    await screen.findByText('No override rules yet')

    await user.click(screen.getByRole('button', { name: 'Add Override Rule' }))
    await user.type(await screen.findByLabelText('Match text'), 'boom')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(
      await screen.findByText('Replacement message cannot be empty')
    ).toBeInTheDocument()
    expect(apiMocks.createErrorMessageOverride).not.toHaveBeenCalled()
  })

  test('deletes a rule after the confirmation is accepted', async () => {
    const user = userEvent.setup()
    apiMocks.getErrorMessageOverrides.mockResolvedValue([buildOverride()])
    apiMocks.deleteErrorMessageOverride.mockResolvedValue({
      success: true,
      message: '',
      data: null,
    })

    renderPage()
    await screen.findByText('insufficient_quota')

    await user.click(screen.getByRole('button', { name: 'Actions' }))
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }))
    await user.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => {
      expect(apiMocks.deleteErrorMessageOverride).toHaveBeenCalledWith(1)
    })
  })

  test('renders Chinese chrome and the add dialog when zh is active', async () => {
    const user = userEvent.setup()
    i18next.options.keySeparator = false
    i18next.options.nsSeparator = false
    i18next.addResourceBundle('zhCN', 'translation', zh.translation, true, true)
    await i18next.changeLanguage('zhCN')
    apiMocks.getErrorMessageOverrides.mockResolvedValue([])

    renderPage()

    expect(
      await screen.findByRole('heading', { name: '报错信息覆盖' })
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: '添加覆盖规则' })
    ).toBeInTheDocument()
    expect(await screen.findByText('暂无覆盖规则')).toBeInTheDocument()
    expect(screen.getByText('匹配文本')).toBeInTheDocument()
    expect(screen.getByText('替换文案')).toBeInTheDocument()
    expect(screen.getByText('生效范围')).toBeInTheDocument()
    expect(
      screen.getByText(
        '当上游报错信息包含匹配文本时，返回给用户的报错会被整段替换。',
        { exact: false }
      )
    ).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '添加覆盖规则' }))

    expect(
      await screen.findByRole('heading', { name: '添加覆盖规则' })
    ).toBeInTheDocument()
    expect(screen.getByLabelText('匹配文本')).toBeInTheDocument()
    expect(screen.getByLabelText('替换文案')).toBeInTheDocument()
  })
})
