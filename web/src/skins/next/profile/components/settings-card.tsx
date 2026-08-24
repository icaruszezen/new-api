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

import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { LanguageSelectRow } from '@/features/profile/components/language-select-row'
import { AccountBindingsTab } from '@/features/profile/components/tabs/account-bindings-tab'
import { NotificationTab } from '@/features/profile/components/tabs/notification-tab'
import type { UserProfile } from '@/features/profile/types'

type NextProfileSettingsCardProps = {
  profile: UserProfile | null
  loading: boolean
  onProfileUpdate: () => void
}

export function NextProfileSettingsCard(props: NextProfileSettingsCardProps) {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState('bindings')

  if (props.loading) {
    return (
      <section
        data-slot='next-profile-settings'
        className='bg-card overflow-hidden rounded-xl border'
      >
        <div className='space-y-2 px-5 py-5'>
          <Skeleton className='h-6 w-24' />
          <Skeleton className='h-4 w-48' />
        </div>
        <div className='space-y-3 border-t px-5 py-5'>
          <Skeleton className='h-16 w-full' />
          <Skeleton className='h-16 w-full' />
        </div>
      </section>
    )
  }

  return (
    <section
      data-slot='next-profile-settings'
      className='bg-card overflow-hidden rounded-xl border'
    >
      <div className='px-5 py-5'>
        <h2 className='text-lg font-medium tracking-tight'>{t('Settings')}</h2>
        <p className='text-muted-foreground mt-1 text-sm'>
          {t('Account bindings and preferences')}
        </p>
      </div>

      <div className='px-5 pb-5'>
        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList
            variant='line'
            className='w-full justify-start rounded-none border-b pb-0 group-data-horizontal/tabs:h-auto'
          >
            <TabsTrigger value='bindings' className='flex-none px-1 pb-2.5'>
              {t('Account Bindings')}
            </TabsTrigger>
            <TabsTrigger value='settings' className='flex-none px-1 pb-2.5'>
              {t('Settings & Preferences')}
            </TabsTrigger>
          </TabsList>

          <TabsContent value='bindings' className='mt-5'>
            <AccountBindingsTab
              profile={props.profile}
              onUpdate={props.onProfileUpdate}
            />
          </TabsContent>
          <TabsContent value='settings' className='mt-5'>
            <NotificationTab
              profile={props.profile}
              onUpdate={props.onProfileUpdate}
            />
          </TabsContent>
        </Tabs>
      </div>

      <div className='border-t px-5 py-4'>
        <LanguageSelectRow
          profile={props.profile}
          onProfileUpdate={props.onProfileUpdate}
        />
      </div>
    </section>
  )
}
