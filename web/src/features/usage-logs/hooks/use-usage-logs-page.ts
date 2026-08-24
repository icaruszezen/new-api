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
import { getRouteApi, useNavigate } from '@tanstack/react-router'
import { useCallback, useMemo } from 'react'

import type { NavGroup } from '@/components/layout/types'
import { useSidebarConfig } from '@/hooks/use-sidebar-config'

import {
  type LogsViewScope,
  useLogsViewScope,
  useUsageLogsContext,
} from '../components/usage-logs-provider'
import {
  isUsageLogsSectionId,
  USAGE_LOGS_DEFAULT_SECTION,
  type UsageLogsSectionId,
} from '../section-registry'

const route = getRouteApi('/_authenticated/usage-logs/$section')
const TASK_LOG_SECTIONS = ['drawing', 'task'] as const

const SECTION_META: Record<UsageLogsSectionId, { titleKey: string }> = {
  common: {
    titleKey: 'Common Logs',
  },
  drawing: {
    titleKey: 'Drawing Logs',
  },
  task: {
    titleKey: 'Task Logs',
  },
}

export function useUsageLogsPage() {
  const navigate = useNavigate()
  const params = route.useParams()
  const activeCategory: UsageLogsSectionId =
    params.section && isUsageLogsSectionId(params.section)
      ? params.section
      : USAGE_LOGS_DEFAULT_SECTION
  const {
    selectedUserId,
    userInfoDialogOpen,
    setUserInfoDialogOpen,
    affinityTarget,
    affinityDialogOpen,
    setAffinityDialogOpen,
  } = useUsageLogsContext()
  const { canManageScope, viewScope, setViewScope } = useLogsViewScope()
  const tabNavGroups = useMemo<NavGroup[]>(
    () => [
      {
        title: 'Task Logs',
        items: TASK_LOG_SECTIONS.map((section) => ({
          title: SECTION_META[section].titleKey,
          url: `/usage-logs/${section}`,
        })),
      },
    ],
    []
  )
  const filteredTabGroups = useSidebarConfig(tabNavGroups)
  const visibleSectionTabs = useMemo(
    () =>
      (filteredTabGroups[0]?.items ?? [])
        .map((item) => {
          if (!('url' in item) || typeof item.url !== 'string') return null
          const section = item.url.split('/').pop() ?? null
          if (!section || !isUsageLogsSectionId(section)) return null
          return {
            id: section,
            titleKey: SECTION_META[section].titleKey,
          }
        })
        .filter(
          (section): section is { id: UsageLogsSectionId; titleKey: string } =>
            section != null
        ),
    [filteredTabGroups]
  )

  const handleSectionChange = useCallback(
    (section: string) => {
      void navigate({
        to: '/usage-logs/$section',
        params: { section: section as UsageLogsSectionId },
      })
    },
    [navigate]
  )

  const handleViewScopeChange = useCallback(
    (scope: string) => {
      if (scope === 'all' || scope === 'self') {
        setViewScope(scope as LogsViewScope)
      }
    },
    [setViewScope]
  )

  const titleKey =
    activeCategory === 'common'
      ? SECTION_META.common.titleKey
      : SECTION_META.task.titleKey
  const showTaskSwitcher =
    activeCategory !== 'common' && visibleSectionTabs.length > 1

  return {
    activeCategory,
    titleKey,
    canManageScope,
    viewScope,
    handleViewScopeChange,
    showTaskSwitcher,
    visibleSectionTabs,
    handleSectionChange,
    selectedUserId,
    userInfoDialogOpen,
    setUserInfoDialogOpen,
    affinityDialogOpen,
    setAffinityDialogOpen,
    affinityDialogTarget: affinityTarget
      ? {
          rule_name: affinityTarget.rule_name || '',
          using_group:
            affinityTarget.using_group || affinityTarget.selected_group || '',
          key_hint: affinityTarget.key_hint || '',
          key_fp: affinityTarget.key_fp || '',
        }
      : null,
  }
}
