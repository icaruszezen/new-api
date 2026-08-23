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

import { RatioTag } from '../components/ratio-tag'

describe('next model square ratio tag', () => {
  test('renders the ratio as a rectangular chip that does not shrink', () => {
    render(<RatioTag ratio={0.8} />)

    const tag = screen.getByText('0.8x')
    expect(tag).toHaveClass('inline-flex', 'shrink-0', 'rounded-[4px]')
    expect(tag.className).not.toMatch(/rounded-full|rounded-4xl|rounded-lg/)
  })

  test('keeps body-size digits inside a compact padded chip', () => {
    render(<RatioTag ratio={1} />)

    const tag = screen.getByText('1x')
    expect(tag).toHaveClass('text-sm', 'px-1.5', 'py-0.5', 'leading-none')
    expect(tag).not.toHaveClass('w-full', 'px-2.5', 'py-1')
  })

  test('applies a frosted-glass backdrop when the browser supports it', () => {
    render(<RatioTag ratio={0.07} />)

    const tag = screen.getByText('0.07x')
    expect(tag).toHaveClass('backdrop-blur-md', 'backdrop-saturate-150')
    expect(tag.className).toMatch(/bg-green-(500|400)\/15/)
  })
})
