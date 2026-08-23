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
import { useTranslation } from 'react-i18next'

import { LanguageSwitcher } from '@/components/language-switcher'
// Imported from the concrete module instead of the `@/components/layout`
// barrel, which reaches the sidebar config and cycles back into features.
import { HeaderLogo } from '@/components/layout/components/header-logo'
import { NotificationPopover } from '@/components/notification-popover'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { ThemeSwitch } from '@/components/theme-switch'
import { Skeleton } from '@/components/ui/skeleton'
import { useNotifications } from '@/hooks/use-notifications'
import { useSystemConfig } from '@/hooks/use-system-config'
import { useTopNavLinks } from '@/hooks/use-top-nav-links'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import { LANDING_MEASURE_CLASS } from '../layout'

const NAV_LINK_CLASS =
  'text-muted-foreground hover:text-foreground rounded-md px-2.5 py-1.5 text-sm transition-colors'

/**
 * Flat header for the next landing page. It reuses the same navigation,
 * notification and profile sources as the classic `PublicHeader` but drops the
 * floating pill, the backdrop blur and the scroll-shrink animation so the page
 * stays quiet. Links that require authentication redirect straight to sign-in
 * instead of opening the classic countdown dialog.
 */
export function MinimalHeader() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const routerState = useRouterState()
  const { auth } = useAuthStore()
  const { systemName, logo, loading, logoLoaded } = useSystemConfig()
  const links = useTopNavLinks()
  const notifications = useNotifications()

  const pathname = routerState.location.pathname
  const isAuthenticated = !!auth.user

  // The entrance animates opacity only: a transform on a sticky element turns
  // it into a containing block and breaks the stick.
  return (
    <header className='border-border/60 bg-background/80 landing-animate-fade-in sticky top-0 z-50 border-b backdrop-blur-md'>
      <nav
        className={cn(
          LANDING_MEASURE_CLASS,
          'flex h-16 items-center justify-between gap-4'
        )}
      >
        <Link to='/' className='group flex shrink-0 items-center gap-2.5'>
          <div className='flex size-6 shrink-0 items-center justify-center transition-transform duration-300 group-hover:scale-105'>
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
          <span className='text-sm font-medium tracking-tight'>
            {loading ? <Skeleton className='h-4 w-16' /> : systemName}
          </span>
        </Link>

        <div className='flex items-center gap-1'>
          <div className='hidden items-center gap-0.5 sm:flex'>
            {links.map((link) => {
              if (link.external) {
                return (
                  <a
                    key={link.href}
                    href={link.href}
                    target='_blank'
                    rel='noopener noreferrer'
                    aria-disabled={link.disabled}
                    tabIndex={link.disabled ? -1 : undefined}
                    className={cn(
                      NAV_LINK_CLASS,
                      link.disabled && 'pointer-events-none opacity-50'
                    )}
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
                    if (!link.requiresAuth) return
                    event.preventDefault()
                    navigate({
                      to: '/sign-in',
                      search: { redirect: link.href },
                    })
                  }}
                  className={cn(
                    NAV_LINK_CLASS,
                    pathname === link.href && 'text-foreground',
                    link.disabled && 'pointer-events-none opacity-50'
                  )}
                >
                  {link.title}
                </Link>
              )
            })}
          </div>

          <div
            aria-hidden='true'
            className='bg-border/70 mx-2 hidden h-4 w-px sm:block'
          />

          <LanguageSwitcher />
          <ThemeSwitch />
          <NotificationPopover
            open={notifications.popoverOpen}
            onOpenChange={notifications.setPopoverOpen}
            unreadCount={notifications.unreadCount}
            activeTab={notifications.activeTab}
            onTabChange={notifications.setActiveTab}
            notice={notifications.notice}
            announcements={notifications.announcements}
            loading={notifications.loading}
          />

          {loading && <Skeleton className='h-8 w-16 rounded-md' />}
          {!loading && isAuthenticated && <ProfileDropdown />}
          {!loading && !isAuthenticated && (
            <Link
              to='/sign-in'
              className='ml-2 text-sm font-medium transition-opacity hover:opacity-70'
            >
              {t('Sign in')}
            </Link>
          )}
        </div>
      </nav>
    </header>
  )
}
