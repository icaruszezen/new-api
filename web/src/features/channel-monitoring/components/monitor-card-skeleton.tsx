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
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

export function MonitorCardSkeleton() {
  return (
    <Card className='rounded-2xl py-5'>
      <CardHeader className='px-5'>
        <div className='flex items-center gap-3'>
          <Skeleton className='size-10 shrink-0 rounded-full' />
          <div className='min-w-0 flex-1 space-y-1.5'>
            <div className='flex items-center justify-between gap-2'>
              <Skeleton className='h-4 w-32' />
              <Skeleton className='h-5 w-12 rounded-full' />
            </div>
            <Skeleton className='h-3 w-40' />
          </div>
        </div>
      </CardHeader>
      <CardContent className='space-y-4 px-5'>
        <div className='grid grid-cols-2 gap-2'>
          <Skeleton className='h-16 rounded-xl' />
          <Skeleton className='h-16 rounded-xl' />
        </div>
        <div className='flex items-center justify-between gap-2'>
          <Skeleton className='h-3 w-16' />
          <Skeleton className='h-7 w-16' />
        </div>
        <div className='space-y-1.5'>
          <div className='flex items-center justify-between gap-2'>
            <Skeleton className='h-3 w-28' />
            <Skeleton className='h-3 w-20' />
          </div>
          <Skeleton className='h-8 w-full rounded-[2px]' />
        </div>
      </CardContent>
    </Card>
  )
}
