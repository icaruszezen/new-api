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
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { Dialog } from '@/components/dialog'
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

export type EpayGatewayData = {
  id: string
  name: string
  pay_address: string
  epay_id: string
  epay_key: string
  custom_callback_address?: string
}

const EPAY_GATEWAY_FORM_ID = 'epay-gateway-form'

// Ids end up in the callback URL path, so keep them restricted to characters
// that need no escaping.
const EPAY_GATEWAY_ID_PATTERN = /^[A-Za-z0-9_-]{1,64}$/

function generateEpayGatewayId() {
  const bytes = new Uint8Array(4)
  crypto.getRandomValues(bytes)
  const suffix = [...bytes]
    .map((byte) => byte.toString(16).padStart(2, '0'))
    .join('')
  return `gw_${suffix}`
}

const createEpayGatewaySchema = (
  t: (key: string) => string,
  requireSecretKey: boolean
) =>
  z.object({
    id: z
      .string()
      .regex(
        EPAY_GATEWAY_ID_PATTERN,
        t('Use only letters, numbers, underscores and hyphens.')
      ),
    name: z.string().min(1, t('Gateway name is required')),
    pay_address: z
      .string()
      .refine(
        (value) => /^https?:\/\//.test(value.trim()),
        t('Provide a valid endpoint starting with http:// or https://')
      ),
    epay_id: z.string().min(1, t('Merchant ID is required')),
    epay_key: z
      .string()
      .refine(
        (value) => !requireSecretKey || value.trim().length > 0,
        t('Secret key is required')
      ),
    custom_callback_address: z.string().refine((value) => {
      const trimmed = value.trim()
      if (!trimmed) return true
      try {
        const url = new URL(trimmed)
        const isHttpProtocol =
          url.protocol === 'http:' || url.protocol === 'https:'
        const hasNoPath = url.pathname === '' || url.pathname === '/'
        return isHttpProtocol && hasNoPath && !url.search && !url.hash
      } catch {
        return false
      }
    }, t('Enter only a top-level callback domain, for example https://api.example.com, without any path.')),
  })

type EpayGatewayFormValues = z.infer<ReturnType<typeof createEpayGatewaySchema>>

type EpayGatewayDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave: (data: EpayGatewayData) => void
  editData?: EpayGatewayData | null
  notifyUrlPreviewBase?: string
}

export function EpayGatewayDialog(props: EpayGatewayDialogProps) {
  const { t } = useTranslation()
  const isEditMode = !!props.editData

  const form = useForm<EpayGatewayFormValues>({
    resolver: zodResolver(createEpayGatewaySchema(t, !isEditMode)),
    defaultValues: {
      id: '',
      name: '',
      pay_address: '',
      epay_id: '',
      epay_key: '',
      custom_callback_address: '',
    },
  })

  useEffect(() => {
    if (!props.open) return
    if (props.editData) {
      form.reset({
        id: props.editData.id,
        name: props.editData.name,
        pay_address: props.editData.pay_address,
        epay_id: props.editData.epay_id,
        epay_key: '',
        custom_callback_address: props.editData.custom_callback_address ?? '',
      })
      return
    }
    form.reset({
      id: generateEpayGatewayId(),
      name: '',
      pay_address: '',
      epay_id: '',
      epay_key: '',
      custom_callback_address: '',
    })
  }, [props.editData, props.open, form])

  const gatewayId = form.watch('id')
  const gatewayCallbackAddress = form.watch('custom_callback_address')
  const notifyUrlBase =
    gatewayCallbackAddress.trim() ||
    props.notifyUrlPreviewBase?.trim() ||
    '<ServerAddress>'

  const handleSubmit = (values: EpayGatewayFormValues) => {
    const data: EpayGatewayData = {
      id: values.id.trim(),
      name: values.name.trim(),
      pay_address: values.pay_address.trim().replace(/\/+$/, ''),
      epay_id: values.epay_id.trim(),
      epay_key: values.epay_key.trim(),
    }
    const callbackAddress = values.custom_callback_address.trim()
    if (callbackAddress) {
      data.custom_callback_address = callbackAddress.replace(/\/+$/, '')
    }
    props.onSave(data)
    props.onOpenChange(false)
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={isEditMode ? t('Edit Epay gateway') : t('Add Epay gateway')}
      description={t(
        'Each gateway keeps its own endpoint, merchant credentials and callback URL.'
      )}
      contentClassName='sm:max-w-[560px]'
      contentHeight='auto'
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
          <Button type='submit' form={EPAY_GATEWAY_FORM_ID}>
            {isEditMode ? t('Update') : t('Add')}
          </Button>
        </>
      }
    >
      <Form {...form}>
        <form
          id={EPAY_GATEWAY_FORM_ID}
          onSubmit={form.handleSubmit(handleSubmit)}
          className='space-y-4'
        >
          <FormField
            control={form.control}
            name='name'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Gateway name')}</FormLabel>
                <FormControl>
                  <Input placeholder={t('e.g., Backup Epay')} {...field} />
                </FormControl>
                <FormDescription>
                  {t('Shown when binding payment methods to this gateway.')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='id'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Gateway ID')}</FormLabel>
                <FormControl>
                  <Input
                    autoComplete='off'
                    readOnly={isEditMode}
                    className={isEditMode ? 'font-mono' : undefined}
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {isEditMode
                    ? t(
                        'The ID is part of the callback URL and cannot be changed after creation.'
                      )
                    : t(
                        'Used in the callback URL. Generated automatically, change it only if needed.'
                      )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='pay_address'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Epay endpoint')}</FormLabel>
                <FormControl>
                  <Input placeholder='https://pay.example.com' {...field} />
                </FormControl>
                <FormDescription>
                  {t('Base address provided by your Epay service')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='epay_id'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Epay merchant ID')}</FormLabel>
                <FormControl>
                  <Input placeholder='10001' autoComplete='off' {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='epay_key'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Epay secret key')}</FormLabel>
                <FormControl>
                  <Input
                    type='password'
                    placeholder={t('Enter new key to update')}
                    autoComplete='new-password'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {isEditMode
                    ? t('Leave blank unless rotating the secret')
                    : t('Required when adding a gateway')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='custom_callback_address'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Callback address')}</FormLabel>
                <FormControl>
                  <Input placeholder='https://gateway.example.com' {...field} />
                </FormControl>
                <FormDescription>
                  {t(
                    'Leave blank to reuse the callback address of the default gateway.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {gatewayId ? (
            <div className='text-muted-foreground bg-muted rounded-md p-3 text-xs'>
              <p className='mb-1 font-medium'>{t('Async notify URL')}</p>
              <code className='break-all'>
                {`${notifyUrlBase.replace(/\/+$/, '')}/api/user/epay/notify/${gatewayId}`}
              </code>
            </div>
          ) : null}
        </form>
      </Form>
    </Dialog>
  )
}
