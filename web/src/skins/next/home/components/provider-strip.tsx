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

import { AnimateInView } from '@/components/animate-in-view'
import { getLobeIcon } from '@/lib/lobe-icon'

// Monochrome `@lobehub/icons` keys (no `.Color` suffix) so the row inherits the
// surrounding text color. Vendor names are proper nouns and stay untranslated.
const PROVIDERS = [
  { icon: 'OpenAI', name: 'OpenAI' },
  { icon: 'Claude', name: 'Anthropic' },
  { icon: 'Gemini', name: 'Google' },
  { icon: 'Azure', name: 'Azure' },
  { icon: 'Aws', name: 'Bedrock' },
  { icon: 'DeepSeek', name: 'DeepSeek' },
] as const

export function ProviderStrip() {
  const { t } = useTranslation()

  return (
    <section
      aria-label={t('Supported providers')}
      className='border-border/60 border-t pt-10 pb-24'
    >
      <AnimateInView
        as='span'
        ignoreReducedMotion
        className='text-muted-foreground block text-[11px] font-medium tracking-[0.16em] uppercase'
      >
        {t('Works with')}
      </AnimateInView>

      <ul className='text-muted-foreground mt-6 flex flex-wrap items-center gap-x-9 gap-y-5'>
        {PROVIDERS.map((provider, index) => (
          <AnimateInView
            key={provider.icon}
            as='li'
            ignoreReducedMotion
            delay={100 + index * 70}
            className='hover:text-foreground flex items-center gap-2 transition-colors duration-300'
          >
            <span aria-hidden='true' className='flex items-center'>
              {getLobeIcon(provider.icon, 16)}
            </span>
            <span className='text-[13px]'>{provider.name}</span>
          </AnimateInView>
        ))}
      </ul>
    </section>
  )
}
