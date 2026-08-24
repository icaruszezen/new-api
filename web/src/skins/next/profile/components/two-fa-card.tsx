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
import { AlertTriangle, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { TwoFABackupDialog } from '@/features/profile/components/dialogs/two-fa-backup-dialog'
import { TwoFADisableDialog } from '@/features/profile/components/dialogs/two-fa-disable-dialog'
import { TwoFASetupDialog } from '@/features/profile/components/dialogs/two-fa-setup-dialog'
import { useTwoFA } from '@/features/profile/hooks'
import { useDialogs } from '@/hooks/use-dialog'

import { NextStatusPill } from './status-pill'

type NextProfileTwoFACardProps = {
  loading: boolean
}

type DialogKey = 'setup' | 'disable' | 'backup'

export function NextProfileTwoFACard(props: NextProfileTwoFACardProps) {
  const { t } = useTranslation()
  const { status, loading, refetch } = useTwoFA(!props.loading)
  const dialogs = useDialogs<DialogKey>()

  if (props.loading || loading) {
    return (
      <section
        data-slot='next-profile-two-fa'
        className='bg-card overflow-hidden rounded-xl border px-5 py-5'
      >
        <Skeleton className='h-6 w-32' />
        <Skeleton className='mt-2 h-4 w-56' />
        <Skeleton className='mt-4 h-9 w-20' />
      </section>
    )
  }

  let twoFaStatusLabel = t('Disabled')
  if (status.locked) {
    twoFaStatusLabel = t('Locked')
  } else if (status.enabled) {
    twoFaStatusLabel = t('Enabled')
  }

  return (
    <>
      <section
        data-slot='next-profile-two-fa'
        className='bg-card overflow-hidden rounded-xl border px-5 py-5'
      >
        <div className='flex items-start justify-between gap-3'>
          <h2 className='text-lg font-medium tracking-tight'>
            {t('Two-Step Verification')}
          </h2>
          <NextStatusPill tone={status.locked ? 'danger' : 'default'}>
            {twoFaStatusLabel}
          </NextStatusPill>
        </div>
        <p className='text-muted-foreground mt-1.5 text-sm'>
          {status.enabled
            ? t('Backup codes remaining: {{count}}', {
                count: status.backup_codes_remaining,
              })
            : t('Add an extra layer of security to your account')}
        </p>

        {!status.enabled && (
          <Button className='mt-4' onClick={() => dialogs.open('setup')}>
            {t('Enable')}
          </Button>
        )}

        {status.enabled && (
          <div className='mt-4 flex flex-col gap-2'>
            <Button variant='outline' onClick={() => dialogs.open('backup')}>
              <RefreshCw className='size-4' aria-hidden='true' />
              {t('Regenerate Backup Codes')}
            </Button>
            <Button
              variant='destructive'
              onClick={() => dialogs.open('disable')}
            >
              <AlertTriangle className='size-4' aria-hidden='true' />
              {t('Disable 2FA')}
            </Button>
          </div>
        )}
      </section>

      <TwoFASetupDialog
        open={dialogs.isOpen('setup')}
        onOpenChange={(open) =>
          open ? dialogs.open('setup') : dialogs.close('setup')
        }
        onSuccess={refetch}
      />
      <TwoFADisableDialog
        open={dialogs.isOpen('disable')}
        onOpenChange={(open) =>
          open ? dialogs.open('disable') : dialogs.close('disable')
        }
        onSuccess={refetch}
      />
      <TwoFABackupDialog
        open={dialogs.isOpen('backup')}
        onOpenChange={(open) =>
          open ? dialogs.open('backup') : dialogs.close('backup')
        }
        onSuccess={refetch}
      />
    </>
  )
}
