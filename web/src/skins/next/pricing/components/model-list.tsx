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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Accordion } from '@/components/ui/accordion'
import type { PricingModel, TokenUnit } from '@/features/pricing/types'
import { getLobeIcon } from '@/lib/lobe-icon'

import { MODEL_LIST_GRID_CLASS } from '../layout'
import {
  OTHER_VENDOR_GROUP_ID,
  groupModelsByVendor,
  type VendorModelGroup,
} from '../lib/group-by-vendor'
import { ModelRow } from './model-row'

export type ModelListProps = {
  models: PricingModel[]
  usableGroup: Record<string, { desc: string; ratio: number }>
  groupRatio: Record<string, number>
  tokenUnit: TokenUnit
  showRechargePrice: boolean
  priceRate: number
  usdExchangeRate: number
  selectedGroup?: string
}

function vendorHeadingId(groupId: string): string {
  return `vendor-heading-${encodeURIComponent(groupId).replaceAll(/[^a-zA-Z0-9_-]/g, '')}`
}

function ListColumnHeader(props: { tokenUnitLabel: string }) {
  const { t } = useTranslation()

  return (
    <div
      className={`${MODEL_LIST_GRID_CLASS} border-border text-muted-foreground border-b px-3 py-2.5 text-[11px] font-medium tracking-wide sm:px-4`}
    >
      <span />
      <span>{t('Model')}</span>
      <span className='text-right'>
        {t('Input (per {{unit}} tokens)', { unit: props.tokenUnitLabel })}
      </span>
      <span className='text-right'>
        {t('Output (per {{unit}} tokens)', { unit: props.tokenUnitLabel })}
      </span>
      <span className='text-right'>
        {t('Cached input (per {{unit}} tokens)', { unit: props.tokenUnitLabel })}
      </span>
      <span />
    </div>
  )
}

function VendorSection(props: {
  group: VendorModelGroup
  usableGroup: ModelListProps['usableGroup']
  groupRatio: ModelListProps['groupRatio']
  tokenUnit: ModelListProps['tokenUnit']
  showRechargePrice: ModelListProps['showRechargePrice']
  priceRate: ModelListProps['priceRate']
  usdExchangeRate: ModelListProps['usdExchangeRate']
  selectedGroup?: ModelListProps['selectedGroup']
  tokenUnitLabel: string
}) {
  const { t } = useTranslation()
  const headingId = vendorHeadingId(props.group.id)
  const title =
    props.group.id === OTHER_VENDOR_GROUP_ID || !props.group.name
      ? t('Other')
      : props.group.name

  return (
    <section aria-labelledby={headingId} className='space-y-3'>
      <h2
        id={headingId}
        className='flex items-center gap-2 text-sm font-medium'
      >
        <span aria-hidden='true' className='flex items-center'>
          {getLobeIcon(props.group.icon, 20)}
        </span>
        <span>{title}</span>
      </h2>
      <div className='border-border overflow-hidden rounded-xl border'>
        <ListColumnHeader tokenUnitLabel={props.tokenUnitLabel} />
        <Accordion className='w-full' multiple={false}>
          {props.group.models.map((model) => (
            <ModelRow
              key={model.model_name}
              model={model}
              usableGroup={props.usableGroup}
              groupRatio={props.groupRatio}
              tokenUnit={props.tokenUnit}
              showRechargePrice={props.showRechargePrice}
              priceRate={props.priceRate}
              usdExchangeRate={props.usdExchangeRate}
              selectedGroup={props.selectedGroup}
            />
          ))}
        </Accordion>
      </div>
    </section>
  )
}

export function ModelList(props: ModelListProps) {
  const tokenUnitLabel = props.tokenUnit === 'K' ? '1K' : '1M'
  const vendorGroups = useMemo(
    () => groupModelsByVendor(props.models),
    [props.models]
  )

  if (props.models.length === 0) {
    return null
  }

  return (
    <div className='space-y-8'>
      {vendorGroups.map((group) => (
        <VendorSection
          key={group.id}
          group={group}
          usableGroup={props.usableGroup}
          groupRatio={props.groupRatio}
          tokenUnit={props.tokenUnit}
          showRechargePrice={props.showRechargePrice}
          priceRate={props.priceRate}
          usdExchangeRate={props.usdExchangeRate}
          selectedGroup={props.selectedGroup}
          tokenUnitLabel={tokenUnitLabel}
        />
      ))}
    </div>
  )
}
