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
import { useEffect, useState } from 'react'
import type { SubmitErrorHandler } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { DateTimePicker } from '@/components/datetime-picker'
import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { ApiKeyGroupCombobox } from '@/features/keys/components/api-key-group-combobox'
import { useApiKeys } from '@/features/keys/components/api-keys-provider'
import { AutoGroupOrderEditor } from '@/features/keys/components/auto-group-order-editor'
import { useApiKeyMutateForm } from '@/features/keys/hooks/use-api-key-mutate-form'
import type { ApiKeyFormValues } from '@/features/keys/lib'

const BASIC_TAB = 'basic'
const QUOTA_TAB = 'quota'
const ADVANCED_TAB = 'advanced'

export function NextApiKeyDialog() {
  const { t } = useTranslation()
  const { open, setOpen, currentRow } = useApiKeys()
  const isOpen = open === 'create' || open === 'update'
  const [tab, setTab] = useState(BASIC_TAB)
  const mutate = useApiKeyMutateForm({
    open: isOpen,
    onOpenChange: (nextOpen) => {
      if (!nextOpen) setOpen(null)
    },
    currentRow: open === 'update' ? currentRow || undefined : undefined,
  })
  const expiredTime = mutate.form.watch('expired_time')

  useEffect(() => {
    if (isOpen) setTab(BASIC_TAB)
  }, [isOpen])

  const onInvalid: SubmitErrorHandler<ApiKeyFormValues> = (errors) => {
    if (
      errors.remain_quota_dollars ||
      errors.expired_time ||
      errors.tokenCount
    ) {
      setTab(QUOTA_TAB)
    } else if (errors.model_limits || errors.allow_ips) {
      setTab(ADVANCED_TAB)
    } else {
      setTab(BASIC_TAB)
    }
    mutate.onInvalid(errors)
  }

  const handleOpenChange = (nextOpen: boolean) => {
    mutate.handleOpenChange(nextOpen)
    if (!nextOpen) setTab(BASIC_TAB)
  }

  let submitLabel = t('Create key')
  if (mutate.isSubmitting) {
    submitLabel = t('Saving...')
  } else if (mutate.isUpdate) {
    submitLabel = t('Save changes')
  }

  return (
    <Dialog open={isOpen} onOpenChange={handleOpenChange}>
      <DialogContent className='flex max-h-[min(85vh,40rem)] w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-xl'>
        <div
          data-slot='next-api-key-dialog'
          className='flex min-h-0 flex-1 flex-col'
        >
          <DialogHeader className='px-5 pt-5 pb-3'>
            <DialogTitle className='text-lg tracking-tight'>
              {mutate.isUpdate ? t('Update API Key') : t('Create API Key')}
            </DialogTitle>
            <DialogDescription>
              {mutate.isUpdate
                ? t(
                    'Update the quota, expiration, and access limits for this key.'
                  )
                : t('Set name, quota, and access limits step by step.')}
            </DialogDescription>
          </DialogHeader>

          <Form {...mutate.form}>
            <form
              id='next-api-key-form'
              onSubmit={mutate.form.handleSubmit(mutate.onSubmit, onInvalid)}
              aria-busy={!mutate.isFormInitialized}
              inert={
                !mutate.isFormInitialized || mutate.isSubmitting
                  ? true
                  : undefined
              }
              className='flex min-h-0 flex-1 flex-col'
            >
              <Tabs
                value={tab}
                onValueChange={setTab}
                className='flex min-h-0 flex-1 flex-col gap-0 px-5'
              >
                <TabsList
                  variant='line'
                  className='w-full shrink-0 justify-start rounded-none border-b pb-0 group-data-horizontal/tabs:h-auto'
                >
                  <TabsTrigger
                    value={BASIC_TAB}
                    className='flex-none px-1 pb-2.5'
                  >
                    {t('Basic Information')}
                  </TabsTrigger>
                  <TabsTrigger
                    value={QUOTA_TAB}
                    className='flex-none px-1 pb-2.5'
                  >
                    {t('Quota Settings')}
                  </TabsTrigger>
                  <TabsTrigger
                    value={ADVANCED_TAB}
                    className='flex-none px-1 pb-2.5'
                  >
                    {t('Advanced Settings')}
                  </TabsTrigger>
                </TabsList>

                <div className='min-h-0 flex-1 overflow-y-auto py-4'>
                  <TabsContent value={BASIC_TAB} className='mt-0 space-y-4'>
                    <FormField
                      control={mutate.form.control}
                      name='name'
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('Name')}</FormLabel>
                          <FormControl>
                            <Input {...field} placeholder={t('Enter a name')} />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={mutate.form.control}
                      name='group'
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('Group')}</FormLabel>
                          <FormControl>
                            <ApiKeyGroupCombobox
                              options={mutate.groups}
                              value={field.value}
                              onValueChange={(group) => {
                                field.onChange(group)
                                if (group === 'auto') {
                                  mutate.form.setValue(
                                    'cross_group_retry',
                                    true,
                                    { shouldDirty: true }
                                  )
                                  return
                                }
                                mutate.form.setValue(
                                  'cross_group_retry',
                                  false,
                                  { shouldDirty: true }
                                )
                              }}
                              placeholder={t('Select a group')}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    {mutate.selectedGroup === 'auto' ? (
                      <FormField
                        control={mutate.form.control}
                        name='auto_groups'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Auto group order')}</FormLabel>
                            <FormDescription>
                              {t(
                                'Choose and order the groups this API key will try.'
                              )}
                            </FormDescription>
                            <FormControl>
                              <AutoGroupOrderEditor
                                value={field.value}
                                mode={mutate.autoGroupsMode}
                                options={mutate.groups}
                                globalOptions={mutate.globalAutoGroupOptions}
                                maxCount={mutate.maxAutoGroups}
                                onChange={(value) => {
                                  mutate.form.setValue(
                                    'auto_groups_mode',
                                    value.mode,
                                    {
                                      shouldDirty: true,
                                      shouldValidate: false,
                                    }
                                  )
                                  mutate.form.setValue(
                                    'auto_groups',
                                    value.groups.slice(0, mutate.maxAutoGroups),
                                    {
                                      shouldDirty: true,
                                      shouldValidate: true,
                                    }
                                  )
                                }}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    ) : null}

                    {mutate.selectedGroup === 'auto' ? (
                      <FormField
                        control={mutate.form.control}
                        name='cross_group_retry'
                        render={({ field }) => (
                          <FormItem className='flex items-center justify-between gap-4'>
                            <div className='flex flex-col gap-0.5'>
                              <FormLabel className='text-sm'>
                                {t('Cross-group retry')}
                              </FormLabel>
                              <FormDescription className='text-xs'>
                                {t(
                                  'When enabled, if channels in the current group fail, it will try channels in the next group in order.'
                                )}
                              </FormDescription>
                            </div>
                            <FormControl>
                              <Switch
                                checked={!!field.value}
                                onCheckedChange={field.onChange}
                              />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                    ) : null}
                  </TabsContent>

                  <TabsContent value={QUOTA_TAB} className='mt-0 space-y-4'>
                    <FormField
                      control={mutate.form.control}
                      name='expired_time'
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('Expiration Time')}</FormLabel>
                          <FormControl>
                            <DateTimePicker
                              value={field.value}
                              onChange={field.onChange}
                              placeholder={t('Never expires')}
                              className='min-w-0 [&_input[type=time]]:w-24 sm:[&_input[type=time]]:w-32'
                            />
                          </FormControl>
                          <div className='flex flex-wrap gap-2'>
                            <Button
                              type='button'
                              variant={expiredTime ? 'outline' : 'default'}
                              size='sm'
                              className='rounded-full px-3 text-xs'
                              onClick={() => mutate.handleSetExpiry(0, 0, 0)}
                            >
                              {t('Never')}
                            </Button>
                            <Button
                              type='button'
                              variant='outline'
                              size='sm'
                              className='rounded-full px-3 text-xs'
                              onClick={() => mutate.handleSetExpiry(1, 0, 0)}
                            >
                              {t('1 Month')}
                            </Button>
                            <Button
                              type='button'
                              variant='outline'
                              size='sm'
                              className='rounded-full px-3 text-xs'
                              onClick={() => mutate.handleSetExpiry(0, 1, 0)}
                            >
                              {t('1 Day')}
                            </Button>
                            <Button
                              type='button'
                              variant='outline'
                              size='sm'
                              className='rounded-full px-3 text-xs'
                              onClick={() => mutate.handleSetExpiry(0, 0, 1)}
                            >
                              {t('1 Hour')}
                            </Button>
                          </div>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    {mutate.isUpdate ? null : (
                      <FormField
                        control={mutate.form.control}
                        name='tokenCount'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t('Quantity')}</FormLabel>
                            <FormControl>
                              <Input
                                {...field}
                                type='number'
                                min='1'
                                placeholder={t('Number of keys to create')}
                                onChange={(event) =>
                                  field.onChange(
                                    Number.parseInt(event.target.value, 10) || 1
                                  )
                                }
                              />
                            </FormControl>
                            <FormDescription>
                              {t(
                                'Create multiple API keys at once (random suffix will be added to names)'
                              )}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    )}

                    <FormField
                      control={mutate.form.control}
                      name='unlimited_quota'
                      render={({ field }) => (
                        <FormItem className='flex items-center justify-between gap-4'>
                          <div className='flex flex-col gap-0.5'>
                            <FormLabel className='text-sm'>
                              {t('Unlimited Quota')}
                            </FormLabel>
                            <FormDescription className='text-xs'>
                              {t('Enable unlimited quota for this API key')}
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

                    {mutate.unlimitedQuota ? null : (
                      <FormField
                        control={mutate.form.control}
                        name='remain_quota_dollars'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{mutate.quotaLabel}</FormLabel>
                            <FormControl>
                              <Input
                                {...field}
                                type='number'
                                step={mutate.tokensOnly ? 1 : 0.01}
                                placeholder={mutate.quotaPlaceholder}
                                onChange={(event) =>
                                  field.onChange(
                                    Number.parseFloat(event.target.value) || 0
                                  )
                                }
                              />
                            </FormControl>
                            <FormDescription>
                              {mutate.tokensOnly
                                ? t('Enter the quota amount in tokens')
                                : t('Enter the quota amount in {{currency}}', {
                                    currency: mutate.currencyLabel,
                                  })}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    )}
                  </TabsContent>

                  <TabsContent value={ADVANCED_TAB} className='mt-0 space-y-4'>
                    <FormField
                      control={mutate.form.control}
                      name='model_limits'
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('Model Limits')}</FormLabel>
                          <FormControl>
                            <MultiSelect
                              options={mutate.models.map((model) => ({
                                label: model,
                                value: model,
                              }))}
                              selected={field.value}
                              onChange={field.onChange}
                              placeholder={t(
                                'Select models (empty for allow all)'
                              )}
                            />
                          </FormControl>
                          <FormDescription>
                            {t('Limit which models can be used with this key')}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={mutate.form.control}
                      name='allow_ips'
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t('IP Whitelist (supports CIDR)')}
                          </FormLabel>
                          <FormControl>
                            <Textarea
                              {...field}
                              className='min-h-20 resize-none'
                              placeholder={t(
                                'One IP per line (empty for no restriction)'
                              )}
                              rows={3}
                            />
                          </FormControl>
                          <FormDescription>
                            {t(
                              'Do not over-trust this feature. IP may be spoofed. Please use with nginx, CDN and other gateways.'
                            )}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </TabsContent>
                </div>
              </Tabs>
            </form>
          </Form>

          <DialogFooter className='mx-0 mb-0'>
            <Button
              type='button'
              variant='outline'
              onClick={() => handleOpenChange(false)}
            >
              {t('Cancel')}
            </Button>
            <Button
              type='button'
              onClick={mutate.form.handleSubmit(mutate.onSubmit, onInvalid)}
              disabled={!mutate.isFormInitialized || mutate.isSubmitting}
            >
              {submitLabel}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  )
}
