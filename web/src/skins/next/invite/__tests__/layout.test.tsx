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
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type { InviteOverview } from '@/features/invite/types'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

const apiMocks = vi.hoisted(() => ({
  getInviteOverview: vi.fn(),
}))

vi.mock('@/features/invite/api', () => ({
  getInviteOverview: apiMocks.getInviteOverview,
}))

const { NextInvite } = await import('../index')

const REGISTER_AMOUNT = 2
const TOPUP_AMOUNT = 5
const TOPUP_THRESHOLD = 50
// 500000 quota per unit, USD display: 1_000_000 quota renders as $2.00.
const REGISTER_QUOTA = 1_000_000
const TOPUP_QUOTA = 2_500_000

function buildOverview(
  overrides: Partial<InviteOverview> = {}
): InviteOverview {
  return {
    aff_code: 'A8K2M9',
    register_amount: REGISTER_AMOUNT,
    topup_amount: TOPUP_AMOUNT,
    topup_threshold: TOPUP_THRESHOLD,
    invitee_count: 2,
    recharged_count: 1,
    rebate_quota: REGISTER_QUOTA * 2 + TOPUP_QUOTA,
    invitees: [
      {
        username: 'l***3',
        registered_at: 1756343400,
        status: 'recharged',
        rebate_quota: REGISTER_QUOTA + TOPUP_QUOTA,
      },
      {
        username: 'u***8',
        registered_at: 1757003400,
        status: 'registered',
        rebate_quota: REGISTER_QUOTA,
      },
    ],
    ...overrides,
  }
}

function renderInvite() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const Wrapper = (props: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {props.children}
    </QueryClientProvider>
  )
  return render(<NextInvite />, { wrapper: Wrapper })
}

describe('next referral page', () => {
  beforeEach(() => {
    useAuthStore.setState((state) => ({
      auth: { ...state.auth, accessToken: 'test-token' },
    }))
    useSystemConfigStore.getState().setConfig({
      currency: {
        displayInCurrency: true,
        quotaDisplayType: 'USD',
        quotaPerUnit: 500_000,
        usdExchangeRate: 7.3,
        customCurrencySymbol: '¤',
        customCurrencyExchangeRate: 1,
      },
    })
    apiMocks.getInviteOverview.mockResolvedValue({
      success: true,
      message: '',
      data: buildOverview(),
    })
  })

  afterEach(() => {
    useAuthStore.getState().auth.reset()
    vi.clearAllMocks()
  })

  test('shows the referral link, rule chips, summary cards and invitee rows', async () => {
    renderInvite()

    await waitFor(() => {
      expect(
        screen.getByDisplayValue(`${window.location.origin}/sign-up?aff=A8K2M9`)
      ).toBeVisible()
    })

    expect(
      screen.getByRole('heading', { name: 'Referral Rebate', level: 1 })
    ).toBeVisible()
    expect(screen.getByText('Sign-up rebate $2.00')).toBeVisible()
    expect(screen.getByText('Top up $50.00 to earn $5.00')).toBeVisible()
    expect(screen.getByText('Once per friend for each reward')).toBeVisible()

    expect(
      screen.getByRole('heading', { name: 'Successful referrals' })
    ).toBeVisible()
    expect(screen.getByRole('heading', { name: 'Rebate earned' })).toBeVisible()
    expect(screen.getByRole('heading', { name: 'Topped up' })).toBeVisible()

    expect(screen.getByText('l***3')).toBeVisible()
    expect(screen.getByText('u***8')).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'Username' })).toBeVisible()
    expect(
      screen.getByRole('columnheader', { name: 'Signed up at' })
    ).toBeVisible()
    expect(
      screen.getByRole('columnheader', { name: 'Rebate amount' })
    ).toBeVisible()
  })

  // The empty state must still offer the link, otherwise a new user has no way
  // to start inviting from this page.
  test('keeps the link actions and explains the empty invitee table', async () => {
    apiMocks.getInviteOverview.mockResolvedValue({
      success: true,
      message: '',
      data: buildOverview({
        invitee_count: 0,
        recharged_count: 0,
        rebate_quota: 0,
        invitees: [],
      }),
    })

    renderInvite()

    await waitFor(() => {
      expect(
        screen.getByText(
          'No invited friends yet. Send the link above to a friend.'
        )
      ).toBeVisible()
    })

    expect(screen.getByRole('button', { name: 'Copy link' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Share link' })).toBeEnabled()
    expect(
      screen.getByText('Friends will show up here once they sign up.')
    ).toBeVisible()
  })

  test('requests the selected invitee filter from the server', async () => {
    renderInvite()

    await waitFor(() => {
      expect(apiMocks.getInviteOverview).toHaveBeenCalledWith('all')
    })

    await userEvent.click(screen.getByRole('button', { name: 'Signed up' }))
    await waitFor(() => {
      expect(apiMocks.getInviteOverview).toHaveBeenCalledWith('registered')
    })

    await userEvent.click(
      screen.getByRole('button', { name: 'Topped up', pressed: false })
    )
    await waitFor(() => {
      expect(apiMocks.getInviteOverview).toHaveBeenCalledWith('recharged')
    })
  })

  // Disabling both tiers must not leave stale reward promises on screen.
  test('drops the rule chips when every rebate tier is disabled', async () => {
    apiMocks.getInviteOverview.mockResolvedValue({
      success: true,
      message: '',
      data: buildOverview({
        register_amount: 0,
        topup_amount: 0,
        topup_threshold: 0,
      }),
    })

    renderInvite()

    await waitFor(() => {
      expect(
        screen.getByText(
          'Referral rebates are currently turned off. You can still share your link, and invited friends stay linked to your account.'
        )
      ).toBeVisible()
    })

    expect(
      document.querySelectorAll('[data-slot="next-invite-rule-chip"]')
    ).toHaveLength(0)
  })

  test('surfaces a load failure instead of an empty page', async () => {
    apiMocks.getInviteOverview.mockRejectedValue(new Error('network down'))

    renderInvite()

    await waitFor(() => {
      expect(
        screen.getByText(
          'Failed to load referral data. Please try again later.'
        )
      ).toBeVisible()
    })
  })
})
