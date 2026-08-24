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
import { useQuery } from '@tanstack/react-query'
import { Check, ChevronsUpDown } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTitle,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Skeleton } from '@/components/ui/skeleton'
import { getApiKey, updateApiKey } from '@/features/keys/api'
import { ERROR_MESSAGES, SUCCESS_MESSAGES } from '@/features/keys/constants'
import type { ApiKey } from '@/features/keys/types'
import { getUserGroups } from '@/lib/api'
import { cn } from '@/lib/utils'
import { RatioTag } from '@/skins/next/pricing/components/ratio-tag'

import { buildApiKeyGroupUpdatePayload } from '../lib/group-update'

type ConsoleKeyGroupCellProps = {
  apiKey: ApiKey
  ratio?: number | string | null
  onSwitched: () => void
  className?: string
}

export function ConsoleKeyGroupCell(props: ConsoleKeyGroupCellProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [pendingGroup, setPendingGroup] = useState<string | null>(null)
  const [searchValue, setSearchValue] = useState('')
  const currentGroup = props.apiKey.group ?? ''
  const groupName = currentGroup.trim()
  const label = groupName === 'auto' ? t('Cross-group') : groupName || '-'

  const groupsQuery = useQuery({
    queryKey: ['user-groups'],
    queryFn: getUserGroups,
    enabled: open,
  })

  const groups = Object.entries(groupsQuery.data?.data ?? {}).map(
    ([value, info]) => ({
      value,
      desc: info.desc || value,
      ratio: info.ratio,
    })
  )

  const filteredGroups = useMemo(() => {
    const search = searchValue.trim().toLowerCase()
    if (!search) return groups
    return groups.filter((group) => {
      const ratioText = String(group.ratio ?? '').toLowerCase()
      return (
        group.value.toLowerCase().includes(search) ||
        group.desc.toLowerCase().includes(search) ||
        ratioText.includes(search)
      )
    })
  }, [groups, searchValue])

  const closePicker = () => {
    setOpen(false)
    setSearchValue('')
  }

  const handleOpenChange = (nextOpen: boolean) => {
    if (pendingGroup && !nextOpen) return
    if (nextOpen) {
      setOpen(true)
      return
    }
    closePicker()
  }

  const handleSelect = async (nextGroup: string) => {
    if (pendingGroup) return
    if (nextGroup === currentGroup) {
      closePicker()
      return
    }

    setPendingGroup(nextGroup)
    try {
      const latest = await getApiKey(props.apiKey.id)
      const source = latest.success && latest.data ? latest.data : props.apiKey
      const result = await updateApiKey(
        buildApiKeyGroupUpdatePayload(source, nextGroup)
      )
      if (result.success) {
        toast.success(t(SUCCESS_MESSAGES.API_KEY_UPDATED))
        props.onSwitched()
        closePicker()
      } else {
        toast.error(result.message || t(ERROR_MESSAGES.UPDATE_FAILED))
      }
    } catch {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setPendingGroup(null)
    }
  }

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger
        render={
          <button
            type='button'
            data-slot='console-key-group-cell'
            role='combobox'
            aria-expanded={open}
            aria-label={t('Switch group')}
            title={label}
            className={cn(
              'border-border bg-muted/40 hover:bg-muted/70 data-popup-open:border-foreground/25 data-popup-open:bg-muted/80 inline-flex h-8 max-w-full min-w-0 cursor-pointer items-center gap-1.5 rounded-lg border px-2 text-start transition-colors',
              props.className
            )}
          />
        }
      >
        <span
          data-slot='console-key-group-name'
          className='min-w-0 truncate text-sm font-medium'
        >
          {label}
        </span>
        {typeof props.ratio === 'number' ? (
          <RatioTag ratio={props.ratio} />
        ) : null}
        <ChevronsUpDown
          aria-hidden='true'
          data-slot='console-key-group-chevrons'
          className='text-muted-foreground size-3.5 shrink-0'
        />
      </PopoverTrigger>
      <PopoverContent
        align='start'
        side='bottom'
        sideOffset={6}
        initialFocus={(openType) => openType === 'keyboard'}
        className='w-80 overflow-hidden rounded-xl p-0 shadow-md'
        onWheel={(event) => event.stopPropagation()}
        onTouchMove={(event) => event.stopPropagation()}
        onPointerDown={(event) => event.stopPropagation()}
      >
        <div data-slot='console-key-group-picker'>
          <PopoverTitle className='sr-only'>{t('Switch group')}</PopoverTitle>
          {groupsQuery.isLoading ? (
            <div className='flex flex-col gap-2 p-2'>
              <Skeleton className='h-8 w-full' />
              <Skeleton className='h-10 w-full' />
              <Skeleton className='h-10 w-full' />
            </div>
          ) : null}
          {!groupsQuery.isLoading && groups.length === 0 ? (
            <p className='text-muted-foreground px-3 py-8 text-center text-sm'>
              {t('No groups available')}
            </p>
          ) : null}
          {!groupsQuery.isLoading && groups.length > 0 ? (
            <div>
              <div className='p-1.5 pb-1'>
                <Input
                  type='search'
                  placeholder={t('Search groups...')}
                  value={searchValue}
                  onChange={(event) => setSearchValue(event.target.value)}
                  autoFocus={false}
                />
              </div>
              <div
                role='listbox'
                aria-label={t('Switch group')}
                className='max-h-64 overflow-y-auto overscroll-contain p-1'
              >
                {filteredGroups.length === 0 ? (
                  <p className='text-muted-foreground px-2 py-6 text-center text-sm'>
                    {t('No group found.')}
                  </p>
                ) : null}
                {filteredGroups.map((group) => {
                  const isCurrent = group.value === currentGroup
                  const optionLabel =
                    group.value === 'auto' ? t('Cross-group') : group.value

                  return (
                    <button
                      key={group.value}
                      type='button'
                      role='option'
                      aria-selected={isCurrent}
                      disabled={pendingGroup !== null}
                      onClick={() => {
                        void handleSelect(group.value)
                      }}
                      className={cn(
                        'hover:bg-muted/60 flex w-full items-center gap-2 rounded-lg px-2 py-2 text-start',
                        isCurrent && 'bg-muted/40',
                        pendingGroup === group.value && 'opacity-70'
                      )}
                    >
                      <span className='min-w-0 flex-1'>
                        <span className='block truncate font-medium'>
                          {optionLabel}
                        </span>
                        {group.desc && group.desc !== group.value ? (
                          <span className='text-muted-foreground block truncate text-xs'>
                            {group.desc}
                          </span>
                        ) : null}
                      </span>
                      {typeof group.ratio === 'number' ? (
                        <RatioTag ratio={group.ratio} />
                      ) : null}
                      {isCurrent ? (
                        <Check className='size-4 shrink-0' aria-hidden='true' />
                      ) : (
                        <span className='size-4 shrink-0' aria-hidden='true' />
                      )}
                    </button>
                  )
                })}
              </div>
            </div>
          ) : null}
        </div>
      </PopoverContent>
    </Popover>
  )
}
