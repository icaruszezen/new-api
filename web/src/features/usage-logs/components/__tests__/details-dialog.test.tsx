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
import { describe, expect, test } from 'vitest'

import { formatLogQuota } from '@/lib/format'

import type { UsageLog } from '../../data/schema'
import type { LogOtherData } from '../../types'
import { DetailsDialog } from '../dialogs/details-dialog'

const WAIVED_NOTE =
  'Not charged: the upstream returned no output and no usage information.'

const CORRECTED_OTHER: LogOtherData = {
  billing_corrected: true,
  admin_info: {
    billing_correction: {
      rule_id: 7,
      channel_id: 5,
      prompt_tokens_threshold: 1_000_000,
      prompt_tokens: 3_456_789,
      original_quota: 1_728_394,
    },
  },
}

function buildConsumeLog(other: LogOtherData): UsageLog {
  return {
    id: 1,
    user_id: 1,
    created_at: 1_700_000_000,
    type: 2,
    content: '',
    username: 'alice',
    token_name: 'default',
    model_name: 'gpt-test',
    quota: 0,
    prompt_tokens: 3_456_789,
    completion_tokens: 0,
    use_time: 30,
    is_stream: true,
    channel: 5,
    channel_name: 'heartbeat-upstream',
    token_id: 1,
    group: 'default',
    ip: '',
    other: JSON.stringify(other),
    request_id: 'req-1',
    upstream_request_id: '',
  }
}

function renderDetails(props: { other: LogOtherData; isAdmin: boolean }) {
  return render(
    <DetailsDialog
      log={buildConsumeLog(props.other)}
      isAdmin={props.isAdmin}
      open
      onOpenChange={() => undefined}
    />
  )
}

describe('usage log details billing correction', () => {
  test('admins see the rule threshold, estimated input, original cost and rule id', async () => {
    renderDetails({ other: CORRECTED_OTHER, isAdmin: true })

    expect(await screen.findByText(WAIVED_NOTE)).toBeInTheDocument()
    expect(screen.getByText('Billing Correction')).toBeInTheDocument()
    expect(screen.getByText('Threshold').nextSibling).toHaveTextContent(
      /^1\D000\D000$/
    )
    expect(
      screen.getByText('Estimated input tokens').nextSibling
    ).toHaveTextContent(/^3\D456\D789$/)
    expect(screen.getByText('Original cost').nextSibling).toHaveTextContent(
      formatLogQuota(1_728_394)
    )
    expect(screen.getByText('Rule ID').nextSibling).toHaveTextContent('#7')
  })

  test('non-admins only see the waived note without the rule audit', async () => {
    renderDetails({ other: CORRECTED_OTHER, isAdmin: false })

    expect(await screen.findByText(WAIVED_NOTE)).toBeInTheDocument()
    expect(screen.queryByText('Billing Correction')).not.toBeInTheDocument()
    expect(screen.queryByText('Rule ID')).not.toBeInTheDocument()
  })

  test('a log without correction markers shows neither section', async () => {
    renderDetails({ other: {}, isAdmin: true })

    expect(await screen.findByText('Log Details')).toBeInTheDocument()
    expect(screen.queryByText(WAIVED_NOTE)).not.toBeInTheDocument()
    expect(screen.queryByText('Billing Correction')).not.toBeInTheDocument()
  })
})
