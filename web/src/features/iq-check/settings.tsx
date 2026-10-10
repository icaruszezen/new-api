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
import { Plus } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { getGroups } from '@/features/users/api'

import { getIqCheckConfig } from './api'
import { CheckDialog } from './components/check-dialog'
import { CheckRow } from './components/check-row'
import { GeneralSettingsCard } from './components/general-settings-card'
import {
  DEFAULT_HISTORY_LIMIT,
  DEFAULT_REQUEST_TIMEOUT_SECONDS,
  IQ_CHECK_CONFIG_QUERY_KEY,
  MAX_CHECKS,
} from './constants'
import { useRunCheck } from './hooks/use-run-check'
import { useSaveChecks } from './hooks/use-save-checks'
import type { CheckFormValues } from './lib/forms'
import type { AdminIqCheck } from './types'

export function IqCheckSettings() {
  const { t } = useTranslation()

  const configQuery = useQuery({
    queryKey: IQ_CHECK_CONFIG_QUERY_KEY,
    queryFn: getIqCheckConfig,
    // A background refetch replaces the rows, which would reset an open edit
    // dialog; the list is refreshed after every save and run request instead.
    refetchOnWindowFocus: false,
  })
  const groupsQuery = useQuery({
    queryKey: ['groups'],
    queryFn: getGroups,
  })
  const { saveChecks, setEnabled, saveGeneral, isSaving } = useSaveChecks()
  const { runCheck, runningId } = useRunCheck()

  const [checks, setChecks] = useState<AdminIqCheck[]>([])
  const [enabled, setEnabledState] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editIndex, setEditIndex] = useState<number | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<AdminIqCheck | null>(null)

  useEffect(() => {
    if (!configQuery.data) return
    setChecks(configQuery.data.checks)
    setEnabledState(configQuery.data.enabled)
  }, [configQuery.data])

  const handleToggleEnabled = async (checked: boolean) => {
    setEnabledState(checked)
    const ok = await setEnabled(checked)
    if (!ok) setEnabledState(!checked)
  }

  const persistInFlight = useRef(false)
  const persist = async (next: AdminIqCheck[]) => {
    // The server treats the list as complete and deletes the history of every
    // check missing from it, so nothing is saved until the real list is loaded.
    if (!configQuery.data || persistInFlight.current) return false
    persistInFlight.current = true
    const previous = checks
    setChecks(next)
    // A rejected save must not leave the optimistic list on screen, otherwise
    // the admin sees a check that the server never accepted.
    try {
      const ok = await saveChecks(next)
      if (!ok) setChecks(previous)
      return ok
    } finally {
      persistInFlight.current = false
    }
  }

  const handleSave = (values: CheckFormValues) => {
    const current = editIndex === null ? undefined : checks[editIndex]
    const saved: AdminIqCheck = {
      // Run state is owned by the server; a new check starts without any.
      last_started_at: 0,
      last_finished_at: 0,
      run_requested_at: 0,
      last_status: '',
      last_checked_at: 0,
      last_error: '',
      ...current,
      id: current?.id ?? '',
      name: values.name,
      group: values.group,
      model: values.model,
      prompt: values.prompt,
      interval_minutes: values.interval_minutes,
      max_tokens: values.max_tokens,
      icon: values.icon,
      enabled: values.enabled,
      sort: current?.sort ?? checks.length,
      resolved_icon: values.icon || current?.resolved_icon || '',
    }
    const next =
      editIndex === null
        ? [...checks, saved]
        : checks.map((check, index) => (index === editIndex ? saved : check))
    return persist(next)
  }

  const handleMove = (index: number, direction: -1 | 1) => {
    const target = index + direction
    if (target < 0 || target >= checks.length) return
    const next = [...checks]
    ;[next[index], next[target]] = [next[target], next[index]]
    void persist(next)
  }

  const openAddDialog = () => {
    if (!configQuery.data) return
    if (checks.length >= MAX_CHECKS) {
      toast.error(
        t('At most {{count}} IQ checks can be configured.', {
          count: MAX_CHECKS,
        })
      )
      return
    }
    setEditIndex(null)
    setDialogOpen(true)
  }

  const requestTimeoutSeconds =
    configQuery.data?.request_timeout_seconds ?? DEFAULT_REQUEST_TIMEOUT_SECONDS

  const renderCheckList = () => {
    if (configQuery.isLoading) {
      return (
        <div className='space-y-2'>
          <Skeleton className='h-16 rounded-lg' />
          <Skeleton className='h-16 rounded-lg' />
        </div>
      )
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
    if (checks.length === 0) {
      return (
        <p className='text-muted-foreground border-border/60 rounded-lg border border-dashed px-4 py-8 text-center text-sm'>
          {t('No IQ checks have been configured yet.')}
        </p>
      )
    }
    return (
      <ul className='space-y-2'>
        {checks.map((check, index) => (
          <CheckRow
            key={check.id || `new-${check.sort}`}
            check={check}
            isFirst={index === 0}
            isLast={index === checks.length - 1}
            featureEnabled={enabled}
            requestTimeoutSeconds={requestTimeoutSeconds}
            runPending={runningId === check.id}
            onRun={() => void runCheck(check.id)}
            onEdit={() => {
              setEditIndex(index)
              setDialogOpen(true)
            }}
            onRemove={() => setDeleteTarget(check)}
            onMove={(direction) => handleMove(index, direction)}
            disabled={isSaving}
          />
        ))}
      </ul>
    )
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('IQ Check Settings')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          type='button'
          onClick={openAddDialog}
          disabled={isSaving || !configQuery.data}
        >
          <Plus className='size-4' />
          {t('Add check')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='mx-auto max-w-3xl space-y-6'>
          <div className='border-border/60 flex flex-wrap items-center justify-between gap-4 rounded-lg border px-4 py-3'>
            <div className='space-y-1'>
              <div className='text-sm font-medium'>
                {t('Enable IQ check page')}
              </div>
              <p className='text-muted-foreground text-xs'>
                {t('Publishes /iq-check and starts running scheduled checks.')}
              </p>
            </div>
            <Switch
              checked={enabled}
              onCheckedChange={(checked) => void handleToggleEnabled(checked)}
              disabled={isSaving || configQuery.isLoading}
              aria-label={t('Enable IQ check page')}
            />
          </div>

          <GeneralSettingsCard
            values={{
              history_limit:
                configQuery.data?.history_limit ?? DEFAULT_HISTORY_LIMIT,
              request_timeout_seconds: requestTimeoutSeconds,
            }}
            disabled={isSaving || !configQuery.data}
            onSave={saveGeneral}
          />

          <section className='space-y-3'>
            <div className='space-y-1'>
              <h3 className='text-sm font-medium'>{t('Checks')}</h3>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Each check sends its prompt to one model in one group through the normal relay path and publishes the answer as one card. Checks run automatically at their interval; editing the prompt, group, model or max tokens queues a fresh run.'
                )}
              </p>
            </div>

            {renderCheckList()}
          </section>
        </div>

        <CheckDialog
          open={dialogOpen}
          onOpenChange={(open) => {
            setDialogOpen(open)
            if (!open) setEditIndex(null)
          }}
          editData={editIndex === null ? undefined : checks[editIndex]}
          groups={groupsQuery.data?.data ?? []}
          onSave={handleSave}
        />

        <ConfirmDialog
          open={deleteTarget !== null}
          onOpenChange={(open) => {
            if (!open) setDeleteTarget(null)
          }}
          title={t('Delete IQ check')}
          desc={t(
            'This will delete {{name}} and all of its recorded results. This action cannot be undone.',
            { name: deleteTarget?.name ?? '' }
          )}
          confirmText={t('Delete')}
          destructive
          isLoading={isSaving}
          handleConfirm={() => {
            if (!deleteTarget) return
            void (async () => {
              const ok = await persist(
                checks.filter((check) =>
                  deleteTarget.id
                    ? check.id !== deleteTarget.id
                    : check !== deleteTarget
                )
              )
              if (ok) setDeleteTarget(null)
            })()
          }}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
