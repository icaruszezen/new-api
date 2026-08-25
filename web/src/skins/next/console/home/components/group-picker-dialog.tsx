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
import { Check } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'
import { RatioTag } from '@/skins/next/pricing/components/ratio-tag'

import { useGroupPickerCatalog } from '../hooks/use-group-picker-catalog'
import {
  GROUP_PICKER_AUTO_SECTION_ID,
  GROUP_PICKER_FLAT_SECTION_ID,
  GROUP_PICKER_OTHER_SECTION_ID,
  filterGroupPickerSections,
  groupUserGroupsByVendor,
  sortGroupsByRatio,
  type GroupPickerOption,
  type GroupPickerSection,
} from '../lib/group-picker'

export type GroupPickerDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  options: GroupPickerOption[]
  value?: string
  onSelect: (value: string) => void
  title: string
  description?: string
  isOptionsLoading?: boolean
  pendingValue?: string | null
}

export function GroupPickerDialog(props: GroupPickerDialogProps) {
  const { t } = useTranslation()
  const [searchValue, setSearchValue] = useState('')
  const catalog = useGroupPickerCatalog(props.open)
  const isLoading = Boolean(props.isOptionsLoading) || catalog.isLoading

  const sections = useMemo(() => {
    if (catalog.isError) {
      return [
        {
          id: GROUP_PICKER_FLAT_SECTION_ID,
          title: '',
          groups: sortGroupsByRatio(props.options),
        },
      ]
    }
    return groupUserGroupsByVendor(props.options, catalog.models)
  }, [catalog.isError, catalog.models, props.options])

  const visibleSections = useMemo(
    () => filterGroupPickerSections(sections, searchValue),
    [searchValue, sections]
  )

  const handleOpenChange = (nextOpen: boolean) => {
    if (props.pendingValue && !nextOpen) return
    if (!nextOpen) setSearchValue('')
    props.onOpenChange(nextOpen)
  }

  return (
    <Dialog open={props.open} onOpenChange={handleOpenChange}>
      <DialogContent
        overlayClassName='z-[60]'
        className='z-[60] flex max-h-[min(85vh,42rem)] w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-2xl'
        onWheel={(event) => event.stopPropagation()}
        onTouchMove={(event) => event.stopPropagation()}
        onPointerDown={(event) => event.stopPropagation()}
      >
        <div
          data-slot='next-group-picker-dialog'
          className='flex min-h-0 flex-1 flex-col'
        >
          <DialogHeader className='px-5 pt-5 pb-3'>
            <p className='text-primary text-[11px] font-medium tracking-[0.16em] uppercase'>
              {t('Group')}
            </p>
            <DialogTitle className='text-lg tracking-tight'>
              {props.title}
            </DialogTitle>
            <DialogDescription
              className={props.description ? undefined : 'sr-only'}
            >
              {props.description || props.title}
            </DialogDescription>
          </DialogHeader>

          <div className='flex min-h-0 flex-1 flex-col px-5 pb-4'>
            {isLoading ? (
              <div className='flex flex-col gap-2'>
                <Skeleton className='h-9 w-full' />
                <Skeleton className='h-16 w-full' />
                <Skeleton className='h-16 w-full' />
              </div>
            ) : null}

            {!isLoading && props.options.length === 0 ? (
              <p className='text-muted-foreground px-3 py-12 text-center text-sm'>
                {t('No groups available')}
              </p>
            ) : null}

            {!isLoading && props.options.length > 0 ? (
              <GroupPickerBody
                pendingValue={props.pendingValue}
                searchValue={searchValue}
                sections={visibleSections}
                selectedValue={props.value}
                onSearchChange={setSearchValue}
                onSelect={props.onSelect}
              />
            ) : null}
          </div>

          <DialogFooter className='mx-0 mb-0'>
            <Button
              type='button'
              variant='outline'
              disabled={Boolean(props.pendingValue)}
              onClick={() => handleOpenChange(false)}
            >
              {t('Cancel')}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function GroupPickerBody(props: {
  pendingValue?: string | null
  searchValue: string
  sections: GroupPickerSection[]
  selectedValue?: string
  onSearchChange: (value: string) => void
  onSelect: (value: string) => void
}) {
  const { t } = useTranslation()

  return (
    <div className='flex min-h-0 flex-1 flex-col gap-3'>
      <Input
        type='search'
        placeholder={t('Search groups...')}
        value={props.searchValue}
        autoFocus={false}
        onChange={(event) => props.onSearchChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Enter') event.preventDefault()
        }}
      />
      <div
        role='listbox'
        aria-label={t('Select a group')}
        className='min-h-0 flex-1 space-y-5 overflow-y-auto overscroll-contain pr-1'
      >
        {props.sections.length === 0 ? (
          <p className='text-muted-foreground px-2 py-10 text-center text-sm'>
            {t('No group found.')}
          </p>
        ) : null}
        {props.sections.map((section) => (
          <GroupPickerSectionList
            key={section.id}
            pendingValue={props.pendingValue}
            section={section}
            selectedValue={props.selectedValue}
            onSelect={props.onSelect}
          />
        ))}
      </div>
    </div>
  )
}

function GroupPickerSectionList(props: {
  pendingValue?: string | null
  section: GroupPickerSection
  selectedValue?: string
  onSelect: (value: string) => void
}) {
  const { t } = useTranslation()
  const isFlat = props.section.id === GROUP_PICKER_FLAT_SECTION_ID
  let heading = props.section.title
  if (props.section.id === GROUP_PICKER_AUTO_SECTION_ID) {
    heading = t('Cross-group')
  } else if (props.section.id === GROUP_PICKER_OTHER_SECTION_ID) {
    heading = t('Other')
  }

  return (
    <section
      data-slot='group-picker-vendor'
      data-vendor={props.section.id}
      className='space-y-2'
    >
      {isFlat || !heading ? null : (
        <h3 className='flex items-center gap-2 text-sm font-medium'>
          {props.section.icon ? (
            <span aria-hidden='true' className='flex items-center'>
              {getLobeIcon(props.section.icon, 20)}
            </span>
          ) : null}
          <span>{heading}</span>
        </h3>
      )}
      <div className='border-border overflow-hidden rounded-xl border'>
        {props.section.groups.map((group) => (
          <GroupPickerOptionRow
            key={`${props.section.id}-${group.value}`}
            group={group}
            isPending={props.pendingValue === group.value}
            isSelected={group.value === props.selectedValue}
            disabled={props.pendingValue != null}
            onSelect={props.onSelect}
          />
        ))}
      </div>
    </section>
  )
}

function GroupPickerOptionRow(props: {
  disabled: boolean
  group: GroupPickerOption
  isPending: boolean
  isSelected: boolean
  onSelect: (value: string) => void
}) {
  const { t } = useTranslation()
  const optionLabel =
    props.group.value === 'auto' ? t('Cross-group') : props.group.label

  return (
    <button
      type='button'
      role='option'
      aria-selected={props.isSelected}
      disabled={props.disabled}
      data-slot='group-picker-option'
      data-group={props.group.value}
      onClick={() => props.onSelect(props.group.value)}
      className={cn(
        'hover:bg-muted/60 flex w-full items-center gap-2 border-border border-b px-3 py-2.5 text-start last:border-b-0',
        props.isSelected && 'bg-muted/40',
        props.isPending && 'opacity-70'
      )}
    >
      <span className='min-w-0 flex-1'>
        <span className='block truncate font-medium'>{optionLabel}</span>
        {props.group.desc && props.group.desc !== props.group.value ? (
          <span className='text-muted-foreground block truncate text-xs'>
            {props.group.desc}
          </span>
        ) : null}
      </span>
      {typeof props.group.ratio === 'number' ? (
        <RatioTag ratio={props.group.ratio} />
      ) : null}
      {props.isSelected ? (
        <Check className='size-4 shrink-0' aria-hidden='true' />
      ) : (
        <span className='size-4 shrink-0' aria-hidden='true' />
      )}
    </button>
  )
}
