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
import { getRouteApi } from '@tanstack/react-router'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { CompactDateTimeRangePicker } from '@/features/usage-logs/components/compact-date-time-range-picker'

import { getDefaultTimeRange } from '../lib/utils'

const route = getRouteApi('/_authenticated/first-token-errors/')

type FirstTokenErrorDraft = {
  sourceKey: string
  username: string
  token: string
  model: string
  channel: string
  group: string
  requestId: string
  start: Date
  end: Date
}

function buildSearchSourceKey(values: {
  username?: string
  token?: string
  model?: string
  channel?: string
  group?: string
  requestId?: string
  startTime?: number
  endTime?: number
}) {
  return [
    values.username,
    values.token,
    values.model,
    values.channel,
    values.group,
    values.requestId,
    values.startTime,
    values.endTime,
  ]
    .map((value) => String(value ?? ''))
    .join('\u001f')
}

function draftFromSearch(search: {
  username?: string
  token?: string
  model?: string
  channel?: string
  group?: string
  requestId?: string
  startTime?: number
  endTime?: number
}): FirstTokenErrorDraft {
  const defaultRange = getDefaultTimeRange()
  return {
    sourceKey: buildSearchSourceKey(search),
    username: search.username ?? '',
    token: search.token ?? '',
    model: search.model ?? '',
    channel: search.channel ?? '',
    group: search.group ?? '',
    requestId: search.requestId ?? '',
    start: search.startTime ? new Date(search.startTime) : defaultRange.start,
    end: search.endTime ? new Date(search.endTime) : defaultRange.end,
  }
}

export function FirstTokenErrorFilterBar() {
  const { t } = useTranslation()
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const searchState = draftFromSearch(search)
  const [draft, setDraft] = useState<FirstTokenErrorDraft>(() => searchState)
  const activeDraft = draft.sourceKey === searchState.sourceKey ? draft : searchState

  const applySearch = (next: FirstTokenErrorDraft) => {
    void navigate({
      search: (prev) => ({
        ...prev,
        username: next.username,
        token: next.token,
        model: next.model,
        channel: next.channel,
        group: next.group,
        requestId: next.requestId,
        startTime: next.start.getTime(),
        endTime: next.end.getTime(),
        page: 1,
      }),
    })
  }

  const updateDraft = (patch: Partial<FirstTokenErrorDraft>) => {
    setDraft((current) => {
      const base = current.sourceKey === searchState.sourceKey ? current : searchState
      return {
        ...base,
        ...patch,
        sourceKey: searchState.sourceKey,
      }
    })
  }

  const handleSearch = () => {
    applySearch(activeDraft)
  }

  const handleReset = () => {
    const defaultRange = getDefaultTimeRange()
    const next: FirstTokenErrorDraft = {
      sourceKey: buildSearchSourceKey({
        startTime: defaultRange.start.getTime(),
        endTime: defaultRange.end.getTime(),
      }),
      username: '',
      token: '',
      model: '',
      channel: '',
      group: '',
      requestId: '',
      start: defaultRange.start,
      end: defaultRange.end,
    }
    setDraft(next)
    applySearch(next)
  }

  return (
    <div className='flex flex-col gap-3'>
      <CompactDateTimeRangePicker
        start={activeDraft.start}
        end={activeDraft.end}
        onChange={(range) => {
          const defaultRange = getDefaultTimeRange()
          updateDraft({
            start: range.start ?? defaultRange.start,
            end: range.end ?? defaultRange.end,
          })
        }}
      />
      <div className='flex flex-wrap gap-2'>
        <Input
          value={activeDraft.username}
          onChange={(event) => updateDraft({ username: event.target.value })}
          onKeyDown={(event) => {
            if (event.key === 'Enter') handleSearch()
          }}
          placeholder={t('User')}
          className='w-36'
        />
        <Input
          value={activeDraft.token}
          onChange={(event) => updateDraft({ token: event.target.value })}
          onKeyDown={(event) => {
            if (event.key === 'Enter') handleSearch()
          }}
          placeholder={t('Token')}
          className='w-36'
        />
        <Input
          value={activeDraft.model}
          onChange={(event) => updateDraft({ model: event.target.value })}
          onKeyDown={(event) => {
            if (event.key === 'Enter') handleSearch()
          }}
          placeholder={t('Model')}
          className='w-36'
        />
        <Input
          value={activeDraft.channel}
          onChange={(event) => updateDraft({ channel: event.target.value })}
          onKeyDown={(event) => {
            if (event.key === 'Enter') handleSearch()
          }}
          placeholder={t('Channel ID')}
          className='w-28'
        />
        <Input
          value={activeDraft.group}
          onChange={(event) => updateDraft({ group: event.target.value })}
          onKeyDown={(event) => {
            if (event.key === 'Enter') handleSearch()
          }}
          placeholder={t('Group')}
          className='w-28'
        />
        <Input
          value={activeDraft.requestId}
          onChange={(event) => updateDraft({ requestId: event.target.value })}
          onKeyDown={(event) => {
            if (event.key === 'Enter') handleSearch()
          }}
          placeholder={t('Request ID')}
          className='w-44'
        />
        <Button onClick={handleSearch}>{t('Search')}</Button>
        <Button variant='outline' onClick={handleReset}>
          {t('Reset')}
        </Button>
      </div>
    </div>
  )
}
