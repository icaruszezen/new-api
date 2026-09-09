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
import { Share2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

type NextInviteLinkCardProps = {
  inviteLink: string
  ruleLabels: string[]
  loading: boolean
}

export function NextInviteLinkCard(props: NextInviteLinkCardProps) {
  const { t } = useTranslation()
  const { copyToClipboard } = useCopyToClipboard()

  // Web Share is only available on secure contexts and mostly on mobile; fall
  // back to copying so the button always does something useful.
  const handleShare = async () => {
    if (!navigator.share) {
      copyToClipboard(props.inviteLink)
      return
    }
    try {
      await navigator.share({
        title: t('Referral Rebate'),
        url: props.inviteLink,
      })
    } catch {
      // The user dismissed the share sheet; nothing to report.
    }
  }

  if (props.loading) {
    return (
      <section
        data-slot='next-invite-link'
        className='bg-card overflow-hidden rounded-xl border'
      >
        <div className='space-y-2 px-5 py-5'>
          <Skeleton className='h-6 w-28' />
          <Skeleton className='h-4 w-64' />
        </div>
        <div className='space-y-3 border-t px-5 py-5'>
          <Skeleton className='h-3 w-24' />
          <Skeleton className='h-10 w-full' />
          <Skeleton className='h-6 w-72' />
        </div>
      </section>
    )
  }

  return (
    <section
      data-slot='next-invite-link'
      className='bg-card overflow-hidden rounded-xl border'
    >
      <div className='flex flex-wrap items-start justify-between gap-3 px-5 py-5'>
        <div className='min-w-0'>
          <h2 className='text-lg font-medium tracking-tight'>
            {t('Referral link')}
          </h2>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t(
              'Send the link to a friend. The referral is bound as soon as they sign up.'
            )}
          </p>
        </div>
        <Button
          type='button'
          variant='outline'
          onClick={handleShare}
          disabled={!props.inviteLink}
          className='h-9 shrink-0 px-3'
        >
          <Share2 className='size-4' aria-hidden='true' />
          {t('Share link')}
        </Button>
      </div>

      <div className='space-y-3 border-t px-5 py-5'>
        <p className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
          {t('Referral link')}
        </p>
        <div className='grid grid-cols-[minmax(0,1fr)_auto_auto] gap-2'>
          <Input
            value={props.inviteLink}
            readOnly
            aria-label={t('Referral link')}
            className='h-10 min-w-0 font-mono text-xs'
          />
          <CopyButton
            value={props.inviteLink}
            variant='outline'
            className='size-10'
            iconClassName='size-4'
            tooltip={t('Copy referral link')}
            aria-label={t('Copy referral link')}
          />
          <Button
            type='button'
            onClick={() => copyToClipboard(props.inviteLink)}
            disabled={!props.inviteLink}
            className='h-10 px-4'
          >
            {t('Copy link')}
          </Button>
        </div>
        {props.ruleLabels.length > 0 ? (
          <ul className='flex flex-wrap gap-2'>
            {props.ruleLabels.map((label) => (
              <li
                key={label}
                data-slot='next-invite-rule-chip'
                className='border-border text-muted-foreground rounded-lg border px-2.5 py-1 text-xs'
              >
                {label}
              </li>
            ))}
          </ul>
        ) : null}
      </div>
    </section>
  )
}
