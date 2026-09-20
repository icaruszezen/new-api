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
import { describe, expect, test } from 'vitest'

import { TooltipProvider } from '@/components/ui/tooltip'

import { BeatBar } from '../components/beat-bar'
import {
  BEAT_STATUS_DOWN,
  BEAT_STATUS_SLOW,
  BEAT_STATUS_UP,
  type Beat,
} from '../types'

function beat(ts: number, status: number, ttftMs = 0): Beat {
  return { ts, status, ttft_ms: ttftMs }
}

function renderedSlots() {
  return [...screen.getByRole('img').children] as HTMLElement[]
}

function slotTrigger(slot: HTMLElement) {
  return (
    slot.querySelector<HTMLElement>('[data-slot="tooltip-trigger"]') ?? slot
  )
}

describe('BeatBar', () => {
  test('pads missing history so short series keep the full bar width', () => {
    render(<BeatBar beats={[beat(10, BEAT_STATUS_UP)]} slots={5} />)

    const rendered = renderedSlots()
    expect(rendered).toHaveLength(5)
    // Padding sits on the left so the newest sample stays anchored to "Now".
    expect(rendered.slice(0, 4).every((el) => el.ariaHidden === 'true')).toBe(
      true
    )
  })

  test('renders one slot per beat when history fills the bar', () => {
    render(
      <BeatBar
        beats={[
          beat(10, BEAT_STATUS_UP),
          beat(20, BEAT_STATUS_SLOW),
          beat(30, BEAT_STATUS_DOWN),
        ]}
        slots={3}
      />
    )

    expect(renderedSlots()).toHaveLength(3)
  })

  test('keeps only the newest slots when history exceeds the bar', () => {
    render(
      <BeatBar
        beats={[
          beat(10, BEAT_STATUS_DOWN),
          beat(20, BEAT_STATUS_UP),
          beat(30, BEAT_STATUS_UP),
        ]}
        slots={2}
      />
    )

    expect(renderedSlots()).toHaveLength(2)
  })

  test('colours a mixed window from request counts, not the representative status', () => {
    render(
      <BeatBar
        beats={[
          {
            ts: 10,
            status: BEAT_STATUS_UP,
            ttft_ms: 120,
            request_total: 10,
            request_up: 1,
            request_down: 9,
          },
        ]}
        slots={1}
      />
    )

    expect(slotTrigger(renderedSlots()[0])).toHaveStyle({
      backgroundColor: 'rgb(248, 113, 113)',
    })
  })

  test('colours each slot by window success rate', () => {
    render(
      <BeatBar
        beats={[
          beat(10, BEAT_STATUS_UP),
          beat(20, BEAT_STATUS_SLOW),
          beat(30, BEAT_STATUS_DOWN),
        ]}
        slots={3}
      />
    )

    const rendered = renderedSlots()
    expect(slotTrigger(rendered[0])).toHaveStyle({
      backgroundColor: 'rgb(22, 163, 74)',
    })
    expect(slotTrigger(rendered[1])).toHaveStyle({
      backgroundColor: 'rgb(22, 163, 74)',
    })
    expect(slotTrigger(rendered[2])).toHaveStyle({
      backgroundColor: 'rgb(239, 67, 67)',
    })
  })

  test('encodes representative status as bar height', () => {
    render(
      <BeatBar
        beats={[
          beat(10, BEAT_STATUS_UP),
          beat(20, BEAT_STATUS_SLOW),
          beat(30, BEAT_STATUS_DOWN),
        ]}
        slots={3}
      />
    )

    const rendered = renderedSlots()
    expect(slotTrigger(rendered[0])).toHaveStyle({ height: '100%' })
    expect(slotTrigger(rendered[1])).toHaveStyle({ height: '65%' })
    expect(slotTrigger(rendered[2])).toHaveStyle({ height: '35%' })
  })

  test('lays every slot on an equal flex track so filled and empty cells stay the same width', () => {
    render(<BeatBar beats={[beat(10, BEAT_STATUS_UP)]} slots={5} />)

    expect(screen.getByRole('img')).toHaveClass('flex')
    expect(screen.getByRole('img')).toHaveClass('items-end')
    expect(
      renderedSlots().every((slot) => slot.className.includes('min-w-0'))
    ).toBe(true)
    expect(
      renderedSlots().every((slot) => slot.className.includes('flex-1'))
    ).toBe(true)
  })

  test('renders past and now axis labels', () => {
    render(<BeatBar beats={[]} slots={3} />)

    expect(screen.getByText('Past')).toBeInTheDocument()
    expect(screen.getByText('Now')).toBeInTheDocument()
  })

  test('renders an all-empty bar when there is no history at all', () => {
    render(<BeatBar beats={[]} slots={4} />)

    const rendered = renderedSlots()
    expect(rendered).toHaveLength(4)
    expect(rendered.every((el) => el.ariaHidden === 'true')).toBe(true)
  })

  test('reveals status and first-token latency on hover without a timestamp', async () => {
    const user = userEvent.setup()
    render(
      <TooltipProvider delay={0}>
        <BeatBar
          beats={[beat(1_700_000_000, BEAT_STATUS_UP, 3230)]}
          slots={1}
        />
      </TooltipProvider>
    )

    await user.hover(slotTrigger(renderedSlots()[0]))

    expect(await screen.findByText('Normal')).toBeInTheDocument()
    expect(screen.getByText('First token 3230 ms')).toBeInTheDocument()
    expect(screen.queryByText('11-15 06:13:20')).not.toBeInTheDocument()
  })

  test('keeps hover details off the page-muted colour so they stay readable', async () => {
    const user = userEvent.setup()
    render(
      <TooltipProvider delay={0}>
        <BeatBar
          beats={[beat(1_700_000_000, BEAT_STATUS_UP, 3230)]}
          slots={1}
        />
      </TooltipProvider>
    )

    await user.hover(slotTrigger(renderedSlots()[0]))

    const tooltip = (await screen.findByText('Normal')).closest(
      '[data-slot="tooltip-content"]'
    )

    // The tooltip surface is inverted, so muted page text vanishes in both themes.
    expect(tooltip).not.toBeNull()
    expect(tooltip?.outerHTML).not.toContain('text-muted-foreground')
  })
})
