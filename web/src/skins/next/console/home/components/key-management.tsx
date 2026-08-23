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
import { useQuery } from '@tanstack/react-query'
import {
  CircleSlash,
  Copy,
  FileText,
  Pencil,
  Plus,
  Send,
  Trash2,
  Zap,
} from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Skeleton } from '@/components/ui/skeleton'
import { useApiInfo } from '@/features/dashboard/hooks/use-status-data'
import { getApiKeys } from '@/features/keys/api'
import type { ApiKey } from '@/features/keys/types'
import { getUserGroups } from '@/lib/api'

import { formatGroupRatio, maskApiKey } from '../lib/mask-key'
import { MISSING_METRIC } from '../lib/stats'

type GroupInfo = { desc: string; ratio: number | string }

function InertIconButton(props: { label: string; children: ReactNode }) {
  return (
    <button
      type='button'
      aria-label={props.label}
      className='text-muted-foreground hover:text-foreground inline-flex size-8 items-center justify-center rounded-md transition-colors'
    >
      {props.children}
    </button>
  )
}

function KeyRow(props: {
  apiKey: ApiKey
  groups: Record<string, GroupInfo>
}) {
  const { t } = useTranslation()
  const group = props.apiKey.group ?? ''
  const groupInfo = group ? props.groups[group] : undefined
  const groupLabel = groupInfo?.desc || group || MISSING_METRIC
  const ratioLabel = formatGroupRatio(groupInfo?.ratio) ?? MISSING_METRIC

  return (
    <tr className='border-border/70 border-t'>
      <td className='px-4 py-3 font-medium'>{props.apiKey.name}</td>
      <td className='text-muted-foreground px-4 py-3'>{t('OpenAI')}</td>
      <td className='px-4 py-3 font-mono text-xs tabular-nums'>
        {maskApiKey(props.apiKey.key)}
      </td>
      <td className='px-4 py-3'>{groupLabel}</td>
      <td className='px-4 py-3 font-mono text-xs tabular-nums'>{ratioLabel}</td>
      <td className='text-muted-foreground px-4 py-3'>{MISSING_METRIC}</td>
      <td className='px-4 py-3'>
        <div className='flex items-center justify-end gap-0.5'>
          <InertIconButton label={t('Copy')}>
            <Copy className='size-3.5' aria-hidden='true' />
          </InertIconButton>
          <InertIconButton label={t('Test')}>
            <Send className='size-3.5' aria-hidden='true' />
          </InertIconButton>
          <InertIconButton label={t('Edit')}>
            <Pencil className='size-3.5' aria-hidden='true' />
          </InertIconButton>
          <InertIconButton label={t('Disable')}>
            <CircleSlash className='size-3.5' aria-hidden='true' />
          </InertIconButton>
          <InertIconButton label={t('Delete')}>
            <Trash2 className='size-3.5' aria-hidden='true' />
          </InertIconButton>
        </div>
      </td>
    </tr>
  )
}

export function ConsoleKeyManagement() {
  const { t } = useTranslation()
  const apiInfo = useApiInfo()

  const keysQuery = useQuery({
    queryKey: ['console', 'home', 'api-keys'],
    queryFn: async () => {
      const result = await getApiKeys({ p: 1, size: 50 })
      return result.success ? (result.data?.items ?? []) : []
    },
    staleTime: 60 * 1000,
  })

  const groupsQuery = useQuery({
    queryKey: ['console', 'home', 'user-groups'],
    queryFn: async () => {
      const result = await getUserGroups()
      return result.success ? (result.data ?? {}) : {}
    },
    staleTime: 60 * 1000,
  })

  const keys = keysQuery.data ?? []
  const groups = groupsQuery.data ?? {}

  return (
    <section className='bg-card overflow-hidden rounded-xl border'>
      <div className='flex flex-col gap-4 px-5 py-5 sm:flex-row sm:items-start sm:justify-between'>
        <div className='min-w-0'>
          <h2 className='text-lg font-medium tracking-tight'>
            {t('Key Management')}
          </h2>
          <p className='text-muted-foreground mt-1 max-w-xl text-sm leading-relaxed'>
            {t(
              'Create multiple keys by usage scenario and bind a platform and billing group to each key.'
            )}
          </p>
        </div>
        <div className='flex shrink-0 flex-wrap items-center gap-2'>
          <button
            type='button'
            className='border-border inline-flex h-9 items-center gap-1.5 rounded-full border px-3.5 text-sm font-medium'
          >
            <FileText className='size-3.5' aria-hidden='true' />
            {t('Usage docs')}
          </button>
          <button
            type='button'
            className='bg-foreground text-background inline-flex h-9 items-center gap-1.5 rounded-full px-3.5 text-sm font-medium transition-opacity hover:opacity-85'
          >
            <Plus className='size-3.5' aria-hidden='true' />
            {t('New key')}
          </button>
        </div>
      </div>

      <div className='border-border/70 flex flex-col gap-3 border-t px-5 py-3 sm:flex-row sm:items-center'>
        <button
          type='button'
          className='border-border inline-flex h-9 shrink-0 items-center gap-1.5 rounded-full border px-3.5 text-sm font-medium'
        >
          <Zap className='size-3.5' aria-hidden='true' />
          {t('Test Connection')}
        </button>
        <div className='flex min-w-0 flex-1 flex-wrap items-center gap-2'>
          {apiInfo.items.map((item) => (
            <div
              key={item.url}
              className='border-border bg-muted/40 flex min-w-0 items-center gap-2 rounded-lg border px-2.5 py-1.5'
            >
              <span className='bg-background text-muted-foreground shrink-0 rounded-md border px-1.5 py-0.5 text-[11px] font-medium'>
                {item.description || item.route || t('Default')}
              </span>
              <code className='min-w-0 truncate font-mono text-xs'>
                {item.url}
              </code>
              <CopyButton
                value={item.url}
                size='icon'
                className='text-muted-foreground hover:text-foreground size-7'
                aria-label={t('Copy')}
              />
            </div>
          ))}
        </div>
      </div>

      <div className='overflow-x-auto'>
        <table className='w-full min-w-[56rem] text-left text-sm'>
          <thead>
            <tr className='text-muted-foreground border-border/70 border-t text-xs font-medium'>
              <th className='px-4 py-2.5 font-medium'>{t('Name')}</th>
              <th className='px-4 py-2.5 font-medium'>{t('Platform')}</th>
              <th className='px-4 py-2.5 font-medium'>{t('API Key')}</th>
              <th className='px-4 py-2.5 font-medium'>{t('Billing group')}</th>
              <th className='px-4 py-2.5 font-medium'>{t('Billing rate')}</th>
              <th className='px-4 py-2.5 font-medium'>{t('Usage')}</th>
              <th className='px-4 py-2.5 text-end font-medium'>
                {t('Actions')}
              </th>
            </tr>
          </thead>
          <tbody>
            {keysQuery.isLoading ? (
              <tr className='border-border/70 border-t'>
                <td colSpan={7} className='px-4 py-6'>
                  <Skeleton className='h-8 w-full' />
                </td>
              </tr>
            ) : null}
            {!keysQuery.isLoading && keys.length === 0 ? (
              <tr className='border-border/70 border-t'>
                <td
                  colSpan={7}
                  className='text-muted-foreground px-4 py-8 text-center'
                >
                  {t('No API keys yet')}
                </td>
              </tr>
            ) : null}
            {keys.map((apiKey) => (
              <KeyRow key={apiKey.id} apiKey={apiKey} groups={groups} />
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}
