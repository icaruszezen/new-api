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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { SUBJECT_MAX_LENGTH } from '@/features/tickets/lib/limits'

type TicketSearchBarProps = {
  /** The applied keyword. Remount with a new `key` to reset the draft. */
  keyword: string
  onSearch: (keyword: string) => void
}

/** Edits a draft keyword; the list only changes on Search or Enter. */
export function TicketSearchBar(props: TicketSearchBarProps) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState(props.keyword)

  return (
    <form
      role='search'
      className='flex w-full items-center gap-2 sm:w-auto'
      onSubmit={(event) => {
        event.preventDefault()
        props.onSearch(draft.trim())
      }}
    >
      <Input
        className='h-8 min-w-0 flex-1 sm:w-64 sm:flex-none'
        aria-label={t('Search tickets')}
        placeholder={t('Ticket ID, subject or username')}
        maxLength={SUBJECT_MAX_LENGTH}
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
      />
      <Button type='submit' size='sm'>
        <Search className='mr-1 h-4 w-4' aria-hidden='true' />
        {t('Search')}
      </Button>
      {props.keyword && (
        <Button
          type='button'
          size='sm'
          variant='outline'
          onClick={() => {
            setDraft('')
            props.onSearch('')
          }}
        >
          {t('Reset')}
        </Button>
      )}
    </form>
  )
}
