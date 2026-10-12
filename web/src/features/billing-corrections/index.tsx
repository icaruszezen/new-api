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
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatNumber, formatTimestampToDate } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import {
  createBillingCorrectionRule,
  deleteBillingCorrectionRule,
  getBillingCorrectionRules,
  updateBillingCorrectionRule,
} from './api'
import { RecordsPanel } from './components/records-panel'
import { RuleEditorDialog } from './components/rule-editor-dialog'
import { getChannelLabel } from './lib/records'
import {
  transformFormValuesToPayload,
  type RuleFormValues,
} from './lib/rule-form'
import type { BillingCorrectionRule } from './types'

const RULES_QUERY_KEY = ['billing-correction-rules']

type PageTab = 'rules' | 'records'

export function BillingCorrections() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [tab, setTab] = useState<PageTab>('rules')
  const [editorOpen, setEditorOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<BillingCorrectionRule | null>(
    null
  )
  const [deletingRule, setDeletingRule] =
    useState<BillingCorrectionRule | null>(null)

  const { data: rules, isLoading } = useQuery({
    queryKey: RULES_QUERY_KEY,
    queryFn: getBillingCorrectionRules,
  })

  const saveMutation = useMutation({
    mutationFn: async (values: RuleFormValues) => {
      const payload = transformFormValuesToPayload(values)
      return editingRule
        ? updateBillingCorrectionRule(editingRule.id, payload)
        : createBillingCorrectionRule(payload)
    },
    onSuccess: (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to save'))
        return
      }
      toast.success(t('Saved successfully'))
      setEditorOpen(false)
      setEditingRule(null)
      void queryClient.invalidateQueries({ queryKey: RULES_QUERY_KEY })
    },
    onError: handleServerError,
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteBillingCorrectionRule(id),
    onSuccess: (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to delete'))
        return
      }
      toast.success(t('Deleted successfully'))
      setDeletingRule(null)
      void queryClient.invalidateQueries({ queryKey: RULES_QUERY_KEY })
    },
    onError: handleServerError,
  })

  const openEditor = (rule: BillingCorrectionRule | null) => {
    setEditingRule(rule)
    setEditorOpen(true)
  }

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Billing Correction')}
        </SectionPageLayout.Title>
        {tab === 'rules' && (
          <SectionPageLayout.Actions>
            <Button size='sm' onClick={() => openEditor(null)}>
              <Plus className='mr-1 h-4 w-4' />
              {t('Add Correction Rule')}
            </Button>
          </SectionPageLayout.Actions>
        )}
        <SectionPageLayout.Content>
          <div className='space-y-4'>
            <Tabs
              value={tab}
              onValueChange={(value) => setTab(value as PageTab)}
            >
              <TabsList>
                <TabsTrigger value='rules'>{t('Correction rules')}</TabsTrigger>
                <TabsTrigger value='records'>
                  {t('Mis-billed records')}
                </TabsTrigger>
              </TabsList>
            </Tabs>

            {tab === 'records' && <RecordsPanel rules={rules ?? []} />}

            {tab === 'rules' && (
              <>
                <Alert>
                  <AlertDescription className='text-xs'>
                    {t(
                      'A request on a channel with an enabled rule is not charged when its input tokens are strictly greater than the threshold, it produced zero output tokens, and the upstream returned no usage information, so tokens were estimated locally.'
                    )}{' '}
                    {t(
                      'Only requests made after a rule is enabled are affected; existing logs are not changed.'
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
                    data={rules ?? []}
                    getRowKey={(rule) => rule.id}
                    emptyClassName='text-muted-foreground py-8'
                    emptyContent={t('No billing correction rules yet')}
                    columns={[
                      {
                        id: 'channel',
                        header: t('Channel'),
                        cellClassName: 'max-w-[280px] font-medium',
                        cell: (rule) =>
                          getChannelLabel(rule.channel_id, rule.channel_name),
                      },
                      {
                        id: 'threshold',
                        header: t('Input token threshold'),
                        cellClassName: 'font-mono tabular-nums',
                        cell: (rule) =>
                          `${formatNumber(rule.prompt_tokens_threshold)} ${t('tokens')}`,
                      },
                      {
                        id: 'status',
                        header: t('Status'),
                        cell: (rule) => (
                          <StatusBadge
                            label={rule.enabled ? t('Enabled') : t('Disabled')}
                            variant={rule.enabled ? 'success' : 'neutral'}
                            copyable={false}
                          />
                        ),
                      },
                      {
                        id: 'upstream',
                        header: t('Upstream check'),
                        cell: (rule) => {
                          if (!rule.upstream_check_enabled) {
                            return (
                              <span className='text-muted-foreground'>-</span>
                            )
                          }
                          return (
                            <StatusBadge
                              label={
                                rule.auto_refund
                                  ? t('On, returns quota automatically')
                                  : t('On')
                              }
                              variant='info'
                              copyable={false}
                            />
                          )
                        },
                      },
                      {
                        id: 'updated',
                        header: t('Updated'),
                        cellClassName: 'font-mono whitespace-nowrap',
                        cell: (rule) =>
                          formatTimestampToDate(rule.updated_time),
                      },
                      {
                        id: 'actions',
                        header: t('Actions'),
                        className: 'text-right',
                        cellClassName: 'text-right',
                        cell: (rule) => (
                          <StaticRowActions
                            editLabel={t('Edit')}
                            deleteLabel={t('Delete')}
                            menuLabel={t('Actions')}
                            onEdit={() => openEditor(rule)}
                            onDelete={() => setDeletingRule(rule)}
                          />
                        ),
                      },
                    ]}
                  />
                )}
              </>
            )}
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <RuleEditorDialog
        open={editorOpen}
        onOpenChange={(open) => {
          setEditorOpen(open)
          if (!open) setEditingRule(null)
        }}
        rule={editingRule}
        isSubmitting={saveMutation.isPending}
        onSubmit={(values) => saveMutation.mutate(values)}
      />

      <ConfirmDialog
        open={deletingRule !== null}
        onOpenChange={(open) => {
          if (!open) setDeletingRule(null)
        }}
        title={t('Delete this billing correction rule?')}
        desc={
          deletingRule
            ? getChannelLabel(
                deletingRule.channel_id,
                deletingRule.channel_name
              )
            : ''
        }
        confirmText={t('Delete')}
        destructive
        isLoading={deleteMutation.isPending}
        handleConfirm={() => {
          if (deletingRule) deleteMutation.mutate(deletingRule.id)
        }}
      />
    </>
  )
}
