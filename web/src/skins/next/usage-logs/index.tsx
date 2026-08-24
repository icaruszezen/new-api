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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { PageFooterProvider } from '@/components/layout/components/page-footer'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { CacheStatsDialog } from '@/features/system-settings/general/channel-affinity/cache-stats-dialog'
import { UserInfoDialog } from '@/features/usage-logs/components/dialogs/user-info-dialog'
import { UsageLogsProvider } from '@/features/usage-logs/components/usage-logs-provider'
import { UsageLogsTable } from '@/features/usage-logs/components/usage-logs-table'
import { useUsageLogsPage } from '@/features/usage-logs/hooks'

import { NextUsageLogsHeading } from './components/page-heading'

function NextUsageLogsContent() {
  const { t } = useTranslation()
  const page = useUsageLogsPage()
  const [footerContainer, setFooterContainer] = useState<HTMLDivElement | null>(
    null
  )

  return (
    <PageFooterProvider container={footerContainer}>
      <div
        data-slot='next-usage-logs'
        className='flex h-full min-h-0 w-full flex-1 flex-col gap-3 px-3 pt-3 pb-3 sm:gap-4 sm:px-4 sm:pt-5 sm:pb-4'
      >
        <NextUsageLogsHeading
          title={t(page.titleKey)}
          canManageScope={page.canManageScope}
          viewScope={page.viewScope}
          onViewScopeChange={page.handleViewScopeChange}
        />

        {page.showTaskSwitcher && (
          <div data-slot='next-usage-logs-switcher'>
            <Tabs
              value={page.activeCategory}
              onValueChange={page.handleSectionChange}
            >
              <TabsList className='h-9 max-w-full flex-wrap justify-start rounded-full p-1 group-data-horizontal/tabs:h-auto'>
                {page.visibleSectionTabs.map((section) => (
                  <TabsTrigger
                    key={section.id}
                    value={section.id}
                    className='rounded-full px-3.5'
                  >
                    {t(section.titleKey)}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          </div>
        )}

        <section
          data-slot='next-usage-logs-card'
          className='bg-card flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border'
        >
          <div className='min-h-0 flex-1 px-5 py-5'>
            <UsageLogsTable
              logCategory={page.activeCategory}
              toolbarClassName='rounded-none border-0 bg-transparent p-0 sm:p-0'
            />
          </div>
          <div
            ref={setFooterContainer}
            data-slot='next-usage-logs-footer'
            className='border-t px-5 py-3 empty:hidden'
          />
        </section>
      </div>

      <UserInfoDialog
        userId={page.selectedUserId}
        open={page.userInfoDialogOpen}
        onOpenChange={page.setUserInfoDialogOpen}
      />

      <CacheStatsDialog
        open={page.affinityDialogOpen}
        onOpenChange={page.setAffinityDialogOpen}
        target={page.affinityDialogTarget}
      />
    </PageFooterProvider>
  )
}

export function NextUsageLogs() {
  return (
    <UsageLogsProvider>
      <NextUsageLogsContent />
    </UsageLogsProvider>
  )
}
