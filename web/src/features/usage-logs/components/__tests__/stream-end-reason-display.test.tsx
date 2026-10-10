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
import i18next from 'i18next'
import { afterEach, beforeEach, describe, expect, test } from 'vitest'

import zh from '@/i18n/locales/zh.json'

import type { UsageLog } from '../../data/schema'
import { DetailsDialog } from '../dialogs/details-dialog'
import { StreamTpsCell } from '../timing-metrics-cell'

function streamLog(endReason: string): UsageLog {
  return {
    id: 1,
    user_id: 1,
    created_at: 1_700_000_000,
    type: 2,
    content: '',
    username: '',
    token_name: '',
    model_name: 'gpt-4o',
    quota: 0,
    prompt_tokens: 0,
    completion_tokens: 0,
    use_time: 1,
    is_stream: true,
    channel: 0,
    channel_name: '',
    token_id: 0,
    group: '',
    ip: '',
    other: JSON.stringify({
      stream_status: {
        status: 'error',
        end_reason: endReason,
        end_error: 'context canceled',
      },
    }),
    request_id: '',
    upstream_request_id: '',
  }
}

async function hoverStreamErrorIcon(container: HTMLElement): Promise<void> {
  const trigger = container.querySelector<HTMLElement>(
    '[data-slot="tooltip-trigger"]'
  )
  expect(trigger).not.toBeNull()
  await userEvent.setup().hover(trigger as HTMLElement)
}

beforeEach(async () => {
  i18next.options.keySeparator = false
  i18next.options.nsSeparator = false
  i18next.addResourceBundle('zhCN', 'translation', zh.translation, true, true)
  await i18next.changeLanguage('zhCN')
})

afterEach(async () => {
  i18next.options.keySeparator = '.'
  i18next.options.nsSeparator = ':'
  await i18next.changeLanguage('en')
})

describe('stream end reason display', () => {
  test('shows a client_gone stream as 客户端主动取消 in the stream tooltip', async () => {
    const rendered = render(
      <StreamTpsCell
        isStream
        streamStatus={{ status: 'error', end_reason: 'client_gone' }}
      />
    )

    await hoverStreamErrorIcon(rendered.container)

    expect(await screen.findByText('客户端主动取消')).toBeInTheDocument()
    expect(screen.queryByText('client_gone')).not.toBeInTheDocument()
  })

  test('keeps other end reasons as the raw backend value in the stream tooltip', async () => {
    const rendered = render(
      <StreamTpsCell
        isStream
        streamStatus={{ status: 'error', end_reason: 'scanner_error' }}
      />
    )

    await hoverStreamErrorIcon(rendered.container)

    expect(await screen.findByText('scanner_error')).toBeInTheDocument()
  })

  test('shows a client_gone stream as 客户端主动取消 in the details end reason row', () => {
    render(
      <DetailsDialog
        log={streamLog('client_gone')}
        isAdmin={false}
        open
        onOpenChange={() => {}}
      />
    )

    expect(screen.getByText('结束原因')).toBeInTheDocument()
    expect(screen.getByText('客户端主动取消')).toBeInTheDocument()
    expect(screen.queryByText('client_gone')).not.toBeInTheDocument()
  })
})
