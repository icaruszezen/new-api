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
import type { ReactNode } from 'react'
import { Toaster, toast } from 'sonner'
import { afterEach, describe, expect, test, vi } from 'vitest'

import type {
  BillingCorrectionRecord,
  BillingCorrectionRecordPage,
  BillingCorrectionRule,
} from '../types'

const apiMocks = vi.hoisted(() => ({
  getBillingCorrectionRecords: vi.fn(),
  markBillingCorrectionRecordsRefunded: vi.fn(),
  unmarkBillingCorrectionRecordsRefunded: vi.fn(),
}))

vi.mock('../api', () => apiMocks)

const { RecordsPanel } = await import('../components/records-panel')

const RULES: BillingCorrectionRule[] = [
  {
    id: 1,
    channel_id: 7,
    prompt_tokens_threshold: 1_000_000,
    enabled: true,
    created_time: 0,
    updated_time: 0,
    channel_name: 'heartbeat-upstream',
  },
  {
    id: 2,
    channel_id: 8,
    prompt_tokens_threshold: 1_000_000,
    enabled: false,
    created_time: 0,
    updated_time: 0,
    channel_name: 'disabled-upstream',
  },
]

function buildRecord(
  overrides: Partial<BillingCorrectionRecord> = {}
): BillingCorrectionRecord {
  return {
    request_id: 'req-pending',
    created_at: 1_760_000_000,
    user_id: 1,
    username: 'alice',
    token_name: 'default',
    model_name: 'gpt-4o',
    channel_id: 7,
    channel_name: 'heartbeat-upstream',
    prompt_tokens: 1_500_000,
    prompt_tokens_threshold: 1_000_000,
    quota: 750_000,
    use_time: 30,
    is_stream: true,
    status: 'pending',
    ...overrides,
  }
}

const PENDING = buildRecord()
const REFUNDED = buildRecord({
  request_id: 'req-refunded',
  username: 'bob',
  status: 'refunded',
  refunded_at: 1_760_100_000,
})
const CORRECTED = buildRecord({
  request_id: 'req-corrected',
  username: 'carol',
  status: 'corrected',
  quota: 0,
  original_quota: 750_000,
})

function buildPage(items: BillingCorrectionRecord[]): {
  success: true
  message: string
  data: BillingCorrectionRecordPage
} {
  return {
    success: true,
    message: '',
    data: {
      items,
      total: items.length,
      page: 1,
      page_size: 20,
      stats: {
        charged_count: 2,
        charged_quota: 1_500_000,
        pending_count: 1,
        pending_quota: 750_000,
        refunded_count: 1,
        refunded_quota: 750_000,
        corrected_count: 1,
      },
    },
  }
}

function renderPanel() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const Wrapper = (props: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {props.children}
      <Toaster duration={60_000} />
    </QueryClientProvider>
  )
  return render(<RecordsPanel rules={RULES} />, { wrapper: Wrapper })
}

afterEach(() => {
  toast.dismiss()
  vi.clearAllMocks()
})

describe('billing correction records panel', () => {
  test('shows the charged, pending, refunded and corrected totals with each record status', async () => {
    apiMocks.getBillingCorrectionRecords.mockResolvedValue(
      buildPage([PENDING, REFUNDED, CORRECTED])
    )

    renderPanel()

    const charged = await screen.findByRole('group', {
      name: 'Wrongly charged',
    })
    expect(within(charged).getByText('2 record(s)')).toBeInTheDocument()
    const corrected = screen.getByRole('group', { name: 'Corrected' })
    expect(within(corrected).getByText('Not charged')).toBeInTheDocument()
    expect(within(corrected).getByText('1 record(s)')).toBeInTheDocument()

    const rows = screen.getAllByRole('row')
    expect(within(rows[1]).getByText('Pending refund')).toBeInTheDocument()
    expect(within(rows[2]).getByText('Refunded')).toBeInTheDocument()
    expect(within(rows[3]).getByText('Corrected')).toBeInTheDocument()
    expect(within(rows[3]).getByText(/^Original cost:/)).toBeInTheDocument()
  })

  test('lists only channels with an enabled rule in the channel filter', async () => {
    apiMocks.getBillingCorrectionRecords.mockResolvedValue(buildPage([]))

    renderPanel()
    await screen.findByText('No matching records')

    const channelSelect = screen.getByRole('combobox', { name: 'Channel' })
    expect(
      within(channelSelect).getByRole('option', { name: 'heartbeat-upstream' })
    ).toBeInTheDocument()
    expect(
      within(channelSelect).queryByRole('option', { name: 'disabled-upstream' })
    ).not.toBeInTheDocument()
  })

  test('searching sends the edited filters and starts again from page one', async () => {
    const user = userEvent.setup()
    apiMocks.getBillingCorrectionRecords.mockResolvedValue(buildPage([]))

    renderPanel()
    await screen.findByText('No matching records')

    await user.type(
      screen.getByRole('textbox', { name: 'Username' }),
      ' alice '
    )
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Channel' }),
      '7'
    )
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Status' }),
      'pending'
    )
    expect(apiMocks.getBillingCorrectionRecords).toHaveBeenCalledTimes(1)
    await user.click(screen.getByRole('button', { name: 'Search' }))

    await waitFor(() => {
      expect(apiMocks.getBillingCorrectionRecords).toHaveBeenLastCalledWith(
        expect.objectContaining({
          p: 1,
          username: 'alice',
          channel: 7,
          status: 'pending',
        })
      )
    })
  })

  test('shows the server message inline when the query is rejected', async () => {
    apiMocks.getBillingCorrectionRecords.mockResolvedValue({
      success: false,
      message: 'Too many matching records, narrow the time range',
      data: null,
    })

    renderPanel()

    expect(
      await screen.findByText(
        'Too many matching records, narrow the time range'
      )
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Mark all pending as refunded' })
    ).not.toBeInTheDocument()
  })

  test('corrected records and legacy records without a request id cannot be selected', async () => {
    apiMocks.getBillingCorrectionRecords.mockResolvedValue(
      buildPage([CORRECTED, buildRecord({ request_id: '' })])
    )

    renderPanel()

    expect(
      await screen.findByRole('checkbox', {
        name: 'Select record req-corrected',
      })
    ).toHaveAttribute('aria-disabled', 'true')
    expect(
      screen.getByRole('checkbox', { name: 'Select record -' })
    ).toHaveAttribute('aria-disabled', 'true')
    expect(
      screen.getByRole('checkbox', { name: 'Select all on this page' })
    ).toHaveAttribute('aria-disabled', 'true')
  })

  test('marks only the selected pending records after confirmation and reloads the list', async () => {
    const user = userEvent.setup()
    apiMocks.getBillingCorrectionRecords.mockResolvedValue(
      buildPage([PENDING, REFUNDED, CORRECTED])
    )
    apiMocks.markBillingCorrectionRecordsRefunded.mockResolvedValue({
      success: true,
      message: '',
      data: { marked: 1, quota: 750_000, skipped: 0 },
    })

    renderPanel()
    await user.click(
      await screen.findByRole('checkbox', { name: 'Select all on this page' })
    )
    expect(screen.getByText('Selected 2')).toBeInTheDocument()
    await user.click(
      screen.getByRole('button', { name: 'Mark selected as refunded' })
    )
    const dialog = await screen.findByRole('alertdialog')
    expect(apiMocks.markBillingCorrectionRecordsRefunded).not.toHaveBeenCalled()
    await user.click(within(dialog).getByRole('button', { name: 'Confirm' }))

    await waitFor(() => {
      expect(
        apiMocks.markBillingCorrectionRecordsRefunded
      ).toHaveBeenCalledWith({ request_ids: ['req-pending'] })
    })
    expect(
      await screen.findByText(/^Marked 1 record\(s\) as refunded/)
    ).toBeInTheDocument()
    await waitFor(() => {
      expect(apiMocks.getBillingCorrectionRecords).toHaveBeenCalledTimes(2)
    })
    expect(screen.queryByText('Selected 2')).not.toBeInTheDocument()
  })

  test('marking every pending record sends the applied filters without the status filter', async () => {
    const user = userEvent.setup()
    apiMocks.getBillingCorrectionRecords.mockResolvedValue(buildPage([PENDING]))
    apiMocks.markBillingCorrectionRecordsRefunded.mockResolvedValue({
      success: true,
      message: '',
      data: { marked: 1, quota: 750_000, skipped: 0 },
    })

    renderPanel()
    await screen.findByText('alice')
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Status' }),
      'pending'
    )
    await user.click(screen.getByRole('button', { name: 'Search' }))
    await waitFor(() => {
      expect(apiMocks.getBillingCorrectionRecords).toHaveBeenCalledTimes(2)
    })
    await user.click(
      await screen.findByRole('button', {
        name: 'Mark all pending as refunded',
      })
    )
    const dialog = await screen.findByRole('alertdialog')
    await user.click(within(dialog).getByRole('button', { name: 'Confirm' }))

    await waitFor(() => {
      expect(
        apiMocks.markBillingCorrectionRecordsRefunded
      ).toHaveBeenCalledTimes(1)
    })
    const payload =
      apiMocks.markBillingCorrectionRecordsRefunded.mock.calls[0][0]
    expect(payload).toMatchObject({
      all: true,
      start_timestamp: expect.any(Number),
      end_timestamp: expect.any(Number),
    })
    expect(payload).not.toHaveProperty('status')
    expect(payload).not.toHaveProperty('request_ids')
  })

  test('removes the refunded mark from the selected refunded records', async () => {
    const user = userEvent.setup()
    apiMocks.getBillingCorrectionRecords.mockResolvedValue(
      buildPage([PENDING, REFUNDED])
    )
    apiMocks.unmarkBillingCorrectionRecordsRefunded.mockResolvedValue({
      success: true,
      message: '',
      data: { unmarked: 1 },
    })

    renderPanel()
    await user.click(
      await screen.findByRole('checkbox', {
        name: 'Select record req-refunded',
      })
    )
    expect(
      screen.getByRole('button', { name: 'Mark selected as refunded' })
    ).toBeDisabled()
    await user.click(
      screen.getByRole('button', { name: 'Remove refunded mark' })
    )
    const dialog = await screen.findByRole('alertdialog')
    await user.click(within(dialog).getByRole('button', { name: 'Confirm' }))

    await waitFor(() => {
      expect(
        apiMocks.unmarkBillingCorrectionRecordsRefunded
      ).toHaveBeenCalledWith(['req-refunded'])
    })
    expect(
      await screen.findByText('Removed the refunded mark from 1 record(s)')
    ).toBeInTheDocument()
  })

  test('warns when records without a request id were skipped', async () => {
    const user = userEvent.setup()
    apiMocks.getBillingCorrectionRecords.mockResolvedValue(buildPage([PENDING]))
    apiMocks.markBillingCorrectionRecordsRefunded.mockResolvedValue({
      success: true,
      message: '',
      data: { marked: 1, quota: 750_000, skipped: 2 },
    })

    renderPanel()
    await user.click(
      await screen.findByRole('button', {
        name: 'Mark all pending as refunded',
      })
    )
    await user.click(
      within(await screen.findByRole('alertdialog')).getByRole('button', {
        name: 'Confirm',
      })
    )

    expect(
      await screen.findByText('2 record(s) without a request ID were skipped')
    ).toBeInTheDocument()
  })
})
