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

import { resetChannelMonitor } from '../api'

export function useResetMonitor() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [isResetting, setIsResetting] = useState(false)

  const resetMonitor = async (id: string) => {
    setIsResetting(true)
    try {
      const response = await resetChannelMonitor(id)
      if (!response.success) {
        toast.error(response.message || t('Failed to reset monitoring history'))
        return false
      }
      queryClient.invalidateQueries({ queryKey: ['channel-monitoring-config'] })
      queryClient.invalidateQueries({ queryKey: ['channel-monitoring-status'] })
      toast.success(t('Monitoring history has been reset.'))
      return true
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to reset monitoring history')
      )
      return false
    } finally {
      setIsResetting(false)
    }
  }

  return { resetMonitor, isResetting }
}
