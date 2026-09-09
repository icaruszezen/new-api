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
import type { ChangeEvent } from 'react'
import type { Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { Alert, AlertDescription } from '@/components/ui/alert'
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
  buildInviteRebateRuleLabels,
  MAX_INVITE_REBATE_AMOUNT,
} from '@/features/invite/lib/rules'

import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import {
  SettingsForm,
  SettingsFormGrid,
  SettingsFormGridItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'

const amountSchema = z.coerce.number().min(0).max(MAX_INVITE_REBATE_AMOUNT)

const inviteRebateSchema = z.object({
  invite_rebate_setting: z.object({
    register_amount: amountSchema,
    topup_amount: amountSchema,
    topup_threshold: amountSchema,
  }),
})

type InviteRebateFormValues = z.infer<typeof inviteRebateSchema>
type AmountInputValue = number | ''

type InviteRebateSettingsSectionProps = {
  defaultValues: InviteRebateFormValues
  currencySymbol: string
  complianceConfirmed?: boolean
}

export function InviteRebateSettingsSection(
  props: InviteRebateSettingsSectionProps
) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const { form, handleSubmit, isDirty, isSubmitting } =
    useSettingsForm<InviteRebateFormValues>({
      resolver: zodResolver(inviteRebateSchema) as Resolver<
        InviteRebateFormValues,
        unknown,
        InviteRebateFormValues
      >,
      defaultValues: props.defaultValues,
      onSubmit: async (_data, changedFields) => {
        // Persist the threshold before the top-up amount so the backend pair
        // check sees a convertible quota when both fields are enabled together.
        const keys = Object.keys(changedFields).sort((left, right) => {
          const leftThreshold = left.endsWith('.topup_threshold')
          const rightThreshold = right.endsWith('.topup_threshold')
          if (leftThreshold === rightThreshold) {
            return 0
          }
          if (leftThreshold) {
            return -1
          }
          return 1
        })
        for (const key of keys) {
          await updateOption.mutateAsync({
            key,
            value: changedFields[key] as string | number | boolean,
          })
        }
      },
    })

  const handleAmountChange =
    (onChange: (value: AmountInputValue) => void) =>
    (event: ChangeEvent<HTMLInputElement>) => {
      const value = event.currentTarget.valueAsNumber
      onChange(Number.isNaN(value) ? '' : value)
    }

  const values = form.watch('invite_rebate_setting')
  const ruleLabels = buildInviteRebateRuleLabels(
    {
      register_amount: Number(values?.register_amount) || 0,
      topup_amount: Number(values?.topup_amount) || 0,
      topup_threshold: Number(values?.topup_threshold) || 0,
    },
    props.currencySymbol,
    t
  )

  return (
    <SettingsSection title={t('Referral Rebate Settings')}>
      <FormNavigationGuard when={isDirty} />

      {props.complianceConfirmed === false ? (
        <Alert variant='destructive'>
          <AlertDescription>
            {t(
              'Non-zero invitation rewards require compliance confirmation in Payment Gateway settings.'
            )}
          </AlertDescription>
        </Alert>
      ) : null}

      <Form {...form}>
        <SettingsForm onSubmit={handleSubmit}>
          <SettingsPageFormActions
            onSave={handleSubmit}
            isSaving={updateOption.isPending || isSubmitting}
            saveLabel='Save rebate settings'
          />
          <FormDirtyIndicator isDirty={isDirty} />

          <SettingsFormGridItem span='full'>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Each invitee triggers at most two payouts: once on sign-up, and once when their cumulative top-up reaches the threshold.'
              )}
            </p>
          </SettingsFormGridItem>

          <SettingsFormGrid>
            <FormField
              control={form.control}
              name='invite_rebate_setting.register_amount'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Sign-up rebate amount')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      max={MAX_INVITE_REBATE_AMOUNT}
                      step='0.01'
                      value={field.value ?? ''}
                      onChange={handleAmountChange(field.onChange)}
                      name={field.name}
                      onBlur={field.onBlur}
                      ref={field.ref}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Paid to the inviter as soon as the invitee signs up. Set to 0 to disable the sign-up rebate. Amount in {{symbol}}.',
                      { symbol: props.currencySymbol }
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='invite_rebate_setting.topup_amount'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Top-up rebate amount')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      max={MAX_INVITE_REBATE_AMOUNT}
                      step='0.01'
                      value={field.value ?? ''}
                      onChange={handleAmountChange(field.onChange)}
                      name={field.name}
                      onBlur={field.onBlur}
                      ref={field.ref}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Paid once the invitee cumulative top-up reaches the threshold. Each invitee triggers this only once. Amount in {{symbol}}.',
                      { symbol: props.currencySymbol }
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='invite_rebate_setting.topup_threshold'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Top-up threshold')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      max={MAX_INVITE_REBATE_AMOUNT}
                      step='0.01'
                      value={field.value ?? ''}
                      onChange={handleAmountChange(field.onChange)}
                      name={field.name}
                      onBlur={field.onBlur}
                      ref={field.ref}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Cumulative successful online top-up an invitee must reach before the top-up rebate is paid. Redemption codes and check-in rewards do not count. Amount in {{symbol}}.',
                      { symbol: props.currencySymbol }
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <SettingsFormGridItem>
              <div className='bg-muted/30 flex h-full min-w-0 flex-col gap-1.5 rounded-xl border px-3 py-2.5'>
                <p className='text-sm font-medium'>{t('User page preview')}</p>
                <p className='text-muted-foreground text-sm'>
                  {ruleLabels.length > 0
                    ? ruleLabels.join(' · ')
                    : t('Referral rebate is currently disabled.')}
                </p>
                <p className='text-muted-foreground text-xs'>
                  {t(
                    'These labels appear as rule chips on the user referral page.'
                  )}
                </p>
              </div>
            </SettingsFormGridItem>

            <SettingsFormGridItem span='full'>
              <div className='bg-muted/30 min-w-0 space-y-1.5 rounded-xl border px-3 py-2.5'>
                <p className='text-sm font-medium'>{t('Payout order')}</p>
                <ol className='text-muted-foreground list-decimal space-y-1 pl-5 text-xs'>
                  <li>
                    {t(
                      'The invitee signs up with the referral link, and the inviter receives the sign-up rebate immediately.'
                    )}
                  </li>
                  <li>
                    {t(
                      'When that invitee cumulative top-up reaches the threshold, the inviter receives the top-up rebate once. Further top-ups by the same invitee do not pay again.'
                    )}
                  </li>
                  <li>
                    {t(
                      'Rebates land directly in the inviter balance and require payment compliance confirmation.'
                    )}
                  </li>
                </ol>
              </div>
            </SettingsFormGridItem>
          </SettingsFormGrid>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
