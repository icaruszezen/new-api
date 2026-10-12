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
import { CustomerService01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import { Link, useRouterState } from '@tanstack/react-router'
import { useId } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { getSelfTicketUnreadCount } from '@/features/tickets/api'
import { TICKET_QUERY_KEYS } from '@/features/tickets/constants'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import {
  isNextFirstTokenErrorsPath,
  isNextStandaloneShellPath,
  isNextTicketsPath,
  isNextUsageLogsPath,
} from '../../console-home-path'
import { useResolvedConsoleSkin } from '../../use-resolved-console-skin'

// Their own composer or send button sits in the bottom-right corner.
const HIDDEN_PATH_PREFIXES = ['/playground', '/chat', '/chat2link']

const UNREAD_POLL_INTERVAL_MS = 60_000

/**
 * Support entry for signed-in users of the Next console. Guests and
 * sessions resolved to the classic console (administrators who are not
 * previewing) never see it, and it never runs the router or unread query
 * for them.
 */
export function NextTicketFab() {
  const userId = useAuthStore((state) => state.auth.user?.id)
  const skin = useResolvedConsoleSkin()

  if (userId === undefined || skin !== 'next') return null

  return <NextTicketFabLink userId={userId} />
}

function NextTicketFabLink(props: { userId: number }) {
  const { t } = useTranslation()
  const unreadLabelId = useId()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const isHidden =
    isNextTicketsPath(pathname) ||
    HIDDEN_PATH_PREFIXES.some(
      (prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`)
    )
  // Full-bleed Next list pages and sidebar-shell pages (whose page footer
  // holds the table pagination) keep their pagination bottom-right.
  const isLifted =
    !isNextStandaloneShellPath(pathname) ||
    isNextUsageLogsPath(pathname) ||
    isNextFirstTokenErrorsPath(pathname)

  const unreadQuery = useQuery({
    queryKey: [...TICKET_QUERY_KEYS.selfUnread, props.userId],
    queryFn: getSelfTicketUnreadCount,
    enabled: !isHidden,
    refetchInterval: UNREAD_POLL_INTERVAL_MS,
    refetchOnWindowFocus: true,
  })

  if (isHidden) return null

  const unreadCount = unreadQuery.data ?? 0
  const hasUnread = unreadCount > 0

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Link
            to='/tickets'
            data-slot='next-ticket-fab'
            aria-label={t('Support tickets')}
            aria-describedby={hasUnread ? unreadLabelId : undefined}
            className={cn(
              'bg-primary text-primary-foreground focus-visible:ring-ring/50 fixed right-4 z-40 mb-[env(safe-area-inset-bottom)] flex size-12 items-center justify-center rounded-full shadow-lg transition-transform outline-none hover:scale-105 focus-visible:ring-3 motion-reduce:transition-none motion-reduce:hover:scale-100 sm:right-6',
              isLifted ? 'bottom-20 sm:bottom-24' : 'bottom-4 sm:bottom-6'
            )}
          />
        }
      >
        <HugeiconsIcon
          icon={CustomerService01Icon}
          strokeWidth={1.75}
          aria-hidden='true'
          className='size-6'
        />
        {hasUnread && (
          <>
            <span
              data-slot='next-ticket-fab-badge'
              aria-hidden='true'
              className='bg-destructive ring-background absolute -top-1 -right-1 flex h-5 min-w-5 items-center justify-center rounded-full px-1 text-[10px] font-medium text-white tabular-nums ring-2'
            >
              {unreadCount > 99 ? '99+' : unreadCount}
            </span>
            <span id={unreadLabelId} className='sr-only'>
              {t('Unread updates: {{count}}', { count: unreadCount })}
            </span>
          </>
        )}
      </TooltipTrigger>
      <TooltipContent side='left'>{t('Support tickets')}</TooltipContent>
    </Tooltip>
  )
}
