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
import { Add01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

export function NextTicketsHeading(props: { onNewTicket: () => void }) {
  const { t } = useTranslation()

  return (
    <header
      data-slot='next-tickets-heading'
      className='flex flex-wrap items-end justify-between gap-x-3 gap-y-4'
    >
      <div className='min-w-0'>
        <h1 className='text-3xl font-medium tracking-tight'>
          {t('Support tickets')}
        </h1>
        <p className='text-muted-foreground mt-1.5 text-sm'>
          {t(
            'Report a problem or ask a question. Our support team replies here, and you can follow up at any time.'
          )}
        </p>
      </div>
      <Button
        type='button'
        onClick={props.onNewTicket}
        className='h-10 shrink-0 px-4'
      >
        <HugeiconsIcon icon={Add01Icon} strokeWidth={2} aria-hidden='true' />
        {t('New ticket')}
      </Button>
    </header>
  )
}
