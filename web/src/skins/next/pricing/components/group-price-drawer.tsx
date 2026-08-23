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
import { Link } from '@tanstack/react-router'
import { ChevronRight } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  getDynamicPriceEntries,
  getDynamicPricingTiers,
  isDynamicPricingModel,
} from '@/features/pricing/lib/dynamic-price'
import {
  getAvailableGroups,
  getConfiguredGroupRatio,
  isTokenBasedModel,
} from '@/features/pricing/lib/model-helpers'
import {
  formatFixedPrice,
  formatGroupPrice,
} from '@/features/pricing/lib/price'
import type { PricingModel, TokenUnit } from '@/features/pricing/types'

import { RatioTag } from './ratio-tag'

const MISSING_PRICE = '—'

const DRAWER_HEAD =
  'text-muted-foreground pb-2 text-left text-[11px] font-medium tracking-wide'

export type GroupPriceDrawerProps = {
  model: PricingModel
  usableGroup: Record<string, { desc: string; ratio: number }>
  groupRatio: Record<string, number>
  tokenUnit: TokenUnit
  showRechargePrice: boolean
  priceRate: number
  usdExchangeRate: number
}

export function GroupPriceDrawer(props: GroupPriceDrawerProps) {
  const { t } = useTranslation()
  const groups = useMemo(
    () => getAvailableGroups(props.model, props.usableGroup),
    [props.model, props.usableGroup]
  )
  const isTokenBased = isTokenBasedModel(props.model)
  const isDynamic = isDynamicPricingModel(props.model)
  const hasCache = isTokenBased && props.model.cache_ratio != null

  return (
    <div className='border-border mx-3 mb-3 rounded-lg border px-3 py-3 sm:mx-4 sm:px-4'>
      <div className='mb-3 flex items-center justify-between gap-3'>
        <h3 className='text-sm font-medium'>{t('Pricing by Group')}</h3>
        <Link
          to='/pricing/$modelId'
          params={{ modelId: props.model.model_name }}
          className='text-muted-foreground hover:text-foreground text-xs underline-offset-4 hover:underline'
        >
          {t('Model details')}
        </Link>
      </div>

      {groups.length === 0 ? (
        <p className='text-muted-foreground text-sm'>
          {t(
            'This model is not available in any group, or no group pricing information is configured.'
          )}
        </p>
      ) : null}

      {groups.length > 0 && isDynamic ? (
        <DynamicGroupTable
          model={props.model}
          groups={groups}
          groupRatio={props.groupRatio}
          tokenUnit={props.tokenUnit}
          showRechargePrice={props.showRechargePrice}
          priceRate={props.priceRate}
          usdExchangeRate={props.usdExchangeRate}
        />
      ) : null}

      {groups.length > 0 && !isDynamic ? (
        <table className='w-full text-sm'>
          <caption className='sr-only'>
            {t('Pricing by Group')} {props.model.model_name}
          </caption>
          <thead>
            <tr className='border-border/60 border-b'>
              <th scope='col' className={DRAWER_HEAD}>
                {t('Group')}
              </th>
              <th scope='col' className={`${DRAWER_HEAD} text-right`}>
                {t('Multiplier')}
              </th>
              {isTokenBased ? (
                <>
                  <th scope='col' className={`${DRAWER_HEAD} text-right`}>
                    {t('Input')}
                  </th>
                  <th scope='col' className={`${DRAWER_HEAD} text-right`}>
                    {t('Output')}
                  </th>
                  <th scope='col' className={`${DRAWER_HEAD} text-right`}>
                    {t('Cached')}
                  </th>
                </>
              ) : (
                <th scope='col' className={`${DRAWER_HEAD} text-right`}>
                  {t('Price')}
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {groups.map((group) => {
              const ratio = getConfiguredGroupRatio(props.groupRatio, group)
              return (
                <tr
                  key={group}
                  className='border-border/50 text-foreground border-b last:border-b-0'
                >
                  <th
                    scope='row'
                    className='py-2 text-left font-mono text-xs font-normal'
                  >
                    {group}
                  </th>
                  <td className='py-2 text-right'>
                    <RatioTag ratio={ratio} />
                  </td>
                  {isTokenBased ? (
                    <>
                      <td className='py-2 text-right font-mono text-xs'>
                        {formatGroupPrice(
                          props.model,
                          group,
                          'input',
                          props.tokenUnit,
                          props.showRechargePrice,
                          props.priceRate,
                          props.usdExchangeRate,
                          props.groupRatio
                        )}
                      </td>
                      <td className='py-2 text-right font-mono text-xs'>
                        {formatGroupPrice(
                          props.model,
                          group,
                          'output',
                          props.tokenUnit,
                          props.showRechargePrice,
                          props.priceRate,
                          props.usdExchangeRate,
                          props.groupRatio
                        )}
                      </td>
                      <td className='py-2 text-right font-mono text-xs'>
                        {hasCache
                          ? formatGroupPrice(
                              props.model,
                              group,
                              'cache',
                              props.tokenUnit,
                              props.showRechargePrice,
                              props.priceRate,
                              props.usdExchangeRate,
                              props.groupRatio
                            )
                          : MISSING_PRICE}
                      </td>
                    </>
                  ) : (
                    <td className='py-2 text-right font-mono text-xs'>
                      {formatFixedPrice(
                        props.model,
                        group,
                        props.showRechargePrice,
                        props.priceRate,
                        props.usdExchangeRate,
                        props.groupRatio
                      )}
                    </td>
                  )}
                </tr>
              )
            })}
          </tbody>
        </table>
      ) : null}
    </div>
  )
}

function DynamicGroupTable(props: {
  model: PricingModel
  groups: string[]
  groupRatio: Record<string, number>
  tokenUnit: TokenUnit
  showRechargePrice: boolean
  priceRate: number
  usdExchangeRate: number
}) {
  const { t } = useTranslation()
  const tiers = getDynamicPricingTiers(props.model)

  if (tiers.length === 0) {
    return (
      <div className='text-muted-foreground space-y-2 text-sm'>
        <p>{t('Special billing expression')}</p>
        <code className='bg-background/80 block max-h-24 overflow-auto rounded-md border px-2 py-1.5 font-mono text-xs break-all'>
          {props.model.billing_expr}
        </code>
      </div>
    )
  }

  const firstTier = tiers[0]
  const fields = getDynamicPriceEntries(firstTier, {
    tokenUnit: props.tokenUnit,
    showRechargePrice: props.showRechargePrice,
    priceRate: props.priceRate,
    usdExchangeRate: props.usdExchangeRate,
    groupRatioMultiplier: 1,
  }).slice(0, 3)
  const hasMultipleTiers = tiers.length > 1
  const rowClass =
    'grid grid-cols-[minmax(0,1.3fr)_4rem_minmax(3.5rem,1fr)_minmax(3.5rem,1fr)_minmax(3.5rem,1fr)] items-center gap-x-2'

  return (
    <div>
      <div className={`${rowClass} border-border/60 border-b`}>
        <span className={DRAWER_HEAD}>{t('Group')}</span>
        <span className={`${DRAWER_HEAD} text-right`}>{t('Multiplier')}</span>
        {fields.map((field) => (
          <span key={field.field} className={`${DRAWER_HEAD} text-right`}>
            {t(field.shortLabel)}
          </span>
        ))}
      </div>
      {props.groups.map((group) => {
        const ratio = getConfiguredGroupRatio(props.groupRatio, group)
        const lowestEntries = getDynamicPriceEntries(firstTier, {
          tokenUnit: props.tokenUnit,
          showRechargePrice: props.showRechargePrice,
          priceRate: props.priceRate,
          usdExchangeRate: props.usdExchangeRate,
          groupRatioMultiplier: ratio,
        })
        const lowestByField = new Map(
          lowestEntries.map((entry) => [entry.field, entry.formatted])
        )
        const rowCells = (
          <>
            <span className='flex min-w-0 items-center gap-1.5 py-2 text-left font-mono text-xs'>
              {hasMultipleTiers ? (
                <ChevronRight
                  aria-hidden
                  className='size-3.5 shrink-0 transition-transform group-aria-expanded/tier-trigger:rotate-90'
                />
              ) : null}
              <span className='truncate'>{group}</span>
            </span>
            <span className='flex justify-end py-2'>
              <RatioTag ratio={ratio} />
            </span>
            {fields.map((field) => (
              <span
                key={field.field}
                className='py-2 text-right font-mono text-xs'
              >
                {lowestByField.get(field.field) ?? MISSING_PRICE}
              </span>
            ))}
          </>
        )

        if (!hasMultipleTiers) {
          return (
            <div
              key={group}
              className={`${rowClass} border-border/50 border-b last:border-b-0`}
            >
              {rowCells}
            </div>
          )
        }

        return (
          <Collapsible
            key={group}
            defaultOpen={false}
            className='border-border/50 border-b last:border-b-0'
          >
            <CollapsibleTrigger
              aria-label={t('Expand {{group}} tier prices', { group })}
              className={`group/tier-trigger hover:bg-muted/15 ${rowClass} w-full`}
            >
              {rowCells}
            </CollapsibleTrigger>
            <CollapsibleContent className='px-1 pt-1 pb-2'>
              <table className='w-full text-sm'>
                <thead>
                  <tr className='border-border/60 border-b'>
                    <th scope='col' className={DRAWER_HEAD}>
                      {t('Tier')}
                    </th>
                    {fields.map((field) => (
                      <th
                        key={field.field}
                        scope='col'
                        className={`${DRAWER_HEAD} text-right`}
                      >
                        {t(field.shortLabel)}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {tiers.map((tier, tierIndex) => {
                    const entries = getDynamicPriceEntries(tier, {
                      tokenUnit: props.tokenUnit,
                      showRechargePrice: props.showRechargePrice,
                      priceRate: props.priceRate,
                      usdExchangeRate: props.usdExchangeRate,
                      groupRatioMultiplier: ratio,
                    })
                    const byField = new Map(
                      entries.map((entry) => [entry.field, entry.formatted])
                    )
                    return (
                      <tr
                        key={`${group}-${tier.label || tierIndex}`}
                        className='border-border/50 border-b last:border-b-0'
                      >
                        <th
                          scope='row'
                          className='text-muted-foreground py-2 text-left text-xs font-normal'
                        >
                          {tier.label || t('Default')}
                        </th>
                        {fields.map((field) => (
                          <td
                            key={field.field}
                            className='py-2 text-right font-mono text-xs'
                          >
                            {byField.get(field.field) ?? MISSING_PRICE}
                          </td>
                        ))}
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </CollapsibleContent>
          </Collapsible>
        )
      })}
    </div>
  )
}
