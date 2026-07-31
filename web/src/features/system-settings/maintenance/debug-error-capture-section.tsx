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
import { useCallback, useEffect, useState } from 'react'
import { RotateCw, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { Alert, AlertDescription } from '@/components/ui/alert'
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
import { SettingsSwitchField } from '../components/settings-form-layout'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  clearDebugCaptureErrors,
  getDebugCaptureErrors,
  type DebugCaptureRecord,
} from './debug-capture-api'
import { DebugCaptureRecords } from './debug-capture-records'

const OPTION_KEY = 'debug_setting.error_capture_enabled'

type Props = {
  defaultEnabled: boolean
}

export function DebugErrorCaptureSection(props: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const [enabled, setEnabled] = useState(props.defaultEnabled)
  const [records, setRecords] = useState<DebugCaptureRecord[]>([])
  const [total, setTotal] = useState(0)
  const [refreshing, setRefreshing] = useState(false)
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    setEnabled(props.defaultEnabled)
  }, [props.defaultEnabled])

  const handleToggle = async (next: boolean) => {
    setEnabled(next)
    try {
      await updateOption.mutateAsync({ key: OPTION_KEY, value: next })
    } catch {
      setEnabled(!next)
    }
  }

  const handleRefresh = useCallback(async () => {
    setRefreshing(true)
    try {
      const res = await getDebugCaptureErrors()
      if (res.success && res.data) {
        setRecords(res.data.records)
        setTotal(res.data.total)
        setLoaded(true)
      }
    } catch {
      toast.error(t('Failed to load records'))
    } finally {
      setRefreshing(false)
    }
  }, [t])

  const handleClear = async () => {
    try {
      const res = await clearDebugCaptureErrors()
      if (res.success) {
        setRecords([])
        setTotal(0)
        setLoaded(true)
        toast.success(t('Records cleared'))
      }
    } catch {
      toast.error(t('Cleanup failed'))
    }
  }

  return (
    <SettingsSection title={t('Error Request Capture')}>
      <SettingsSwitchField
        checked={enabled}
        onCheckedChange={handleToggle}
        disabled={updateOption.isPending}
        label={t('Capture failed and unbilled requests')}
        description={t(
          'When enabled, automatically retains the full request body and upstream response of every request that errors, or where the upstream returned no billing info (e.g. upstream timeout). No record limit.'
        )}
      />

      <Alert>
        <AlertDescription>
          {t(
            'Captured records are kept in memory without a count limit and contain user prompts and upstream responses. Keeping this enabled long-term may grow memory usage; clear records regularly.'
          )}
        </AlertDescription>
      </Alert>

      <div className='flex flex-wrap items-center gap-2'>
        <Button
          type='button'
          size='sm'
          variant='outline'
          onClick={handleRefresh}
          disabled={refreshing}
        >
          <RotateCw className={cn('size-3.5', refreshing && 'animate-spin')} />
          <span>{t('Refresh')}</span>
        </Button>

        <AlertDialog>
          <AlertDialogTrigger
            render={
              <Button
                type='button'
                size='sm'
                variant='outline'
                disabled={records.length === 0}
              >
                <Trash2 className='size-3.5' />
                <span>{t('Clear All')}</span>
              </Button>
            }
          />
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>{t('Clear all records?')}</AlertDialogTitle>
              <AlertDialogDescription>
                {t('This will permanently remove all captured error records.')}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
              <AlertDialogAction onClick={handleClear}>
                {t('Clear All')}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        {loaded ? (
          <span className='text-muted-foreground text-xs'>
            {t('Total')}: {total}
          </span>
        ) : null}
      </div>

      {loaded ? (
        <DebugCaptureRecords
          records={records}
          emptyMessage={t('No error requests captured yet.')}
        />
      ) : (
        <div className='text-muted-foreground rounded-md border border-dashed p-6 text-center text-sm'>
          {t('Click Refresh to load captured records.')}
        </div>
      )}
    </SettingsSection>
  )
}
