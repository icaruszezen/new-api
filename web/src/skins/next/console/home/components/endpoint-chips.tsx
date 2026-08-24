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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import type { ApiInfoItem } from '@/features/dashboard/types'
import { cn } from '@/lib/utils'

import { assignEndpointTones } from '../lib/endpoint-tone'

type ConsoleEndpointChipsProps = {
  items: ApiInfoItem[]
}

export function ConsoleEndpointChips(props: ConsoleEndpointChipsProps) {
  const { t } = useTranslation()
  const tones = useMemo(() => assignEndpointTones(props.items), [props.items])

  if (props.items.length === 0) return null

  return (
    <div
      data-slot='console-endpoint-chips'
      className='flex min-w-0 flex-1 flex-wrap items-center gap-2'
    >
      {props.items.map((item, index) => {
        const tone = tones[index]
        if (!tone) return null
        return (
          <div
            key={item.url}
            data-slot='console-endpoint-chip'
            data-tone={tone.id}
            className={cn(
              'flex min-w-0 items-center gap-2 rounded-lg border px-2.5 py-1.5',
              tone.chip
            )}
          >
            <span
              className={cn(
                'shrink-0 rounded-md border px-1.5 py-0.5 text-[11px] font-medium whitespace-nowrap',
                tone.label
              )}
            >
              {item.description || item.route || t('Default')}
            </span>
            <code className='min-w-0 truncate font-mono text-xs'>
              {item.url}
            </code>
            <CopyButton
              value={item.url}
              size='icon'
              className='text-muted-foreground hover:text-foreground size-7 shrink-0'
              aria-label={t('Copy')}
            />
          </div>
        )
      })}
    </div>
  )
}
