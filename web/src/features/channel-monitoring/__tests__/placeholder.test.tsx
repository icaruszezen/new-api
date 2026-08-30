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

import { ChannelMonitoring } from '../index'
import { ChannelMonitoringSettings } from '../settings'

describe('channel monitoring placeholders', () => {
  test('renders the public page title and empty-state copy', () => {
    render(<ChannelMonitoring />)

    expect(
      screen.getByRole('heading', { name: 'Channel Monitoring' })
    ).toBeInTheDocument()
    expect(
      screen.getByText('Channel monitoring is under development.')
    ).toBeInTheDocument()
  })

  test('renders the settings page title and empty-state copy', () => {
    render(<ChannelMonitoringSettings />)

    expect(
      screen.getByRole('heading', { name: 'Channel Monitoring Settings' })
    ).toBeInTheDocument()
    expect(
      screen.getByText('Channel monitoring settings will be available here.')
    ).toBeInTheDocument()
  })
})
