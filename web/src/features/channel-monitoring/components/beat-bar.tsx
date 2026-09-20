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
along with this program. If you did not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useTranslation } from 'react-i18next'

import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import { beatBarColor, beatHeightPct } from '../lib/monitor-colors'
import {
  BEAT_STATUS_DOWN,
  BEAT_STATUS_SLOW,
  BEAT_STATUS_UP,
  type Beat,
} from '../types'

type BeatBarProps = {
  beats: Beat[]
  /** Number of slots to render; older history is trimmed, gaps are padded. */
  slots: number
  className?: string
}

/**
 * Availability trend strip. The newest sample sits on the right so the bar
 * reads left-to-right as past-to-now; missing history is padded to keep every
 * card the same width. Colour follows the window success rate; height follows
 * the representative status.
 */
export function BeatBar(props: BeatBarProps) {
  const { t } = useTranslation()

  const visible = props.beats.slice(-props.slots)
  const gapCount = Math.max(0, props.slots - visible.length)
  const slots: { id: string; beat?: Beat }[] = [
    ...Array.from({ length: gapCount }, (_, offset) => ({
      id: `gap-${gapCount - offset}`,
    })),
    ...visible.map((beat) => ({ id: `beat-${beat.ts}`, beat })),
  ]

  const statusLabel = (status: number) => {
    if (status === BEAT_STATUS_UP) return t('Normal')
    if (status === BEAT_STATUS_SLOW) return t('Slow')
    if (status === BEAT_STATUS_DOWN) return t('Failed')
    return t('No data')
  }

  return (
    <div className={cn('space-y-1.5', props.className)}>
      <div
        className='flex h-8 items-end gap-[2px]'
        role='img'
        aria-label={t('Availability trend')}
      >
        {slots.map((slot) => {
          const heightPct = beatHeightPct(slot.beat?.status)
          const color = beatBarColor(slot.beat)
          if (!slot.beat) {
            return (
              <div
                key={slot.id}
                className='min-w-0 flex-1 rounded-[2px]'
                style={{ height: `${heightPct}%`, backgroundColor: color }}
                aria-hidden='true'
              />
            )
          }
          const beat = slot.beat
          return (
            <div key={slot.id} className='flex h-full min-w-0 flex-1 items-end'>
              <Tooltip>
                <TooltipTrigger
                  render={
                    <div
                      className='w-full rounded-[2px] transition-opacity hover:opacity-70'
                      style={{
                        height: `${heightPct}%`,
                        backgroundColor: color,
                      }}
                    />
                  }
                />
                {/*
                  The tooltip surface is inverted (bg-foreground), so page-level
                  muted text would be unreadable in both themes; secondary lines
                  dim the tooltip's own foreground instead.
                */}
                <TooltipContent
                  side='top'
                  className='flex flex-col items-start gap-0.5 text-xs'
                >
                  <span className='font-medium'>{statusLabel(beat.status)}</span>
                  {beat.ttft_ms > 0 && (
                    <span className='text-background/70 tabular-nums'>
                      {t('First token')} {beat.ttft_ms} ms
                    </span>
                  )}
                </TooltipContent>
              </Tooltip>
            </div>
          )
        })}
      </div>
      <div className='text-muted-foreground flex items-center justify-between text-[10px] font-medium tracking-[0.14em] uppercase'>
        <span>{t('Past')}</span>
        <span>{t('Now')}</span>
      </div>
    </div>
  )
}
