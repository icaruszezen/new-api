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

import { NextPublicShell } from '../public-shell'
import { LandingAtmosphere } from './components/atmosphere'
import { Hero } from './components/hero'
import { ProviderStrip } from './components/provider-strip'

/**
 * Next landing page: a quiet, typography-led single column.
 *
 * `PublicLayout` always renders the classic `PublicHeader`, so next public
 * pages use `NextPublicShell`. Administrator-configured home pages
 * (iframe / HTML / Markdown) keep the classic rendering.
 */
export function NextHome() {
  const { t } = useTranslation()
  const { auth } = useAuthStore()
  const { content, isLoaded } = useHomePageContent()

  if (!isLoaded) {
    return (
      <NextPublicShell>
        <div className='flex min-h-[50vh] items-center justify-center'>
          <span className='text-muted-foreground text-sm'>
            {t('Loading...')}
          </span>
        </div>
      </NextPublicShell>
    )
  }

  if (content) {
    return <Home />
  }

  return (
    <NextPublicShell atmosphere={<LandingAtmosphere />}>
      <Hero isAuthenticated={!!auth.user} />
      <ProviderStrip />
    </NextPublicShell>
  )
}
