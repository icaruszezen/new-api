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
import { FileText, Plus, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { useApiInfo } from '@/features/dashboard/hooks/use-status-data'
import { getApiKeys } from '@/features/keys/api'
import { ApiKeysDialogs } from '@/features/keys/components/api-keys-dialogs'
import {
  ApiKeysProvider,
  useApiKeys,
} from '@/features/keys/components/api-keys-provider'
import { useStatus } from '@/hooks/use-status'

import { ConsoleEndpointChips } from './endpoint-chips'
import { ConsoleKeyList } from './key-list'

function KeyManagementCard() {
  const { t } = useTranslation()
  const apiInfo = useApiInfo()
  const { status } = useStatus()
  const { refreshTrigger, setOpen } = useApiKeys()
  const docsLink = status?.docs_link as string | undefined

  const keysQuery = useQuery({
    queryKey: ['console', 'home', 'api-keys', refreshTrigger],
    queryFn: async () => {
      const result = await getApiKeys({ p: 1, size: 50 })
      return result.success ? (result.data?.items ?? []) : []
    },
    staleTime: 60 * 1000,
  })

  const keys = keysQuery.data ?? []

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
            <Zap className='size-3.5' aria-hidden='true' />
            {t('Test Connection')}
          </button>
          <a
            href={docsLink || '/docs'}
            target={docsLink ? '_blank' : undefined}
            rel={docsLink ? 'noopener noreferrer' : undefined}
            className='border-border inline-flex h-9 items-center gap-1.5 rounded-full border px-3.5 text-sm font-medium'
          >
            <FileText className='size-3.5' aria-hidden='true' />
            {t('Usage docs')}
          </a>
          <button
            type='button'
            className='bg-foreground text-background inline-flex h-9 items-center gap-1.5 rounded-full px-3.5 text-sm font-medium transition-opacity hover:opacity-85'
            onClick={() => setOpen('create')}
          >
            <Plus className='size-3.5' aria-hidden='true' />
            {t('New key')}
          </button>
        </div>
      </div>

      <div className='border-border/70 flex flex-col gap-3 border-t px-5 py-3 sm:flex-row sm:items-center'>
        <ConsoleEndpointChips items={apiInfo.items} />
      </div>

      <ConsoleKeyList isLoading={keysQuery.isLoading} keys={keys} />
    </section>
  )
}

export function ConsoleKeyManagement() {
  return (
    <ApiKeysProvider>
      <KeyManagementCard />
      <ApiKeysDialogs />
    </ApiKeysProvider>
  )
}
