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
import { Outlet, useRouterState } from '@tanstack/react-router'
import { useEffect } from 'react'

import { SkipToMain } from '@/components/skip-to-main'
import { cn } from '@/lib/utils'

import { isNextUsageLogsPath } from '../../console-home-path'
import { MinimalHeader } from '../home/components/minimal-header'
import { LANDING_MEASURE_CLASS } from '../home/layout'

type NextConsoleShellProps = {
  previewOffset?: boolean
}

/**
 * Standalone next console chrome for the user homepage. Public pages use
 * `data-next-public`; this page mirrors `data-next-console` onto body so
 * portaled menus inherit the charcoal tokens without touching classic.
 */
export function NextConsoleShell(props: NextConsoleShellProps) {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const isFullBleed = isNextUsageLogsPath(pathname)

  useEffect(() => {
    document.body.setAttribute('data-next-console', '')
    return () => {
      document.body.removeAttribute('data-next-console')
    }
  }, [])

  return (
    <div
      data-skin='next'
      className={cn(
        'bg-background text-foreground relative isolate flex min-h-svh flex-col',
        props.previewOffset && 'pt-9 [&_header]:top-9'
      )}
    >
      <SkipToMain />
      <MinimalHeader />
      <main
        id='content'
        className={cn(
          'flex min-h-0 flex-1 flex-col',
          isFullBleed ? 'w-full px-0 py-0' : `${LANDING_MEASURE_CLASS} py-8`
        )}
      >
        <Outlet />
      </main>
    </div>
  )
}
