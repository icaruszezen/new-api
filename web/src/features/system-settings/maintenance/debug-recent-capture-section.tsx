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
import { RotateCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { SettingsSwitchField } from '../components/settings-form-layout'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  getDebugCaptureRecent,
  type DebugCaptureRecord,
} from './debug-capture-api'
import { DebugCaptureRecords } from './debug-capture-records'

const OPTION_KEY = 'debug_setting.recent_capture_enabled'

type Props = {
  defaultEnabled: boolean
}

export function DebugRecentCaptureSection(props: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const [enabled, setEnabled] = useState(props.defaultEnabled)
  const [records, setRecords] = useState<DebugCaptureRecord[]>([])
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
      const res = await getDebugCaptureRecent()
      if (res.success && res.data) {
        setRecords(res.data.records)
        setLoaded(true)
      }
    } catch {
      toast.error(t('Failed to load records'))
    } finally {
      setRefreshing(false)
    }
  }, [t])

  return (
    <SettingsSection title={t('Recent Request Capture')}>
      <SettingsSwitchField
        checked={enabled}
        onCheckedChange={handleToggle}
        disabled={updateOption.isPending}
        label={t('Keep the most recent 20 requests')}
        description={t(
          'When enabled, the latest 20 requests are continuously retained in memory (full request body and upstream response). The list does not auto-refresh; click Refresh to load the most recent 20.'
        )}
      />

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
      </div>

      {loaded ? (
        <DebugCaptureRecords
          records={records}
          emptyMessage={t('No requests captured yet.')}
        />
      ) : (
        <div className='text-muted-foreground rounded-md border border-dashed p-6 text-center text-sm'>
          {t('Click Refresh to load the most recent requests.')}
        </div>
      )}
    </SettingsSection>
  )
}
