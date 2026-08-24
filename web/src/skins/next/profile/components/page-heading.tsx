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

import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Skeleton } from '@/components/ui/skeleton'
import { getDisplayName } from '@/features/profile/lib'
import type { UserProfile } from '@/features/profile/types'
import { getUserAvatarFallback, getUserAvatarStyle } from '@/lib/avatar'
import { getRoleLabel } from '@/lib/roles'

import { NextStatusPill } from './status-pill'

type NextProfileHeadingProps = {
  profile: UserProfile | null
  loading: boolean
}

export function NextProfileHeading(props: NextProfileHeadingProps) {
  const { t } = useTranslation()

  if (props.loading) {
    return (
      <header data-slot='next-profile-heading' className='flex flex-col gap-5'>
        <div className='space-y-2'>
          <Skeleton className='h-8 w-40' />
          <Skeleton className='h-4 w-72' />
        </div>
        <div className='flex items-center gap-3'>
          <Skeleton className='size-12 rounded-xl' />
          <div className='space-y-2'>
            <Skeleton className='h-5 w-36' />
            <Skeleton className='h-4 w-48' />
          </div>
        </div>
      </header>
    )
  }

  if (!props.profile) return null

  const displayName = getDisplayName(props.profile)
  const avatarName = props.profile.username || displayName

  return (
    <header data-slot='next-profile-heading' className='flex flex-col gap-5'>
      <div>
        <h1 className='text-3xl font-medium tracking-tight'>
          {t('Personal Center')}
        </h1>
        <p className='text-muted-foreground mt-1.5 text-sm'>
          {t('Manage account, security, and preferences')}
        </p>
      </div>

      <div
        data-slot='next-profile-identity'
        className='flex items-center gap-3'
      >
        <Avatar className='size-12 rounded-xl'>
          <AvatarFallback
            className='rounded-xl text-sm font-medium text-white'
            style={getUserAvatarStyle(avatarName)}
          >
            {getUserAvatarFallback(avatarName)}
          </AvatarFallback>
        </Avatar>
        <div className='min-w-0 space-y-1'>
          <div className='flex min-w-0 flex-wrap items-center gap-2'>
            <p className='truncate text-lg font-medium tracking-tight'>
              {displayName}
            </p>
            <NextStatusPill>{getRoleLabel(props.profile.role)}</NextStatusPill>
            <NextStatusPill>
              {t('User ID')} {props.profile.id}
            </NextStatusPill>
          </div>
          <p className='text-muted-foreground truncate text-sm'>
            @{props.profile.username}
            {props.profile.group ? ` · ${props.profile.group}` : ''}
          </p>
        </div>
      </div>
    </header>
  )
}
