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

import { SkinnedChannelMonitoring } from '../channel-monitoring'

vi.mock('../classic/channel-monitoring', () => ({
  ClassicChannelMonitoring: () => (
    <div data-testid='classic-channel-monitoring' />
  ),
}))

vi.mock('../next/channel-monitoring', () => ({
  NextChannelMonitoring: () => <div data-testid='next-channel-monitoring' />,
}))

function renderChannelMonitoring(uiSkin: unknown) {
  const store = useSystemConfigStore.getState()
  useSystemConfigStore.setState({
    config: { ...store.config, uiSkin: uiSkin as never },
  })

  return render(<SkinnedChannelMonitoring />)
}

describe('channel monitoring skin dispatch', () => {
  afterEach(() => {
    useSystemConfigStore.setState({
      config: { ...useSystemConfigStore.getState().config, uiSkin: 'classic' },
    })
    document.body.removeAttribute('data-ui-skin')
  })

  test('renders the classic placeholder when the administrator chose classic', () => {
    renderChannelMonitoring('classic')

    expect(screen.getByTestId('classic-channel-monitoring')).toBeInTheDocument()
    expect(screen.queryByTestId('next-channel-monitoring')).toBeNull()
    expect(document.body.getAttribute('data-ui-skin')).toBe('classic')
  })

  test('renders the next placeholder when the administrator chose next', () => {
    renderChannelMonitoring('next')

    expect(screen.getByTestId('next-channel-monitoring')).toBeInTheDocument()
    expect(screen.queryByTestId('classic-channel-monitoring')).toBeNull()
    expect(document.body.getAttribute('data-ui-skin')).toBe('next')
  })

  test('renders the classic placeholder while the status response is unavailable', () => {
    renderChannelMonitoring(undefined)

    expect(screen.getByTestId('classic-channel-monitoring')).toBeInTheDocument()
    expect(screen.queryByTestId('next-channel-monitoring')).toBeNull()
  })
})
