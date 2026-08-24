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
import type { ApiInfoItem } from '@/features/dashboard/types'
import { API_KEY_STATUS } from '@/features/keys/constants'
import type { ApiKey } from '@/features/keys/types'

import { maskApiKey } from './mask-key'

export const TEST_PROMPT = 'are you ok?'
export const CHAT_COMPLETIONS_FORMAT = '/chat/completions'
export const RESPONSES_FORMAT = '/responses'
export const MESSAGES_FORMAT = '/messages'

export type TestRequestFormat =
  | typeof CHAT_COMPLETIONS_FORMAT
  | typeof RESPONSES_FORMAT
  | typeof MESSAGES_FORMAT

export type TestEndpointOption = {
  id: string
  label: string
  baseUrl: string
}

export type TestConnectionPhase = 'idle' | 'running' | 'success' | 'error'

function stripTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '')
}

export function resolveFallbackOrigin(serverAddress?: string): string {
  const trimmed = serverAddress?.trim()
  if (trimmed) return stripTrailingSlash(trimmed)
  if (typeof window !== 'undefined' && window.location.origin) {
    return window.location.origin
  }
  return ''
}

export function normalizeApiBase(
  sourceUrl: string | undefined,
  fallbackOrigin: string
): string {
  const fallback = fallbackOrigin ? `${stripTrailingSlash(fallbackOrigin)}/v1` : '/v1'
  const trimmed = sourceUrl?.trim()
  if (!trimmed) return fallback

  const withoutSlash = stripTrailingSlash(trimmed)
  if (withoutSlash.endsWith('/v1/chat/completions')) {
    return withoutSlash.slice(0, -'/chat/completions'.length)
  }
  if (withoutSlash.endsWith('/v1/responses')) {
    return withoutSlash.slice(0, -'/responses'.length)
  }
  if (withoutSlash.endsWith('/v1/messages')) {
    return withoutSlash.slice(0, -'/messages'.length)
  }
  if (withoutSlash.endsWith('/v1')) {
    return withoutSlash
  }
  return `${withoutSlash}/v1`
}

export function buildEndpointOptions(
  items: ApiInfoItem[],
  fallbackOrigin: string,
  defaultLabel: string
): TestEndpointOption[] {
  if (items.length === 0) {
    const baseUrl = normalizeApiBase(undefined, fallbackOrigin)
    return [
      {
        id: baseUrl,
        label: `${defaultLabel} · ${baseUrl}`,
        baseUrl,
      },
    ]
  }

  return items.map((item, index) => {
    const baseUrl = normalizeApiBase(item.url, fallbackOrigin)
    const name = item.description || item.route || defaultLabel
    return {
      id: `${index}:${baseUrl}`,
      label: `${name} · ${baseUrl}`,
      baseUrl,
    }
  })
}

export function formatKeyOptionLabel(
  key: Pick<ApiKey, 'group' | 'name' | 'key'>
): string {
  const group = key.group?.trim() || '-'
  return `${group} · ${key.name} · ${maskApiKey(key.key)}`
}

export function getPreferredApiKey(keys: ApiKey[]): ApiKey | null {
  return keys.find((key) => key.status === API_KEY_STATUS.ENABLED) ?? keys[0] ?? null
}

export function parseModelLimits(
  key: Pick<ApiKey, 'model_limits_enabled' | 'model_limits'>
): string[] | null {
  if (!key.model_limits_enabled) return null
  const models = (key.model_limits ?? '')
    .split(',')
    .map((model) => model.trim())
    .filter(Boolean)
  return models.length > 0 ? models : null
}

export function pickModel(models: string[], current?: string): string {
  if (current && models.includes(current)) return current
  return models[0] ?? ''
}

export function isOpenAIModel(model: string): boolean {
  const name = model.toLowerCase()
  if (name.includes('gpt-') || name.includes('chatgpt-')) return true
  return /\bo[134](?:[-.]|$)/.test(name)
}

export function isClaudeModel(model: string): boolean {
  const name = model.toLowerCase()
  return name.includes('claude') || name.includes('anthropic')
}

export function formatsForModel(model: string): TestRequestFormat[] {
  if (isClaudeModel(model)) {
    return [MESSAGES_FORMAT, CHAT_COMPLETIONS_FORMAT]
  }
  if (isOpenAIModel(model)) {
    return [RESPONSES_FORMAT, CHAT_COMPLETIONS_FORMAT]
  }
  return [CHAT_COMPLETIONS_FORMAT]
}

export function pickFormat(
  formats: string[],
  current?: string
): TestRequestFormat {
  if (current && formats.includes(current)) {
    return current as TestRequestFormat
  }
  return (formats[0] as TestRequestFormat | undefined) ?? CHAT_COMPLETIONS_FORMAT
}

export function buildTestUrl(baseUrl: string, format: string): string {
  return `${stripTrailingSlash(baseUrl)}${format}`
}

export function buildTestPayload(
  model: string,
  format: string = CHAT_COMPLETIONS_FORMAT
): Record<string, unknown> {
  if (format === RESPONSES_FORMAT) {
    return {
      model,
      input: TEST_PROMPT,
      stream: true,
    }
  }
  if (format === MESSAGES_FORMAT) {
    return {
      model,
      max_tokens: 1024,
      messages: [{ role: 'user', content: TEST_PROMPT }],
      stream: true,
    }
  }
  return {
    model,
    messages: [{ role: 'user', content: TEST_PROMPT }],
    stream: true,
  }
}

export function buildRequestHeaders(
  apiKey: string,
  format: string = CHAT_COMPLETIONS_FORMAT
): Record<string, string> {
  const headers: Record<string, string> = {
    Accept: 'text/event-stream',
    Authorization: `Bearer ${apiKey}`,
    'Content-Type': 'application/json',
  }
  if (format === MESSAGES_FORMAT) {
    headers['anthropic-version'] = '2023-06-01'
  }
  return headers
}

export function buildCurlCommand(args: {
  endpoint: string
  apiKey: string
  model: string
  format?: string
}): string {
  const format = args.format ?? CHAT_COMPLETIONS_FORMAT
  const payload = JSON.stringify(buildTestPayload(args.model, format))
  const lines = [
    `curl ${args.endpoint} \\`,
    '  -H "Content-Type: application/json" \\',
    `  -H "Authorization: Bearer ${args.apiKey}" \\`,
  ]
  if (format === MESSAGES_FORMAT) {
    lines.push('  -H "anthropic-version: 2023-06-01" \\')
  }
  lines.push(`  -d '${payload}'`)
  return lines.join('\n')
}
