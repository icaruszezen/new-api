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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { runIqCheck } from '../api'
import {
  IQ_CHECK_CONFIG_QUERY_KEY,
  IQ_CHECK_STATUS_QUERY_KEY,
} from '../constants'

/** Asks the scheduler to run one check on its next pass. */
export function useRunCheck() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [runningId, setRunningId] = useState<string | null>(null)

  const runCheck = async (id: string) => {
    setRunningId(id)
    try {
      const response = await runIqCheck(id)
      if (!response.success) {
        toast.error(response.message || t('Failed to queue the IQ check'))
        return false
      }
      queryClient.invalidateQueries({ queryKey: IQ_CHECK_CONFIG_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: IQ_CHECK_STATUS_QUERY_KEY })
      toast.success(response.message || t('IQ check queued.'))
      return true
    } catch {
      // Transport errors were already reported by the shared API client.
      return false
    } finally {
      setRunningId(null)
    }
  }

  return { runCheck, runningId }
}
