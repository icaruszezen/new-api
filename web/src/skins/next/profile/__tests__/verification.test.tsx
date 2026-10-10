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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import type { SecurityProof } from '@/features/auth/secure-verification'

const mocks = vi.hoisted(() => ({
  requestVerification: vi.fn(),
  cancelVerification: vi.fn(),
  refetchTwoFA: vi.fn(),
  register: vi.fn(),
  remove: vi.fn(),
  fetchStatus: vi.fn(),
  passkey: {
    enabled: false,
    statusError: null as string | null,
  },
}))

vi.mock('@/features/auth/passkey', () => ({
  usePasskeyManagement: () => ({
    status: null,
    statusError: mocks.passkey.statusError,
    fetchStatus: mocks.fetchStatus,
    loading: false,
    registering: false,
    removing: false,
    supported: true,
    enabled: mocks.passkey.enabled,
    lastUsed: null,
    register: mocks.register,
    remove: mocks.remove,
  }),
}))

// The real hook returns stable callbacks; fresh mocks per render would re-run
// effects that depend on them.
vi.mock('@/features/auth/secure-verification', () => {
  const verification = {
    requestVerification: mocks.requestVerification,
    requestLoginVerification: vi.fn(),
    cancel: mocks.cancelVerification,
    isActive: false,
    dialogProps: {},
  }
  return {
    useSecureVerification: () => verification,
    SecureVerificationDialog: () => null,
  }
})

vi.mock('@/features/security/hooks/use-two-fa', () => ({
  useTwoFA: () => ({
    status: { enabled: false, locked: false, backup_codes_remaining: 0 },
    loading: false,
    error: undefined,
    refetch: mocks.refetchTwoFA,
  }),
}))

vi.mock('@/features/security/components/dialogs/two-fa-setup-dialog', () => ({
  TwoFASetupDialog: () => null,
}))

vi.mock('@/features/security/components/dialogs/two-fa-disable-dialog', () => ({
  TwoFADisableDialog: () => null,
}))

vi.mock('@/features/security/components/dialogs/two-fa-backup-dialog', () => ({
  TwoFABackupDialog: () => null,
}))

const { NextProfilePasskeyCard } = await import('../components/passkey-card')
const { NextProfileTwoFACard } = await import('../components/two-fa-card')

function proofFor(scope: SecurityProof['scope']): SecurityProof {
  return {
    proof_token: `${scope}-proof`,
    expires_at: 0,
    method: '2fa',
    scope,
  }
}

describe('next profile secure verification', () => {
  beforeEach(() => {
    mocks.requestVerification.mockReset()
    mocks.cancelVerification.mockReset()
    mocks.refetchTwoFA.mockReset()
    mocks.register.mockReset().mockResolvedValue(undefined)
    mocks.remove.mockReset().mockResolvedValue(undefined)
    mocks.fetchStatus.mockReset()
    mocks.passkey.enabled = false
    mocks.passkey.statusError = null
  })

  test('enabling a passkey registers it with the verified proof token', async () => {
    mocks.requestVerification.mockResolvedValue(proofFor('passkey.register'))
    render(<NextProfilePasskeyCard loading={false} />)

    await userEvent.click(
      screen.getByRole('button', { name: 'Enable Passkey' })
    )

    expect(mocks.requestVerification).toHaveBeenCalledWith({
      scope: 'passkey.register',
    })
    await waitFor(() =>
      expect(mocks.register).toHaveBeenCalledWith('passkey.register-proof')
    )
  })

  test('cancelling passkey verification leaves the passkey unregistered', async () => {
    mocks.requestVerification.mockResolvedValue(null)
    render(<NextProfilePasskeyCard loading={false} />)

    await userEvent.click(
      screen.getByRole('button', { name: 'Enable Passkey' })
    )

    await waitFor(() => expect(mocks.requestVerification).toHaveBeenCalled())
    expect(mocks.register).not.toHaveBeenCalled()
  })

  test('confirming passkey removal deletes it with the verified proof token', async () => {
    mocks.passkey.enabled = true
    mocks.requestVerification.mockResolvedValue(proofFor('passkey.delete'))
    render(<NextProfilePasskeyCard loading={false} />)

    await userEvent.click(
      screen.getByRole('button', { name: 'Remove Passkey' })
    )
    await userEvent.click(await screen.findByRole('button', { name: 'Remove' }))

    expect(mocks.requestVerification).toHaveBeenCalledWith(
      expect.objectContaining({ scope: 'passkey.delete' })
    )
    await waitFor(() =>
      expect(mocks.remove).toHaveBeenCalledWith('passkey.delete-proof')
    )
  })

  test('passkey status failure shows the error and retries on demand', async () => {
    mocks.passkey.statusError = 'Failed to load Passkey status'
    render(<NextProfilePasskeyCard loading={false} />)

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Failed to load Passkey status'
    )
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))

    expect(mocks.fetchStatus).toHaveBeenCalledTimes(1)
    expect(
      screen.queryByRole('button', { name: 'Enable Passkey' })
    ).not.toBeInTheDocument()
  })

  test('enabling two-step verification first requests 2fa setup verification', async () => {
    mocks.requestVerification.mockResolvedValue(null)
    render(<NextProfileTwoFACard loading={false} />)

    await userEvent.click(screen.getByRole('button', { name: 'Enable' }))

    expect(mocks.requestVerification).toHaveBeenCalledWith({
      scope: '2fa.setup',
    })
  })
})
