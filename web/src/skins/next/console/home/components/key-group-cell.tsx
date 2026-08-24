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

import { TruncatedCell } from '@/components/data-table'
import { StatusBadge } from '@/components/status-badge'
import { RatioTag } from '@/skins/next/pricing/components/ratio-tag'

type ConsoleKeyGroupCellProps = {
  group: string
  ratio?: number | string | null
}

export function ConsoleKeyGroupCell(props: ConsoleKeyGroupCellProps) {
  const { t } = useTranslation()
  const groupName = props.group.trim()
  const label = groupName === 'auto' ? t('Cross-group') : groupName || '-'

  return (
    <TruncatedCell
      className='-ml-1.5'
      tooltipContent={label}
      tooltipClassName='break-all'
    >
      <span
        data-slot='console-key-group-cell'
        className='inline-flex max-w-full min-w-0 items-center gap-2'
      >
        <StatusBadge
          label={label}
          variant='neutral'
          copyable={false}
          showDot={false}
          className='min-w-0 shrink overflow-hidden'
        />
        {typeof props.ratio === 'number' ? (
          <RatioTag ratio={props.ratio} />
        ) : null}
      </span>
    </TruncatedCell>
  )
}
