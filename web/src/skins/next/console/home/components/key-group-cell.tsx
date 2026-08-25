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
import { ChevronsUpDown } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { getApiKey, updateApiKey } from '@/features/keys/api'
import { ERROR_MESSAGES, SUCCESS_MESSAGES } from '@/features/keys/constants'
import type { ApiKey } from '@/features/keys/types'
import { getUserGroups } from '@/lib/api'
import { cn } from '@/lib/utils'
import { RatioTag } from '@/skins/next/pricing/components/ratio-tag'

import { buildApiKeyGroupUpdatePayload } from '../lib/group-update'
import { GroupPickerDialog } from './group-picker-dialog'

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
      label: value,
      desc: info.desc || value,
      ratio: info.ratio,
    })
  )

  const closePicker = () => {
    setOpen(false)
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
    <>
      <button
        type='button'
        data-slot='console-key-group-cell'
        aria-haspopup='dialog'
        aria-expanded={open}
        aria-label={t('Switch group')}
        title={label}
        className={cn(
          'border-border bg-muted/40 hover:bg-muted/70 data-popup-open:border-foreground/25 data-popup-open:bg-muted/80 inline-flex h-8 max-w-full min-w-0 cursor-pointer items-center gap-1.5 rounded-lg border px-2 text-start transition-colors',
          open && 'border-foreground/25 bg-muted/80',
          props.className
        )}
        onClick={() => handleOpenChange(true)}
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
      </button>
      <GroupPickerDialog
        open={open}
        onOpenChange={handleOpenChange}
        options={groups}
        value={currentGroup}
        title={t('Switch group')}
        isOptionsLoading={groupsQuery.isLoading}
        pendingValue={pendingGroup}
        onSelect={(group) => {
          void handleSelect(group)
        }}
      />
    </>
  )
}
