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
import { ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import {
  getDynamicDisplayGroupRatio,
  getDynamicPricingSummary,
  isDynamicPricingModel,
} from '@/features/pricing/lib/dynamic-price'
import { isTokenBasedModel } from '@/features/pricing/lib/model-helpers'
import { formatPrice, formatRequestPrice } from '@/features/pricing/lib/price'
import type { PricingModel, TokenUnit } from '@/features/pricing/types'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import { MODEL_LIST_GRID_CLASS } from '../layout'
import { GroupPriceDrawer } from './group-price-drawer'

const MISSING_PRICE = '—'

export type ModelRowProps = {
  model: PricingModel
  usableGroup: Record<string, { desc: string; ratio: number }>
  groupRatio: Record<string, number>
  tokenUnit: TokenUnit
  showRechargePrice: boolean
  priceRate: number
  usdExchangeRate: number
  selectedGroup?: string
}

function PriceCell(props: { children: string; className?: string }) {
  return (
    <span
      className={cn(
        'text-right font-mono text-xs tabular-nums sm:text-sm',
        props.className
      )}
    >
      {props.children}
    </span>
  )
}

export function ModelRow(props: ModelRowProps) {
  const { t } = useTranslation()
  const isTokenBased = isTokenBasedModel(props.model)
  const isDynamic = isDynamicPricingModel(props.model)
  const hasCache = isTokenBased && props.model.cache_ratio != null
  const groupCount = (props.model.enable_groups || []).length
  const showLowestBadge = groupCount > 1
  const tokenUnit = props.tokenUnit
  const dynamicSummary = isDynamic
    ? getDynamicPricingSummary(props.model, {
        tokenUnit,
        showRechargePrice: props.showRechargePrice,
        priceRate: props.priceRate,
        usdExchangeRate: props.usdExchangeRate,
        groupRatioMultiplier: getDynamicDisplayGroupRatio(
          props.model,
          props.selectedGroup
        ),
      })
    : null

  let inputPrice = MISSING_PRICE
  let outputPrice = MISSING_PRICE
  let cachePrice = MISSING_PRICE

  if (dynamicSummary?.isSpecialExpression) {
    inputPrice = t('Dynamic Pricing')
  } else if (dynamicSummary) {
    const inputEntry = dynamicSummary.primaryEntries.find(
      (entry) => entry.field === 'inputPrice'
    )
    const outputEntry = dynamicSummary.primaryEntries.find(
      (entry) => entry.field === 'outputPrice'
    )
    const cacheEntry = dynamicSummary.entries.find(
      (entry) => entry.field === 'cacheReadPrice'
    )
    if (inputEntry) inputPrice = inputEntry.formatted
    if (outputEntry) outputPrice = outputEntry.formatted
    if (cacheEntry) cachePrice = cacheEntry.formatted
  } else if (isTokenBased) {
    inputPrice = formatPrice(
      props.model,
      'input',
      tokenUnit,
      props.showRechargePrice,
      props.priceRate,
      props.usdExchangeRate,
      props.selectedGroup
    )
    outputPrice = formatPrice(
      props.model,
      'output',
      tokenUnit,
      props.showRechargePrice,
      props.priceRate,
      props.usdExchangeRate,
      props.selectedGroup
    )
    if (hasCache) {
      cachePrice = formatPrice(
        props.model,
        'cache',
        tokenUnit,
        props.showRechargePrice,
        props.priceRate,
        props.usdExchangeRate,
        props.selectedGroup
      )
    }
  } else {
    inputPrice = formatRequestPrice(
      props.model,
      props.showRechargePrice,
      props.priceRate,
      props.usdExchangeRate,
      props.selectedGroup
    )
  }

  return (
    <AccordionItem
      value={props.model.model_name}
      className='border-border not-last:border-b'
    >
      <AccordionTrigger
        aria-label={t('Expand {{model}} pricing', {
          model: props.model.model_name,
        })}
        className={cn(
          MODEL_LIST_GRID_CLASS,
          'hover:bg-muted/15 rounded-none border-0 px-3 py-4 hover:no-underline sm:px-4',
          '**:data-[slot=accordion-trigger-icon]:hidden'
        )}
      >
        <ChevronRight
          aria-hidden
          className='text-muted-foreground size-4 shrink-0 transition-transform group-aria-expanded/accordion-trigger:rotate-90'
        />
        <span className='flex min-w-0 items-center gap-2 text-left'>
          <span aria-hidden='true' className='flex shrink-0 items-center'>
            {getLobeIcon(props.model.icon || props.model.vendor_icon, 16)}
          </span>
          <span className='min-w-0 truncate font-mono text-sm font-medium'>
            {props.model.model_name}
          </span>
        </span>
        <PriceCell>{inputPrice}</PriceCell>
        <PriceCell>{outputPrice}</PriceCell>
        <PriceCell>{cachePrice}</PriceCell>
        <span className='flex justify-end'>
          {showLowestBadge ? (
            <span className='text-muted-foreground rounded-md border px-1.5 py-0.5 text-[10px] tracking-wide'>
              {t('lowest')}
            </span>
          ) : null}
        </span>
      </AccordionTrigger>
      <AccordionContent className='pb-0'>
        <GroupPriceDrawer
          model={props.model}
          usableGroup={props.usableGroup}
          groupRatio={props.groupRatio}
          tokenUnit={props.tokenUnit}
          showRechargePrice={props.showRechargePrice}
          priceRate={props.priceRate}
          usdExchangeRate={props.usdExchangeRate}
        />
      </AccordionContent>
    </AccordionItem>
  )
}
