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

import { updateIqCheckConfig } from '../api'
import {
  IQ_CHECK_CONFIG_QUERY_KEY,
  IQ_CHECK_STATUS_QUERY_KEY,
} from '../constants'
import type { IqCheck, IqCheckConfigUpdate } from '../types'

/**
 * Persists IQ check configuration through the admin endpoint so the server
 * validates groups, models and bounds and resolves provider icons.
 */
export function useSaveChecks() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [isSaving, setIsSaving] = useState(false)
  const savingRef = useRef(false)

  const save = async (update: IqCheckConfigUpdate) => {
    if (savingRef.current) {
      return false
    }
    savingRef.current = true
    setIsSaving(true)
    try {
      const response = await updateIqCheckConfig(update)
      if (!response.success) {
        toast.error(response.message || t('Failed to update setting'))
        return false
      }
      queryClient.invalidateQueries({ queryKey: IQ_CHECK_CONFIG_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: IQ_CHECK_STATUS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: ['status'] })
      toast.success(t('Setting updated successfully'))
      return true
    } catch {
      // Transport errors were already reported by the shared API client.
      return false
    } finally {
      savingRef.current = false
      setIsSaving(false)
    }
  }

  const saveChecks = (checks: IqCheck[]) =>
    save({
      // Sort is rewritten from list order so reordering rows is enough to
      // change the card order on the public page.
      checks: checks.map((check, index) => ({
        id: check.id,
        name: check.name,
        group: check.group,
        model: check.model,
        prompt: check.prompt,
        interval_minutes: check.interval_minutes,
        max_tokens: check.max_tokens,
        endpoint_type: check.endpoint_type,
        reasoning_effort: check.reasoning_effort,
        reasoning_summary: check.reasoning_summary,
        icon: check.icon,
        enabled: check.enabled,
        sort: index,
      })),
    })

  const setEnabled = (enabled: boolean) => save({ enabled })

  const saveGeneral = (general: {
    history_limit: number
    request_timeout_seconds: number
  }) => save(general)

  return { saveChecks, setEnabled, saveGeneral, isSaving }
}
