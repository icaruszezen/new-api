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
import { cn } from '@/lib/utils'

export type RatioTagProps = {
  ratio: number
  className?: string
}

export function RatioTag(props: RatioTagProps) {
  return (
    <span
      className={cn(
        'inline-flex shrink-0 items-center rounded-[4px] border border-green-500/25 bg-green-500/15 px-1.5 py-0.5 font-mono text-sm leading-none tabular-nums text-green-700 shadow-sm backdrop-blur-md backdrop-saturate-150 dark:border-green-400/25 dark:bg-green-400/15 dark:text-green-400',
        props.className
      )}
    >
      {props.ratio}x
    </span>
  )
}
