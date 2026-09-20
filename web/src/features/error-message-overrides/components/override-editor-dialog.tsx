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
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import { getChannelOptions } from '../api'
import {
  MATCH_SUBSTRING_MAX_LENGTH,
  OVERRIDE_FORM_DEFAULT_VALUES,
  OVERRIDE_SCOPE_ALL,
  OVERRIDE_SCOPE_CHANNEL,
  getOverrideFormSchema,
  transformOverrideToFormValues,
  type OverrideFormValues,
} from '../lib/override-form'
import { ALL_CHANNELS_ID, type ErrorMessageOverride } from '../types'

const OVERRIDE_FORM_ID = 'error-message-override-form'

type OverrideEditorDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  override: ErrorMessageOverride | null
  isSubmitting: boolean
  onSubmit: (values: OverrideFormValues) => void
}

export function OverrideEditorDialog(props: OverrideEditorDialogProps) {
  const { t } = useTranslation()
  const isUpdate = props.override !== null

  const form = useForm<OverrideFormValues>({
    resolver: zodResolver(getOverrideFormSchema(t)),
    defaultValues: OVERRIDE_FORM_DEFAULT_VALUES,
  })

  useEffect(() => {
    if (!props.open) return
    form.reset(
      props.override
        ? transformOverrideToFormValues(props.override)
        : OVERRIDE_FORM_DEFAULT_VALUES
    )
  }, [props.open, props.override, form])

  const scope = form.watch('scope')

  const { data: channels, isLoading: isLoadingChannels } = useQuery({
    queryKey: ['error-message-override-channel-options'],
    queryFn: getChannelOptions,
    enabled: props.open && scope === OVERRIDE_SCOPE_CHANNEL,
  })

  const channelComboboxOptions = useMemo(
    () =>
      (channels ?? []).map((channel) => ({
        value: String(channel.id),
        label: `#${channel.id} ${channel.name}`,
      })),
    [channels]
  )

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={isUpdate ? t('Edit Override Rule') : t('Add Override Rule')}
      description={t(
        'When an upstream error message contains the match text, the whole message returned to the user is replaced.'
      )}
      contentClassName='sm:max-w-xl'
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
            form={OVERRIDE_FORM_ID}
            disabled={props.isSubmitting}
          >
            {props.isSubmitting ? t('Saving...') : t('Save')}
          </Button>
        </>
      }
    >
      <Form {...form}>
        <form
          id={OVERRIDE_FORM_ID}
          onSubmit={form.handleSubmit(props.onSubmit)}
          className='space-y-4'
        >
          <FormField
            control={form.control}
            name='match_substring'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Match text')}</FormLabel>
                <FormControl>
                  <Input
                    {...field}
                    maxLength={MATCH_SUBSTRING_MAX_LENGTH}
                    placeholder={t('e.g. insufficient_quota')}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Case-insensitive substring match against the upstream error message.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='replacement_message'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Replacement message')}</FormLabel>
                <FormControl>
                  <Textarea
                    {...field}
                    rows={3}
                    placeholder={t('e.g. Service is busy, please retry later')}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Replaces the entire upstream error message. Logs keep the original message.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='scope'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Applies to')}</FormLabel>
                <Select
                  value={field.value}
                  onValueChange={(value) => {
                    if (value === null) return
                    field.onChange(value)
                    if (value === OVERRIDE_SCOPE_ALL) {
                      form.setValue('channel_id', ALL_CHANNELS_ID)
                    }
                  }}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      <SelectItem value={OVERRIDE_SCOPE_ALL}>
                        {t('All channels')}
                      </SelectItem>
                      <SelectItem value={OVERRIDE_SCOPE_CHANNEL}>
                        {t('A specific channel')}
                      </SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <FormDescription>
                  {t(
                    'Channel rules are matched before rules that apply to all channels.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {scope === OVERRIDE_SCOPE_CHANNEL && (
            <FormField
              control={form.control}
              name='channel_id'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Channel')}</FormLabel>
                  <FormControl>
                    <Combobox
                      options={channelComboboxOptions}
                      value={field.value > 0 ? String(field.value) : ''}
                      onValueChange={(value) =>
                        field.onChange(Number(value) || ALL_CHANNELS_ID)
                      }
                      placeholder={
                        isLoadingChannels
                          ? t('Loading...')
                          : t('Select a channel')
                      }
                      emptyText={t('No channels found')}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}

          <FormField
            control={form.control}
            name='priority'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Priority')}</FormLabel>
                <FormControl>
                  <Input
                    {...field}
                    type='number'
                    min={0}
                    onChange={(event) =>
                      field.onChange(
                        Number.parseInt(event.target.value, 10) || 0
                      )
                    }
                  />
                </FormControl>
                <FormDescription>
                  {t('Lower values are matched first within the same scope.')}
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
                    {t('Disabled rules are kept but never applied.')}
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
