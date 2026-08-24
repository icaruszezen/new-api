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
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { useStatus } from '@/hooks/use-status'
import { cn } from '@/lib/utils'

const CHAT_COMPLETIONS_PATH = '/v1/chat/completions'

const HEADLINE_LINES = [
  { text: 'One API.', muted: false, riseDelay: 0 },
  { text: 'Every model.', muted: true, riseDelay: 120 },
] as const

/**
 * Public chat completions endpoint shown on the landing page. `ServerAddress`
 * is the administrator-configured site address; deployments that never set it
 * fall back to the origin the visitor is already browsing.
 */
function useChatCompletionsEndpoint(): string {
  const { status } = useStatus()
  const configured = status?.server_address

  let base = ''
  if (typeof configured === 'string' && configured.trim()) {
    base = configured.trim()
  } else if (typeof window !== 'undefined') {
    base = window.location.origin
  }

  return `${base.replace(/\/+$/, '')}${CHAT_COMPLETIONS_PATH}`
}

export function Hero(props: { isAuthenticated: boolean }) {
  const { t } = useTranslation()
  const endpoint = useChatCompletionsEndpoint()

  return (
    <section className='pt-24 pb-16 md:pt-32 md:pb-24'>
      <h1 className='text-[clamp(2.75rem,7vw,4.5rem)] leading-[0.98] font-medium tracking-[-0.035em]'>
        {HEADLINE_LINES.map((line) => (
          <span
            key={line.text}
            className='landing-animate-fade-up block'
            style={{ animationDelay: `${line.riseDelay}ms` }}
          >
            <span
              data-landing-motion={line.muted ? undefined : 'always'}
              className={cn(
                line.muted
                  ? 'text-muted-foreground'
                  : 'landing-headline-sheen'
              )}
            >
              {t(line.text)}
            </span>
          </span>
        ))}
      </h1>

      <p
        className='text-muted-foreground landing-animate-fade-up mt-7 max-w-md text-[15px] leading-relaxed'
        style={{ animationDelay: '320ms' }}
      >
        {t('Access every major AI model through a single, unified API.')}
      </p>

      <div
        className='landing-animate-fade-up mt-9 flex flex-wrap items-center gap-x-6 gap-y-3'
        style={{ animationDelay: '420ms' }}
      >
        <Link
          to={props.isAuthenticated ? '/dashboard' : '/sign-up'}
          className='bg-foreground text-background inline-flex h-10 items-center rounded-full px-5 text-sm font-medium transition-[opacity,transform] duration-200 hover:opacity-85 active:scale-[0.98]'
        >
          {props.isAuthenticated ? t('Go to Dashboard') : t('Get Started')}
        </Link>

        <Link
          to='/pricing'
          className='text-muted-foreground hover:text-foreground group inline-flex items-center gap-1.5 text-sm transition-colors'
        >
          {t('View Pricing')}
          <ArrowRight
            aria-hidden='true'
            className='size-3.5 transition-transform group-hover:translate-x-0.5'
          />
        </Link>
      </div>

      <div
        className='border-border bg-muted/25 landing-animate-fade-up hover:bg-muted/45 mt-16 flex max-w-2xl items-center gap-3 rounded-xl border py-2.5 pr-2 pl-4 transition-colors duration-300'
        style={{ animationDelay: '520ms' }}
      >
        <span className='text-muted-foreground shrink-0 font-mono text-[11px] font-semibold tracking-wider'>
          POST
        </span>
        <code
          aria-label={t('Chat completions endpoint')}
          className='min-w-0 flex-1 truncate font-mono text-xs sm:text-[13px]'
        >
          {endpoint}
        </code>
        <CopyButton
          value={endpoint}
          size='icon'
          className='text-muted-foreground hover:text-foreground size-8'
          iconClassName='size-3.5'
        />
      </div>
    </section>
  )
}
