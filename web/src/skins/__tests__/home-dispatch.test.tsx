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

import { SkinnedHome } from '../home'

vi.mock('../classic/home', () => ({
  ClassicHome: () => <div data-testid='classic-home' />,
}))

vi.mock('../next/home', () => ({
  NextHome: () => <div data-testid='next-home' />,
}))

function renderHome(uiSkin: unknown) {
  const store = useSystemConfigStore.getState()
  useSystemConfigStore.setState({
    config: { ...store.config, uiSkin: uiSkin as never },
  })

  return render(<SkinnedHome />)
}

describe('landing page skin dispatch', () => {
  afterEach(() => {
    useSystemConfigStore.setState({
      config: { ...useSystemConfigStore.getState().config, uiSkin: 'classic' },
    })
    document.body.removeAttribute('data-ui-skin')
  })

  test('renders the classic landing page when the administrator chose classic', () => {
    renderHome('classic')

    expect(screen.getByTestId('classic-home')).toBeInTheDocument()
    expect(screen.queryByTestId('next-home')).toBeNull()
    expect(document.body.getAttribute('data-ui-skin')).toBe('classic')
  })

  test('renders the next landing page when the administrator chose next', () => {
    renderHome('next')

    expect(screen.getByTestId('next-home')).toBeInTheDocument()
    expect(screen.queryByTestId('classic-home')).toBeNull()
    expect(document.body.getAttribute('data-ui-skin')).toBe('next')
  })

  test('renders the classic landing page while the status response is unavailable', () => {
    renderHome(undefined)

    expect(screen.getByTestId('classic-home')).toBeInTheDocument()
    expect(screen.queryByTestId('next-home')).toBeNull()
  })
})
