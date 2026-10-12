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
import i18next from 'i18next'
import type { ReactNode } from 'react'
import { Toaster, toast } from 'sonner'
import { afterEach, describe, expect, test, vi } from 'vitest'

import zh from '@/i18n/locales/zh.json'

import type { BillingCorrectionRule } from '../types'

const apiMocks = vi.hoisted(() => ({
  getBillingCorrectionRules: vi.fn(),
  createBillingCorrectionRule: vi.fn(),
  updateBillingCorrectionRule: vi.fn(),
  deleteBillingCorrectionRule: vi.fn(),
  getChannelOptions: vi.fn(),
  getBillingCorrectionRecords: vi.fn(),
  markBillingCorrectionRecordsRefunded: vi.fn(),
  unmarkBillingCorrectionRecordsRefunded: vi.fn(),
  creditBillingCorrectionRecords: vi.fn(),
  checkBillingCorrectionRecordsUpstream: vi.fn(),
  testBillingCorrectionUpstream: vi.fn(),
}))

vi.mock('../api', () => apiMocks)

const { BillingCorrections } = await import('..')

function buildRule(
  overrides: Partial<BillingCorrectionRule> = {}
): BillingCorrectionRule {
  return {
    id: 1,
    channel_id: 5,
    prompt_tokens_threshold: 2_000_000,
    enabled: true,
    upstream_check_enabled: false,
    upstream_base_url: '',
    upstream_user_id: 0,
    upstream_access_token_set: false,
    upstream_check_since: 0,
    auto_refund: false,
    created_time: 0,
    updated_time: 0,
    channel_name: 'heartbeat-upstream',
    ...overrides,
  }
}

const NO_UPSTREAM_PAYLOAD = {
  upstream_check_enabled: false,
  upstream_base_url: '',
  upstream_user_id: 0,
  auto_refund: false,
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const Wrapper = (props: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {props.children}
      <Toaster duration={60_000} />
    </QueryClientProvider>
  )
  return render(<BillingCorrections />, { wrapper: Wrapper })
}

afterEach(async () => {
  toast.dismiss()
  vi.clearAllMocks()
  i18next.options.keySeparator = '.'
  i18next.options.nsSeparator = ':'
  if (i18next.language !== 'en') {
    await i18next.changeLanguage('en')
  }
})

describe('billing correction page', () => {
  test('shows an empty state when no rules are configured', async () => {
    apiMocks.getBillingCorrectionRules.mockResolvedValue([])

    renderPage()

    expect(
      await screen.findByText('No billing correction rules yet')
    ).toBeInTheDocument()
  })

  test('lists a rule with its channel name and a grouped token threshold', async () => {
    apiMocks.getBillingCorrectionRules.mockResolvedValue([buildRule()])

    renderPage()

    expect(await screen.findByText('heartbeat-upstream')).toBeInTheDocument()
    expect(screen.getByText(/^2\D000\D000 tokens$/)).toBeInTheDocument()
    expect(screen.getByText('Enabled')).toBeInTheDocument()
  })

  test('falls back to the channel id when the channel was deleted', async () => {
    apiMocks.getBillingCorrectionRules.mockResolvedValue([
      buildRule({ channel_id: 42, channel_name: '', enabled: false }),
    ])

    renderPage()

    expect(await screen.findByText('#42')).toBeInTheDocument()
    expect(screen.getByText('Disabled')).toBeInTheDocument()
  })

  test('opening the add dialog loads channels and prefills the default threshold', async () => {
    const user = userEvent.setup()
    apiMocks.getBillingCorrectionRules.mockResolvedValue([])
    apiMocks.getChannelOptions.mockResolvedValue([{ id: 5, name: 'primary' }])

    renderPage()
    await screen.findByText('No billing correction rules yet')
    expect(apiMocks.getChannelOptions).not.toHaveBeenCalled()

    await user.click(
      screen.getByRole('button', { name: 'Add Correction Rule' })
    )

    expect(
      await screen.findByRole('heading', { name: 'Add Correction Rule' })
    ).toBeInTheDocument()
    expect(screen.getByLabelText('Input token threshold')).toHaveValue(
      1_000_000
    )
    expect(screen.getByRole('switch', { name: 'Enabled' })).toBeChecked()
    await waitFor(() => {
      expect(apiMocks.getChannelOptions).toHaveBeenCalledTimes(1)
    })
  })

  test('creates a rule for the picked channel with the default threshold', async () => {
    const user = userEvent.setup()
    apiMocks.getBillingCorrectionRules.mockResolvedValue([])
    apiMocks.getChannelOptions.mockResolvedValue([{ id: 5, name: 'primary' }])
    apiMocks.createBillingCorrectionRule.mockResolvedValue({
      success: true,
      message: '',
      data: buildRule(),
    })

    renderPage()
    await screen.findByText('No billing correction rules yet')

    await user.click(
      screen.getByRole('button', { name: 'Add Correction Rule' })
    )
    await user.click(await screen.findByRole('combobox', { name: 'Channel' }))
    await user.click(await screen.findByRole('option', { name: '#5 primary' }))
    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(apiMocks.createBillingCorrectionRule).toHaveBeenCalledWith({
        channel_id: 5,
        prompt_tokens_threshold: 1_000_000,
        enabled: true,
        ...NO_UPSTREAM_PAYLOAD,
      })
    })
  })

  test('blocks saving when no channel is picked', async () => {
    const user = userEvent.setup()
    apiMocks.getBillingCorrectionRules.mockResolvedValue([])
    apiMocks.getChannelOptions.mockResolvedValue([])

    renderPage()
    await screen.findByText('No billing correction rules yet')

    await user.click(
      screen.getByRole('button', { name: 'Add Correction Rule' })
    )
    await screen.findByRole('heading', { name: 'Add Correction Rule' })
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(
      await screen.findByText('Please select a channel')
    ).toBeInTheDocument()
    expect(apiMocks.createBillingCorrectionRule).not.toHaveBeenCalled()
  })

  test('keeps the dialog open and shows the server message when saving is rejected', async () => {
    const user = userEvent.setup()
    apiMocks.getBillingCorrectionRules.mockResolvedValue([buildRule()])
    apiMocks.getChannelOptions.mockResolvedValue([
      { id: 5, name: 'heartbeat-upstream' },
    ])
    apiMocks.updateBillingCorrectionRule.mockResolvedValue({
      success: false,
      message: 'This channel already has a billing correction rule',
      data: null,
    })

    renderPage()
    await screen.findByText('heartbeat-upstream')

    await user.click(screen.getByRole('button', { name: 'Edit' }))
    const thresholdInput = await screen.findByLabelText('Input token threshold')
    await waitFor(() => {
      expect(thresholdInput).toHaveValue(2_000_000)
    })
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(
      await screen.findByText(
        'This channel already has a billing correction rule'
      )
    ).toBeInTheDocument()
    expect(apiMocks.updateBillingCorrectionRule).toHaveBeenCalledWith(1, {
      channel_id: 5,
      prompt_tokens_threshold: 2_000_000,
      enabled: true,
      ...NO_UPSTREAM_PAYLOAD,
    })
    expect(
      screen.getByRole('heading', { name: 'Edit Correction Rule' })
    ).toBeInTheDocument()
  })

  test('deletes a rule after the confirmation is accepted', async () => {
    apiMocks.getBillingCorrectionRules.mockResolvedValue([buildRule()])
    apiMocks.deleteBillingCorrectionRule.mockResolvedValue({
      success: true,
      message: '',
      data: null,
    })

    renderPage()
    await screen.findByText('heartbeat-upstream')

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Actions' }))
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }))
    const confirmDialog = await screen.findByRole('alertdialog')
    expect(
      within(confirmDialog).getByText('heartbeat-upstream')
    ).toBeInTheDocument()
    await user.click(
      within(confirmDialog).getByRole('button', { name: 'Delete' })
    )

    await waitFor(() => {
      expect(apiMocks.deleteBillingCorrectionRule).toHaveBeenCalledWith(1)
    })
  })

  test('keeps the confirmation open and shows the server message when deletion is rejected', async () => {
    const user = userEvent.setup()
    apiMocks.getBillingCorrectionRules.mockResolvedValue([buildRule()])
    apiMocks.deleteBillingCorrectionRule.mockResolvedValue({
      success: false,
      message: 'Billing correction rule does not exist',
      data: null,
    })

    renderPage()
    await screen.findByText('heartbeat-upstream')

    await user.click(screen.getByRole('button', { name: 'Actions' }))
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }))
    const confirmDialog = await screen.findByRole('alertdialog')
    await user.click(
      within(confirmDialog).getByRole('button', { name: 'Delete' })
    )

    expect(
      await screen.findByText('Billing correction rule does not exist')
    ).toBeInTheDocument()
    expect(screen.getByRole('alertdialog')).toBeInTheDocument()
  })

  test('marks a zero threshold invalid and does not save it', async () => {
    const user = userEvent.setup()
    apiMocks.getBillingCorrectionRules.mockResolvedValue([buildRule()])
    apiMocks.getChannelOptions.mockResolvedValue([
      { id: 5, name: 'heartbeat-upstream' },
    ])

    renderPage()
    await screen.findByText('heartbeat-upstream')

    await user.click(screen.getByRole('button', { name: 'Edit' }))
    const thresholdInput = await screen.findByLabelText('Input token threshold')
    await waitFor(() => {
      expect(thresholdInput).toHaveValue(2_000_000)
    })
    await user.clear(thresholdInput)
    await user.type(thresholdInput, '0')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(
      await screen.findByText(
        /^Threshold must be between 1 and 1\D000\D000\D000 tokens$/
      )
    ).toBeInTheDocument()
    expect(thresholdInput).toHaveAttribute('aria-invalid', 'true')
    expect(apiMocks.updateBillingCorrectionRule).not.toHaveBeenCalled()
  })

  test('the records tab loads the mis-billed records and hides the add rule action', async () => {
    const user = userEvent.setup()
    apiMocks.getBillingCorrectionRules.mockResolvedValue([buildRule()])
    apiMocks.getBillingCorrectionRecords.mockResolvedValue({
      success: true,
      message: '',
      data: {
        items: [],
        total: 0,
        page: 1,
        page_size: 20,
        stats: {
          charged_count: 0,
          charged_quota: 0,
          rule_matched_count: 0,
          rule_matched_quota: 0,
          pending_count: 0,
          pending_quota: 0,
          refunded_count: 0,
          refunded_quota: 0,
          credited_count: 0,
          credited_quota: 0,
          corrected_count: 0,
          upstream_not_charged_count: 0,
          upstream_not_charged_quota: 0,
        },
      },
    })

    renderPage()
    await screen.findByText('heartbeat-upstream')
    expect(apiMocks.getBillingCorrectionRecords).not.toHaveBeenCalled()

    await user.click(screen.getByRole('tab', { name: 'Mis-billed records' }))

    expect(await screen.findByText('No matching records')).toBeInTheDocument()
    expect(apiMocks.getBillingCorrectionRecords).toHaveBeenCalledWith(
      expect.objectContaining({ p: 1, page_size: 20 })
    )
    expect(
      screen.queryByRole('button', { name: 'Add Correction Rule' })
    ).not.toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Mark all pending as refunded' })
    ).toBeDisabled()
  })

  test('shows whether each rule checks the upstream and returns quota automatically', async () => {
    apiMocks.getBillingCorrectionRules.mockResolvedValue([
      buildRule({ upstream_check_enabled: true, auto_refund: true }),
    ])

    renderPage()

    expect(
      await screen.findByText('On, returns quota automatically')
    ).toBeInTheDocument()
  })

  test('turning on the upstream check asks for the user id and token, then saves them', async () => {
    const user = userEvent.setup()
    apiMocks.getBillingCorrectionRules.mockResolvedValue([buildRule()])
    apiMocks.getChannelOptions.mockResolvedValue([
      { id: 5, name: 'heartbeat-upstream' },
    ])
    apiMocks.updateBillingCorrectionRule.mockResolvedValue({
      success: true,
      message: '',
      data: buildRule({ upstream_check_enabled: true }),
    })

    renderPage()
    await screen.findByText('heartbeat-upstream')
    await user.click(screen.getByRole('button', { name: 'Edit' }))
    await waitFor(() => {
      expect(screen.getByLabelText('Input token threshold')).toHaveValue(
        2_000_000
      )
    })
    expect(screen.queryByLabelText('Upstream user ID')).not.toBeInTheDocument()

    await user.click(
      screen.getByRole('switch', {
        name: 'Check charges on the upstream new-api (experimental)',
      })
    )
    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(
      await screen.findByText('Enter the upstream user ID')
    ).toBeInTheDocument()
    expect(
      screen.getByText('Enter the upstream access token')
    ).toBeInTheDocument()
    expect(apiMocks.updateBillingCorrectionRule).not.toHaveBeenCalled()

    await user.type(screen.getByLabelText('Upstream user ID'), '9')
    await user.type(
      screen.getByLabelText('Upstream system access token'),
      'secret-token'
    )
    await user.click(
      screen.getByRole('switch', { name: 'Return quota automatically' })
    )
    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(apiMocks.updateBillingCorrectionRule).toHaveBeenCalledWith(1, {
        channel_id: 5,
        prompt_tokens_threshold: 2_000_000,
        enabled: true,
        upstream_check_enabled: true,
        upstream_base_url: '',
        upstream_user_id: 9,
        upstream_access_token: 'secret-token',
        auto_refund: true,
      })
    })
  })

  test('a saved access token can stay empty and is used to test the connection', async () => {
    const user = userEvent.setup()
    apiMocks.getBillingCorrectionRules.mockResolvedValue([
      buildRule({
        upstream_check_enabled: true,
        upstream_user_id: 9,
        upstream_access_token_set: true,
      }),
    ])
    apiMocks.getChannelOptions.mockResolvedValue([
      { id: 5, name: 'heartbeat-upstream' },
    ])
    apiMocks.testBillingCorrectionUpstream.mockResolvedValue({
      success: true,
      message: '',
      data: { username: 'relay-bot' },
    })

    renderPage()
    await screen.findByText('heartbeat-upstream')
    await user.click(screen.getByRole('button', { name: 'Edit' }))
    const tokenInput = await screen.findByLabelText(
      'Upstream system access token'
    )
    expect(tokenInput).toHaveValue('')
    expect(tokenInput).toHaveAttribute(
      'placeholder',
      'Saved. Leave empty to keep it.'
    )

    await user.click(screen.getByRole('button', { name: 'Test connection' }))

    await waitFor(() => {
      expect(apiMocks.testBillingCorrectionUpstream).toHaveBeenCalledWith({
        rule_id: 1,
        channel_id: 5,
        upstream_base_url: '',
        upstream_user_id: 9,
        upstream_access_token: undefined,
      })
    })
    expect(
      await screen.findByText('Connected to the upstream as relay-bot')
    ).toBeInTheDocument()
  })

  test('renders the Chinese page title when zh is active', async () => {
    i18next.options.keySeparator = false
    i18next.options.nsSeparator = false
    i18next.addResourceBundle('zhCN', 'translation', zh.translation, true, true)
    await i18next.changeLanguage('zhCN')
    apiMocks.getBillingCorrectionRules.mockResolvedValue([])

    renderPage()

    expect(
      await screen.findByRole('heading', { name: '计费修正' })
    ).toBeInTheDocument()
    expect(await screen.findByText('暂无计费修正规则')).toBeInTheDocument()
  })
})
