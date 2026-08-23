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
import { useEffect, type ReactNode } from 'react'

import { MinimalFooter } from './home/components/minimal-footer'
import { MinimalHeader } from './home/components/minimal-header'
import { LANDING_MEASURE_CLASS } from './home/layout'

type NextPublicShellProps = {
  children: ReactNode
  atmosphere?: ReactNode
  alwaysLandingMotion?: boolean
}

/**
 * Shared next-skin chrome for public pages. `PublicLayout` always mounts the
 * classic header, so next pages assemble this shell instead.
 *
 * `data-skin="next"` scopes the near-black tokens on the page itself.
 * `data-next-public` is mirrored onto `document.body` so portaled sheets and
 * menus inherit the same tokens without touching the console.
 */
export function NextPublicShell(props: NextPublicShellProps) {
  useEffect(() => {
    document.body.setAttribute('data-next-public', '')
    return () => {
      document.body.removeAttribute('data-next-public')
    }
  }, [])

  return (
    <div
      data-skin='next'
      data-landing-motion={props.alwaysLandingMotion ? 'always' : undefined}
      className='bg-background text-foreground relative isolate flex min-h-svh flex-col'
    >
      {props.atmosphere}
      <MinimalHeader />
      <main className={`${LANDING_MEASURE_CLASS} flex-1`}>
        {props.children}
      </main>
      <MinimalFooter />
    </div>
  )
}
