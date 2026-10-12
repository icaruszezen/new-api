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
import { useMutation, useQuery } from '@tanstack/react-query'
import { useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

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
import { handleServerError } from '@/lib/handle-server-error'

import { getChannelOptions, testBillingCorrectionUpstream } from '../api'
import {
  PROMPT_TOKENS_THRESHOLD_MAX,
  PROMPT_TOKENS_THRESHOLD_MIN,
  RULE_FORM_DEFAULT_VALUES,
  UNSELECTED_CHANNEL_ID,
  getRuleFormSchema,
  transformFormValuesToPayload,
  transformRuleToFormValues,
  type RuleFormValues,
} from '../lib/rule-form'
import type {
  BillingCorrectionRule,
  BillingCorrectionUpstreamTestPayload,
} from '../types'

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

  const upstreamTokenSaved = props.rule?.upstream_access_token_set === true
  const form = useForm<RuleFormValues>({
    resolver: zodResolver(getRuleFormSchema(t, { upstreamTokenSaved })),
    defaultValues: RULE_FORM_DEFAULT_VALUES,
  })
  const upstreamCheckEnabled = form.watch('upstream_check_enabled')

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

  const testMutation = useMutation({
    mutationFn: (payload: BillingCorrectionUpstreamTestPayload) =>
      testBillingCorrectionUpstream(payload),
    onSuccess: (response) => {
      if (!response.success) {
        toast.error(response.message || t('Connection failed'))
        return
      }
      toast.success(
        t('Connected to the upstream as {{username}}', {
          username: response.data.username,
        })
      )
    },
    onError: handleServerError,
  })

  const testUpstream = async () => {
    const valid = await form.trigger([
      'channel_id',
      'upstream_base_url',
      'upstream_user_id',
      'upstream_access_token',
    ])
    if (!valid) return
    const payload = transformFormValuesToPayload(form.getValues())
    testMutation.mutate({
      rule_id: props.rule?.id,
      channel_id: payload.channel_id,
      upstream_base_url: payload.upstream_base_url,
      upstream_user_id: payload.upstream_user_id,
      upstream_access_token: payload.upstream_access_token,
    })
  }

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

          <div className='space-y-4 rounded-lg border p-3'>
            <FormField
              control={form.control}
              name='upstream_check_enabled'
              render={({ field }) => (
                <FormItem className='flex flex-row items-center justify-between gap-4'>
                  <div className='space-y-1'>
                    <FormLabel>
                      {t(
                        'Check charges on the upstream new-api (experimental)'
                      )}
                    </FormLabel>
                    <FormDescription>
                      {t(
                        'For channels whose upstream is also new-api: looks up zero-output requests in the upstream usage logs by the upstream request ID to see whether the upstream charged them.'
                      )}
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

            {upstreamCheckEnabled && (
              <>
                <FormField
                  control={form.control}
                  name='upstream_base_url'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Upstream address')}</FormLabel>
                      <FormControl>
                        <Input
                          placeholder='https://upstream.example.com'
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>
                        {t('Leave empty to use the channel Base URL.')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='upstream_user_id'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Upstream user ID')}</FormLabel>
                      <FormControl>
                        <Input inputMode='numeric' {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name='upstream_access_token'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Upstream system access token')}</FormLabel>
                      <FormControl>
                        <Input
                          type='password'
                          autoComplete='off'
                          placeholder={
                            upstreamTokenSaved
                              ? t('Saved. Leave empty to keep it.')
                              : ''
                          }
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          "Generate it under personal settings on the upstream. It is used only to read that user's usage logs and is never shown again."
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  disabled={testMutation.isPending}
                  onClick={() => void testUpstream()}
                >
                  {testMutation.isPending
                    ? t('Testing...')
                    : t('Test connection')}
                </Button>

                <FormField
                  control={form.control}
                  name='auto_refund'
                  render={({ field }) => (
                    <FormItem className='flex flex-row items-center justify-between gap-4'>
                      <div className='space-y-1'>
                        <FormLabel>{t('Return quota automatically')}</FormLabel>
                        <FormDescription>
                          {t(
                            'Zero-output requests made after the check is turned on are checked in the background; when the upstream did not charge one, its quota goes straight back to the user. Records checked by hand are never refunded automatically.'
                          )}
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
              </>
            )}
          </div>
        </form>
      </Form>
    </Dialog>
  )
}
