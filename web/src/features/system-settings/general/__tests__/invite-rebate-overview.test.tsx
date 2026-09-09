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
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type { InviteAdminStats } from '@/features/invite/types'
import { useSystemConfigStore } from '@/stores/system-config-store'

const apiMocks = vi.hoisted(() => ({
  getInviteAdminStats: vi.fn(),
}))

vi.mock('@/features/invite/api', () => ({
  getInviteAdminStats: apiMocks.getInviteAdminStats,
}))

const { InviteRebateOverviewSection } = await import(
  '../invite-rebate-overview-section'
)

const REGISTER_QUOTA = 1_000_000
const TOPUP_QUOTA = 2_500_000

function emptyStats(): InviteAdminStats {
  return {
    summary: {
      inviter_count: 0,
      invitee_count: 0,
      recharged_count: 0,
      rebate_quota: 0,
      rebate_payout_count: 0,
      register_rebate_quota: 0,
      register_rebate_count: 0,
      topup_rebate_quota: 0,
      topup_rebate_count: 0,
    },
    invite_rankings: [],
    rebate_rankings: [],
    recent_invitees: [],
  }
}

function populatedStats(): InviteAdminStats {
  return {
    summary: {
      inviter_count: 2,
      invitee_count: 3,
      recharged_count: 1,
      rebate_quota: REGISTER_QUOTA * 2 + TOPUP_QUOTA,
      rebate_payout_count: 3,
      register_rebate_quota: REGISTER_QUOTA * 2,
      register_rebate_count: 2,
      topup_rebate_quota: TOPUP_QUOTA,
      topup_rebate_count: 1,
    },
    invite_rankings: [
      {
        rank: 1,
        user_id: 11,
        username: 'alice',
        invitee_count: 2,
        rebate_quota: REGISTER_QUOTA + TOPUP_QUOTA,
      },
      {
        rank: 2,
        user_id: 12,
        username: 'bob',
        invitee_count: 1,
        rebate_quota: REGISTER_QUOTA,
      },
    ],
    rebate_rankings: [
      {
        rank: 1,
        user_id: 11,
        username: 'alice',
        rebate_quota: REGISTER_QUOTA + TOPUP_QUOTA,
        payout_count: 2,
        invitee_count: 2,
      },
      {
        rank: 2,
        user_id: 12,
        username: 'bob',
        rebate_quota: REGISTER_QUOTA,
        payout_count: 1,
        invitee_count: 1,
      },
    ],
    recent_invitees: [
      {
        invitee_id: 21,
        invitee_username: 'carol',
        inviter_id: 11,
        inviter_username: 'alice',
        registered_at: 1_757_003_400,
        status: 'recharged',
        rebate_quota: REGISTER_QUOTA + TOPUP_QUOTA,
      },
      {
        invitee_id: 22,
        invitee_username: 'dave',
        inviter_id: 12,
        inviter_username: 'bob',
        registered_at: 1_756_343_400,
        status: 'registered',
        rebate_quota: REGISTER_QUOTA,
      },
    ],
  }
}

function renderOverview() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const Wrapper = (props: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {props.children}
    </QueryClientProvider>
  )
  return render(<InviteRebateOverviewSection />, { wrapper: Wrapper })
}

describe('invite rebate admin overview', () => {
  beforeEach(() => {
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
    apiMocks.getInviteAdminStats.mockResolvedValue({
      success: true,
      message: '',
      data: populatedStats(),
    })
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  test('keeps summary cards as skeletons until the snapshot loads', () => {
    apiMocks.getInviteAdminStats.mockReturnValue(new Promise(() => {}))

    renderOverview()

    expect(
      screen.getByRole('heading', { name: 'Referral overview' })
    ).toBeVisible()
    expect(document.querySelectorAll('[data-slot="invite-admin-stat-card"]')).toHaveLength(4)
    expect(screen.queryByText('alice')).not.toBeInTheDocument()
    expect(screen.queryByText('No referral rankings yet.')).not.toBeInTheDocument()
  })

  test('shows site-wide totals, both rankings and recent referrals', async () => {
    renderOverview()

    await waitFor(() => {
      expect(screen.getAllByText('alice').length).toBeGreaterThan(0)
    })

    const cards = document.querySelectorAll(
      '[data-slot="invite-admin-stat-card"]'
    )
    expect(cards).toHaveLength(4)
    expect(cards[0]).toHaveTextContent('Inviters')
    expect(cards[0]).toHaveTextContent('2')
    expect(cards[1]).toHaveTextContent('Invitees')
    expect(cards[1]).toHaveTextContent('3')
    expect(cards[2]).toHaveTextContent('Topped-up invitees')
    expect(cards[2]).toHaveTextContent('1')
    expect(cards[3]).toHaveTextContent('Rebate issued')
    expect(cards[3]).toHaveTextContent('$9')
    expect(cards[3]).toHaveTextContent('Sign-up $4 · Top-up $5')

    expect(screen.getByText('Referral rankings')).toBeVisible()
    expect(screen.getByText('Rebate rankings')).toBeVisible()
    expect(screen.getAllByText('bob').length).toBeGreaterThan(0)
    expect(screen.getByRole('columnheader', { name: 'Invites' })).toBeVisible()
    expect(screen.getByRole('columnheader', { name: 'Payouts' })).toBeVisible()

    expect(screen.getByText('Recent referrals')).toBeVisible()
    expect(screen.getByText('carol')).toBeVisible()
    expect(screen.getByText('dave')).toBeVisible()
    expect(screen.getByText('Topped up')).toBeVisible()
    expect(screen.getByText('Signed up')).toBeVisible()
  })

  test('explains empty rankings and the empty recent table', async () => {
    apiMocks.getInviteAdminStats.mockResolvedValue({
      success: true,
      message: '',
      data: emptyStats(),
    })

    renderOverview()

    await waitFor(() => {
      expect(screen.getByText('No referral rankings yet.')).toBeVisible()
    })

    expect(screen.getByText('No rebate rankings yet.')).toBeVisible()
    expect(screen.getByText('No referrals yet.')).toBeVisible()
    expect(screen.getByRole('heading', { name: 'Inviters' })).toBeVisible()
  })

  test('surfaces a load failure instead of empty rankings', async () => {
    apiMocks.getInviteAdminStats.mockRejectedValue(new Error('network down'))

    renderOverview()

    await waitFor(() => {
      expect(
        screen.getByText(
          'Failed to load referral overview. Please try again later.'
        )
      ).toBeVisible()
    })

    expect(screen.queryByText('Referral rankings')).not.toBeInTheDocument()
    expect(screen.queryByText('No referrals yet.')).not.toBeInTheDocument()
  })
})
