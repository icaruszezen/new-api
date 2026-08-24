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
import type { ReactElement } from 'react'
import { describe, expect, test } from 'vitest'

import { TooltipProvider } from '@/components/ui/tooltip'

import { ConsoleKeyGroupCell } from '../components/key-group-cell'

function renderGroupCell(ui: ReactElement) {
  return render(<TooltipProvider>{ui}</TooltipProvider>)
}

describe('next console key group cell', () => {
  test('renders every group name in the same muted badge color', () => {
    const { rerender } = renderGroupCell(
      <ConsoleKeyGroupCell group='special' ratio={0.12} />
    )

    const special = screen
      .getByText('special')
      .closest('[data-slot="status-badge"]')
    expect(special).toHaveClass('text-muted-foreground')

    rerender(
      <TooltipProvider>
        <ConsoleKeyGroupCell group='vip' ratio={3} />
      </TooltipProvider>
    )

    const vip = screen.getByText('vip').closest('[data-slot="status-badge"]')
    expect(vip).toHaveClass('text-muted-foreground')
    expect(vip?.className).toBe(special?.className)
  })

  test('shows the model-square ratio tag beside a numeric group ratio', () => {
    renderGroupCell(<ConsoleKeyGroupCell group='special' ratio={0.12} />)

    const ratio = screen.getByText('0.12x')
    expect(ratio).toHaveClass(
      'inline-flex',
      'shrink-0',
      'rounded-[4px]',
      'text-sm',
      'backdrop-blur-md',
      'backdrop-saturate-150'
    )
    expect(ratio.className).toMatch(/bg-green-(500|400)\/15/)
    expect(ratio.className).not.toMatch(/rounded-full/)
  })

  test('keeps the auto group monochrome and hides a non-numeric ratio', () => {
    renderGroupCell(<ConsoleKeyGroupCell group='auto' ratio='Auto' />)

    const badge = screen
      .getByText('Cross-group')
      .closest('[data-slot="status-badge"]')
    expect(badge).toHaveClass('text-muted-foreground')
    expect(screen.queryByText(/x$/)).toBeNull()
    expect(screen.queryByText('Auto Ratio')).toBeNull()
  })
})
