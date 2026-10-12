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
import { Cancel01Icon, ImageAdd01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

import { useAddTicketImages } from '../hooks/use-add-ticket-images'
import { ACCEPTED_IMAGE_TYPES, MAX_IMAGES_PER_MESSAGE } from '../lib/limits'

type TicketImagePickerProps = {
  files: File[]
  onChange: (files: File[]) => void
  disabled?: boolean
  className?: string
}

const fileKeys = new WeakMap<File, number>()
let lastFileKey = 0

/** Stable React key for a File object; File has no id of its own. */
function getFileKey(file: File): number {
  let key = fileKeys.get(file)
  if (key === undefined) {
    lastFileKey += 1
    key = lastFileKey
    fileKeys.set(file, key)
  }
  return key
}

/**
 * Controlled image picker for ticket messages. Parents that also accept
 * pasted or dropped images should feed them through `useAddTicketImages`
 * with the same `files`/`onChange` pair.
 */
export function TicketImagePicker(props: TicketImagePickerProps) {
  const { t } = useTranslation()
  const inputRef = useRef<HTMLInputElement>(null)
  const addFiles = useAddTicketImages(props.files, props.onChange)
  const isFull = props.files.length >= MAX_IMAGES_PER_MESSAGE

  return (
    <div
      data-slot='ticket-image-picker'
      className={cn('flex flex-wrap items-center gap-2', props.className)}
    >
      {props.files.map((file, index) => (
        <TicketImagePreview
          key={getFileKey(file)}
          file={file}
          disabled={props.disabled}
          onRemove={() =>
            props.onChange(props.files.filter((_, i) => i !== index))
          }
        />
      ))}
      <input
        ref={inputRef}
        type='file'
        multiple
        accept={ACCEPTED_IMAGE_TYPES.join(',')}
        className='hidden'
        tabIndex={-1}
        aria-hidden='true'
        disabled={props.disabled}
        onChange={(event) => {
          const incoming = [...(event.target.files ?? [])]
          event.target.value = ''
          addFiles(incoming)
        }}
      />
      <Button
        type='button'
        variant='outline'
        size='sm'
        disabled={props.disabled || isFull}
        onClick={() => inputRef.current?.click()}
      >
        <HugeiconsIcon
          icon={ImageAdd01Icon}
          strokeWidth={2}
          aria-hidden='true'
        />
        {t('Add images')}
      </Button>
      <span className='text-muted-foreground text-xs tabular-nums'>
        {t('{{count}}/{{max}} images', {
          count: props.files.length,
          max: MAX_IMAGES_PER_MESSAGE,
        })}
      </span>
    </div>
  )
}

function TicketImagePreview(props: {
  file: File
  disabled?: boolean
  onRemove: () => void
}) {
  const { t } = useTranslation()
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)

  useEffect(() => {
    const url = URL.createObjectURL(props.file)
    setPreviewUrl(url)
    return () => {
      URL.revokeObjectURL(url)
      setPreviewUrl(null)
    }
  }, [props.file])

  return (
    <div className='bg-muted relative size-16 shrink-0 overflow-hidden rounded-lg border'>
      {previewUrl && (
        <img
          src={previewUrl}
          alt={props.file.name}
          draggable={false}
          className='size-full object-cover'
        />
      )}
      <button
        type='button'
        aria-label={t('Remove image')}
        disabled={props.disabled}
        onClick={props.onRemove}
        className='bg-background/90 text-foreground hover:bg-background focus-visible:ring-ring/50 absolute top-1 right-1 flex size-5 items-center justify-center rounded-full border outline-none focus-visible:ring-2 disabled:pointer-events-none disabled:opacity-50'
      >
        <HugeiconsIcon icon={Cancel01Icon} strokeWidth={2} className='size-3' />
      </button>
    </div>
  )
}
