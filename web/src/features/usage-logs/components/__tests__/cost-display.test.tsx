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
import type React from 'react'
import { beforeAll, describe, expect, test } from 'vitest'

import { formatLogQuota } from '@/lib/format'

import { LogCostDisplay } from '../log-cost-display'

function renderCost(
  props: React.ComponentProps<typeof LogCostDisplay>
): ReturnType<typeof render> {
  return render(<LogCostDisplay {...props} />)
}

function normalizedText(value: string | null): string {
  return (value ?? '').replaceAll(/\s/g, '')
}

describe('log cost display', () => {
  beforeAll(() => {
    i18next.addResourceBundle('en', 'translation', {
      Subscription: 'Subscription',
      'Deducted by subscription': 'Deducted by subscription',
      'Includes tool-call surcharge': 'Includes tool-call surcharge',
      Waived: 'Waived',
      'Not charged: the upstream returned no output and no usage information.':
        'Not charged: the upstream returned no output and no usage information.',
    })
  })

  test('keeps the regular cost visible and adds an accessible surcharge marker', () => {
    const rendered = renderCost({
      quota: 12500,
      other: {
        tool_surcharges: [{ name: 'lookup_customer', count: 1, price: 5 }],
      },
    })

    expect(
      normalizedText(rendered.container.textContent).includes(
        normalizedText(formatLogQuota(12500))
      )
    ).toBe(true)
    const marker = screen.getByRole('img', {
      name: 'Includes tool-call surcharge',
    })
    expect(marker).toHaveAttribute('data-tool-surcharge-indicator', 'true')
    expect(marker).toHaveAttribute('tabindex', '0')
  })

  test('preserves the subscription badge and adds the same legacy surcharge marker', () => {
    renderCost({
      quota: 5000,
      other: {
        billing_source: 'subscription',
        web_search: true,
        web_search_call_count: 1,
        web_search_price: 10,
      },
    })

    expect(screen.getByText('Subscription')).toBeInTheDocument()
    expect(
      screen.getByRole('img', { name: 'Includes tool-call surcharge' })
    ).toHaveAttribute('data-tool-surcharge-indicator', 'true')
  })

  test('marks a billing-corrected request as waived next to its zero cost', () => {
    const rendered = renderCost({
      quota: 0,
      other: { billing_corrected: true },
    })

    expect(
      normalizedText(rendered.container.textContent).includes(
        normalizedText(formatLogQuota(0))
      )
    ).toBe(true)
    const marker = screen.getByText('Waived')
    expect(marker).toHaveAttribute('data-billing-corrected-indicator', 'true')
    expect(marker).toHaveAttribute('tabindex', '0')
  })

  test('explains why the request was waived when the marker is hovered', async () => {
    const user = userEvent.setup()
    renderCost({ quota: 0, other: { billing_corrected: true } })

    await user.hover(screen.getByText('Waived'))

    expect(
      await screen.findByText(
        'Not charged: the upstream returned no output and no usage information.'
      )
    ).toBeInTheDocument()
  })

  test('does not mark a regular zero-cost request as waived', () => {
    renderCost({ quota: 0, other: { billing_corrected: false } })

    expect(screen.queryByText('Waived')).not.toBeInTheDocument()
  })
})
