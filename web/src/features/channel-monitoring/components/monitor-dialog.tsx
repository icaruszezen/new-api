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
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Combobox } from '@/components/ui/combobox'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { getLobeIcon } from '@/lib/lobe-icon'

import { getGroupModels } from '../api'
import { MAX_MONITOR_NAME_LENGTH, monitorPairKey } from '../constants'
import type { ChannelMonitor } from '../types'

const MONITOR_FORM_ID = 'channel-monitor-form'

const monitorFormSchema = z.object({
  name: z.string().trim().min(1).max(MAX_MONITOR_NAME_LENGTH),
  group: z.string().trim().min(1),
  model: z.string().trim().min(1),
  icon: z.string().trim().max(128),
  enabled: z.boolean(),
})

export type MonitorFormValues = z.infer<typeof monitorFormSchema>

type MonitorDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Monitor being edited, or undefined when adding a new one. */
  editData?: ChannelMonitor
  groups: string[]
  /** Group + model pairs already taken by other monitors. */
  takenPairs: string[]
  onSave: (values: MonitorFormValues) => void
}

const EMPTY_VALUES: MonitorFormValues = {
  name: '',
  group: '',
  model: '',
  icon: '',
  enabled: true,
}

export function MonitorDialog(props: MonitorDialogProps) {
  const { t } = useTranslation()
  const isEditMode = Boolean(props.editData)

  const form = useForm<MonitorFormValues>({
    resolver: zodResolver(monitorFormSchema),
    defaultValues: EMPTY_VALUES,
  })

  const selectedGroup = form.watch('group')

  useEffect(() => {
    if (!props.open) return
    if (props.editData) {
      form.reset({
        name: props.editData.name,
        group: props.editData.group,
        model: props.editData.model,
        icon: props.editData.icon ?? '',
        enabled: props.editData.enabled,
      })
      return
    }
    form.reset(EMPTY_VALUES)
  }, [props.editData, props.open, form])

  const groupModelsQuery = useQuery({
    queryKey: ['group-models', selectedGroup],
    queryFn: () => getGroupModels(selectedGroup),
    enabled: props.open && selectedGroup !== '',
    staleTime: 60_000,
  })

  const groupOptions = useMemo(
    () => props.groups.map((group) => ({ value: group, label: group })),
    [props.groups]
  )

  const modelOptions = useMemo(
    () =>
      (groupModelsQuery.data ?? []).map((modelName) => ({
        value: modelName,
        label: modelName,
      })),
    [groupModelsQuery.data]
  )

  const handleSubmit = (values: MonitorFormValues) => {
    if (props.takenPairs.includes(monitorPairKey(values.group, values.model))) {
      form.setError('model', {
        message: t('This group and model pair is already monitored.'),
      })
      return
    }
    props.onSave(values)
    props.onOpenChange(false)
  }

  const iconPreview = form.watch('icon')

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={isEditMode ? t('Edit monitor') : t('Add monitor')}
      description={t(
        'Monitor one model inside one group and control how it appears on the public status page.'
      )}
      contentClassName='sm:max-w-[520px]'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button type='submit' form={MONITOR_FORM_ID}>
            {isEditMode ? t('Update') : t('Add')}
          </Button>
        </>
      }
    >
      <Form {...form}>
        <form
          id={MONITOR_FORM_ID}
          onSubmit={form.handleSubmit(handleSubmit)}
          className='space-y-4'
        >
          <FormField
            control={form.control}
            name='name'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Display name')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={t('Name shown on the status page')}
                    maxLength={MAX_MONITOR_NAME_LENGTH}
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Visitors only see this name; the underlying group stays private.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='group'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Group')}</FormLabel>
                <FormControl>
                  <Combobox
                    options={groupOptions}
                    value={field.value}
                    onValueChange={(value) => {
                      if (value === null) return
                      field.onChange(value)
                      // The model list is scoped to the group, so any previous
                      // selection may no longer be routable.
                      form.setValue('model', '', { shouldDirty: true })
                    }}
                    placeholder={t('Select a group')}
                    searchPlaceholder={t('Search groups...')}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='model'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Model')}</FormLabel>
                <FormControl>
                  <Combobox
                    options={modelOptions}
                    value={field.value}
                    onValueChange={(value) => {
                      if (value === null) return
                      field.onChange(value)
                    }}
                    searchPlaceholder={
                      selectedGroup === ''
                        ? t('Select a group first')
                        : t('Search models...')
                    }
                    emptyText={t('No models available in this group')}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Only streaming requests to this model in this group are recorded.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='icon'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Icon override')}</FormLabel>
                <div className='flex items-center gap-2'>
                  <span
                    aria-hidden='true'
                    className='flex shrink-0 items-center'
                  >
                    {getLobeIcon(iconPreview, 20)}
                  </span>
                  <FormControl>
                    <Input placeholder={t('Auto-detected from model')} {...field} />
                  </FormControl>
                </div>
                <FormDescription>
                  {t(
                    'Leave empty to derive the provider icon from the model name.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='enabled'
            render={({ field }) => (
              <FormItem className='flex flex-row items-center justify-between gap-4'>
                <div className='space-y-1'>
                  <FormLabel>{t('Enabled')}</FormLabel>
                  <FormDescription>
                    {t('Disabled monitors are hidden and stop being probed.')}
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </FormItem>
            )}
          />
        </form>
      </Form>
    </Dialog>
  )
}
