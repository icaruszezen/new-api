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
import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type { UserProfile } from '@/features/profile/types'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

const profileFixture: UserProfile = {
  id: 1,
  username: 'zezen',
  display_name: 'Root User',
  role: ROLE.SUPER_ADMIN,
  group: 'default',
  quota: 5002621425000,
  used_quota: 4081075000,
  request_count: 81000,
  status: 1,
  aff_count: 0,
  aff_quota: 0,
  aff_history_quota: 0,
  created_time: 1,
}

vi.mock('@/features/profile/hooks', () => ({
  useProfile: () => ({
    profile: profileFixture,
    loading: false,
    refreshProfile: vi.fn(),
  }),
  useTwoFA: () => ({
    status: {
      enabled: false,
      locked: false,
      backup_codes_remaining: 0,
    },
    loading: false,
    refetch: vi.fn(),
  }),
}))

vi.mock('@/features/auth/passkey', () => ({
  usePasskeyManagement: () => ({
    status: null,
    loading: false,
    registering: false,
    removing: false,
    supported: true,
    enabled: false,
    lastUsed: null,
    register: vi.fn(),
    remove: vi.fn(),
  }),
}))

vi.mock('@/features/auth/secure-verification', () => ({
  useSecureVerification: () => ({
    open: false,
    setOpen: vi.fn(),
    methods: { has2FA: false, hasPasskey: false, passkeySupported: true },
    state: {},
    startVerification: vi.fn(),
    executeVerification: vi.fn(),
    cancel: vi.fn(),
    setCode: vi.fn(),
    switchMethod: vi.fn(),
    fetchVerificationMethods: vi.fn(),
  }),
  SecureVerificationDialog: () => null,
}))

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({
    status: { checkin_enabled: false },
    loading: false,
  }),
}))

vi.mock('@/features/profile/components/tabs/account-bindings-tab', () => ({
  AccountBindingsTab: () => <div data-testid='account-bindings' />,
}))

vi.mock('@/features/profile/components/tabs/notification-tab', () => ({
  NotificationTab: () => <div data-testid='notification-settings' />,
}))

vi.mock('@/features/profile/components/language-select-row', () => ({
  LanguageSelectRow: () => (
    <div data-testid='language-select-row'>Interface Language</div>
  ),
}))

vi.mock('@/features/profile/components/login-sessions-card', () => ({
  LoginSessionsCard: () => <div data-testid='login-sessions' />,
}))

vi.mock('@/features/profile/components/checkin-calendar-card', () => ({
  CheckinCalendarCard: () => <div data-testid='checkin-calendar' />,
}))

vi.mock('@/features/profile/components/sidebar-modules-card', () => ({
  SidebarModulesCard: () => <div data-testid='sidebar-modules' />,
}))

vi.mock('@/features/profile/components/dialogs/change-password-dialog', () => ({
  ChangePasswordDialog: () => null,
}))

vi.mock('@/features/profile/components/dialogs/access-token-dialog', () => ({
  AccessTokenDialog: () => null,
}))

vi.mock('@/features/profile/components/dialogs/delete-account-dialog', () => ({
  DeleteAccountDialog: () => null,
}))

vi.mock('@/features/profile/components/dialogs/two-fa-setup-dialog', () => ({
  TwoFASetupDialog: () => null,
}))

vi.mock('@/features/profile/components/dialogs/two-fa-disable-dialog', () => ({
  TwoFADisableDialog: () => null,
}))

vi.mock('@/features/profile/components/dialogs/two-fa-backup-dialog', () => ({
  TwoFABackupDialog: () => null,
}))

const { NextProfile } = await import('../index')

describe('next profile layout', () => {
  beforeEach(() => {
    useAuthStore.getState().auth.setUser({
      id: 1,
      username: 'zezen',
      role: ROLE.USER,
      permissions: { sidebar_settings: false },
    })
  })

  afterEach(() => {
    useAuthStore.getState().auth.reset()
  })

  test('renders the heading, three stats, settings, and security columns', () => {
    render(<NextProfile />)

    const page = document.querySelector('[data-slot="next-profile"]')
    expect(page).toBeInTheDocument()
    expect(page).toHaveClass('flex')
    expect(page).toHaveClass('flex-col')
    expect(page).not.toHaveClass('overflow-hidden')

    expect(
      screen.getByRole('heading', { name: 'Personal Center' })
    ).toBeVisible()
    expect(
      screen.getByText('Manage account, security, and preferences')
    ).toBeVisible()
    expect(screen.getByText('Root User')).toBeVisible()
    expect(screen.getByText('@zezen · default')).toBeVisible()
    expect(screen.getByText('Super Admin')).toBeVisible()
    expect(screen.getByText('User ID 1')).toBeVisible()

    const stats = screen.getByRole('region', { name: 'Account usage' })
    expect(stats).toHaveAttribute('data-slot', 'next-profile-stats')
    expect(stats).toHaveClass('sm:grid-cols-3')
    expect(
      screen.getByRole('heading', { name: 'Current Balance' })
    ).toBeVisible()
    expect(screen.getByRole('heading', { name: 'Total Usage' })).toBeVisible()
    expect(screen.getByRole('heading', { name: 'API Requests' })).toBeVisible()
    expect(
      stats.querySelectorAll('[data-slot="next-profile-stat-card"]')
    ).toHaveLength(3)
    expect(stats.querySelector('[data-slot="icon-badge"]')).toBeNull()

    const settings = screen
      .getByRole('heading', { name: 'Settings' })
      .closest('[data-slot="next-profile-settings"]')
    expect(settings).toBeInTheDocument()
    expect(settings).toHaveClass('rounded-xl')
    expect(settings).toHaveClass('border')
    expect(screen.getByRole('tab', { name: 'Account Bindings' })).toBeVisible()
    expect(
      screen.getByRole('tab', { name: 'Settings & Preferences' })
    ).toBeVisible()
    expect(screen.getByTestId('account-bindings')).toBeVisible()
    expect(screen.getByTestId('language-select-row')).toBeVisible()
    expect(settings).toContainElement(screen.getByTestId('language-select-row'))

    const columns = document.querySelector('[data-slot="next-profile-columns"]')
    expect(columns).toHaveClass(
      'xl:grid-cols-[minmax(0,1fr)_minmax(320px,0.42fr)]'
    )
    expect(screen.getByTestId('login-sessions').parentElement).toBe(
      settings?.parentElement
    )

    expect(screen.getByRole('heading', { name: 'Passkey Login' })).toBeVisible()
    expect(screen.getByRole('button', { name: 'Enable Passkey' })).toBeVisible()
    expect(
      screen.getByRole('heading', { name: 'Two-Step Verification' })
    ).toBeVisible()
    expect(
      screen.getByRole('heading', { name: 'Security Settings' })
    ).toBeVisible()
    expect(
      screen.getByRole('button', { name: /Change Password/ })
    ).toBeVisible()
    expect(screen.getByRole('button', { name: /Access Token/ })).toBeVisible()
    expect(screen.getByRole('button', { name: /Delete Account/ })).toBeVisible()

    expect(screen.queryByTestId('checkin-calendar')).toBeNull()
    expect(screen.queryByTestId('sidebar-modules')).toBeNull()
    expect(document.querySelector('main')).toBeNull()
  })
})
