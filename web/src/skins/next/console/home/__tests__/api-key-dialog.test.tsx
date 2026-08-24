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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useEffect } from 'react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import {
  ApiKeysProvider,
  useApiKeys,
} from '@/features/keys/components/api-keys-provider'
import type { ApiKey } from '@/features/keys/types'

const apiMocks = vi.hoisted(() => ({
  createApiKey: vi.fn(),
  updateApiKey: vi.fn(),
  getApiKey: vi.fn(),
  getTokenAutoGroups: vi.fn(),
  getUserGroups: vi.fn(),
  getUserModels: vi.fn(),
  getStatus: vi.fn(),
}))

vi.mock('@/features/keys/api', () => ({
  createApiKey: apiMocks.createApiKey,
  updateApiKey: apiMocks.updateApiKey,
  getApiKey: apiMocks.getApiKey,
  getTokenAutoGroups: apiMocks.getTokenAutoGroups,
}))

vi.mock('@/lib/api', () => ({
  getUserGroups: apiMocks.getUserGroups,
  getUserModels: apiMocks.getUserModels,
  getStatus: apiMocks.getStatus,
}))

const { NextApiKeyDialog } = await import('../components/api-key-dialog')

const sampleKey: ApiKey = {
  id: 1,
  name: 'prod-gateway',
  key: 'd7af123443e0',
  status: 1,
  remain_quota: 0,
  used_quota: 0,
  unlimited_quota: true,
  expired_time: -1,
  created_time: 1,
  accessed_time: 0,
  group: 'default',
  auto_groups: null,
  cross_group_retry: false,
  model_limits_enabled: false,
  model_limits: '',
  allow_ips: '',
}

const groupsFixture = {
  auto: { desc: 'Automatic routing', ratio: 'auto' },
  default: { desc: 'Standard access', ratio: 1 },
}

type DialogMode = 'create' | 'update'

function DialogHarness(props: { mode: DialogMode }) {
  const { setOpen, setCurrentRow } = useApiKeys()

  useEffect(() => {
    if (props.mode === 'update') {
      setCurrentRow(sampleKey)
      setOpen('update')
      return
    }
    setOpen('create')
  }, [props.mode, setCurrentRow, setOpen])

  return <NextApiKeyDialog />
}

function seedQueries(queryClient: QueryClient, mode: DialogMode): void {
  const freshAt = Date.now() + 60_000
  queryClient.setQueryData(
    ['status'],
    { default_use_auto_group: true },
    { updatedAt: freshAt }
  )
  queryClient.setQueryData(
    ['user-models'],
    { success: true, data: [] },
    { updatedAt: freshAt }
  )
  queryClient.setQueryData(
    ['user-groups'],
    { success: true, data: groupsFixture },
    { updatedAt: freshAt }
  )
  queryClient.setQueryData(
    ['token-auto-groups'],
    { success: true, data: { groups: ['default'], max_count: 3 } },
    { updatedAt: freshAt }
  )
  if (mode === 'update') {
    queryClient.setQueryData(
      ['api-key', sampleKey.id],
      { success: true, data: sampleKey },
      { updatedAt: freshAt }
    )
  }
}

function renderDialog(mode: DialogMode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  seedQueries(queryClient, mode)
  render(
    <QueryClientProvider client={queryClient}>
      <ApiKeysProvider>
        <DialogHarness mode={mode} />
      </ApiKeysProvider>
    </QueryClientProvider>
  )
  return queryClient
}

function findButton(text: string): HTMLButtonElement {
  const button = screen
    .queryAllByRole<HTMLButtonElement>('button')
    .find((candidate) => candidate.textContent?.includes(text))
  if (!button) {
    throw new Error(`Expected button containing "${text}"`)
  }
  return button
}

function getControlByLabel(labelText: string): HTMLElement {
  const label = [...document.querySelectorAll<HTMLLabelElement>('label')].find(
    (candidate) => candidate.textContent?.trim() === labelText
  )
  if (!label) {
    throw new Error(`Expected label "${labelText}"`)
  }

  const control =
    label.control ??
    label
      .closest('[data-slot="form-item"]')
      ?.querySelector<HTMLElement>(
        '[data-slot="form-control"], input, textarea, button[role="combobox"]'
      )
  if (!control) {
    throw new Error(`Expected control for label "${labelText}"`)
  }
  return control
}

describe('next API key dialog', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  test('opens a centered dialog with only name and group on the basic tab', async () => {
    renderDialog('create')

    await waitFor(() => {
      expect(findButton('Create key')).toBeEnabled()
    })

    expect(screen.getByRole('dialog')).toBeVisible()
    expect(
      document.querySelector('[data-slot="next-api-key-dialog"]')
    ).toBeTruthy()
    expect(document.querySelector('[data-slot="sheet-content"]')).toBeNull()
    expect(getControlByLabel('Name')).toBeTruthy()
    expect(getControlByLabel('Group')).toBeTruthy()
    expect(screen.queryByRole('tab', { name: 'Quota Settings' })).toBeVisible()
    expect(screen.queryByLabelText('Expiration Time')).toBeNull()
    expect(screen.queryByLabelText('Quantity')).toBeNull()
  })

  test('creates one never-expiring unlimited key from the basic tab', async () => {
    const createdPayloads: Array<Record<string, unknown>> = []
    apiMocks.createApiKey.mockImplementation(async (data) => {
      createdPayloads.push(data as Record<string, unknown>)
      return { success: true, data: {} }
    })
    renderDialog('create')

    await waitFor(() => {
      expect(findButton('Create key')).toBeEnabled()
    })

    fireEvent.input(getControlByLabel('Name'), {
      target: { value: 'prod-gateway' },
    })
    fireEvent.click(findButton('Create key'))

    await waitFor(() => expect(createdPayloads).toHaveLength(1))
    expect(createdPayloads[0]?.name).toBe('prod-gateway')
    expect(createdPayloads[0]?.expired_time).toBe(-1)
    expect(createdPayloads[0]?.unlimited_quota).toBe(true)
    expect(createdPayloads[0]?.remain_quota).toBe(0)
  })

  test('lets the quota tab change quantity before create', async () => {
    const user = userEvent.setup()
    const createdPayloads: Array<Record<string, unknown>> = []
    apiMocks.createApiKey.mockImplementation(async (data) => {
      createdPayloads.push(data as Record<string, unknown>)
      return { success: true, data: {} }
    })
    renderDialog('create')

    await waitFor(() => {
      expect(findButton('Create key')).toBeEnabled()
    })

    fireEvent.input(getControlByLabel('Name'), { target: { value: 'batch' } })
    await user.click(screen.getByRole('tab', { name: 'Quota Settings' }))
    expect(getControlByLabel('Expiration Time')).toBeTruthy()
    fireEvent.input(getControlByLabel('Quantity'), { target: { value: '2' } })
    fireEvent.click(findButton('Create key'))

    await waitFor(() => expect(createdPayloads).toHaveLength(2))
    expect(createdPayloads[0]?.name).toBe('batch')
    expect(createdPayloads[0]?.expired_time).toBe(-1)
  })

  test('hides quantity when editing an existing key', async () => {
    const user = userEvent.setup()
    apiMocks.getApiKey.mockResolvedValue({ success: true, data: sampleKey })
    renderDialog('update')

    await waitFor(() => {
      expect(screen.getByText('Update API Key')).toBeVisible()
    })
    await waitFor(() => {
      expect(findButton('Save changes')).toBeEnabled()
    })

    expect(screen.getByRole('dialog')).toBeVisible()
    expect(document.querySelector('[data-slot="sheet-content"]')).toBeNull()
    await user.click(screen.getByRole('tab', { name: 'Quota Settings' }))
    expect(getControlByLabel('Expiration Time')).toBeTruthy()
    expect(screen.queryByLabelText('Quantity')).toBeNull()
  })
})
