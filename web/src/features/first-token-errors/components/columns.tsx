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
/* eslint-disable react-refresh/only-export-components */
import type { ColumnDef } from '@tanstack/react-table'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { formatLogQuota, formatTimestampToDate } from '@/lib/format'

import type { FirstTokenErrorLog } from '../types'
import { FirstTokenErrorDetailsDialog } from './details-dialog'

export function useFirstTokenErrorColumns(): ColumnDef<FirstTokenErrorLog>[] {
  const { t } = useTranslation()

  return [
    {
      accessorKey: 'created_at',
      header: t('Time'),
      cell: ({ row }) => formatTimestampToDate(row.original.created_at),
    },
    {
      accessorKey: 'username',
      header: t('User'),
    },
    {
      accessorKey: 'token_name',
      header: t('Token'),
    },
    {
      accessorKey: 'model_name',
      header: t('Model'),
    },
    {
      accessorKey: 'channel_name',
      header: t('Channel'),
    },
    {
      accessorKey: 'group',
      header: t('Group'),
    },
    {
      accessorKey: 'prompt_tokens',
      header: t('Input tokens'),
    },
    {
      accessorKey: 'completion_tokens',
      header: t('Output tokens'),
    },
    {
      accessorKey: 'would_be_quota',
      header: t('Would-be quota'),
      cell: ({ row }) => formatLogQuota(row.original.would_be_quota),
    },
    {
      accessorKey: 'use_time',
      header: t('Duration'),
      cell: ({ row }) => `${row.original.use_time}s`,
    },
    {
      accessorKey: 'error_message',
      header: t('Error Message'),
      cell: ({ row }) => (
        <span className='line-clamp-2 max-w-64'>
          {row.original.error_message || '-'}
        </span>
      ),
    },
    {
      accessorKey: 'request_id',
      header: t('Request ID'),
    },
    {
      accessorKey: 'has_request_body',
      header: t('Request body'),
      cell: ({ row }) => (row.original.has_request_body ? t('Yes') : t('No')),
    },
    {
      id: 'details',
      header: t('Details'),
      cell: ({ row }) => <DetailsCell log={row.original} />,
    },
  ]
}

function DetailsCell(props: { log: FirstTokenErrorLog }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  return (
    <>
      <Button variant='ghost' size='sm' onClick={() => setOpen(true)}>
        {t('Details')}
      </Button>
      <FirstTokenErrorDetailsDialog
        log={props.log}
        open={open}
        onOpenChange={setOpen}
      />
    </>
  )
}
