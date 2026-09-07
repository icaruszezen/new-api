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
import { useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { updateChannelMonitoringConfig } from '../api'
import type { ChannelMonitor } from '../types'

/**
 * Persists channel monitoring configuration through the admin endpoint so the
 * server validates groups and models and resolves provider icons.
 */
export function useSaveMonitors() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [isSaving, setIsSaving] = useState(false)
  const savingRef = useRef(false)

  const save = async (update: {
    enabled?: boolean
    monitors?: ChannelMonitor[]
  }) => {
    if (savingRef.current) {
      return false
    }
    savingRef.current = true
    setIsSaving(true)
    try {
      const response = await updateChannelMonitoringConfig(update)
      if (!response.success) {
        toast.error(response.message || t('Failed to update setting'))
        return false
      }
      queryClient.invalidateQueries({ queryKey: ['channel-monitoring-config'] })
      queryClient.invalidateQueries({ queryKey: ['channel-monitoring-status'] })
      queryClient.invalidateQueries({ queryKey: ['status'] })
      toast.success(t('Setting updated successfully'))
      return true
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to update setting')
      )
      return false
    } finally {
      savingRef.current = false
      setIsSaving(false)
    }
  }

  const saveMonitors = (monitors: ChannelMonitor[]) =>
    save({
      // Sort is rewritten from list order so reordering rows is enough to
      // change the card order on the status page.
      monitors: monitors.map((monitor, index) => ({
        id: monitor.id,
        name: monitor.name,
        group: monitor.group,
        model: monitor.model,
        icon: monitor.icon,
        enabled: monitor.enabled,
        uptime_scope: monitor.uptime_scope,
        sort: index,
      })),
    })

  const setEnabled = (enabled: boolean) => save({ enabled })

  return { saveMonitors, setEnabled, isSaving }
}
