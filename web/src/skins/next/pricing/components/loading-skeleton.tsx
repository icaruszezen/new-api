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
import { Skeleton } from '@/components/ui/skeleton'

export function NextPricingSkeleton() {
  return (
    <div className='pt-16 pb-20'>
      <div className='mx-auto mb-10 flex max-w-2xl flex-col items-center gap-6'>
        <Skeleton className='h-12 w-56' />
        <Skeleton className='h-11 w-full rounded-xl' />
      </div>
      <div className='overflow-hidden rounded-xl border'>
        {[
          'row-a',
          'row-b',
          'row-c',
          'row-d',
          'row-e',
          'row-f',
          'row-g',
          'row-h',
        ].map((rowId) => (
          <div
            key={rowId}
            className='flex items-center justify-between gap-4 border-b px-4 py-4 last:border-b-0'
          >
            <Skeleton className='h-4 w-36' />
            <div className='flex gap-8'>
              <Skeleton className='h-4 w-14' />
              <Skeleton className='h-4 w-14' />
              <Skeleton className='h-4 w-14' />
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
