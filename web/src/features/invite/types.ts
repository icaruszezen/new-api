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

/** Invitee lifecycle stage shown on the referral page. */
export type InviteeStatus = 'registered' | 'recharged'

/** Filter applied to the invitee list; `all` is the default view. */
export type InviteeFilter = 'all' | InviteeStatus

/**
 * One invitee row. The backend masks the username before it leaves the server,
 * so no field here can identify the invited account.
 */
export type InviteeSummary = {
  username: string
  /** Unix seconds. */
  registered_at: number
  status: InviteeStatus
  rebate_quota: number
}

/**
 * Rebate rule amounts, expressed in the site display currency the admin used
 * when configuring them.
 */
export type InviteRebateRules = {
  register_amount: number
  topup_amount: number
  topup_threshold: number
}

export type InviteOverview = InviteRebateRules & {
  aff_code: string
  invitee_count: number
  recharged_count: number
  rebate_quota: number
  invitees: InviteeSummary[]
}

export type InviteOverviewResponse = {
  success: boolean
  message: string
  data?: InviteOverview
}

export type InviteAdminSummary = {
  inviter_count: number
  invitee_count: number
  recharged_count: number
  rebate_quota: number
  rebate_payout_count: number
  register_rebate_quota: number
  register_rebate_count: number
  topup_rebate_quota: number
  topup_rebate_count: number
}

export type InviteCountRank = {
  rank: number
  user_id: number
  username: string
  invitee_count: number
  rebate_quota: number
}

export type InviteRebateRank = {
  rank: number
  user_id: number
  username: string
  rebate_quota: number
  payout_count: number
  invitee_count: number
}

export type InviteAdminInvitee = {
  invitee_id: number
  invitee_username: string
  inviter_id: number
  inviter_username: string
  registered_at: number
  status: InviteeStatus
  rebate_quota: number
}

export type InviteAdminStats = {
  summary: InviteAdminSummary
  invite_rankings: InviteCountRank[]
  rebate_rankings: InviteRebateRank[]
  recent_invitees: InviteAdminInvitee[]
}

export type InviteAdminStatsResponse = {
  success: boolean
  message: string
  data?: InviteAdminStats
}
