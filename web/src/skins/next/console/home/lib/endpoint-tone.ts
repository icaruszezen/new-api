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
export type EndpointToneId =
  | 'sky'
  | 'violet'
  | 'emerald'
  | 'amber'
  | 'rose'
  | 'cyan'
  | 'orange'
  | 'indigo'

export type EndpointTone = {
  id: EndpointToneId
  chip: string
  label: string
}

export type EndpointToneSource = {
  url: string
  route?: string
  color?: string
}

export const ENDPOINT_TONES: readonly EndpointTone[] = [
  {
    id: 'sky',
    chip: 'border-sky-200/80 bg-sky-50 dark:border-sky-800/70 dark:bg-sky-950/45',
    label:
      'border-sky-200/80 bg-sky-100 text-sky-800 dark:border-sky-700 dark:bg-sky-900/80 dark:text-sky-100',
  },
  {
    id: 'violet',
    chip: 'border-violet-200/80 bg-violet-50 dark:border-violet-800/70 dark:bg-violet-950/45',
    label:
      'border-violet-200/80 bg-violet-100 text-violet-800 dark:border-violet-700 dark:bg-violet-900/80 dark:text-violet-100',
  },
  {
    id: 'emerald',
    chip: 'border-emerald-200/80 bg-emerald-50 dark:border-emerald-800/70 dark:bg-emerald-950/45',
    label:
      'border-emerald-200/80 bg-emerald-100 text-emerald-800 dark:border-emerald-700 dark:bg-emerald-900/80 dark:text-emerald-100',
  },
  {
    id: 'amber',
    chip: 'border-amber-200/80 bg-amber-50 dark:border-amber-800/70 dark:bg-amber-950/45',
    label:
      'border-amber-200/80 bg-amber-100 text-amber-900 dark:border-amber-700 dark:bg-amber-900/80 dark:text-amber-100',
  },
  {
    id: 'rose',
    chip: 'border-rose-200/80 bg-rose-50 dark:border-rose-800/70 dark:bg-rose-950/45',
    label:
      'border-rose-200/80 bg-rose-100 text-rose-800 dark:border-rose-700 dark:bg-rose-900/80 dark:text-rose-100',
  },
  {
    id: 'cyan',
    chip: 'border-cyan-200/80 bg-cyan-50 dark:border-cyan-800/70 dark:bg-cyan-950/45',
    label:
      'border-cyan-200/80 bg-cyan-100 text-cyan-800 dark:border-cyan-700 dark:bg-cyan-900/80 dark:text-cyan-100',
  },
  {
    id: 'orange',
    chip: 'border-orange-200/80 bg-orange-50 dark:border-orange-800/70 dark:bg-orange-950/45',
    label:
      'border-orange-200/80 bg-orange-100 text-orange-800 dark:border-orange-700 dark:bg-orange-900/80 dark:text-orange-100',
  },
  {
    id: 'indigo',
    chip: 'border-indigo-200/80 bg-indigo-50 dark:border-indigo-800/70 dark:bg-indigo-950/45',
    label:
      'border-indigo-200/80 bg-indigo-100 text-indigo-800 dark:border-indigo-700 dark:bg-indigo-900/80 dark:text-indigo-100',
  },
]

const TONE_BY_ID = new Map(
  ENDPOINT_TONES.map((tone) => [tone.id, tone] as const)
)

const PREFERRED_TONE_BY_COLOR: Record<string, EndpointToneId> = {
  amber: 'amber',
  blue: 'sky',
  cyan: 'cyan',
  green: 'emerald',
  indigo: 'indigo',
  'light-blue': 'sky',
  'light-green': 'emerald',
  lime: 'emerald',
  orange: 'orange',
  pink: 'rose',
  purple: 'violet',
  red: 'rose',
  teal: 'cyan',
  violet: 'violet',
  yellow: 'amber',
}

function hashString(value: string): number {
  let hash = 0
  for (let i = 0; i < value.length; i++) {
    hash = (hash * 31 + value.charCodeAt(i)) >>> 0
  }
  return hash
}

function nextUnusedTone(
  startIndex: number,
  used: Set<EndpointToneId>
): EndpointTone {
  for (let offset = 0; offset < ENDPOINT_TONES.length; offset++) {
    const tone = ENDPOINT_TONES[(startIndex + offset) % ENDPOINT_TONES.length]
    if (!used.has(tone.id)) return tone
  }
  return ENDPOINT_TONES[startIndex % ENDPOINT_TONES.length]
}

export function assignEndpointTones(
  items: EndpointToneSource[]
): EndpointTone[] {
  const used = new Set<EndpointToneId>()

  return items.map((item) => {
    const preferredId = item.color
      ? PREFERRED_TONE_BY_COLOR[item.color]
      : undefined
    const preferredTone = preferredId ? TONE_BY_ID.get(preferredId) : undefined
    if (preferredTone && !used.has(preferredTone.id)) {
      used.add(preferredTone.id)
      return preferredTone
    }

    const seed = item.url || item.route || ''
    const tone = nextUnusedTone(hashString(seed) % ENDPOINT_TONES.length, used)
    used.add(tone.id)
    return tone
  })
}
