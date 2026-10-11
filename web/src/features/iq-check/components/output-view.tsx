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

import { parseOutputSegments } from '../lib/output-segments'
import { HtmlPreview } from './html-preview'

type OutputViewProps = {
  content: string
  /** Renders HTML previews at reading size, as in the expanded dialog. */
  expanded?: boolean
}

/**
 * Shows a model answer in its original order: prose stays plain text and every
 * HTML page found in it is rendered in a sandboxed preview box.
 */
export function OutputView(props: OutputViewProps) {
  const { t } = useTranslation()
  const segments = useMemo(
    () => parseOutputSegments(props.content),
    [props.content]
  )

  if (segments.length === 0) {
    return <p className='text-muted-foreground text-sm'>{t('No output')}</p>
  }

  return (
    <div className='space-y-3'>
      {segments.map((segment, index) => {
        // Segments are derived from immutable content, so their position is a
        // stable identity.
        const key = `${segment.type}-${index}`
        if (segment.type === 'html') {
          return (
            <HtmlPreview
              key={key}
              html={segment.html}
              complete={segment.complete}
              expanded={props.expanded}
            />
          )
        }
        return (
          <div
            key={key}
            className='text-sm leading-relaxed break-words whitespace-pre-wrap'
          >
            {segment.text}
          </div>
        )
      })}
    </div>
  )
}
