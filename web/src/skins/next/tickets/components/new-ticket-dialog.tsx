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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useId, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'
import { createSelfTicket } from '@/features/tickets/api'
import { TicketImagePicker } from '@/features/tickets/components/ticket-image-picker'
import { TICKET_QUERY_KEYS } from '@/features/tickets/constants'
import { useAddTicketImages } from '@/features/tickets/hooks/use-add-ticket-images'
import { getTransferFiles } from '@/features/tickets/lib/attachments'
import {
  CONTENT_MAX_LENGTH,
  MAX_IMAGE_MB,
  MAX_IMAGES_PER_MESSAGE,
  SUBJECT_MAX_LENGTH,
} from '@/features/tickets/lib/limits'
import { cn } from '@/lib/utils'

import {
  getNewTicketSchema,
  type NewTicketFormValues,
} from '../lib/new-ticket-schema'

const EMPTY_VALUES: NewTicketFormValues = { subject: '', content: '' }

type NextNewTicketDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

/**
 * New-ticket form. Closing the dialog keeps the draft; it is cleared only
 * once the ticket was created, and the user lands on the new ticket.
 */
export function NextNewTicketDialog(props: NextNewTicketDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const imagesLabelId = useId()
  const [files, setFiles] = useState<File[]>([])
  const [isDragging, setIsDragging] = useState(false)
  const addFiles = useAddTicketImages(files, setFiles)
  const schema = useMemo(() => getNewTicketSchema(t), [t])
  const form = useForm({
    resolver: zodResolver(schema),
    defaultValues: EMPTY_VALUES,
  })
  const contentLength = form.watch('content').length

  const createMutation = useMutation({
    mutationFn: createSelfTicket,
    onSuccess: (response) => {
      if (!response.success) {
        toast.error(response.message || t('Request failed'))
        return
      }
      toast.success(t('Ticket submitted'))
      void queryClient.invalidateQueries({
        queryKey: TICKET_QUERY_KEYS.selfList,
      })
      void queryClient.invalidateQueries({
        queryKey: TICKET_QUERY_KEYS.selfUnread,
      })
      form.reset(EMPTY_VALUES)
      setFiles([])
      props.onOpenChange(false)
      void navigate({
        to: '/tickets/$ticketId',
        params: { ticketId: String(response.data.id) },
      })
    },
    onError: (error) => toast.error(error.message || t('Request failed')),
  })
  const isPending = createMutation.isPending

  return (
    <Dialog
      open={props.open}
      onOpenChange={(open) => {
        if (!open && isPending) return
        props.onOpenChange(open)
      }}
    >
      <DialogContent className='flex max-h-[min(90vh,46rem)] w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-xl'>
        <DialogHeader className='px-5 pt-5 pb-3'>
          <DialogTitle className='text-lg tracking-tight'>
            {t('New ticket')}
          </DialogTitle>
          <DialogDescription>
            {t(
              'Tell us what happened. Screenshots help us find the problem faster.'
            )}
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form
            data-slot='next-new-ticket-form'
            noValidate
            onSubmit={form.handleSubmit((values) =>
              createMutation.mutate({ ...values, files })
            )}
            onDragOver={(event) => {
              if (!event.dataTransfer.types.includes('Files')) return
              event.preventDefault()
              if (!isPending) setIsDragging(true)
            }}
            onDragLeave={(event) => {
              const next = event.relatedTarget
              if (next instanceof Node && event.currentTarget.contains(next)) {
                return
              }
              setIsDragging(false)
            }}
            onDrop={(event) => {
              if (!event.dataTransfer.types.includes('Files')) return
              // Swallow file drops so the browser never navigates to them.
              event.preventDefault()
              setIsDragging(false)
              if (!isPending) addFiles(getTransferFiles(event.dataTransfer))
            }}
            className={cn(
              'flex min-h-0 flex-1 flex-col transition-shadow',
              isDragging && 'ring-ring/50 ring-3 ring-inset'
            )}
          >
            <fieldset
              disabled={isPending}
              className='min-h-0 flex-1 space-y-4 overflow-y-auto border-t px-5 py-4'
            >
              <FormField
                control={form.control}
                name='subject'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Subject')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        maxLength={SUBJECT_MAX_LENGTH}
                        autoComplete='off'
                        placeholder={t('Summarize the problem in one line')}
                        className='h-10'
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='content'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Issue description')}</FormLabel>
                    <FormControl>
                      <Textarea
                        {...field}
                        maxLength={CONTENT_MAX_LENGTH}
                        placeholder={t(
                          'What happened, when, and what did you expect? Include error messages if you have them.'
                        )}
                        onPaste={(event) => {
                          if (isPending) return
                          addFiles(getTransferFiles(event.clipboardData))
                        }}
                        className='min-h-36'
                      />
                    </FormControl>
                    <div className='flex items-start gap-3'>
                      <FormMessage />
                      <span className='text-muted-foreground ml-auto text-xs tabular-nums'>
                        {contentLength}/{CONTENT_MAX_LENGTH}
                      </span>
                    </div>
                  </FormItem>
                )}
              />

              <div
                role='group'
                aria-labelledby={imagesLabelId}
                className='space-y-2'
              >
                <p id={imagesLabelId} className='text-sm font-medium'>
                  {t('Images')}
                </p>
                <TicketImagePicker
                  files={files}
                  onChange={setFiles}
                  disabled={isPending}
                />
                <p className='text-muted-foreground text-xs'>
                  {t(
                    'Optional. Up to {{images}} PNG, JPEG, GIF or WebP images, {{size}} MB each. You can also paste or drop them here.',
                    { images: MAX_IMAGES_PER_MESSAGE, size: MAX_IMAGE_MB }
                  )}
                </p>
              </div>
            </fieldset>

            <div className='flex flex-col-reverse gap-2 border-t px-5 py-4 sm:flex-row sm:justify-end'>
              <Button
                type='button'
                variant='outline'
                disabled={isPending}
                onClick={() => props.onOpenChange(false)}
                className='h-10 px-4'
              >
                {t('Cancel')}
              </Button>
              <Button type='submit' disabled={isPending} className='h-10 px-4'>
                {isPending && <Spinner aria-hidden='true' />}
                {t('Submit ticket')}
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
