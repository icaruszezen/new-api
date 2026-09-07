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
import { ArrowDown, ArrowUp, Pencil, Plus, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { getGroups } from '@/features/users/api'
import { getLobeIcon } from '@/lib/lobe-icon'

import { getChannelMonitoringConfig } from './api'
import {
  MonitorDialog,
  type MonitorFormValues,
} from './components/monitor-dialog'
import { MAX_MONITORS, monitorPairKey } from './constants'
import { useSaveMonitors } from './hooks/use-save-monitors'
import type { AdminMonitor } from './types'

function MonitorRow(props: {
  monitor: AdminMonitor
  isFirst: boolean
  isLast: boolean
  onEdit: () => void
  onRemove: () => void
  onMove: (direction: -1 | 1) => void
}) {
  const { t } = useTranslation()

  return (
    <li className='border-border/60 bg-card flex flex-wrap items-center gap-3 rounded-lg border px-3 py-2.5'>
      <span aria-hidden='true' className='flex shrink-0 items-center'>
        {getLobeIcon(props.monitor.resolved_icon, 20)}
      </span>

      <div className='min-w-0 flex-1'>
        <div className='flex items-center gap-2'>
          <span className='truncate text-sm font-medium'>
            {props.monitor.name}
          </span>
          {!props.monitor.enabled && (
            <Badge variant='secondary'>{t('Disabled')}</Badge>
          )}
        </div>
        <div className='text-muted-foreground truncate text-xs'>
          {props.monitor.group} · {props.monitor.model}
        </div>
      </div>

      <div className='flex shrink-0 items-center gap-1'>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          aria-label={t('Move up')}
          disabled={props.isFirst}
          onClick={() => props.onMove(-1)}
        >
          <ArrowUp className='size-4' />
        </Button>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          aria-label={t('Move down')}
          disabled={props.isLast}
          onClick={() => props.onMove(1)}
        >
          <ArrowDown className='size-4' />
        </Button>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          aria-label={t('Edit')}
          onClick={props.onEdit}
        >
          <Pencil className='size-4' />
        </Button>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          aria-label={t('Delete')}
          onClick={props.onRemove}
        >
          <Trash2 className='text-destructive size-4' />
        </Button>
      </div>
    </li>
  )
}

export function ChannelMonitoringSettings() {
  const { t } = useTranslation()

  const configQuery = useQuery({
    queryKey: ['channel-monitoring-config'],
    queryFn: getChannelMonitoringConfig,
  })
  const groupsQuery = useQuery({
    queryKey: ['groups'],
    queryFn: getGroups,
  })
  const { saveMonitors, setEnabled, isSaving } = useSaveMonitors()

  const [monitors, setMonitors] = useState<AdminMonitor[]>([])
  const [enabled, setEnabledState] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editIndex, setEditIndex] = useState<number | null>(null)

  useEffect(() => {
    if (!configQuery.data) return
    setMonitors(configQuery.data.monitors)
    setEnabledState(configQuery.data.enabled)
  }, [configQuery.data])

  const handleToggleEnabled = async (checked: boolean) => {
    setEnabledState(checked)
    const ok = await setEnabled(checked)
    if (!ok) setEnabledState(!checked)
  }

  const persist = async (next: AdminMonitor[]) => {
    const previous = monitors
    setMonitors(next)
    // A rejected save must not leave the optimistic list on screen, otherwise
    // the admin sees a monitor that the server never accepted.
    if (!(await saveMonitors(next))) {
      setMonitors(previous)
    }
  }

  const handleSave = (values: MonitorFormValues) => {
    const base: AdminMonitor = {
      id: editIndex === null ? '' : monitors[editIndex].id,
      name: values.name,
      group: values.group,
      model: values.model,
      icon: values.icon,
      enabled: values.enabled,
      sort: editIndex === null ? monitors.length : monitors[editIndex].sort,
      resolved_icon: values.icon,
    }
    const next =
      editIndex === null
        ? [...monitors, base]
        : monitors.map((monitor, index) =>
            index === editIndex ? base : monitor
          )
    void persist(next)
  }

  const handleMove = (index: number, direction: -1 | 1) => {
    const target = index + direction
    if (target < 0 || target >= monitors.length) return
    const next = [...monitors]
    ;[next[index], next[target]] = [next[target], next[index]]
    void persist(next)
  }

  const openAddDialog = () => {
    if (monitors.length >= MAX_MONITORS) {
      toast.error(t('At most {{count}} monitors can be configured.', {
        count: MAX_MONITORS,
      }))
      return
    }
    setEditIndex(null)
    setDialogOpen(true)
  }

  // Editing a row must not clash with itself, so its own pair is excluded.
  const takenPairs = monitors
    .filter((_, index) => index !== editIndex)
    .map((monitor) => monitorPairKey(monitor.group, monitor.model))

  const renderMonitorList = () => {
    if (configQuery.isLoading) {
      return (
        <div className='space-y-2'>
          <Skeleton className='h-14 rounded-lg' />
          <Skeleton className='h-14 rounded-lg' />
        </div>
      )
    }
    if (monitors.length === 0) {
      return (
        <p className='text-muted-foreground border-border/60 rounded-lg border border-dashed px-4 py-8 text-center text-sm'>
          {t('No monitors have been configured yet.')}
        </p>
      )
    }
    return (
      <ul className='space-y-2'>
        {monitors.map((monitor, index) => (
          <MonitorRow
            key={monitor.id || monitorPairKey(monitor.group, monitor.model)}
            monitor={monitor}
            isFirst={index === 0}
            isLast={index === monitors.length - 1}
            onEdit={() => {
              setEditIndex(index)
              setDialogOpen(true)
            }}
            onRemove={() =>
              void persist(monitors.filter((_, i) => i !== index))
            }
            onMove={(direction) => handleMove(index, direction)}
          />
        ))}
      </ul>
    )
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Channel Monitoring Settings')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button type='button' onClick={openAddDialog} disabled={isSaving}>
          <Plus className='size-4' />
          {t('Add monitor')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='mx-auto max-w-3xl space-y-6'>
          <div className='border-border/60 flex flex-wrap items-center justify-between gap-4 rounded-lg border px-4 py-3'>
            <div className='space-y-1'>
              <div className='text-sm font-medium'>
                {t('Enable public status page')}
              </div>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Publishes /channel-monitoring and starts collecting samples from streaming requests.'
                )}
              </p>
            </div>
            <Switch
              checked={enabled}
              onCheckedChange={(checked) => void handleToggleEnabled(checked)}
              disabled={isSaving || configQuery.isLoading}
              aria-label={t('Enable public status page')}
            />
          </div>

          <section className='space-y-3'>
            <div className='space-y-1'>
              <h3 className='text-sm font-medium'>{t('Monitors')}</h3>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Each monitor watches one model inside one group and renders as one card. Endpoint ping measures the network latency to the upstream the group resolves to, or to its proxy when one is configured.'
                )}
              </p>
            </div>

            {renderMonitorList()}
          </section>
        </div>

        <MonitorDialog
          open={dialogOpen}
          onOpenChange={(open) => {
            setDialogOpen(open)
            if (!open) setEditIndex(null)
          }}
          editData={editIndex === null ? undefined : monitors[editIndex]}
          groups={groupsQuery.data?.data ?? []}
          takenPairs={takenPairs}
          onSave={handleSave}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
