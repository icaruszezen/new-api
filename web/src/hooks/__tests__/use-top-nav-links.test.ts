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
import { renderHook } from '@testing-library/react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { useSystemConfigStore } from '@/stores/system-config-store'

import { useTopNavLinks } from '../use-top-nav-links'

let mockStatus: Record<string, unknown> | null = null

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: mockStatus, loading: false, error: null }),
}))

function setUiSkin(uiSkin: unknown) {
  const store = useSystemConfigStore.getState()
  useSystemConfigStore.setState({
    config: { ...store.config, uiSkin: uiSkin as never },
  })
}

describe('useTopNavLinks channel monitoring', () => {
  afterEach(() => {
    setUiSkin('classic')
    mockStatus = null
  })

  test('adds Channel Monitoring between Rankings and Docs when next skin has it enabled', () => {
    setUiSkin('next')
    mockStatus = {
      channel_monitoring_enabled: true,
      HeaderNavModules: JSON.stringify({
        rankings: { enabled: true, requireAuth: false },
      }),
    }

    const { result } = renderHook(() => useTopNavLinks())
    const hrefs = result.current.map((link) => link.href)
    const titles = result.current.map((link) => link.title)

    expect(titles).toContain('Channel Monitoring')
    expect(hrefs.indexOf('/channel-monitoring')).toBe(
      hrefs.indexOf('/rankings') + 1
    )
    expect(hrefs.indexOf('/docs')).toBe(
      hrefs.indexOf('/channel-monitoring') + 1
    )
  })

  test('omits Channel Monitoring when the site skin is classic', () => {
    setUiSkin('classic')
    mockStatus = { channel_monitoring_enabled: true }

    const { result } = renderHook(() => useTopNavLinks())
    const hrefs = result.current.map((link) => link.href)

    expect(hrefs).not.toContain('/channel-monitoring')
  })

  test('omits Channel Monitoring on next skin while the status page stays disabled', () => {
    setUiSkin('next')
    mockStatus = { channel_monitoring_enabled: false }

    const { result } = renderHook(() => useTopNavLinks())
    const hrefs = result.current.map((link) => link.href)

    expect(hrefs).not.toContain('/channel-monitoring')
  })
})
