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
import { Brain, ChevronDown, CircleAlert, RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

import type { ResultDetailState } from '../hooks/use-result-details'
import type { IqCheckResult } from '../types'
import { OutputView } from './output-view'

function ReasoningSection(props: { reasoning: string; truncated: boolean }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger
        render={
          <button
            type='button'
            className='text-muted-foreground hover:text-foreground flex items-center gap-1.5 text-xs font-medium transition-colors'
          />
        }
      >
        <Brain className='size-3.5' aria-hidden='true' />
        {t('Thinking')}
        <ChevronDown
          className={cn('size-3.5 transition-transform', open && 'rotate-180')}
          aria-hidden='true'
        />
      </CollapsibleTrigger>
      <CollapsibleContent className='mt-2'>
        <div className='bg-muted/40 text-muted-foreground rounded-lg px-3 py-2 text-xs leading-relaxed break-words whitespace-pre-wrap'>
          {props.reasoning}
        </div>
        {props.truncated && (
          <p className='text-muted-foreground mt-1 text-xs'>
            {t('The thinking was too long and has been truncated.')}
          </p>
        )}
      </CollapsibleContent>
    </Collapsible>
  )
}

type ResultBodyProps = {
  result?: IqCheckResult
  state: ResultDetailState
  onRetry: () => void
  /** Renders HTML previews at reading size, as in the expanded dialog. */
  expanded?: boolean
}

/** Output of one run: error, thinking and the answer itself. */
export function ResultBody(props: ResultBodyProps) {
  const { t } = useTranslation()

  if (props.state === 'loading') {
    return (
      <div className='space-y-2' aria-busy='true'>
        <Skeleton className='h-4 w-3/4' />
        <Skeleton className='h-4 w-full' />
        <Skeleton className='h-4 w-2/3' />
      </div>
    )
  }

  if (props.state === 'error') {
    return (
      <div className='flex flex-wrap items-center gap-2'>
        <p className='text-muted-foreground text-sm'>
          {t('Failed to load the output.')}
        </p>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={props.onRetry}
        >
          <RefreshCw aria-hidden='true' />
          {t('Retry')}
        </Button>
      </div>
    )
  }

  const result = props.result
  if (props.state === 'unavailable' || !result) {
    return (
      <p className='text-muted-foreground text-sm'>
        {t('This result is no longer available.')}
      </p>
    )
  }

  const failed = result.status === 'failed'
  // A failed run without any partial answer is fully explained by its error.
  const showOutput = !failed || result.content !== ''

  return (
    <div className='space-y-3'>
      {failed && (
        <Alert variant='destructive'>
          <CircleAlert aria-hidden='true' />
          <AlertTitle>{t('Check failed')}</AlertTitle>
          {result.error_message && (
            <AlertDescription className='break-words whitespace-pre-wrap'>
              {result.error_message}
            </AlertDescription>
          )}
        </Alert>
      )}
      {result.reasoning && (
        <ReasoningSection
          reasoning={result.reasoning}
          truncated={result.reasoning_truncated}
        />
      )}
      {showOutput && (
        <OutputView content={result.content} expanded={props.expanded} />
      )}
      {result.content_truncated && (
        <p className='text-muted-foreground text-xs'>
          {t('The output was too long and has been truncated.')}
        </p>
      )}
    </div>
  )
}
