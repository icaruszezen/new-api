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
import {
  ChevronRight,
  Key,
  Shield,
  Trash2,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import type { UserProfile } from '@/features/profile/types'
import { ChangePasswordDialog } from '@/features/security/components/dialogs/change-password-dialog'
import { DeleteAccountDialog } from '@/features/security/components/dialogs/delete-account-dialog'
import { useDialogs } from '@/hooks/use-dialog'
import { cn } from '@/lib/utils'

type NextProfileSecurityListProps = {
  profile: UserProfile | null
  loading: boolean
  onProfileUpdate: () => void
}

type DialogKey = 'password' | 'delete'

type SecurityAction = {
  key: DialogKey | 'tokens'
  icon: LucideIcon
  title: string
  description: string
  destructive?: boolean
}

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

  const hasPassword = props.profile.has_password !== false
  const actions: SecurityAction[] = [
    {
      key: 'password',
      icon: Shield,
      title: hasPassword ? t('Change Password') : t('Set Password'),
      description: hasPassword
        ? t('Update your password to keep your account secure')
        : t('Add a password after verifying your identity'),
    },
    {
      // Scoped access tokens are managed on the Security & Access page.
      key: 'tokens',
      icon: Key,
      title: t('Access tokens'),
      description: t('Create and manage scoped access tokens'),
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
            const rowClassName = cn(
              'hover:bg-muted/40 flex w-full items-center gap-3 px-5 py-3.5 text-left transition-colors',
              item.destructive && 'text-destructive'
            )
            const content = (
              <>
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
              </>
            )

            if (item.key === 'tokens') {
              return (
                <Link key={item.key} to='/security' className={rowClassName}>
                  {content}
                </Link>
              )
            }

            const dialogKey = item.key
            return (
              <button
                key={item.key}
                type='button'
                onClick={() => dialogs.open(dialogKey)}
                className={rowClassName}
              >
                {content}
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
        hasPassword={props.profile.has_password}
        onSuccess={props.onProfileUpdate}
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
