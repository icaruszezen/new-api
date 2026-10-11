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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
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
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'

import { getIqCheckConfig, updateIqCheckConfig } from './api'
import { getIqCheckEntrySchema, type IqCheckEntryValues } from './lib/entry-url'

const IQ_CHECK_CONFIG_QUERY_KEY = ['iq-check-config']

/**
 * The IQ check itself runs on an external website; this page only turns the
 * navigation entry on and sets the address it opens.
 */
export function IqCheckSettings() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const schema = useMemo(() => getIqCheckEntrySchema(t), [t])

  const configQuery = useQuery({
    queryKey: IQ_CHECK_CONFIG_QUERY_KEY,
    queryFn: getIqCheckConfig,
  })

  const form = useForm<IqCheckEntryValues>({
    resolver: zodResolver(schema),
    defaultValues: { enabled: false, url: '' },
  })

  const config = configQuery.data
  useEffect(() => {
    if (config) form.reset({ enabled: config.enabled, url: config.url })
  }, [form, config])

  const saveMutation = useMutation({
    mutationFn: updateIqCheckConfig,
    onSuccess: (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to update setting'))
        return
      }
      queryClient.setQueryData(IQ_CHECK_CONFIG_QUERY_KEY, response.data)
      void queryClient.invalidateQueries({ queryKey: ['status'] })
      toast.success(t('Setting updated successfully'))
    },
  })

  const renderContent = () => {
    if (configQuery.isLoading) {
      return <Skeleton className='h-48 rounded-lg' />
    }
    if (configQuery.isError) {
      return (
        <div className='space-y-3 text-center'>
          <p className='text-muted-foreground text-sm'>
            {t('Failed to load IQ check settings.')}
          </p>
          <Button
            type='button'
            variant='outline'
            onClick={() => void configQuery.refetch()}
          >
            {t('Retry')}
          </Button>
        </div>
      )
    }
    return (
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit((values) => saveMutation.mutate(values))}
          className='space-y-4'
          noValidate
        >
          <FormField
            control={form.control}
            name='enabled'
            render={({ field }) => (
              <FormItem className='border-border/60 flex flex-row items-center justify-between gap-4 rounded-lg border px-4 py-3'>
                <div className='space-y-1'>
                  <FormLabel>{t('Show IQ check entry')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Adds an IQ Check link to the top navigation that opens the website below in a new tab.'
                    )}
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={saveMutation.isPending}
                  />
                </FormControl>
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='url'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('IQ check website')}</FormLabel>
                <FormControl>
                  <Input
                    {...field}
                    type='url'
                    inputMode='url'
                    placeholder='https://'
                    disabled={saveMutation.isPending}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'The external website that runs the IQ checks. Use a full address starting with http:// or https://.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className='flex justify-end'>
            <Button
              type='submit'
              disabled={saveMutation.isPending || !form.formState.isDirty}
            >
              {t('Save')}
            </Button>
          </div>
        </form>
      </Form>
    )
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('IQ Check Settings')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='mx-auto max-w-3xl'>{renderContent()}</div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
