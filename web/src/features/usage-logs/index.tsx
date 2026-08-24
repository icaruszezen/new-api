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
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { CacheStatsDialog } from '@/features/system-settings/general/channel-affinity/cache-stats-dialog'

import { UserInfoDialog } from './components/dialogs/user-info-dialog'
import { UsageLogsProvider } from './components/usage-logs-provider'
import { UsageLogsTable } from './components/usage-logs-table'
import { useUsageLogsPage } from './hooks'

function UsageLogsContent() {
  const { t } = useTranslation()
  const page = useUsageLogsPage()

  return (
    <>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>{t(page.titleKey)}</SectionPageLayout.Title>
        {page.canManageScope && (
          <SectionPageLayout.Actions>
            <Tabs
              value={page.viewScope}
              onValueChange={page.handleViewScopeChange}
            >
              <TabsList>
                <TabsTrigger value='all'>{t('All')}</TabsTrigger>
                <TabsTrigger value='self'>{t('Only Mine')}</TabsTrigger>
              </TabsList>
            </Tabs>
          </SectionPageLayout.Actions>
        )}
        <SectionPageLayout.Content>
          <div className='flex h-full min-h-0 flex-col gap-4'>
            {page.showTaskSwitcher && (
              <Tabs
                value={page.activeCategory}
                onValueChange={page.handleSectionChange}
              >
                <TabsList className='max-w-full flex-wrap justify-start group-data-horizontal/tabs:h-auto'>
                  {page.visibleSectionTabs.map((section) => (
                    <TabsTrigger key={section.id} value={section.id}>
                      {t(section.titleKey)}
                    </TabsTrigger>
                  ))}
                </TabsList>
              </Tabs>
            )}
            <div className='min-h-0 flex-1'>
              <UsageLogsTable logCategory={page.activeCategory} />
            </div>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

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
    </>
  )
}

export function UsageLogs() {
  return (
    <UsageLogsProvider>
      <UsageLogsContent />
    </UsageLogsProvider>
  )
}
