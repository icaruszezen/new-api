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

import { getChannelOptions } from '../api'
import {
  PROMPT_TOKENS_THRESHOLD_MAX,
  PROMPT_TOKENS_THRESHOLD_MIN,
  RULE_FORM_DEFAULT_VALUES,
  UNSELECTED_CHANNEL_ID,
  getRuleFormSchema,
  transformRuleToFormValues,
  type RuleFormValues,
} from '../lib/rule-form'
import type { BillingCorrectionRule } from '../types'

const RULE_FORM_ID = 'billing-correction-rule-form'

type RuleEditorDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  rule: BillingCorrectionRule | null
  isSubmitting: boolean
  onSubmit: (values: RuleFormValues) => void
}

export function RuleEditorDialog(props: RuleEditorDialogProps) {
  const { t } = useTranslation()
  const isUpdate = props.rule !== null

  const form = useForm<RuleFormValues>({
    resolver: zodResolver(getRuleFormSchema(t)),
    defaultValues: RULE_FORM_DEFAULT_VALUES,
  })

  useEffect(() => {
    if (!props.open) return
    form.reset(
      props.rule
        ? transformRuleToFormValues(props.rule)
        : RULE_FORM_DEFAULT_VALUES
    )
  }, [props.open, props.rule, form])

  const { data: channels, isLoading: isLoadingChannels } = useQuery({
    queryKey: ['billing-correction-channel-options'],
    queryFn: getChannelOptions,
    enabled: props.open,
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
      title={isUpdate ? t('Edit Correction Rule') : t('Add Correction Rule')}
      description={t(
        'Waives the charge when a request on this channel exceeds the input token threshold, produces no output, and the upstream returns no usage.'
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
            form={RULE_FORM_ID}
            disabled={props.isSubmitting}
          >
            {props.isSubmitting ? t('Saving...') : t('Save')}
          </Button>
        </>
      }
    >
      <Form {...form}>
        <form
          id={RULE_FORM_ID}
          noValidate
          onSubmit={form.handleSubmit(props.onSubmit)}
          className='space-y-4'
        >
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
                      field.onChange(Number(value) || UNSELECTED_CHANNEL_ID)
                    }
                    placeholder={
                      isLoadingChannels
                        ? t('Loading...')
                        : t('Select a channel')
                    }
                    emptyText={t('No channels found')}
                  />
                </FormControl>
                <FormDescription>
                  {t('Each channel can have at most one correction rule.')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='prompt_tokens_threshold'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Input token threshold')}</FormLabel>
                <FormControl>
                  <Input
                    type='number'
                    inputMode='numeric'
                    min={PROMPT_TOKENS_THRESHOLD_MIN}
                    max={PROMPT_TOKENS_THRESHOLD_MAX}
                    step={1}
                    name={field.name}
                    ref={field.ref}
                    onBlur={field.onBlur}
                    value={Number.isFinite(field.value) ? field.value : ''}
                    onChange={(event) =>
                      field.onChange(event.target.valueAsNumber)
                    }
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Only requests whose input tokens are strictly greater than this value can be waived.'
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
