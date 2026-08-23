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

import { Home } from '@/features/home'
import { useHomePageContent } from '@/features/home/hooks'
import { useAuthStore } from '@/stores/auth-store'

import { LandingAtmosphere } from './components/atmosphere'
import { Hero } from './components/hero'
import { MinimalFooter } from './components/minimal-footer'
import { MinimalHeader } from './components/minimal-header'
import { ProviderStrip } from './components/provider-strip'
import { LANDING_MEASURE_CLASS } from './layout'

/**
 * Next landing page: a quiet, typography-led single column.
 *
 * `PublicLayout` always renders the classic `PublicHeader`, so this shell is
 * assembled by hand to carry the flat header instead. Administrator-configured
 * home pages (iframe / HTML / Markdown) keep the classic rendering, so this
 * skin only owns the default landing page.
 */
export function NextHome() {
  const { t } = useTranslation()
  const { auth } = useAuthStore()
  const { content, isLoaded } = useHomePageContent()

  if (!isLoaded) {
    return (
      <div className='bg-background text-foreground flex min-h-svh items-center justify-center'>
        <span className='text-muted-foreground text-sm'>{t('Loading...')}</span>
      </div>
    )
  }

  if (content) {
    return <Home />
  }

  // The entrance reveal is this page's whole design, so `data-landing-motion`
  // keeps it playing even for a visitor who asked for reduced motion. The
  // console and the classic pages still honour the preference.
  return (
    <div
      data-landing-motion='always'
      className='bg-background text-foreground relative isolate flex min-h-svh flex-col'
    >
      <LandingAtmosphere />
      <MinimalHeader />
      {/* The shell owns the single measure the page is built on, so the header
          logo, the headline and the provider row all share one left edge. */}
      <main className={`${LANDING_MEASURE_CLASS} flex-1`}>
        <Hero isAuthenticated={!!auth.user} />
        <ProviderStrip />
      </main>
      <MinimalFooter />
    </div>
  )
}
