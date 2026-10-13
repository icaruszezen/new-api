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
import { SentIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { type FormEvent, type ReactNode, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

import { useAddTicketImages } from '../hooks/use-add-ticket-images'
import { getTransferFiles } from '../lib/attachments'
import { CONTENT_MAX_LENGTH } from '../lib/limits'
import type { TicketReplyValues } from '../types'
import { TicketImagePicker } from './ticket-image-picker'

type TicketReplyComposerProps = {
  /** Resolve `true` to clear the draft; `false` or a rejection keeps it. */
  onSubmit: (values: TicketReplyValues) => Promise<boolean>
  disabled?: boolean
  submitLabel?: string
  /** Rendered next to the submit button (e.g. a status select). */
  extraActions?: ReactNode
  placeholder?: string
  className?: string
}

/**
 * Reply box shared by the user and support views: text plus up to four
 * images from the picker, the clipboard, or drag and drop. A reply may be
 * text only, images only, or both.
 */
export function TicketReplyComposer(props: TicketReplyComposerProps) {
  const { t } = useTranslation()
  const counterId = useId()
  const [content, setContent] = useState('')
  const [files, setFiles] = useState<File[]>([])
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isDragging, setIsDragging] = useState(false)
  const addFiles = useAddTicketImages(files, setFiles)

  const isLocked = Boolean(props.disabled) || isSubmitting
  const hasDraft = content.trim().length > 0 || files.length > 0
  const canSubmit = !isLocked && hasDraft

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!canSubmit) return

    setIsSubmitting(true)
    try {
      const succeeded = await props.onSubmit({ content: content.trim(), files })
      if (succeeded) {
        setContent('')
        setFiles([])
      }
    } catch {
      // The caller reports the failure; the draft stays for a retry.
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form
      data-slot='ticket-reply-composer'
      onSubmit={handleSubmit}
      onDragOver={(event) => {
        if (!event.dataTransfer.types.includes('Files')) return
        event.preventDefault()
        if (!isLocked) setIsDragging(true)
      }}
      onDragLeave={(event) => {
        const next = event.relatedTarget
        if (next instanceof Node && event.currentTarget.contains(next)) return
        setIsDragging(false)
      }}
      onDrop={(event) => {
        if (!event.dataTransfer.types.includes('Files')) return
        // Always swallow file drops so the browser never navigates to them.
        event.preventDefault()
        setIsDragging(false)
        if (!isLocked) addFiles(getTransferFiles(event.dataTransfer))
      }}
      className={cn(
        'bg-card rounded-xl border transition-shadow',
        isDragging && 'ring-ring/50 ring-3',
        props.className
      )}
    >
      <Textarea
        aria-label={t('Reply')}
        aria-describedby={counterId}
        value={content}
        maxLength={CONTENT_MAX_LENGTH}
        placeholder={props.placeholder ?? t('Write a reply...')}
        disabled={props.disabled}
        readOnly={isSubmitting}
        onChange={(event) => setContent(event.target.value)}
        onPaste={(event) => {
          if (isLocked) return
          addFiles(getTransferFiles(event.clipboardData))
        }}
        onKeyDown={(event) => {
          if (event.key !== 'Enter' || !(event.metaKey || event.ctrlKey)) {
            return
          }
          event.preventDefault()
          event.currentTarget.form?.requestSubmit()
        }}
        className='min-h-28 resize-none rounded-b-none border-0 bg-transparent px-4 py-3 shadow-none focus-visible:ring-0 dark:bg-transparent'
      />
      <TicketImagePicker
        files={files}
        onChange={setFiles}
        disabled={isLocked}
        className='px-4 pb-3'
      />
      <div className='flex flex-wrap items-center gap-2 border-t px-4 py-3'>
        <span
          id={counterId}
          className='text-muted-foreground text-xs tabular-nums'
        >
          {content.length}/{CONTENT_MAX_LENGTH}
        </span>
        <div className='ml-auto flex flex-wrap items-center gap-2'>
          {props.extraActions}
          <Button type='submit' disabled={!canSubmit}>
            {isSubmitting ? (
              <Spinner aria-hidden='true' />
            ) : (
              <HugeiconsIcon
                icon={SentIcon}
                strokeWidth={2}
                aria-hidden='true'
              />
            )}
            {props.submitLabel ?? t('Send reply')}
          </Button>
        </div>
      </div>
    </form>
  )
}
