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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

/** Prompts longer than this are clamped until the visitor asks for more. */
const CLAMPED_LINES = 4
const CLAMPED_CHARACTERS = 280

type PromptBlockProps = {
  prompt: string
  /** The shown run was sent a prompt other than the one configured now. */
  outdated: boolean
}

export function PromptBlock(props: PromptBlockProps) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const collapsible =
    props.prompt.split('\n').length > CLAMPED_LINES ||
    props.prompt.length > CLAMPED_CHARACTERS

  return (
    <section className='space-y-1.5'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <h3 className='text-muted-foreground text-xs font-medium'>
          {t('Prompt')}
        </h3>
        {props.outdated && (
          <span className='text-xs text-amber-700 dark:text-amber-300'>
            {t('This run used an earlier version of the prompt.')}
          </span>
        )}
      </div>
      <p
        className={cn(
          'bg-muted/40 rounded-lg px-3 py-2 text-sm leading-relaxed break-words whitespace-pre-wrap',
          collapsible && !expanded && 'line-clamp-4'
        )}
      >
        {props.prompt}
      </p>
      {collapsible && (
        <Button
          type='button'
          variant='link'
          size='xs'
          className='h-auto px-0'
          aria-expanded={expanded}
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? t('Show less') : t('Show more')}
        </Button>
      )}
    </section>
  )
}
