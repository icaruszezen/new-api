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

import { Alert, AlertDescription } from '@/components/ui/alert'
import { useInviteOverview } from '@/features/invite/hooks/use-invite-overview'
import {
  buildInviteRebateRuleLabels,
  getInviteRebateCurrencySymbol,
  isInviteRebateActive,
} from '@/features/invite/lib/rules'

import { NextInviteeList } from './components/invitee-list'
import { NextInviteLinkCard } from './components/link-card'
import { NextInviteHeading } from './components/page-heading'
import { NextInviteStatsRow } from './components/stats-row'

export function NextInvite() {
  const { t } = useTranslation()
  const page = useInviteOverview()

  const rules = {
    register_amount: page.overview?.register_amount ?? 0,
    topup_amount: page.overview?.topup_amount ?? 0,
    topup_threshold: page.overview?.topup_threshold ?? 0,
  }
  const ruleLabels = buildInviteRebateRuleLabels(
    rules,
    getInviteRebateCurrencySymbol(),
    t
  )
  if (ruleLabels.length > 0) {
    ruleLabels.push(t('Once per friend for each reward'))
  }

  return (
    <div data-slot='next-invite' className='flex flex-col gap-5 pb-10'>
      <NextInviteHeading
        hasInvitees={(page.overview?.invitee_count ?? 0) > 0}
        rebateActive={isInviteRebateActive(rules)}
      />

      {page.error ? (
        <Alert variant='destructive'>
          <AlertDescription>
            {t('Failed to load referral data. Please try again later.')}
          </AlertDescription>
        </Alert>
      ) : null}

      <NextInviteLinkCard
        inviteLink={page.inviteLink}
        ruleLabels={ruleLabels}
        loading={page.loading}
      />

      <NextInviteStatsRow
        inviteeCount={page.overview?.invitee_count ?? 0}
        rechargedCount={page.overview?.recharged_count ?? 0}
        rebateQuota={page.overview?.rebate_quota ?? 0}
        loading={page.loading}
      />

      <NextInviteeList
        invitees={page.overview?.invitees ?? []}
        filter={page.filter}
        onFilterChange={page.setFilter}
        hasInvitees={(page.overview?.invitee_count ?? 0) > 0}
        loading={page.loading}
      />
    </div>
  )
}
