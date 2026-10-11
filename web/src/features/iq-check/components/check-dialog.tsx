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
import { Textarea } from '@/components/ui/textarea'
import { getGroupModels } from '@/features/channel-monitoring/api'
import { getLobeIcon } from '@/lib/lobe-icon'

import {
  MAX_CHECK_NAME_LENGTH,
  MAX_INTERVAL_MINUTES,
  MAX_MAX_TOKENS,
  MAX_PROMPT_LENGTH,
  MIN_INTERVAL_MINUTES,
} from '../constants'
import {
  CHECK_FORM_DEFAULT_VALUES,
  type CheckFormValues,
  checkToFormValues,
  countCharacters,
  getCheckFormSchema,
} from '../lib/forms'
import type { IqCheck } from '../types'

const CHECK_FORM_ID = 'iq-check-form'

type CheckDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Check being edited, or undefined when adding a new one. */
  editData?: IqCheck
  groups: string[]
  /** Resolves to whether the server accepted the save. */
  onSave: (values: CheckFormValues) => Promise<boolean>
}

export function CheckDialog(props: CheckDialogProps) {
  const { t } = useTranslation()
  const isEditMode = Boolean(props.editData)
  const schema = useMemo(() => getCheckFormSchema(t), [t])

  const form = useForm<CheckFormValues>({
    resolver: zodResolver(schema),
    defaultValues: CHECK_FORM_DEFAULT_VALUES,
  })

  const selectedGroup = form.watch('group')
  const promptLength = countCharacters(form.watch('prompt'))
  const iconPreview = form.watch('icon')

  // Reset only when the dialog opens or moves to another check. A config
  // refetch or a rolled-back save replaces the row object, and must not wipe
  // what the admin has typed.
  const editId = props.editData?.id
  useEffect(() => {
    if (!props.open) return
    form.reset(
      props.editData
        ? checkToFormValues(props.editData)
        : CHECK_FORM_DEFAULT_VALUES
    )
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editId, props.open, form])

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

  // The dialog stays open when the server rejects the save, so the admin can
  // fix the problem without retyping a long prompt.
  const handleSubmit = async (values: CheckFormValues) => {
    if (await props.onSave(values)) props.onOpenChange(false)
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={isEditMode ? t('Edit IQ check') : t('Add IQ check')}
      description={t(
        'Send a fixed prompt to one model in one group on a schedule and publish its answers.'
      )}
      contentClassName='sm:max-w-[640px]'
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
          <Button
            type='submit'
            form={CHECK_FORM_ID}
            disabled={form.formState.isSubmitting}
          >
            {isEditMode ? t('Update') : t('Add')}
          </Button>
        </>
      }
    >
      <Form {...form}>
        <form
          id={CHECK_FORM_ID}
          onSubmit={form.handleSubmit(handleSubmit)}
          className='space-y-4'
          noValidate
        >
          <FormField
            control={form.control}
            name='name'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Display name')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={t('Name shown on the IQ check page')}
                    maxLength={MAX_CHECK_NAME_LENGTH}
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Visitors only see this name and the model; the group stays private.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className='grid gap-4 sm:grid-cols-2'>
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
                        // The model list is scoped to the group, so any
                        // previous selection may no longer be routable.
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
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <FormField
            control={form.control}
            name='prompt'
            render={({ field }) => (
              <FormItem>
                <div className='flex items-center justify-between gap-2'>
                  <FormLabel>{t('Prompt')}</FormLabel>
                  <span
                    className='text-muted-foreground text-xs tabular-nums'
                    aria-live='polite'
                  >
                    {promptLength} / {MAX_PROMPT_LENGTH}
                  </span>
                </div>
                <FormControl>
                  <Textarea
                    rows={6}
                    className='max-h-72 min-h-32 font-mono text-xs md:text-xs'
                    placeholder={t(
                      'e.g. "How many letters r are in strawberry? Reply with the number only." or "Write a single-file HTML page that draws a working analog clock."'
                    )}
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Ask for plain text or for an HTML page; HTML in the answer is rendered as a live preview on the IQ check page.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className='grid gap-4 sm:grid-cols-2'>
            <FormField
              control={form.control}
              name='interval_minutes'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Check interval (minutes)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      inputMode='numeric'
                      min={MIN_INTERVAL_MINUTES}
                      max={MAX_INTERVAL_MINUTES}
                      step={1}
                      name={field.name}
                      ref={field.ref}
                      onBlur={field.onBlur}
                      value={Number.isNaN(field.value) ? '' : field.value}
                      onChange={(event) =>
                        field.onChange(event.target.valueAsNumber)
                      }
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Between {{min}} minutes and {{days}} days.', {
                      min: MIN_INTERVAL_MINUTES,
                      days: MAX_INTERVAL_MINUTES / (24 * 60),
                    })}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='max_tokens'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Max output tokens')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      inputMode='numeric'
                      min={0}
                      max={MAX_MAX_TOKENS}
                      step={1}
                      name={field.name}
                      ref={field.ref}
                      onBlur={field.onBlur}
                      value={Number.isNaN(field.value) ? '' : field.value}
                      onChange={(event) =>
                        field.onChange(event.target.valueAsNumber)
                      }
                    />
                  </FormControl>
                  <FormDescription>
                    {t('0 = do not send a limit')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

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
                    <Input
                      placeholder={t('Auto-detected from model')}
                      {...field}
                    />
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
                    {t('Disabled checks are hidden and stop running.')}
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
