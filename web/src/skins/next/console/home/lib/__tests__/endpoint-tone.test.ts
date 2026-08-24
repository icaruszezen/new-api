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
import { describe, expect, test } from 'vitest'

import { assignEndpointTones, ENDPOINT_TONES } from '../endpoint-tone'

describe('assignEndpointTones', () => {
  test('returns no tones when there are no endpoints', () => {
    expect(assignEndpointTones([])).toEqual([])
  })

  test('keeps the same tone for the same single endpoint', () => {
    const first = assignEndpointTones([{ url: 'https://api.example.com' }])
    const second = assignEndpointTones([{ url: 'https://api.example.com' }])

    expect(first).toHaveLength(1)
    expect(first[0]).toBe(second[0])
    expect(ENDPOINT_TONES).toContain(first[0])
  })

  test('gives different endpoints distinct tones', () => {
    const tones = assignEndpointTones([
      { url: 'https://api.example.com/v1' },
      { url: 'https://relay.example.com/v1' },
      { url: 'https://openai.example.com' },
    ])

    const ids = tones.map((tone) => tone.id)
    expect(new Set(ids).size).toBe(3)
  })

  test('uses configured colors when they are still unused', () => {
    const tones = assignEndpointTones([
      { url: 'https://a.example.com', color: 'purple' },
      { url: 'https://b.example.com', color: 'green' },
    ])

    expect(tones[0]?.id).toBe('violet')
    expect(tones[1]?.id).toBe('emerald')
  })

  test('does not reuse a preferred color already taken by another endpoint', () => {
    const tones = assignEndpointTones([
      { url: 'https://a.example.com', color: 'blue' },
      { url: 'https://b.example.com', color: 'blue' },
    ])

    expect(tones[0]?.id).toBe('sky')
    expect(tones[1]?.id).not.toBe('sky')
    expect(tones[0]?.id).not.toBe(tones[1]?.id)
  })

  test('keeps the first palette of tones unique when the list fills the palette', () => {
    const items = ENDPOINT_TONES.map((_, index) => ({
      url: `https://endpoint-${index}.example.com`,
    }))

    const ids = assignEndpointTones(items).map((tone) => tone.id)
    expect(new Set(ids).size).toBe(ENDPOINT_TONES.length)
  })
})
