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
import {
  ChevronRight,
  Key,
  Shield,
  Trash2,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { AccessTokenDialog } from '@/features/profile/components/dialogs/access-token-dialog'
import { ChangePasswordDialog } from '@/features/profile/components/dialogs/change-password-dialog'
import { DeleteAccountDialog } from '@/features/profile/components/dialogs/delete-account-dialog'
import type { UserProfile } from '@/features/profile/types'
import { useDialogs } from '@/hooks/use-dialog'
import { cn } from '@/lib/utils'

type NextProfileSecurityListProps = {
  profile: UserProfile | null
  loading: boolean
}

type DialogKey = 'password' | 'token' | 'delete'

export function NextProfileSecurityList(props: NextProfileSecurityListProps) {
  const { t } = useTranslation()
  const dialogs = useDialogs<DialogKey>()

  if (props.loading) {
    return (
      <section
        data-slot='next-profile-security'
        className='bg-card overflow-hidden rounded-xl border'
      >
        <div className='space-y-2 px-5 py-5'>
          <Skeleton className='h-6 w-32' />
          <Skeleton className='h-4 w-48' />
        </div>
        <div className='space-y-2 border-t px-5 py-4'>
          <Skeleton className='h-14 w-full' />
          <Skeleton className='h-14 w-full' />
        </div>
      </section>
    )
  }

  if (!props.profile) return null

  const actions: {
    key: DialogKey
    icon: LucideIcon
    title: string
    description: string
    destructive?: boolean
  }[] = [
    {
      key: 'password',
      icon: Shield,
      title: t('Change Password'),
      description: t('Update your password to keep your account secure'),
    },
    {
      key: 'token',
      icon: Key,
      title: t('Access Token'),
      description: t('Generate and manage your API access token'),
    },
    {
      key: 'delete',
      icon: Trash2,
      title: t('Delete Account'),
      description: t('Permanently delete your account and all data'),
      destructive: true,
    },
  ]

  return (
    <>
      <section
        data-slot='next-profile-security'
        className='bg-card overflow-hidden rounded-xl border'
      >
        <div className='px-5 py-5'>
          <h2 className='text-lg font-medium tracking-tight'>
            {t('Security Settings')}
          </h2>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Manage your security settings and account access')}
          </p>
        </div>
        <div className='border-t'>
          {actions.map((item) => {
            const Icon = item.icon
            return (
              <button
                key={item.key}
                type='button'
                onClick={() => dialogs.open(item.key)}
                className={cn(
                  'hover:bg-muted/40 flex w-full items-center gap-3 px-5 py-3.5 text-left transition-colors',
                  item.destructive && 'text-destructive'
                )}
              >
                <span
                  className={cn(
                    'flex size-8 shrink-0 items-center justify-center rounded-lg',
                    item.destructive
                      ? 'bg-destructive/10 text-destructive'
                      : 'bg-muted text-muted-foreground'
                  )}
                >
                  <Icon className='size-4' aria-hidden='true' />
                </span>
                <span className='min-w-0 flex-1'>
                  <span className='block text-sm font-medium'>
                    {item.title}
                  </span>
                  <span
                    className={cn(
                      'mt-0.5 block truncate text-xs',
                      item.destructive
                        ? 'text-destructive/80'
                        : 'text-muted-foreground'
                    )}
                  >
                    {item.description}
                  </span>
                </span>
                <ChevronRight
                  className='text-muted-foreground size-4 shrink-0'
                  aria-hidden='true'
                />
              </button>
            )
          })}
        </div>
      </section>

      <ChangePasswordDialog
        open={dialogs.isOpen('password')}
        onOpenChange={(open) =>
          open ? dialogs.open('password') : dialogs.close('password')
        }
        username={props.profile.username}
      />
      <AccessTokenDialog
        open={dialogs.isOpen('token')}
        onOpenChange={(open) =>
          open ? dialogs.open('token') : dialogs.close('token')
        }
      />
      <DeleteAccountDialog
        open={dialogs.isOpen('delete')}
        onOpenChange={(open) =>
          open ? dialogs.open('delete') : dialogs.close('delete')
        }
        username={props.profile.username}
      />
    </>
  )
}
