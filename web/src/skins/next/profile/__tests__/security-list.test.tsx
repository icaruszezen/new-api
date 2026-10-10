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
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { describe, expect, test, vi } from 'vitest'

import type { UserProfile } from '@/features/profile/types'
import { ROLE } from '@/lib/roles'

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { to: string; children: ReactNode; className?: string }) => (
    <a href={props.to} className={props.className}>
      {props.children}
    </a>
  ),
}))

vi.mock(
  '@/features/security/components/dialogs/change-password-dialog',
  () => ({
    ChangePasswordDialog: (props: {
      open: boolean
      hasPassword?: boolean
      onSuccess?: () => void
    }) =>
      props.open ? (
        <div
          role='dialog'
          aria-label={props.hasPassword === false ? 'set' : 'change'}
        >
          <button type='button' onClick={props.onSuccess}>
            Save password
          </button>
        </div>
      ) : null,
  })
)

vi.mock('@/features/security/components/dialogs/delete-account-dialog', () => ({
  DeleteAccountDialog: () => null,
}))

const { NextProfileSecurityList } = await import('../components/security-list')

function profileWith(hasPassword: boolean): UserProfile {
  return {
    id: 1,
    username: 'zezen',
    display_name: 'Root User',
    role: ROLE.USER,
    group: 'default',
    quota: 0,
    used_quota: 0,
    request_count: 0,
    status: 1,
    aff_count: 0,
    aff_quota: 0,
    aff_history_quota: 0,
    created_time: 1,
    has_password: hasPassword,
  }
}

describe('next profile security list', () => {
  test('account with a password offers Change Password', () => {
    render(
      <NextProfileSecurityList
        profile={profileWith(true)}
        loading={false}
        onProfileUpdate={vi.fn()}
      />
    )

    expect(
      screen.getByRole('button', { name: /Change Password/ })
    ).toBeVisible()
    expect(screen.queryByRole('button', { name: /Set Password/ })).toBeNull()
  })

  test('account without a password sets one and refreshes the profile', async () => {
    const onProfileUpdate = vi.fn()
    render(
      <NextProfileSecurityList
        profile={profileWith(false)}
        loading={false}
        onProfileUpdate={onProfileUpdate}
      />
    )

    await userEvent.click(screen.getByRole('button', { name: /Set Password/ }))
    expect(screen.getByRole('dialog', { name: 'set' })).toBeVisible()

    await userEvent.click(screen.getByRole('button', { name: 'Save password' }))
    expect(onProfileUpdate).toHaveBeenCalledTimes(1)
  })

  test('access tokens row opens the Security & Access page', () => {
    render(
      <NextProfileSecurityList
        profile={profileWith(true)}
        loading={false}
        onProfileUpdate={vi.fn()}
      />
    )

    expect(screen.getByRole('link', { name: /Access tokens/ })).toHaveAttribute(
      'href',
      '/security'
    )
  })
})
