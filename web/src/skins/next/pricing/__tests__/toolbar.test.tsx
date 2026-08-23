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
import { describe, expect, test, vi } from 'vitest'

import { NextPricingToolbar } from '../components/toolbar'

vi.mock('@/features/pricing/components/pricing-sidebar', () => ({
  PricingSidebar: () => <div data-testid='pricing-sidebar' />,
}))

function renderToolbar() {
  return render(
    <NextPricingToolbar
      filteredCount={12}
      totalCount={20}
      quotaTypeFilter=''
      endpointTypeFilter=''
      vendorFilter=''
      groupFilter=''
      tagFilter=''
      onQuotaTypeChange={vi.fn()}
      onEndpointTypeChange={vi.fn()}
      onVendorChange={vi.fn()}
      onGroupChange={vi.fn()}
      onTagChange={vi.fn()}
      vendors={[]}
      groups={[]}
      tags={[]}
      models={[]}
      hasActiveFilters
      activeFilterCount={2}
      onClearFilters={vi.fn()}
    />
  )
}

describe('next model square toolbar', () => {
  test('opens the filter sheet and shows the active count without a primary badge', async () => {
    const user = userEvent.setup()
    renderToolbar()

    const filter = screen.getByRole('button', { name: /Filter/ })
    expect(filter).toHaveTextContent('2')
    expect(filter.querySelector('[data-slot="badge"]')).toBeNull()
    expect(screen.queryByRole('group', { name: 'Price display mode' })).toBeNull()
    expect(screen.queryByRole('group', { name: 'Token unit' })).toBeNull()
    expect(screen.queryByRole('button', { name: /Sort|Name/ })).toBeNull()
    expect(screen.queryByText('Standard')).toBeNull()
    expect(screen.queryByText('Recharge')).toBeNull()
    expect(screen.queryByText('/1M')).toBeNull()
    expect(screen.queryByText('/1K')).toBeNull()

    await user.click(filter)

    expect(await screen.findByRole('heading', { name: 'Filter' })).toBeVisible()
    expect(screen.getByTestId('pricing-sidebar')).toBeInTheDocument()
  })
})
