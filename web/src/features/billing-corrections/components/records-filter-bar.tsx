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
import { Search } from 'lucide-react'
import { useState, type KeyboardEvent } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { CompactDateTimeRangePicker } from '@/features/usage-logs/components/compact-date-time-range-picker'

import { getChannelLabel, getDefaultRecordFilters } from '../lib/records'
import type {
  BillingCorrectionRecordFilters,
  BillingCorrectionRule,
} from '../types'

type RecordsFilterBarProps = {
  rules: BillingCorrectionRule[]
  filters: BillingCorrectionRecordFilters
  onApply: (filters: BillingCorrectionRecordFilters) => void
  /** Shows the upstream result filter when any rule checks the upstream. */
  upstreamEnabled: boolean
}

/** Edits a draft of the filters; nothing is queried until Search or Enter. */
export function RecordsFilterBar(props: RecordsFilterBarProps) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState(props.filters)

  const updateDraft = (patch: Partial<BillingCorrectionRecordFilters>) => {
    setDraft((previous) => ({ ...previous, ...patch }))
  }

  const applyOnEnter = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter') props.onApply(draft)
  }

  const reset = () => {
    const defaults = getDefaultRecordFilters()
    setDraft(defaults)
    props.onApply(defaults)
  }

  return (
    <div className='flex flex-wrap items-center gap-2'>
      <CompactDateTimeRangePicker
        className='w-auto'
        start={draft.start}
        end={draft.end}
        onChange={(range) =>
          updateDraft({ start: range.start, end: range.end })
        }
      />
      <Input
        className='h-8 w-36'
        placeholder={t('Username')}
        aria-label={t('Username')}
        value={draft.username}
        onChange={(event) => updateDraft({ username: event.target.value })}
        onKeyDown={applyOnEnter}
      />
      <Input
        className='h-8 w-36'
        placeholder={t('Token')}
        aria-label={t('Token')}
        value={draft.token}
        onChange={(event) => updateDraft({ token: event.target.value })}
        onKeyDown={applyOnEnter}
      />
      <Input
        className='h-8 w-40'
        placeholder={t('Model')}
        aria-label={t('Model')}
        value={draft.model}
        onChange={(event) => updateDraft({ model: event.target.value })}
        onKeyDown={applyOnEnter}
      />
      <NativeSelect
        size='sm'
        aria-label={t('Channel')}
        value={draft.channel}
        onChange={(event) => updateDraft({ channel: event.target.value })}
      >
        <NativeSelectOption value=''>{t('All channels')}</NativeSelectOption>
        {props.rules
          .filter((rule) => rule.enabled)
          .map((rule) => (
            <NativeSelectOption key={rule.id} value={String(rule.channel_id)}>
              {getChannelLabel(rule.channel_id, rule.channel_name)}
            </NativeSelectOption>
          ))}
      </NativeSelect>
      <NativeSelect
        size='sm'
        aria-label={t('Status')}
        value={draft.status}
        onChange={(event) =>
          updateDraft({
            status: event.target
              .value as BillingCorrectionRecordFilters['status'],
          })
        }
      >
        <NativeSelectOption value=''>{t('All Status')}</NativeSelectOption>
        <NativeSelectOption value='pending'>
          {t('Pending refund')}
        </NativeSelectOption>
        <NativeSelectOption value='credited'>
          {t('Quota returned')}
        </NativeSelectOption>
        <NativeSelectOption value='refunded'>
          {t('Marked as refunded')}
        </NativeSelectOption>
        <NativeSelectOption value='corrected'>
          {t('Corrected')}
        </NativeSelectOption>
      </NativeSelect>
      <NativeSelect
        size='sm'
        aria-label={t('Condition')}
        value={draft.match}
        onChange={(event) =>
          updateDraft({
            match: event.target
              .value as BillingCorrectionRecordFilters['match'],
          })
        }
      >
        <NativeSelectOption value=''>{t('All conditions')}</NativeSelectOption>
        <NativeSelectOption value='rule'>
          {t('Matches rule')}
        </NativeSelectOption>
        <NativeSelectOption value='zero_output'>
          {t('Zero output only')}
        </NativeSelectOption>
      </NativeSelect>
      {props.upstreamEnabled && (
        <NativeSelect
          size='sm'
          aria-label={t('Upstream result')}
          value={draft.upstream}
          onChange={(event) =>
            updateDraft({
              upstream: event.target
                .value as BillingCorrectionRecordFilters['upstream'],
            })
          }
        >
          <NativeSelectOption value=''>
            {t('All upstream results')}
          </NativeSelectOption>
          <NativeSelectOption value='not_charged'>
            {t('Upstream not charged')}
          </NativeSelectOption>
          <NativeSelectOption value='charged'>
            {t('Upstream charged')}
          </NativeSelectOption>
          <NativeSelectOption value='unchecked'>
            {t('Not checked')}
          </NativeSelectOption>
          <NativeSelectOption value='not_found'>
            {t('Not found upstream')}
          </NativeSelectOption>
          <NativeSelectOption value='failed'>
            {t('Upstream check failed')}
          </NativeSelectOption>
          <NativeSelectOption value='unsupported'>
            {t('Upstream unsupported')}
          </NativeSelectOption>
          <NativeSelectOption value='unavailable'>
            {t('Cannot check')}
          </NativeSelectOption>
        </NativeSelect>
      )}
      <Button size='sm' onClick={() => props.onApply(draft)}>
        <Search className='mr-1 h-4 w-4' />
        {t('Search')}
      </Button>
      <Button size='sm' variant='outline' onClick={reset}>
        {t('Reset')}
      </Button>
    </div>
  )
}
