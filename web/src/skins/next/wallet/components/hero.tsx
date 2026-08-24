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

import { Skeleton } from '@/components/ui/skeleton'
import { formatCurrencyFromUSD } from '@/lib/currency'

type NextWalletHeroProps = {
  amount: number
  loading: boolean
}

export function NextWalletHero(props: NextWalletHeroProps) {
  const { t } = useTranslation()

  return (
    <header
      data-slot='next-wallet-hero'
      className='hidden lg:flex lg:flex-col lg:self-center'
    >
      <h1
        className='landing-animate-fade-up text-5xl font-medium tracking-tight'
        style={{ animationDelay: '0ms' }}
      >
        {t('Add Funds')}
      </h1>
      {props.loading ? (
        <Skeleton
          className='landing-animate-fade-up mt-8 h-16 w-48'
          style={{ animationDelay: '180ms' }}
        />
      ) : (
        <p className='mt-8 text-6xl font-medium tracking-tight tabular-nums'>
          <span
            key={props.amount}
            className='landing-animate-fade-up inline-block'
          >
            {formatCurrencyFromUSD(props.amount)}
          </span>
        </p>
      )}
      <p
        className='text-muted-foreground landing-animate-fade-up mt-4 max-w-sm text-sm'
        style={{ animationDelay: '270ms' }}
      >
        {t('Choose an amount and payment method. Quota arrives instantly.')}
      </p>
    </header>
  )
}
