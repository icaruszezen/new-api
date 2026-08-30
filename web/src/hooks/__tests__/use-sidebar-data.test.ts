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
import { afterEach, describe, expect, test } from 'vitest'

import { useSystemConfigStore } from '@/stores/system-config-store'

import { useSidebarData } from '../use-sidebar-data'

function setUiSkin(uiSkin: unknown) {
  const store = useSystemConfigStore.getState()
  useSystemConfigStore.setState({
    config: { ...store.config, uiSkin: uiSkin as never },
  })
}

function adminItemUrls() {
  const { result } = renderHook(() => useSidebarData())
  const admin = result.current.navGroups.find((group) => group.id === 'admin')
  return (admin?.items ?? [])
    .map((item) => ('url' in item ? item.url : undefined))
    .filter((url): url is string => typeof url === 'string')
}

describe('useSidebarData channel monitoring settings', () => {
  afterEach(() => {
    setUiSkin('classic')
  })

  test('adds Channel Monitoring Settings after Channels when the site skin is next', () => {
    setUiSkin('next')

    const urls = adminItemUrls()

    expect(urls.indexOf('/channel-monitoring-settings')).toBe(
      urls.indexOf('/channels') + 1
    )
  })

  test('omits Channel Monitoring Settings when the site skin is classic', () => {
    setUiSkin('classic')

    const urls = adminItemUrls()

    expect(urls).not.toContain('/channel-monitoring-settings')
  })
})
