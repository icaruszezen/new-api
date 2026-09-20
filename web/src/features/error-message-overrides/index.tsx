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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { StaticDataTable, StaticRowActions } from '@/components/data-table'
import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { handleServerError } from '@/lib/handle-server-error'

import {
  createErrorMessageOverride,
  deleteErrorMessageOverride,
  getErrorMessageOverrides,
  updateErrorMessageOverride,
} from './api'
import { OverrideEditorDialog } from './components/override-editor-dialog'
import {
  transformFormValuesToPayload,
  type OverrideFormValues,
} from './lib/override-form'
import type { ErrorMessageOverride } from './types'

const OVERRIDES_QUERY_KEY = ['error-message-overrides']

export function ErrorMessageOverrides() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [editorOpen, setEditorOpen] = useState(false)
  const [editingOverride, setEditingOverride] =
    useState<ErrorMessageOverride | null>(null)
  const [deletingOverride, setDeletingOverride] =
    useState<ErrorMessageOverride | null>(null)

  const { data: overrides, isLoading } = useQuery({
    queryKey: OVERRIDES_QUERY_KEY,
    queryFn: getErrorMessageOverrides,
  })

  const saveMutation = useMutation({
    mutationFn: async (values: OverrideFormValues) => {
      const payload = transformFormValuesToPayload(values)
      return editingOverride
        ? updateErrorMessageOverride(editingOverride.id, payload)
        : createErrorMessageOverride(payload)
    },
    onSuccess: (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to save'))
        return
      }
      toast.success(t('Saved successfully'))
      setEditorOpen(false)
      setEditingOverride(null)
      void queryClient.invalidateQueries({ queryKey: OVERRIDES_QUERY_KEY })
    },
    onError: handleServerError,
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteErrorMessageOverride(id),
    onSuccess: (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to delete'))
        return
      }
      toast.success(t('Deleted successfully'))
      setDeletingOverride(null)
      void queryClient.invalidateQueries({ queryKey: OVERRIDES_QUERY_KEY })
    },
    onError: handleServerError,
  })

  const openEditor = (override: ErrorMessageOverride | null) => {
    setEditingOverride(override)
    setEditorOpen(true)
  }

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Error Message Override')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <Button size='sm' onClick={() => openEditor(null)}>
            <Plus className='mr-1 h-4 w-4' />
            {t('Add Override Rule')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='space-y-4'>
            <Alert>
              <AlertDescription className='text-xs'>
                {t(
                  'When an upstream error message contains the match text, the whole message returned to the user is replaced.'
                )}{' '}
                {t(
                  'Error logs, channel auto-disable and diagnostics keep the original upstream message.'
                )}
              </AlertDescription>
            </Alert>

            {isLoading ? (
              <div className='space-y-2'>
                <Skeleton className='h-10 w-full' />
                <Skeleton className='h-10 w-full' />
                <Skeleton className='h-10 w-full' />
              </div>
            ) : (
              <StaticDataTable
                tableClassName='min-w-max'
                data={overrides ?? []}
                getRowKey={(override) => override.id}
                emptyClassName='text-muted-foreground py-8'
                emptyContent={t('No override rules yet')}
                columns={[
                  {
                    id: 'match',
                    header: t('Match text'),
                    cellClassName: 'max-w-[280px] font-medium',
                    cell: (override) => override.match_substring,
                  },
                  {
                    id: 'replacement',
                    header: t('Replacement message'),
                    cellClassName: 'max-w-[320px]',
                    cell: (override) => override.replacement_message,
                  },
                  {
                    id: 'scope',
                    header: t('Applies to'),
                    cell: (override) => (
                      <ScopeCell
                        channelId={override.channel_id}
                        channelName={override.channel_name}
                      />
                    ),
                  },
                  {
                    id: 'priority',
                    header: t('Priority'),
                    cell: (override) => override.priority,
                  },
                  {
                    id: 'status',
                    header: t('Status'),
                    cell: (override) => (
                      <StatusBadge
                        label={override.enabled ? t('Enabled') : t('Disabled')}
                        variant={override.enabled ? 'success' : 'neutral'}
                        copyable={false}
                      />
                    ),
                  },
                  {
                    id: 'actions',
                    header: t('Actions'),
                    className: 'text-right',
                    cellClassName: 'text-right',
                    cell: (override) => (
                      <StaticRowActions
                        editLabel={t('Edit')}
                        deleteLabel={t('Delete')}
                        menuLabel={t('Actions')}
                        onEdit={() => openEditor(override)}
                        onDelete={() => setDeletingOverride(override)}
                      />
                    ),
                  },
                ]}
              />
            )}
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <OverrideEditorDialog
        open={editorOpen}
        onOpenChange={(open) => {
          setEditorOpen(open)
          if (!open) setEditingOverride(null)
        }}
        override={editingOverride}
        isSubmitting={saveMutation.isPending}
        onSubmit={(values) => saveMutation.mutate(values)}
      />

      <ConfirmDialog
        open={deletingOverride !== null}
        onOpenChange={(open) => {
          if (!open) setDeletingOverride(null)
        }}
        title={t('Delete this override rule?')}
        desc={deletingOverride?.match_substring ?? ''}
        confirmText={t('Delete')}
        destructive
        isLoading={deleteMutation.isPending}
        handleConfirm={() => {
          if (deletingOverride) deleteMutation.mutate(deletingOverride.id)
        }}
      />
    </>
  )
}

function ScopeCell(props: { channelId: number; channelName?: string }) {
  const { t } = useTranslation()

  if (props.channelId <= 0) {
    return (
      <StatusBadge label={t('All channels')} variant='info' copyable={false} />
    )
  }
  return (
    <StatusBadge
      label={props.channelName || `#${props.channelId}`}
      variant='neutral'
      copyable={false}
    />
  )
}
