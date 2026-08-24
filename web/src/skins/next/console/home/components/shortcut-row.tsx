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
import { Link } from '@tanstack/react-router'
import { CreditCard, FileText, UserRound, type LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { USAGE_LOGS_DEFAULT_SECTION } from '@/features/usage-logs/section-registry'

const SHORTCUT_CLASS =
  'bg-card hover:bg-muted/40 flex min-h-14 items-center justify-center gap-2.5 rounded-xl border px-4 py-3 text-sm font-medium transition-colors lg:h-full'

type ShortcutButtonProps = {
  label: string
  icon: LucideIcon
  to: '/profile' | '/wallet' | '/usage-logs/$section'
  params?: { section: string }
}

function ShortcutButton(props: ShortcutButtonProps) {
  const Icon = props.icon

  return (
    <Link to={props.to} params={props.params} className={SHORTCUT_CLASS}>
      <span className='bg-muted text-muted-foreground flex size-8 items-center justify-center rounded-lg'>
        <Icon className='size-4' aria-hidden='true' />
      </span>
      {props.label}
    </Link>
  )
}

export function ConsoleShortcutRow() {
  const { t } = useTranslation()

  return (
    <nav
      aria-label={t('Quick actions')}
      className='grid grid-cols-1 gap-3 lg:h-full lg:grid-rows-3'
    >
      <ShortcutButton
        label={t('Personal Center')}
        icon={UserRound}
        to='/profile'
      />
      <ShortcutButton
        label={t('Usage records')}
        icon={FileText}
        to='/usage-logs/$section'
        params={{ section: USAGE_LOGS_DEFAULT_SECTION }}
      />
      <ShortcutButton label={t('Recharge')} icon={CreditCard} to='/wallet' />
    </nav>
  )
}
