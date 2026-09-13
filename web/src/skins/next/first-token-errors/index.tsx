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
import { useState } from 'react'

import { PageFooterProvider } from '@/components/layout/components/page-footer'
import { FirstTokenErrorTable } from '@/features/first-token-errors/components/table'

import { NextFirstTokenErrorsHeading } from './components/page-heading'

export function NextFirstTokenErrors() {
  const [footerContainer, setFooterContainer] = useState<HTMLDivElement | null>(
    null
  )

  return (
    <PageFooterProvider container={footerContainer}>
      <div
        data-slot='next-first-token-errors'
        className='flex h-full min-h-0 w-full flex-1 flex-col gap-3 px-3 pt-3 pb-3 sm:gap-4 sm:px-4 sm:pt-5 sm:pb-4'
      >
        <NextFirstTokenErrorsHeading />
        <section
          data-slot='next-first-token-errors-card'
          className='bg-card flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border'
        >
          <div className='min-h-0 flex-1 px-5 py-5'>
            <FirstTokenErrorTable />
          </div>
          <div
            ref={setFooterContainer}
            data-slot='next-first-token-errors-footer'
            className='border-t px-5 py-3 empty:hidden'
          />
        </section>
      </div>
    </PageFooterProvider>
  )
}
