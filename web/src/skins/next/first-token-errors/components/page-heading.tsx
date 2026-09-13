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

import { FirstTokenErrorSettingsBar } from '@/features/first-token-errors/components/settings-bar'

export function NextFirstTokenErrorsHeading() {
  const { t } = useTranslation()

  return (
    <header
      data-slot='next-first-token-errors-heading'
      className='flex flex-col gap-3'
    >
      <div className='flex flex-wrap items-center justify-between gap-x-3 gap-y-2'>
        <h1 className='text-3xl font-medium tracking-tight'>
          {t('First-token errors')}
        </h1>
      </div>
      <FirstTokenErrorSettingsBar />
    </header>
  )
}
