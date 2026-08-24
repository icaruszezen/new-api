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
import {
  buildRequestHeaders,
  buildTestPayload,
  CHAT_COMPLETIONS_FORMAT,
} from './test-connection'

export function isAbortError(error: unknown): boolean {
  if (error instanceof DOMException && error.name === 'AbortError') return true
  return error instanceof Error && error.name === 'AbortError'
}

type StreamChoice = {
  delta?: { content?: string }
  message?: { content?: string }
}

type StreamDelta = {
  content?: string
  text?: string
}

type StreamPayload = {
  type?: string
  error?: { message?: string } | string
  message?: string
  delta?: string | StreamDelta
  choices?: StreamChoice[]
}

export function consumeSseBuffer(buffer: string): {
  content: string
  rest: string
  done: boolean
  error?: string
} {
  const parts = buffer.split('\n\n')
  const rest = parts.pop() ?? ''
  let content = ''
  let done = false
  let error: string | undefined

  for (const block of parts) {
    const result = parseSseBlock(block)
    if (result.content) content += result.content
    if (result.done) done = true
    if (result.error) {
      error = result.error
      break
    }
  }

  return { content, rest, done, error }
}

function parseSseBlock(block: string): {
  content?: string
  done?: boolean
  error?: string
} {
  const dataLines = block
    .split('\n')
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice(5).trimStart())

  if (dataLines.length === 0) {
    const trimmed = block.trim()
    if (trimmed.startsWith('{')) return parseStreamPayload(trimmed)
    return {}
  }

  const data = dataLines.join('\n')
  if (data === '[DONE]') return { done: true }
  return parseStreamPayload(data)
}

function parseStreamPayload(data: string): {
  content?: string
  done?: boolean
  error?: string
} {
  try {
    const parsed = JSON.parse(data) as StreamPayload
    if (parsed.error) {
      const message =
        typeof parsed.error === 'string' ? parsed.error : parsed.error.message
      return { error: message || data }
    }
    if (
      parsed.type === 'response.completed' ||
      parsed.type === 'message_stop'
    ) {
      return { done: true }
    }
    const chunk = extractStreamText(parsed)
    return chunk ? { content: chunk } : {}
  } catch {
    return {}
  }
}

function extractStreamText(parsed: StreamPayload): string | undefined {
  const chat =
    parsed.choices?.[0]?.delta?.content ??
    parsed.choices?.[0]?.message?.content
  if (chat) return chat

  if (parsed.type === 'response.output_text.delta') {
    if (typeof parsed.delta === 'string' && parsed.delta) return parsed.delta
  }

  if (parsed.type === 'content_block_delta') {
    if (
      parsed.delta &&
      typeof parsed.delta === 'object' &&
      parsed.delta.text
    ) {
      return parsed.delta.text
    }
  }

  return undefined
}

export async function readErrorResponse(response: Response): Promise<string> {
  const text = await response.text()
  if (!text) return `HTTP ${response.status}`
  try {
    const parsed = JSON.parse(text) as StreamPayload
    if (typeof parsed.error === 'string' && parsed.error) return parsed.error
    if (typeof parsed.error === 'object' && parsed.error?.message) {
      return parsed.error.message
    }
    if (parsed.message) return parsed.message
  } catch {
    return text
  }
  return text
}

export async function streamChatCompletion(args: {
  url: string
  apiKey: string
  model: string
  format?: string
  signal: AbortSignal
  onContent: (chunk: string) => void
}): Promise<void> {
  const format = args.format ?? CHAT_COMPLETIONS_FORMAT
  let response: Response
  try {
    response = await fetch(args.url, {
      method: 'POST',
      headers: buildRequestHeaders(args.apiKey, format),
      body: JSON.stringify(buildTestPayload(args.model, format)),
      signal: args.signal,
    })
  } catch (error) {
    if (isAbortError(error)) return
    throw new Error(error instanceof Error ? error.message : 'Failed to fetch')
  }

  if (!response.ok) {
    throw new Error(await readErrorResponse(response))
  }
  if (!response.body) {
    throw new Error('Empty response')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const parsed = consumeSseBuffer(buffer)
    buffer = parsed.rest
    if (parsed.content) args.onContent(parsed.content)
    if (parsed.error) throw new Error(parsed.error)
    if (parsed.done) return
  }

  const parsed = consumeSseBuffer(`${buffer}\n\n`)
  if (parsed.content) args.onContent(parsed.content)
  if (parsed.error) throw new Error(parsed.error)
}
