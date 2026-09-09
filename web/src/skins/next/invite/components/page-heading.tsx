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

type NextInviteHeadingProps = {
  hasInvitees: boolean
  rebateActive: boolean
}

export function NextInviteHeading(props: NextInviteHeadingProps) {
  const { t } = useTranslation()

  let description = t(
    'Share your referral link. Earn once when a friend signs up, and again once their cumulative top-up reaches the threshold.'
  )
  if (!props.rebateActive) {
    description = t(
      'Referral rebates are currently turned off. You can still share your link, and invited friends stay linked to your account.'
    )
  } else if (!props.hasInvitees) {
    description = t(
      'No friends have signed up through your link yet. Copy the link and send it out to start earning rebates.'
    )
  }

  return (
    <header data-slot='next-invite-heading'>
      <h1 className='text-3xl font-medium tracking-tight'>
        {t('Referral Rebate')}
      </h1>
      <p className='text-muted-foreground mt-1.5 text-sm'>{description}</p>
    </header>
  )
}
