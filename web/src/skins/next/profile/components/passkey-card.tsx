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
import { AlertTriangle, Loader2, ShieldAlert } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { usePasskeyManagement } from '@/features/auth/passkey'
import {
  SecureVerificationDialog,
  useSecureVerification,
  type VerificationMethod,
  type VerificationMethods,
} from '@/features/auth/secure-verification'
import dayjs from '@/lib/dayjs'

import { NextStatusPill } from './status-pill'

type NextProfilePasskeyCardProps = {
  loading: boolean
}

export function NextProfilePasskeyCard(props: NextProfilePasskeyCardProps) {
  const { t } = useTranslation()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [restrictedMethod, setRestrictedMethod] =
    useState<VerificationMethod | null>(null)

  const {
    status,
    loading,
    registering,
    removing,
    supported,
    enabled,
    lastUsed,
    register,
    remove,
  } = usePasskeyManagement()

  const {
    open: verificationOpen,
    setOpen: setVerificationOpen,
    methods: verificationMethods,
    state: verificationState,
    startVerification,
    executeVerification,
    cancel: cancelVerification,
    setCode,
    switchMethod,
    fetchVerificationMethods,
  } = useSecureVerification({
    onSuccess: () => {
      setRestrictedMethod(null)
    },
  })

  const dialogMethods = useMemo<VerificationMethods>(() => {
    if (!restrictedMethod) return verificationMethods
    return {
      ...verificationMethods,
      has2FA: restrictedMethod === '2fa' && verificationMethods.has2FA,
      hasPasskey:
        restrictedMethod === 'passkey' && verificationMethods.hasPasskey,
    }
  }, [restrictedMethod, verificationMethods])

  const handleRegister = useCallback(async () => {
    if (!supported) {
      toast.info(t('This device does not support Passkey'))
      return
    }

    const methods = await fetchVerificationMethods()
    if (!methods.has2FA) {
      await register()
      return
    }

    setRestrictedMethod('2fa')
    await startVerification(register, {
      scope: 'passkey.register',
      preferredMethod: '2fa',
      title: t('Security verification'),
      description: t(
        'Confirm your identity with Two-factor Authentication before registering a Passkey.'
      ),
    })
  }, [fetchVerificationMethods, register, startVerification, supported, t])

  const handleRemove = useCallback(async () => {
    const methods = await fetchVerificationMethods()
    let required: VerificationMethod | null = null
    if (methods.has2FA) {
      required = '2fa'
    } else if (methods.hasPasskey) {
      required = 'passkey'
    }

    if (!required) {
      toast.error(
        t(
          'Please enable Two-factor Authentication or Passkey before proceeding'
        )
      )
      return
    }

    if (required === 'passkey' && !methods.passkeySupported) {
      toast.info(t('This device does not support Passkey'))
      return
    }

    setConfirmOpen(false)
    setRestrictedMethod(required)
    await startVerification(remove, {
      scope: 'passkey.delete',
      preferredMethod: required,
      title: t('Security verification'),
      description: t(
        'Confirm your identity before removing this Passkey from your account.'
      ),
    })
  }, [fetchVerificationMethods, remove, startVerification, t])

  const handleVerificationCancel = useCallback(() => {
    setRestrictedMethod(null)
    cancelVerification()
  }, [cancelVerification])

  const handleVerificationOpenChange = useCallback(
    (next: boolean) => {
      if (!next) {
        setRestrictedMethod(null)
      }
      setVerificationOpen(next)
    },
    [setVerificationOpen]
  )

  const handleDialogVerify = useCallback(
    async (method: VerificationMethod, code?: string) => {
      try {
        await executeVerification(method, code)
      } catch {
        // Errors are already surfaced by useSecureVerification via toast.
      }
    },
    [executeVerification]
  )

  if (props.loading || loading) {
    return (
      <section
        data-slot='next-profile-passkey'
        className='bg-card overflow-hidden rounded-xl border px-5 py-5'
      >
        <Skeleton className='h-6 w-28' />
        <Skeleton className='mt-2 h-4 w-56' />
        <Skeleton className='mt-4 h-9 w-32' />
      </section>
    )
  }

  const formattedLastUsed =
    lastUsed && !Number.isNaN(Date.parse(lastUsed))
      ? dayjs(lastUsed).fromNow()
      : t('Not used yet')
  const showUnsupportedNotice = !supported && !enabled
  let backupLabel: string | null = null
  if (status?.backup_eligible !== undefined) {
    backupLabel = t('No backup')
    if (status.backup_eligible) {
      backupLabel = status.backup_state ? t('Backed up') : t('Not backed up')
    }
  }

  return (
    <>
      <section
        data-slot='next-profile-passkey'
        className='bg-card overflow-hidden rounded-xl border px-5 py-5'
      >
        <div className='flex items-start justify-between gap-3'>
          <h2 className='text-lg font-medium tracking-tight'>
            {t('Passkey Login')}
          </h2>
          <div className='flex flex-wrap justify-end gap-1.5'>
            <NextStatusPill>
              {enabled ? t('Enabled') : t('Disabled')}
            </NextStatusPill>
            {backupLabel && <NextStatusPill>{backupLabel}</NextStatusPill>}
          </div>
        </div>
        <p className='text-muted-foreground mt-1.5 text-sm'>
          {enabled
            ? `${t('Last used:')} ${formattedLastUsed}`
            : t('Use Passkey to sign in without entering your password.')}
        </p>

        {!enabled && (
          <Button
            className='mt-4'
            onClick={handleRegister}
            disabled={!supported || registering}
          >
            {registering && (
              <Loader2 className='size-4 animate-spin' aria-hidden='true' />
            )}
            {t('Enable Passkey')}
          </Button>
        )}

        {enabled && (
          <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
            <AlertDialogTrigger
              render={
                <Button
                  variant='destructive'
                  className='mt-4'
                  disabled={removing}
                />
              }
            >
              {removing ? (
                <Loader2 className='size-4 animate-spin' aria-hidden='true' />
              ) : (
                <AlertTriangle className='size-4' aria-hidden='true' />
              )}
              {t('Remove Passkey')}
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{t('Remove Passkey?')}</AlertDialogTitle>
                <AlertDialogDescription>
                  {t(
                    'Removing Passkey will require you to sign in with your password next time. You can re-register anytime.'
                  )}
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel disabled={removing}>
                  {t('Cancel')}
                </AlertDialogCancel>
                <AlertDialogAction
                  variant='destructive'
                  disabled={removing}
                  onClick={(event) => {
                    event.preventDefault()
                    handleRemove()
                  }}
                >
                  {t('Remove')}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        )}

        {showUnsupportedNotice && (
          <div className='bg-muted/60 text-muted-foreground mt-4 flex items-start gap-3 rounded-md p-3 text-sm'>
            <ShieldAlert
              className='mt-0.5 size-4 shrink-0'
              aria-hidden='true'
            />
            <div>
              <p className='text-foreground font-medium'>
                {t('Passkey not supported on this device')}
              </p>
              <p>
                {t(
                  'Use a compatible browser or device with biometric authentication or a security key to register a Passkey.'
                )}
              </p>
            </div>
          </div>
        )}
      </section>

      <SecureVerificationDialog
        open={verificationOpen}
        onOpenChange={handleVerificationOpenChange}
        methods={dialogMethods}
        state={verificationState}
        onVerify={handleDialogVerify}
        onCancel={handleVerificationCancel}
        onCodeChange={setCode}
        onMethodChange={switchMethod}
      />
    </>
  )
}
