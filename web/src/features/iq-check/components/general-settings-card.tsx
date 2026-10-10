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
import { useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
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
  MAX_HISTORY_LIMIT,
  MAX_REQUEST_TIMEOUT_SECONDS,
  MIN_HISTORY_LIMIT,
  MIN_REQUEST_TIMEOUT_SECONDS,
} from '../constants'
import {
  type GeneralSettingsValues,
  getGeneralSettingsSchema,
} from '../lib/forms'

type GeneralSettingsCardProps = {
  values: GeneralSettingsValues
  disabled: boolean
  onSave: (values: GeneralSettingsValues) => Promise<boolean>
}

/** History size and per-run timeout shared by every check. */
export function GeneralSettingsCard(props: GeneralSettingsCardProps) {
  const { t } = useTranslation()
  const schema = useMemo(() => getGeneralSettingsSchema(t), [t])

  const form = useForm<GeneralSettingsValues>({
    resolver: zodResolver(schema),
    defaultValues: props.values,
  })

  const historyLimit = props.values.history_limit
  const requestTimeoutSeconds = props.values.request_timeout_seconds
  useEffect(() => {
    form.reset({
      history_limit: historyLimit,
      request_timeout_seconds: requestTimeoutSeconds,
    })
  }, [form, historyLimit, requestTimeoutSeconds])

  const handleSubmit = async (values: GeneralSettingsValues) => {
    if (await props.onSave(values)) {
      form.reset(values)
    }
  }

  return (
    <section className='border-border/60 space-y-4 rounded-lg border px-4 py-3'>
      <div className='space-y-1'>
        <h3 className='text-sm font-medium'>{t('General')}</h3>
        <p className='text-muted-foreground text-xs'>
          {t('These limits apply to every check.')}
        </p>
      </div>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(handleSubmit)}
          className='space-y-4'
          noValidate
        >
          <div className='grid gap-4 sm:grid-cols-2'>
            <FormField
              control={form.control}
              name='history_limit'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('History size')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      inputMode='numeric'
                      min={MIN_HISTORY_LIMIT}
                      max={MAX_HISTORY_LIMIT}
                      step={1}
                      name={field.name}
                      ref={field.ref}
                      onBlur={field.onBlur}
                      value={Number.isNaN(field.value) ? '' : field.value}
                      onChange={(event) =>
                        field.onChange(event.target.valueAsNumber)
                      }
                      disabled={props.disabled}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Results kept per check ({{min}}-{{max}}).', {
                      min: MIN_HISTORY_LIMIT,
                      max: MAX_HISTORY_LIMIT,
                    })}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='request_timeout_seconds'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Request timeout (seconds)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      inputMode='numeric'
                      min={MIN_REQUEST_TIMEOUT_SECONDS}
                      max={MAX_REQUEST_TIMEOUT_SECONDS}
                      step={1}
                      name={field.name}
                      ref={field.ref}
                      onBlur={field.onBlur}
                      value={Number.isNaN(field.value) ? '' : field.value}
                      onChange={(event) =>
                        field.onChange(event.target.valueAsNumber)
                      }
                      disabled={props.disabled}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'A run that takes longer is recorded as failed ({{min}}-{{max}}).',
                      {
                        min: MIN_REQUEST_TIMEOUT_SECONDS,
                        max: MAX_REQUEST_TIMEOUT_SECONDS,
                      }
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <div className='flex justify-end'>
            <Button
              type='submit'
              size='sm'
              disabled={props.disabled || !form.formState.isDirty}
            >
              {t('Save')}
            </Button>
          </div>
        </form>
      </Form>
    </section>
  )
}
