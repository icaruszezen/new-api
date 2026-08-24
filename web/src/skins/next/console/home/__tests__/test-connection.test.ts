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
import { afterEach, describe, expect, test, vi } from 'vitest'

import type { ApiKey } from '@/features/keys/types'

import {
  consumeSseBuffer,
  readErrorResponse,
  streamChatCompletion,
} from '../lib/test-connection-stream'
import {
  buildCurlCommand,
  buildEndpointOptions,
  buildTestPayload,
  buildTestUrl,
  CHAT_COMPLETIONS_FORMAT,
  formatsForModel,
  formatKeyOptionLabel,
  getPreferredApiKey,
  MESSAGES_FORMAT,
  normalizeApiBase,
  parseModelLimits,
  pickFormat,
  pickModel,
  RESPONSES_FORMAT,
  resolveFallbackOrigin,
  TEST_PROMPT,
} from '../lib/test-connection'

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
    model_limits_enabled: false,
    model_limits: '',
    allow_ips: '',
    ...overrides,
  }
}

describe('normalizeApiBase', () => {
  test('strips a chat completions suffix down to /v1', () => {
    expect(
      normalizeApiBase('https://api.example.com/v1/chat/completions', '')
    ).toBe('https://api.example.com/v1')
    expect(normalizeApiBase('https://api.example.com/v1/responses', '')).toBe(
      'https://api.example.com/v1'
    )
    expect(normalizeApiBase('https://api.example.com/v1/messages', '')).toBe(
      'https://api.example.com/v1'
    )
  })

  test('keeps an existing /v1 base and appends /v1 to a bare origin', () => {
    expect(normalizeApiBase('https://api.example.com/v1/', '')).toBe(
      'https://api.example.com/v1'
    )
    expect(normalizeApiBase('https://api.example.com', '')).toBe(
      'https://api.example.com/v1'
    )
  })

  test('falls back to the server origin when the url is blank', () => {
    expect(normalizeApiBase('  ', 'https://ai.example.com')).toBe(
      'https://ai.example.com/v1'
    )
  })
})

describe('buildEndpointOptions', () => {
  test('uses api_info labels and a default fallback when the list is empty', () => {
    expect(
      buildEndpointOptions([], 'https://ai.example.com', 'Default')
    ).toEqual([
      {
        id: 'https://ai.example.com/v1',
        label: 'Default · https://ai.example.com/v1',
        baseUrl: 'https://ai.example.com/v1',
      },
    ])

    const options = buildEndpointOptions(
      [
        {
          url: 'https://api.example.com/v1',
          route: 'default',
          description: 'Default',
          color: '',
        },
      ],
      'https://unused.example.com',
      'Default'
    )
    expect(options[0]?.label).toBe('Default · https://api.example.com/v1')
  })
})

describe('key and model helpers', () => {
  test('formats a key option as group, name, and masked secret', () => {
    expect(formatKeyOptionLabel(makeKey())).toBe(
      'plus · openai · sk-d7af***43e0'
    )
  })

  test('prefers the first enabled key and falls back to the first row', () => {
    const disabled = makeKey({ id: 1, status: 2, name: 'off' })
    const enabled = makeKey({ id: 2, name: 'on' })
    expect(getPreferredApiKey([disabled, enabled])?.id).toBe(2)
    expect(getPreferredApiKey([disabled])?.id).toBe(1)
    expect(getPreferredApiKey([])).toBeNull()
  })

  test('reads model limits only when the key enables a non-empty list', () => {
    expect(parseModelLimits(makeKey())).toBeNull()
    expect(
      parseModelLimits(
        makeKey({ model_limits_enabled: true, model_limits: 'gpt-4o, claude' })
      )
    ).toEqual(['gpt-4o', 'claude'])
    expect(
      parseModelLimits(makeKey({ model_limits_enabled: true, model_limits: '' }))
    ).toBeNull()
  })

  test('keeps the current model when it is still available', () => {
    expect(pickModel(['gpt-4o', 'gpt-4.1'], 'gpt-4.1')).toBe('gpt-4.1')
    expect(pickModel(['gpt-4o'], 'missing')).toBe('gpt-4o')
    expect(pickModel([])).toBe('')
  })

  test('offers /responses for OpenAI models and /messages for Claude models', () => {
    expect(formatsForModel('gpt-4o')).toEqual([
      RESPONSES_FORMAT,
      CHAT_COMPLETIONS_FORMAT,
    ])
    expect(formatsForModel('o3-mini')).toEqual([
      RESPONSES_FORMAT,
      CHAT_COMPLETIONS_FORMAT,
    ])
    expect(formatsForModel('claude-sonnet-4')).toEqual([
      MESSAGES_FORMAT,
      CHAT_COMPLETIONS_FORMAT,
    ])
    expect(formatsForModel('deepseek-chat')).toEqual([CHAT_COMPLETIONS_FORMAT])
    expect(pickFormat(formatsForModel('gpt-4o'))).toBe(RESPONSES_FORMAT)
    expect(pickFormat(formatsForModel('claude-3'), CHAT_COMPLETIONS_FORMAT)).toBe(
      CHAT_COMPLETIONS_FORMAT
    )
  })
})

describe('request builders', () => {
  test('builds the streaming chat completions url, payload, and curl', () => {
    expect(buildTestUrl('https://api.example.com/v1', '/chat/completions')).toBe(
      'https://api.example.com/v1/chat/completions'
    )
    expect(buildTestPayload('gpt-4o')).toEqual({
      model: 'gpt-4o',
      messages: [{ role: 'user', content: TEST_PROMPT }],
      stream: true,
    })
    expect(buildTestPayload('gpt-4o', RESPONSES_FORMAT)).toEqual({
      model: 'gpt-4o',
      input: TEST_PROMPT,
      stream: true,
    })
    expect(buildTestPayload('claude-sonnet', MESSAGES_FORMAT)).toEqual({
      model: 'claude-sonnet',
      max_tokens: 1024,
      messages: [{ role: 'user', content: TEST_PROMPT }],
      stream: true,
    })
    expect(
      buildCurlCommand({
        endpoint: 'https://api.example.com/v1/chat/completions',
        apiKey: 'sk-secret',
        model: 'gpt-4o',
      })
    ).toContain('Authorization: Bearer sk-secret')
    expect(
      buildCurlCommand({
        endpoint: 'https://api.example.com/v1/messages',
        apiKey: 'sk-secret',
        model: 'claude-sonnet',
        format: MESSAGES_FORMAT,
      })
    ).toContain('anthropic-version: 2023-06-01')
  })

  test('uses the configured server address before the page origin', () => {
    expect(resolveFallbackOrigin(' https://ai.example.com/ ')).toBe(
      'https://ai.example.com'
    )
    expect(resolveFallbackOrigin()).toBe(window.location.origin)
  })
})

describe('consumeSseBuffer', () => {
  test('extracts streamed content and stops at DONE', () => {
    const parsed = consumeSseBuffer(
      'data: {"choices":[{"delta":{"content":"Yes"}}]}\n\ndata: [DONE]\n\n'
    )
    expect(parsed.content).toBe('Yes')
    expect(parsed.done).toBe(true)
    expect(parsed.rest).toBe('')
  })

  test('surfaces an SSE error payload and leaves a partial frame in rest', () => {
    const parsed = consumeSseBuffer(
      'data: {"error":{"message":"quota exceeded"}}\n\ndata: {"choices"'
    )
    expect(parsed.error).toBe('quota exceeded')
    expect(parsed.rest).toBe('data: {"choices"')
  })

  test('extracts OpenAI Responses and Claude Messages stream deltas', () => {
    const responses = consumeSseBuffer(
      'data: {"type":"response.output_text.delta","delta":"Yes"}\n\ndata: {"type":"response.completed"}\n\n'
    )
    expect(responses.content).toBe('Yes')
    expect(responses.done).toBe(true)

    const claude = consumeSseBuffer(
      'data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Okay"}}\n\ndata: {"type":"message_stop"}\n\n'
    )
    expect(claude.content).toBe('Okay')
    expect(claude.done).toBe(true)
  })
})

describe('streamChatCompletion', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  test('appends delta content from a successful SSE response', async () => {
    const chunks: string[] = []
    const encoder = new TextEncoder()
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(
          encoder.encode(
            'data: {"choices":[{"delta":{"content":"Yes, I\'m okay"}}]}\n\n'
          )
        )
        controller.enqueue(encoder.encode('data: [DONE]\n\n'))
        controller.close()
      },
    })
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(body, { status: 200 }))
    )

    await streamChatCompletion({
      url: 'https://api.example.com/v1/chat/completions',
      apiKey: 'sk-secret',
      model: 'gpt-4o',
      signal: new AbortController().signal,
      onContent: (chunk) => {
        chunks.push(chunk)
      },
    })

    expect(chunks.join('')).toBe("Yes, I'm okay")
    expect(fetch).toHaveBeenCalledWith(
      'https://api.example.com/v1/chat/completions',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({
          Authorization: 'Bearer sk-secret',
        }),
      })
    )
  })

  test('posts an OpenAI Responses body when that format is selected', async () => {
    const encoder = new TextEncoder()
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(
          encoder.encode(
            'data: {"type":"response.output_text.delta","delta":"Yes"}\n\n'
          )
        )
        controller.enqueue(
          encoder.encode('data: {"type":"response.completed"}\n\n')
        )
        controller.close()
      },
    })
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(body, { status: 200 }))
    )

    await streamChatCompletion({
      url: 'https://api.example.com/v1/responses',
      apiKey: 'sk-secret',
      model: 'gpt-4o',
      format: RESPONSES_FORMAT,
      signal: new AbortController().signal,
      onContent: () => undefined,
    })

    const init = (fetch as unknown as ReturnType<typeof vi.fn>).mock
      .calls[0]?.[1] as RequestInit
    expect(JSON.parse(String(init.body))).toEqual({
      model: 'gpt-4o',
      input: TEST_PROMPT,
      stream: true,
    })
  })

  test('throws the server error body when the response is not ok', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: { message: 'invalid key' } }), {
          status: 401,
        })
      )
    )

    await expect(
      streamChatCompletion({
        url: 'https://api.example.com/v1/chat/completions',
        apiKey: 'sk-bad',
        model: 'gpt-4o',
        signal: new AbortController().signal,
        onContent: () => undefined,
      })
    ).rejects.toThrow('invalid key')
  })

  test('reads a plain HTTP status when the error body is empty', async () => {
    expect(await readErrorResponse(new Response('', { status: 502 }))).toBe(
      'HTTP 502'
    )
  })
})
