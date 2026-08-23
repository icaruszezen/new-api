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
import { afterEach, describe, expect, test, vi } from 'vitest'

import { useSystemConfigStore } from '@/stores/system-config-store'

import { SkinnedPricing } from '../pricing'

vi.mock('../classic/pricing', () => ({
  ClassicPricing: () => <div data-testid='classic-pricing' />,
}))

vi.mock('../next/pricing', () => ({
  NextPricing: () => <div data-testid='next-pricing' />,
}))

function renderPricing(uiSkin: unknown) {
  const store = useSystemConfigStore.getState()
  useSystemConfigStore.setState({
    config: { ...store.config, uiSkin: uiSkin as never },
  })

  return render(<SkinnedPricing />)
}

describe('model square skin dispatch', () => {
  afterEach(() => {
    useSystemConfigStore.setState({
      config: { ...useSystemConfigStore.getState().config, uiSkin: 'classic' },
    })
    document.body.removeAttribute('data-ui-skin')
  })

  test('renders the classic model square when the administrator chose classic', () => {
    renderPricing('classic')

    expect(screen.getByTestId('classic-pricing')).toBeInTheDocument()
    expect(screen.queryByTestId('next-pricing')).toBeNull()
    expect(document.body.getAttribute('data-ui-skin')).toBe('classic')
  })

  test('renders the next model square when the administrator chose next', () => {
    renderPricing('next')

    expect(screen.getByTestId('next-pricing')).toBeInTheDocument()
    expect(screen.queryByTestId('classic-pricing')).toBeNull()
    expect(document.body.getAttribute('data-ui-skin')).toBe('next')
  })

  test('renders the classic model square while the status response is unavailable', () => {
    renderPricing(undefined)

    expect(screen.getByTestId('classic-pricing')).toBeInTheDocument()
    expect(screen.queryByTestId('next-pricing')).toBeNull()
  })
})
