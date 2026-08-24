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
import {
  AlertCircle,
  CheckCircle2,
  Copy,
  FileText,
  Play,
  Zap,
} from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Combobox } from '@/components/ui/combobox'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { ApiInfoItem } from '@/features/dashboard/types'
import type { ApiKey } from '@/features/keys/types'
import { cn } from '@/lib/utils'

import { useTestConnection } from '../hooks/use-test-connection'
import { formatKeyOptionLabel, TEST_PROMPT } from '../lib/test-connection'

export function TestConnectionDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  keys: ApiKey[]
  apiInfoItems: ApiInfoItem[]
  serverAddress?: string
}) {
  const { t } = useTranslation()
  const test = useTestConnection({
    open: props.open,
    keys: props.keys,
    apiInfoItems: props.apiInfoItems,
    serverAddress: props.serverAddress,
  })
  const hasKeys = props.keys.length > 0
  const canAct = hasKeys && test.canSubmit && !test.isModelsLoading

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='flex max-h-[min(85vh,40rem)] w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-xl'>
        <div
          data-slot='next-test-connection-dialog'
          className='flex min-h-0 flex-1 flex-col'
        >
          <DialogHeader className='px-5 pt-5 pb-3'>
            <p className='text-primary text-[11px] font-medium tracking-[0.16em] uppercase'>
              {t('API KEY TEST')}
            </p>
            <DialogTitle className='text-lg tracking-tight'>
              {t('Test request link')}
            </DialogTitle>
            <DialogDescription className='sr-only'>
              {t(
                'Send a streaming test request with a selected key, endpoint, and model.'
              )}
            </DialogDescription>
          </DialogHeader>

          <div className='flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-5 pb-4'>
            <Field label={t('Test API key')}>
              <Select
                items={props.keys.map((key) => ({
                  value: String(key.id),
                  label: formatKeyOptionLabel(key),
                }))}
                value={
                  test.selectedKeyId != null ? String(test.selectedKeyId) : null
                }
                onValueChange={(value) => {
                  if (value) test.setSelectedKeyId(Number(value))
                }}
                disabled={!hasKeys}
              >
                <SelectTrigger
                  className='h-9 w-full'
                  aria-label={t('Test API key')}
                >
                  <SelectValue placeholder={t('Select an API key')} />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {props.keys.map((key) => (
                      <SelectItem key={key.id} value={String(key.id)}>
                        {formatKeyOptionLabel(key)}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>

            <Field label={t('Request address')}>
              <Select
                items={test.endpoints.map((item) => ({
                  value: item.id,
                  label: item.label,
                }))}
                value={test.selectedEndpointId}
                onValueChange={(value) => {
                  if (value) test.setSelectedEndpointId(value)
                }}
              >
                <SelectTrigger
                  className='h-9 w-full'
                  aria-label={t('Request address')}
                >
                  <SelectValue placeholder={t('Select an endpoint')} />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {test.endpoints.map((item) => (
                      <SelectItem key={item.id} value={item.id}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>

            <div className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
              <Field htmlFor='test-connection-model' label={t('Test Model')}>
                <Combobox
                  id='test-connection-model'
                  options={test.models.map((model) => ({
                    value: model,
                    label: model,
                  }))}
                  value={test.selectedModel}
                  onValueChange={(value) => {
                    if (value) test.setModelOverride(value)
                  }}
                  placeholder={t('Select a model')}
                  emptyText='No models available for this key.'
                  className='h-9 w-full'
                />
              </Field>
              <Field label={t('Request format')}>
                <Select
                  items={test.formats.map((format) => ({
                    value: format,
                    label: format,
                  }))}
                  value={test.selectedFormat}
                  onValueChange={(value) => {
                    if (value) test.setFormatOverride(value)
                  }}
                >
                  <SelectTrigger
                    className='h-9 w-full'
                    aria-label={t('Request format')}
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      {test.formats.map((format) => (
                        <SelectItem key={format} value={format}>
                          {format}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
            </div>

            <ResultPanel
              errorMessage={test.errorMessage}
              hasKeys={hasKeys}
              phase={test.phase}
              responseText={test.responseText}
            />
          </div>

          <DialogFooter className='mx-0 mb-0'>
            <Button
              type='button'
              variant='outline'
              disabled={!canAct}
              onClick={() => {
                void test.copyTestLink()
              }}
            >
              <Copy aria-hidden='true' />
              {t('Copy test link')}
            </Button>
            <Button
              type='button'
              disabled={!canAct}
              onClick={() => {
                void test.startTest()
              }}
            >
              <Zap aria-hidden='true' />
              {t('Start Test')}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function Field(props: {
  label: string
  htmlFor?: string
  children: ReactNode
}) {
  return (
    <div className='flex flex-col gap-2'>
      <Label
        htmlFor={props.htmlFor}
        className='text-muted-foreground text-xs font-medium'
      >
        {props.label}
      </Label>
      {props.children}
    </div>
  )
}

function ResultPanel(props: {
  errorMessage: string
  hasKeys: boolean
  phase: 'idle' | 'running' | 'success' | 'error'
  responseText: string
}) {
  const { t } = useTranslation()
  const isReady = props.phase === 'idle'
  const showExchange = props.phase !== 'idle'

  let statusIcon = <Play className='size-3.5' aria-hidden='true' />
  let statusLabel = t('Ready to test')
  if (props.phase === 'success') {
    statusIcon = <CheckCircle2 className='size-3.5' aria-hidden='true' />
    statusLabel = t('Returned content')
  } else if (props.phase === 'error') {
    statusIcon = <AlertCircle className='size-3.5' aria-hidden='true' />
    statusLabel = t('Returned content')
  } else if (props.phase === 'running') {
    statusLabel = t('Testing in progress')
  }

  let resultBody = props.responseText
  if (props.phase === 'error') {
    resultBody = props.errorMessage
  } else if (props.phase === 'running' && !props.responseText) {
    resultBody = t('Testing in progress')
  }

  return (
    <div
      data-slot='test-connection-result'
      data-phase={props.phase}
      role='status'
      aria-live='polite'
      aria-busy={props.phase === 'running'}
      className={cn(
        'min-h-36 rounded-xl border px-4 py-3 text-sm',
        props.phase === 'success' &&
          'border-emerald-200 bg-emerald-50/80 dark:border-emerald-900 dark:bg-emerald-950/30',
        props.phase === 'error' && 'border-destructive/30 bg-destructive/5',
        props.phase === 'running' && 'border-primary/20 bg-primary/5',
        isReady && 'border-border bg-muted/40'
      )}
    >
      {isReady ? (
        <div className='text-muted-foreground space-y-2'>
          <p className='flex items-center gap-1.5 font-medium'>
            {statusIcon}
            {statusLabel}
          </p>
          <p>
            {props.hasKeys
              ? t('Click Start Test to begin...')
              : t('Create an API key before testing the connection.')}
          </p>
        </div>
      ) : null}

      {showExchange ? (
        <div className='space-y-3'>
          <div>
            <p className='text-muted-foreground mb-1 flex items-center gap-1.5 text-xs font-medium'>
              <FileText className='size-3.5' aria-hidden='true' />
              {t('Request content')}
            </p>
            <p className='font-mono text-[13px]'>{TEST_PROMPT}</p>
          </div>
          <div className='border-border border-dashed border-t' />
          <div>
            <p
              className={cn(
                'mb-1 flex items-center gap-1.5 text-xs font-medium',
                props.phase === 'success' &&
                  'text-emerald-700 dark:text-emerald-400',
                props.phase === 'error' && 'text-destructive',
                props.phase === 'running' && 'text-primary'
              )}
            >
              {statusIcon}
              {statusLabel}
            </p>
            <p className='font-mono text-[13px] whitespace-pre-wrap'>
              {resultBody}
            </p>
          </div>
        </div>
      ) : null}
    </div>
  )
}
