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
import { useTranslation } from 'react-i18next'

import type { TicketUser } from '@/features/tickets/types'

type TicketUserSummaryProps = {
  user: TicketUser
}

/** The ticket owner's account details, shown to support only. */
export function TicketUserSummary(props: TicketUserSummaryProps) {
  const { t } = useTranslation()
  const fields = [
    { label: t('Username'), value: props.user.username },
    { label: t('Display Name'), value: props.user.display_name },
    { label: t('Email'), value: props.user.email },
    { label: t('Group'), value: props.user.group },
    { label: t('User ID'), value: String(props.user.id) },
  ]

  return (
    <section className='space-y-2'>
      <h3 className='text-muted-foreground text-xs font-medium'>
        {t('Submitted by')}
      </h3>
      <dl className='bg-muted/30 grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1.5 rounded-lg border px-4 py-3 text-sm'>
        {fields.map((field) => (
          <div key={field.label} className='contents'>
            <dt className='text-muted-foreground'>{field.label}</dt>
            <dd className='break-all'>{field.value || '-'}</dd>
          </div>
        ))}
      </dl>
    </section>
  )
}
