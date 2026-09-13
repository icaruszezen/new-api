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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { updateSystemOption } from '@/features/system-settings/api'
import { useAuthStore } from '@/stores/auth-store'

import { getFirstTokenErrorStat } from '../api'
import { canEditFirstTokenErrorSettings } from '../lib/access'

export function FirstTokenErrorSettingsBar() {
  const { t } = useTranslation()
  const role = useAuthStore((state) => state.auth.user?.role)
  const canEdit = canEditFirstTokenErrorSettings(role)
  const queryClient = useQueryClient()
  const statQuery = useQuery({
    queryKey: ['first-token-errors-stat'],
    queryFn: async () => {
      const result = await getFirstTokenErrorStat()
      if (!result.success || !result.data) {
        toast.error(result.message || t('Failed to load'))
        throw new Error(result.message || t('Failed to load'))
      }
      return result
    },
  })
  const stat = statQuery.data?.data
  const update = useMutation({
    mutationFn: (request: { key: string; value: string | boolean | number }) =>
      updateSystemOption(request),
    onSuccess: (data) => {
      if (!data.success) {
        toast.error(data.message || t('Failed to update setting'))
        return
      }
      toast.success(t('Setting updated successfully'))
      void queryClient.invalidateQueries({ queryKey: ['first-token-errors-stat'] })
      void queryClient.invalidateQueries({ queryKey: ['first-token-errors'] })
      void queryClient.invalidateQueries({ queryKey: ['system-options'] })
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to update setting'))
    },
  })

  if (statQuery.isError) {
    return (
      <div
        data-slot='first-token-errors-settings'
        className='text-muted-foreground text-sm'
      >
        {t('Failed to load')}
      </div>
    )
  }

  if (!stat) {
    return (
      <div
        data-slot='first-token-errors-settings'
        className='text-muted-foreground text-sm'
      >
        {t('Loading...')}
      </div>
    )
  }

  return (
    <div
      data-slot='first-token-errors-settings'
      className='flex flex-wrap items-start gap-x-6 gap-y-3 text-sm'
    >
      {!canEdit ? (
        <p className='text-muted-foreground w-full text-xs'>
          {t('Only super admins can change these settings.')}
        </p>
      ) : null}
      <label className='flex max-w-64 items-center gap-2'>
        <Switch
          checked={stat.correction_enabled}
          disabled={!canEdit || update.isPending}
          onCheckedChange={(checked) =>
            update.mutate({
              key: 'FirstTokenErrorCorrectionEnabled',
              value: checked,
            })
          }
        />
        <span>
          <span className='font-medium'>
            {t('Correct first-token billing errors')}
          </span>
          <span className='text-muted-foreground block text-xs'>
            {t(
              'When off, 1-token upstream errors are billed as usual. Default on.'
            )}
          </span>
        </span>
      </label>
      <label className='flex max-w-64 items-center gap-2'>
        <Switch
          checked={stat.treat_all_output_one_enabled}
          disabled={!canEdit || update.isPending}
          onCheckedChange={(checked) =>
            update.mutate({
              key: 'FirstTokenErrorTreatAllOutputOneEnabled',
              value: checked,
            })
          }
        />
        <span>
          <span className='font-medium'>
            {t('Treat every 1-token output as an upstream error')}
          </span>
          <span className='text-muted-foreground block text-xs'>
            {t(
              'When on, a genuine completed response with output_tokens=1 is also corrected. Default off.'
            )}
          </span>
        </span>
      </label>
      <label className='flex max-w-64 items-center gap-2'>
        <Switch
          checked={stat.log_enabled}
          disabled={!canEdit || update.isPending}
          onCheckedChange={(checked) =>
            update.mutate({
              key: 'FirstTokenErrorLogEnabled',
              value: checked,
            })
          }
        />
        <span>
          <span className='font-medium'>
            {t('Write first-token error records')}
          </span>
          <span className='text-muted-foreground block text-xs'>
            {t(
              'When off, billing is still corrected and users still see an error log. New admin snapshots stop.'
            )}
          </span>
        </span>
      </label>
      <div className='flex max-w-48 flex-col gap-1'>
        <Label htmlFor='first-token-error-max-keep'>
          {t('Maximum records to keep')}
        </Label>
        <Input
          id='first-token-error-max-keep'
          type='number'
          min={1}
          max={100000}
          disabled={!canEdit || update.isPending}
          defaultValue={stat.log_max_keep}
          key={stat.log_max_keep}
          onBlur={(event) => {
            const next = Number(event.target.value)
            if (!Number.isInteger(next) || next === stat.log_max_keep) return
            if (next < 1 || next > 100000) {
              toast.error(t('Enter a number from 1 to 100000.'))
              return
            }
            update.mutate({
              key: 'FirstTokenErrorLogMaxKeep',
              value: next,
            })
          }}
        />
        <span className='text-muted-foreground text-xs'>
          {t('Older rows are deleted automatically. Default 5000.')}
        </span>
      </div>
      <label className='flex max-w-72 items-center gap-2'>
        <Switch
          checked={stat.body_capture_enabled}
          disabled={!canEdit || update.isPending}
          onCheckedChange={(checked) =>
            update.mutate({
              key: 'FirstTokenErrorBodyCaptureEnabled',
              value: checked,
            })
          }
        />
        <span>
          <span className='font-medium'>
            {t('Store the last 10 request bodies')}
          </span>
          <span className='text-muted-foreground block text-xs'>
            {t(
              'Bodies include user conversations. Use only for debugging. {{count}}/10 stored globally.',
              { count: stat.body_count }
            )}
          </span>
        </span>
      </label>
    </div>
  )
}
