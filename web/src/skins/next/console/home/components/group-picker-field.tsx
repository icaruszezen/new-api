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
import { ChevronsUpDown } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { RatioTag } from '@/skins/next/pricing/components/ratio-tag'

import type { GroupPickerOption } from '../lib/group-picker'
import { GroupPickerDialog } from './group-picker-dialog'

export type GroupPickerFieldProps = {
  options: GroupPickerOption[]
  value?: string
  onValueChange: (value: string) => void
  placeholder?: string
  disabled?: boolean
}

export function GroupPickerField(props: GroupPickerFieldProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const selected = props.options.find((option) => option.value === props.value)
  const isAuto = selected?.value === 'auto'
  const placeholder = props.placeholder || t('Select a group')
  const selectedLabel = isAuto
    ? t('Cross-group')
    : selected?.label || placeholder

  return (
    <div className='w-full'>
      <Button
        type='button'
        variant='outline'
        aria-haspopup='dialog'
        aria-expanded={open}
        aria-label={placeholder}
        disabled={props.disabled}
        data-slot='next-group-picker-trigger'
        onClick={() => setOpen(true)}
        className='border-input bg-muted/40 hover:bg-muted/55 hover:text-foreground h-auto min-h-14 w-full justify-between gap-2 rounded-lg px-3 py-2 text-start shadow-none sm:min-h-20 sm:gap-3 sm:px-4 sm:py-3'
      >
        <span className='flex min-w-0 flex-1 items-center justify-between gap-2 sm:gap-3'>
          <span className='min-w-0'>
            <span className='block truncate font-medium'>{selectedLabel}</span>
            {selected?.desc ? (
              <span className='text-muted-foreground block truncate text-[11px] sm:text-xs'>
                {selected.desc}
              </span>
            ) : null}
          </span>
          {typeof selected?.ratio === 'number' ? (
            <span className='hidden sm:block'>
              <RatioTag ratio={selected.ratio} />
            </span>
          ) : null}
        </span>
        <ChevronsUpDown
          aria-hidden='true'
          className='size-4 shrink-0 opacity-50'
        />
      </Button>
      <GroupPickerDialog
        open={open}
        onOpenChange={setOpen}
        options={props.options}
        value={props.value}
        title={placeholder}
        onSelect={(group) => {
          props.onValueChange(group)
          setOpen(false)
        }}
      />
    </div>
  )
}
