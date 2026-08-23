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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { useStatus } from '@/hooks/use-status'
import { useSystemConfig } from '@/hooks/use-system-config'
import { cn } from '@/lib/utils'

import { LANDING_MEASURE_CLASS } from '../layout'

const ATTRIBUTION_LINK = 'https://github.com/QuantumNous/new-api'
const NOTICE_ROTATION_MS = 6000

// Assembled the same way as the classic footer so the protected project
// attribution key is not greppable as a single literal.
const NEW_API_FOOTER_ATTRIBUTION_KEY = [
  'footer',
  'new' + 'api',
  'projectAttributionSuffix',
].join('.')

/**
 * Footer for the next landing page. The site copyright and the project
 * attribution share one slot: the leaving notice sinks and dissolves while the
 * arriving one rises into its place. Both notices stay in the DOM verbatim, so
 * neither is ever absent from the page or the accessibility tree.
 */
export function MinimalFooter() {
  const { t } = useTranslation()
  const { systemName, footerHtml } = useSystemConfig()
  const { status } = useStatus()

  const [activeNotice, setActiveNotice] = useState(0)
  // The rotating slot holds a link, so pointer or keyboard attention freezes it
  // rather than letting the target slide away mid-click.
  const [held, setHeld] = useState(false)

  const currentYear = new Date().getFullYear()

  const notices = [
    {
      key: 'site',
      content: (
        <>
          &copy; {currentYear} {systemName}. {t('footer.defaultCopyright')}
        </>
      ),
    },
    {
      key: 'project',
      content: (
        <>
          &copy; {currentYear}{' '}
          <a
            href={ATTRIBUTION_LINK}
            target='_blank'
            rel='noopener noreferrer'
            className='hover:text-foreground font-medium transition-colors'
          >
            {t('New API')}
          </a>
          . {t(NEW_API_FOOTER_ATTRIBUTION_KEY)}
        </>
      ),
    },
  ]
  const noticeCount = notices.length

  useEffect(() => {
    if (held) return

    const intervalId = window.setInterval(() => {
      setActiveNotice((current) => (current + 1) % noticeCount)
    }, NOTICE_ROTATION_MS)

    return () => window.clearInterval(intervalId)
  }, [held, noticeCount])

  const legalLinks: { key: string; label: string; href: string }[] = []
  if (status?.user_agreement_enabled) {
    legalLinks.push({
      key: 'user-agreement',
      label: t('User Agreement'),
      href: '/user-agreement',
    })
  }
  if (status?.privacy_policy_enabled) {
    legalLinks.push({
      key: 'privacy-policy',
      label: t('Privacy Policy'),
      href: '/privacy-policy',
    })
  }

  return (
    <footer className='border-border/60 border-t'>
      <AnimateInView
        ignoreReducedMotion
        className={cn(LANDING_MEASURE_CLASS, 'py-8')}
      >
        {/* Same trust model as the classic footer: this markup can only be set
            by an administrator through the site settings. */}
        {footerHtml && (
          <div
            className='custom-footer text-muted-foreground mb-5 text-xs'
            dangerouslySetInnerHTML={{ __html: footerHtml }}
          />
        )}

        <div className='text-muted-foreground flex flex-col items-start justify-between gap-x-6 gap-y-3 text-xs sm:flex-row sm:items-center'>
          {/* Rotating notices share one grid cell, so the row keeps the height
              of the tallest notice and nothing shifts as they hand over. The
              arriving notice is delayed so the leaving one clears first. */}
          <div
            className='grid'
            onMouseEnter={() => setHeld(true)}
            onMouseLeave={() => setHeld(false)}
            onFocus={() => setHeld(true)}
            onBlur={() => setHeld(false)}
          >
            {notices.map((notice, index) => (
              <span
                key={notice.key}
                data-testid={`footer-notice-${notice.key}`}
                className={cn(
                  'col-start-1 row-start-1 transition-[opacity,transform] duration-500 ease-out will-change-[transform,opacity]',
                  index === activeNotice
                    ? 'translate-y-0 opacity-100 delay-200'
                    : 'translate-y-1.5 opacity-0'
                )}
              >
                {notice.content}
              </span>
            ))}
          </div>

          {legalLinks.length > 0 && (
            <div className='flex shrink-0 items-center gap-x-4'>
              {legalLinks.map((link) => (
                <Link
                  key={link.key}
                  to={link.href}
                  className='hover:text-foreground transition-colors'
                >
                  {link.label}
                </Link>
              ))}
            </div>
          )}
        </div>
      </AnimateInView>
    </footer>
  )
}
