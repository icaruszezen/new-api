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
import { ExternalLink, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'

type NextWalletRedeemCardProps = {
  code: string
  onCodeChange: (value: string) => void
  onRedeem: () => void
  redeeming: boolean
  topupLink?: string
  enabled: boolean
  loading: boolean
}

export function NextWalletRedeemCard(props: NextWalletRedeemCardProps) {
  const { t } = useTranslation()

  if (props.loading) {
    return (
      <section
        data-slot='next-wallet-redeem'
        className='bg-card overflow-hidden rounded-xl border lg:col-start-2'
      >
        <div className='space-y-2 px-5 py-5'>
          <Skeleton className='h-6 w-24' />
          <Skeleton className='h-4 w-48' />
        </div>
        <div className='border-t px-5 py-5'>
          <Skeleton className='h-10 w-full' />
        </div>
      </section>
    )
  }

  return (
    <section
      data-slot='next-wallet-redeem'
      className='bg-card overflow-hidden rounded-xl border lg:col-start-2'
    >
      <div className='px-5 py-5'>
        <h2 className='text-lg font-medium tracking-tight'>
          {t('Redemption Code')}
        </h2>
        <p className='text-muted-foreground mt-1 text-sm'>
          {t('Have a Code?')}
        </p>
      </div>

      <div className='border-t px-5 py-5'>
        {props.enabled ? (
          <div className='space-y-3'>
            <div className='grid grid-cols-[minmax(0,1fr)_auto] gap-2'>
              <Input
                id='next-wallet-redemption-code'
                value={props.code}
                onChange={(event) => props.onCodeChange(event.target.value)}
                placeholder={t('Enter your redemption code')}
                className='h-10 min-w-0'
              />
              <Button
                type='button'
                onClick={props.onRedeem}
                disabled={props.redeeming || !props.code}
                className='h-10 px-4'
              >
                {props.redeeming ? (
                  <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                ) : null}
                {t('Redeem')}
              </Button>
            </div>
            {props.topupLink ? (
              <p className='text-muted-foreground text-xs'>
                {t('Need a redemption code?')}{' '}
                <a
                  href={props.topupLink}
                  target='_blank'
                  rel='noopener noreferrer'
                  className='inline-flex items-center gap-1 underline-offset-4 hover:underline'
                >
                  {t('Get one here')}
                  <ExternalLink className='h-3 w-3' aria-hidden='true' />
                </a>
              </p>
            ) : null}
          </div>
        ) : (
          <Alert>
            <AlertDescription>
              {t(
                'Redemption codes are disabled until the administrator confirms compliance terms.'
              )}
            </AlertDescription>
          </Alert>
        )}
      </div>
    </section>
  )
}
