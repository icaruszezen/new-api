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
import { Link, useNavigate, useRouterState } from '@tanstack/react-router'
import { Menu } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { LanguageSwitcher } from '@/components/language-switcher'
// Imported from the concrete module instead of the `@/components/layout`
// barrel, which reaches the sidebar config and cycles back into features.
import { HeaderLogo } from '@/components/layout/components/header-logo'
import { NotificationPopover } from '@/components/notification-popover'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { ThemeSwitch } from '@/components/theme-switch'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { useNotifications } from '@/hooks/use-notifications'
import { useSystemConfig } from '@/hooks/use-system-config'
import { useTopNavLinks, type TopNavLink } from '@/hooks/use-top-nav-links'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import { isNextConsoleHomePath } from '../../../console-home-path'
import { LANDING_MEASURE_CLASS } from '../layout'

const NAV_LINK_CLASS =
  'text-muted-foreground hover:text-foreground rounded-none px-2.5 py-1.5 text-sm transition-colors'

const ACTIVE_NAV_LINK_CLASS =
  'text-foreground underline decoration-foreground underline-offset-8'

type HeaderNavLinksProps = {
  links: TopNavLink[]
  pathname: string
  onAuthLink: (href: string) => void
  onNavigate?: () => void
  className?: string
  linkClassName?: string
}

function HeaderNavLinks(props: HeaderNavLinksProps) {
  return (
    <div className={props.className}>
      {props.links.map((link) => {
        const isActive =
          props.pathname === link.href ||
          (isNextConsoleHomePath(link.href) &&
            isNextConsoleHomePath(props.pathname))
        const className = cn(
          NAV_LINK_CLASS,
          props.linkClassName,
          isActive && ACTIVE_NAV_LINK_CLASS,
          link.disabled && 'pointer-events-none opacity-50'
        )

        if (link.external) {
          return (
            <a
              key={link.href}
              href={link.href}
              target='_blank'
              rel='noopener noreferrer'
              aria-disabled={link.disabled}
              tabIndex={link.disabled ? -1 : undefined}
              className={className}
              onClick={props.onNavigate}
            >
              {link.title}
            </a>
          )
        }

        return (
          <Link
            key={link.href}
            to={link.href}
            disabled={link.disabled}
            onClick={(event) => {
              if (link.requiresAuth) {
                event.preventDefault()
                props.onAuthLink(link.href)
              }
              props.onNavigate?.()
            }}
            className={className}
          >
            {link.title}
          </Link>
        )
      })}
    </div>
  )
}

/**
 * Flat header for the next public and console-home chrome. Logo sits left,
 * configured nav links sit in the center on wide viewports, and account
 * controls stay on the right. Narrow viewports drop language/theme from the
 * bar, keep notifications and account, and open nav plus those utilities in
 * a sheet.
 */
export function MinimalHeader() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const routerState = useRouterState()
  const { auth } = useAuthStore()
  const { systemName, logo, loading, logoLoaded } = useSystemConfig()
  const links = useTopNavLinks()
  const notifications = useNotifications()
  const [menuOpen, setMenuOpen] = useState(false)

  const pathname = routerState.location.pathname
  const isAuthenticated = !!auth.user

  const handleAuthLink = (href: string) => {
    navigate({
      to: '/sign-in',
      search: { redirect: href },
    })
  }

  return (
    <header className='border-border bg-background/80 sticky top-0 z-50 border-b pt-[env(safe-area-inset-top)] backdrop-blur-md'>
      <nav
        className={cn(
          LANDING_MEASURE_CLASS,
          'flex h-14 touch-manipulation items-center justify-between gap-3',
          'sm:grid sm:h-16 sm:grid-cols-[1fr_auto_1fr] sm:gap-4'
        )}
      >
        <Link
          to='/'
          className='group flex min-w-0 items-center gap-2 justify-self-start sm:gap-2.5'
        >
          <div className='flex size-6 shrink-0 items-center justify-center'>
            {loading ? (
              <Skeleton className='size-full rounded-md' />
            ) : (
              <HeaderLogo
                src={logo}
                loading={loading}
                logoLoaded={logoLoaded}
                className='size-full rounded-md object-contain'
              />
            )}
          </div>
          <span className='max-w-[40vw] min-w-0 truncate text-sm font-medium tracking-tight sm:max-w-[16rem]'>
            {loading ? <Skeleton className='h-4 w-16' /> : systemName}
          </span>
        </Link>

        <HeaderNavLinks
          links={links}
          pathname={pathname}
          onAuthLink={handleAuthLink}
          className='hidden items-center justify-center gap-0.5 sm:flex'
        />

        <div className='flex shrink-0 items-center justify-end gap-2'>
          <div className='hidden items-center gap-2 sm:flex'>
            <LanguageSwitcher />
            <ThemeSwitch />
          </div>
          <NotificationPopover
            className='size-10 sm:size-9'
            open={notifications.popoverOpen}
            onOpenChange={notifications.setPopoverOpen}
            unreadCount={notifications.unreadCount}
            activeTab={notifications.activeTab}
            onTabChange={notifications.setActiveTab}
            notice={notifications.notice}
            announcements={notifications.announcements}
            loading={notifications.loading}
          />

          {loading && (
            <Skeleton className='size-10 rounded-md sm:h-8 sm:w-16' />
          )}
          {!loading && isAuthenticated && (
            <ProfileDropdown triggerClassName='size-10 sm:size-6' />
          )}
          {!loading && !isAuthenticated && (
            <Link
              to='/sign-in'
              className='px-1.5 text-sm font-medium transition-opacity hover:opacity-70'
            >
              {t('Sign in')}
            </Link>
          )}
          <Button
            type='button'
            variant='ghost'
            size='icon'
            className='size-11 sm:hidden'
            aria-label={t('Open menu')}
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen(true)}
          >
            <Menu className='size-5' aria-hidden='true' />
          </Button>
        </div>
      </nav>

      <Sheet open={menuOpen} onOpenChange={setMenuOpen}>
        <SheetContent side='right' className='sm:max-w-xs'>
          <SheetHeader>
            <SheetTitle>{systemName}</SheetTitle>
            <SheetDescription className='sr-only'>
              {t('Open menu')}
            </SheetDescription>
          </SheetHeader>
          <HeaderNavLinks
            links={links}
            pathname={pathname}
            onAuthLink={handleAuthLink}
            onNavigate={() => setMenuOpen(false)}
            className='flex flex-1 flex-col items-stretch gap-1 px-4'
            linkClassName='px-1 py-2.5 text-base'
          />
          <SheetFooter className='border-border border-t'>
            <p className='text-muted-foreground text-xs'>{t('Preferences')}</p>
            <div className='flex items-center gap-2'>
              <LanguageSwitcher />
              <ThemeSwitch />
            </div>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </header>
  )
}
