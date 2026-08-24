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
import { CheckinCalendarCard } from '@/features/profile/components/checkin-calendar-card'
import { LoginSessionsCard } from '@/features/profile/components/login-sessions-card'
import { SidebarModulesCard } from '@/features/profile/components/sidebar-modules-card'
import { useProfile } from '@/features/profile/hooks'
import { useStatus } from '@/hooks/use-status'
import { useAuthStore } from '@/stores/auth-store'

import { NextProfileHeading } from './components/page-heading'
import { NextProfilePasskeyCard } from './components/passkey-card'
import { NextProfileSecurityList } from './components/security-list'
import { NextProfileSettingsCard } from './components/settings-card'
import { NextProfileStatsRow } from './components/stats-row'
import { NextProfileTwoFACard } from './components/two-fa-card'

export function NextProfile() {
  const { profile, loading, refreshProfile } = useProfile()
  const { status } = useStatus()
  const permissions = useAuthStore((state) => state.auth.user?.permissions)

  const checkinEnabled = status?.checkin_enabled === true
  const turnstileEnabled = !!(
    status?.turnstile_check && status?.turnstile_site_key
  )
  const turnstileSiteKey = status?.turnstile_site_key || ''
  const canConfigureSidebar = permissions?.sidebar_settings !== false

  return (
    <div data-slot='next-profile' className='flex flex-col gap-5 pb-10'>
      <NextProfileHeading profile={profile} loading={loading} />
      <NextProfileStatsRow profile={profile} loading={loading} />

      <div
        data-slot='next-profile-columns'
        className='grid items-start gap-5 xl:grid-cols-[minmax(0,1fr)_minmax(320px,0.42fr)]'
      >
        <div className='flex flex-col gap-5'>
          <NextProfileSettingsCard
            profile={profile}
            loading={loading}
            onProfileUpdate={refreshProfile}
          />
          <LoginSessionsCard />
        </div>

        <div className='flex flex-col gap-5 xl:sticky xl:top-6'>
          <NextProfilePasskeyCard loading={loading} />
          <NextProfileTwoFACard loading={loading} />
          <NextProfileSecurityList profile={profile} loading={loading} />
          {checkinEnabled && (
            <CheckinCalendarCard
              checkinEnabled={checkinEnabled}
              turnstileEnabled={turnstileEnabled}
              turnstileSiteKey={turnstileSiteKey}
            />
          )}
          {canConfigureSidebar && <SidebarModulesCard />}
        </div>
      </div>
    </div>
  )
}
