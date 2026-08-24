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
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { ApiKeysProvider } from '@/features/keys/components/api-keys-provider'
import type { ApiKey } from '@/features/keys/types'
import type { ApiInfoItem } from '@/features/dashboard/types'

const apiMocks = vi.hoisted(() => ({
  fetchTokenKey: vi.fn(),
  getUserModels: vi.fn(),
  copyToClipboard: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}))

vi.mock('@/features/keys/api', () => ({
  fetchTokenKey: apiMocks.fetchTokenKey,
  fetchTokenKeysBatch: vi.fn(),
}))

vi.mock('@/lib/api', () => ({
  getUserModels: apiMocks.getUserModels,
}))

vi.mock('@/lib/copy-to-clipboard', () => ({
  copyToClipboard: apiMocks.copyToClipboard,
}))

vi.mock('sonner', () => ({
  toast: {
    success: (...args: unknown[]) => apiMocks.toastSuccess(...args),
    error: (...args: unknown[]) => apiMocks.toastError(...args),
  },
}))

const { TestConnectionDialog } = await import(
  '../components/test-connection-dialog'
)

function makeKey(overrides: Partial<ApiKey> = {}): ApiKey {
  return {
    id: 1,
    name: 'openai',
    key: 'd7af123443e0',
    status: 1,
    remain_quota: 0,
    used_quota: 0,
    unlimited_quota: true,
    expired_time: -1,
    created_time: 1,
    accessed_time: 0,
    group: 'plus',
    auto_groups: null,
    cross_group_retry: false,
    model_limits_enabled: true,
    model_limits: 'gpt-4o',
    allow_ips: '',
    ...overrides,
  }
}

const apiInfo: ApiInfoItem[] = [
  {
    url: 'https://ai.example.com/v1',
    route: 'default',
    description: 'Default',
    color: '',
  },
]

function renderDialog(args?: {
  keys?: ApiKey[]
  apiInfoItems?: ApiInfoItem[]
  open?: boolean
}) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const onOpenChange = vi.fn()
  render(
    <QueryClientProvider client={client}>
      <ApiKeysProvider>
        <TestConnectionDialog
          open={args?.open ?? true}
          onOpenChange={onOpenChange}
          keys={args?.keys ?? [makeKey()]}
          apiInfoItems={args?.apiInfoItems ?? apiInfo}
          serverAddress='https://ai.example.com'
        />
      </ApiKeysProvider>
    </QueryClientProvider>
  )
  return { onOpenChange }
}

function createSseResponse(chunks: string[]): Response {
  const encoder = new TextEncoder()
  const body = new ReadableStream<Uint8Array>({
    start(controller) {
      for (const chunk of chunks) {
        controller.enqueue(encoder.encode(chunk))
      }
      controller.close()
    },
  })
  return new Response(body, { status: 200 })
}

describe('next test connection dialog', () => {
  beforeEach(() => {
    apiMocks.fetchTokenKey.mockResolvedValue({
      success: true,
      data: { key: 'realkey1234abcd' },
    })
    apiMocks.getUserModels.mockResolvedValue({
      success: true,
      data: ['gpt-4o-mini'],
    })
    apiMocks.copyToClipboard.mockResolvedValue(true)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  test('renders the fields and idle result when the dialog is open', () => {
    renderDialog()

    expect(screen.getByRole('dialog')).toBeVisible()
    expect(screen.getByText('API KEY TEST')).toBeVisible()
    expect(screen.getByText('Test request link')).toBeVisible()
    expect(screen.getByLabelText('Test API key')).toBeVisible()
    expect(screen.getByLabelText('Request address')).toBeVisible()
    expect(screen.getByLabelText('Test Model')).toBeVisible()
    expect(screen.getByLabelText('Request format')).toBeVisible()
    expect(screen.getByLabelText('Request format')).toHaveTextContent(
      '/responses'
    )
    expect(screen.getByText('Ready to test')).toBeVisible()
    expect(screen.getByText('Click Start Test to begin...')).toBeVisible()
    expect(
      document.querySelector('[data-slot="test-connection-result"]')
    ).toHaveAttribute('data-phase', 'idle')
  })

  test('disables start and copy when the user has no API keys', () => {
    renderDialog({ keys: [] })

    expect(
      screen.getByText('Create an API key before testing the connection.')
    ).toBeVisible()
    expect(screen.getByRole('button', { name: 'Start Test' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Copy test link' })).toBeDisabled()
  })

  test('replaces the model list when a limited key is selected', async () => {
    const user = userEvent.setup()
    renderDialog({
      keys: [
        makeKey({ id: 1, name: 'openai', model_limits: 'gpt-4o' }),
        makeKey({
          id: 2,
          name: 'claude',
          key: 'abcd9999ef01',
          model_limits: 'claude-sonnet',
        }),
      ],
    })

    expect(screen.getByLabelText('Test Model')).toHaveValue('gpt-4o')

    await user.click(screen.getByLabelText('Test API key'))
    await user.click(await screen.findByRole('option', { name: /claude/ }))

    await waitFor(() => {
      expect(screen.getByLabelText('Test Model')).toHaveValue('claude-sonnet')
    })
    expect(screen.getByLabelText('Request format')).toHaveTextContent(
      '/messages'
    )
    expect(apiMocks.getUserModels).not.toHaveBeenCalled()
  })

  test('offers /messages for a Claude key and sends that request format', async () => {
    const user = userEvent.setup()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        createSseResponse([
          'data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"I am okay"}}\n\n',
          'data: {"type":"message_stop"}\n\n',
        ])
      )
    )
    renderDialog({
      keys: [
        makeKey({
          name: 'claude',
          model_limits: 'claude-sonnet',
        }),
      ],
    })

    expect(screen.getByLabelText('Request format')).toHaveTextContent(
      '/messages'
    )
    await user.click(screen.getByRole('button', { name: 'Start Test' }))

    await waitFor(() => {
      expect(screen.getByText('I am okay')).toBeVisible()
    })
    expect(fetch).toHaveBeenCalledWith(
      'https://ai.example.com/v1/messages',
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer sk-realkey1234abcd',
          'anthropic-version': '2023-06-01',
        }),
      })
    )
    const body = JSON.parse(
      (fetch as unknown as ReturnType<typeof vi.fn>).mock.calls[0]?.[1]
        ?.body as string
    )
    expect(body).toMatchObject({
      model: 'claude-sonnet',
      max_tokens: 1024,
      stream: true,
    })
  })

  test('streams the model reply into the result box after start', async () => {
    const user = userEvent.setup()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        createSseResponse([
          'data: {"type":"response.output_text.delta","delta":"Yes, I\'m okay"}\n\n',
          'data: {"type":"response.completed"}\n\n',
        ])
      )
    )
    renderDialog()

    await user.click(screen.getByRole('button', { name: 'Start Test' }))

    await waitFor(() => {
      expect(screen.getByText("Yes, I'm okay")).toBeVisible()
    })
    expect(screen.getByText('are you ok?')).toBeVisible()
    expect(
      document.querySelector('[data-slot="test-connection-result"]')
    ).toHaveAttribute('data-phase', 'success')
    expect(fetch).toHaveBeenCalledWith(
      'https://ai.example.com/v1/responses',
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer sk-realkey1234abcd',
        }),
      })
    )
    const body = JSON.parse(
      (fetch as unknown as ReturnType<typeof vi.fn>).mock.calls[0]?.[1]
        ?.body as string
    )
    expect(body).toEqual({
      model: 'gpt-4o',
      input: 'are you ok?',
      stream: true,
    })
  })

  test('shows testing in progress until the first streamed character arrives', async () => {
    const user = userEvent.setup()
    let resolveFetch: (value: Response) => void = () => {}
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(
        () =>
          new Promise<Response>((resolve) => {
            resolveFetch = resolve
          })
      )
    )
    renderDialog()

    await user.click(screen.getByRole('button', { name: 'Start Test' }))

    await waitFor(() => {
      expect(
        document.querySelector('[data-slot="test-connection-result"]')
      ).toHaveAttribute('data-phase', 'running')
    })
    expect(screen.queryByText('Click Start Test to begin...')).toBeNull()
    expect(screen.getAllByText('Testing in progress').length).toBeGreaterThan(0)

    resolveFetch(
      createSseResponse([
        'data: {"type":"response.output_text.delta","delta":"Yes"}\n\n',
        'data: {"type":"response.completed"}\n\n',
      ])
    )

    await waitFor(() => {
      expect(screen.getByText('Yes')).toBeVisible()
    })
    await waitFor(() => {
      expect(
        document.querySelector('[data-slot="test-connection-result"]')
      ).toHaveAttribute('data-phase', 'success')
    })
    expect(screen.queryByText('Click Start Test to begin...')).toBeNull()
    expect(screen.queryByText('Testing in progress')).toBeNull()
  })

  test('shows the server error text when the stream fails', async () => {
    const user = userEvent.setup()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: { message: 'model not found' } }), {
          status: 400,
        })
      )
    )
    renderDialog()

    await user.click(screen.getByRole('button', { name: 'Start Test' }))

    await waitFor(() => {
      expect(screen.getByText('model not found')).toBeVisible()
    })
    expect(screen.queryByText('Click Start Test to begin...')).toBeNull()
    expect(screen.queryByText('Testing in progress')).toBeNull()
    expect(
      document.querySelector('[data-slot="test-connection-result"]')
    ).toHaveAttribute('data-phase', 'error')
  })

  test('copies a curl command that includes the resolved key', async () => {
    const user = userEvent.setup()
    renderDialog()

    await user.click(screen.getByRole('button', { name: 'Copy test link' }))

    await waitFor(() => {
      expect(apiMocks.copyToClipboard).toHaveBeenCalled()
    })
    const copied = apiMocks.copyToClipboard.mock.calls[0]?.[0] as string
    expect(copied).toContain('Authorization: Bearer sk-realkey1234abcd')
    expect(copied).toContain('https://ai.example.com/v1/responses')
    expect(copied).toContain('are you ok?')
    expect(copied).toContain('"input":')
    expect(apiMocks.toastSuccess).toHaveBeenCalledWith('Copied to clipboard')
  })
})
