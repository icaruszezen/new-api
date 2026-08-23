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
import { ArrowUpDown, Check, Filter } from 'lucide-react'
import { useCallback, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  sideDrawerContentClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { PricingSidebar } from '@/features/pricing/components/pricing-sidebar'
import { getSortLabels, type SortOption } from '@/features/pricing/constants'
import type { PricingModel, PricingVendor, TokenUnit } from '@/features/pricing/types'
import { cn } from '@/lib/utils'

type SegmentOption = {
  value: string
  label: string
}

export type NextPricingToolbarProps = {
  filteredCount: number
  totalCount?: number
  sortBy: string
  onSortChange: (value: string) => void
  tokenUnit: TokenUnit
  onTokenUnitChange: (value: TokenUnit) => void
  showRechargePrice: boolean
  onRechargePriceChange: (value: boolean) => void
  quotaTypeFilter: string
  endpointTypeFilter: string
  vendorFilter: string
  groupFilter: string
  tagFilter: string
  onQuotaTypeChange: (value: string) => void
  onEndpointTypeChange: (value: string) => void
  onVendorChange: (value: string) => void
  onGroupChange: (value: string) => void
  onTagChange: (value: string) => void
  vendors: PricingVendor[]
  groups: string[]
  groupRatios?: Record<string, number>
  tags: string[]
  models: PricingModel[]
  hasActiveFilters: boolean
  activeFilterCount: number
  onClearFilters: () => void
}

function SegmentedControl(props: {
  options: SegmentOption[]
  value: string
  onChange: (value: string) => void
  ariaLabel: string
}) {
  return (
    <div
      role='group'
      aria-label={props.ariaLabel}
      className='inline-flex h-8 items-center gap-0.5'
    >
      {props.options.map((option) => {
        const isActive = option.value === props.value
        return (
          <button
            key={option.value}
            type='button'
            onClick={() => props.onChange(option.value)}
            aria-pressed={isActive}
            className={cn(
              'inline-flex h-full items-center rounded-sm px-2.5 text-xs font-medium transition-colors',
              isActive
                ? 'text-foreground'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            {option.label}
          </button>
        )
      })}
    </div>
  )
}

export function NextPricingToolbar(props: NextPricingToolbarProps) {
  const { t } = useTranslation()
  const [filtersOpen, setFiltersOpen] = useState(false)
  const sortLabels = getSortLabels(t)

  const handleTokenUnitChange = useCallback(
    (value: string) => props.onTokenUnitChange(value as TokenUnit),
    [props]
  )

  const handleRechargePriceChange = useCallback(
    (value: string) => props.onRechargePriceChange(value === 'recharge'),
    [props]
  )

  return (
    <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
      <div className='flex items-center gap-2'>
        <Button
          type='button'
          variant='ghost'
          size='sm'
          onClick={() => setFiltersOpen(true)}
          className='text-muted-foreground hover:text-foreground gap-1.5'
        >
          <Filter className='size-3.5' />
          {t('Filter')}
          {props.activeFilterCount > 0 ? (
            <span className='text-muted-foreground tabular-nums'>
              {props.activeFilterCount}
            </span>
          ) : null}
        </Button>
        <div className='text-muted-foreground flex items-baseline gap-1 text-sm'>
          <span className='text-foreground font-medium tabular-nums'>
            {props.filteredCount.toLocaleString()}
          </span>
          <span>{props.filteredCount === 1 ? t('model') : t('models')}</span>
        </div>
      </div>

      <div className='flex flex-wrap items-center gap-2'>
        <SegmentedControl
          options={[
            { value: 'standard', label: t('Standard') },
            { value: 'recharge', label: t('Recharge') },
          ]}
          value={props.showRechargePrice ? 'recharge' : 'standard'}
          onChange={handleRechargePriceChange}
          ariaLabel={t('Price display mode')}
        />
        <SegmentedControl
          options={[
            { value: 'M', label: '/1M' },
            { value: 'K', label: '/1K' },
          ]}
          value={props.tokenUnit}
          onChange={handleTokenUnitChange}
          ariaLabel={t('Token unit')}
        />
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                type='button'
                variant='ghost'
                size='sm'
                className='text-muted-foreground hover:text-foreground h-8 gap-1.5 px-3 text-xs'
              />
            }
          >
            <ArrowUpDown className='size-3.5' />
            <span>{sortLabels[props.sortBy as SortOption] || t('Sort')}</span>
          </DropdownMenuTrigger>
          <DropdownMenuContent align='end' className='w-44'>
            {Object.entries(sortLabels).map(([value, label]) => (
              <DropdownMenuItem
                key={value}
                onClick={() => props.onSortChange(value)}
                className='gap-2'
              >
                <Check
                  className={cn(
                    'size-4 shrink-0',
                    props.sortBy === value ? 'opacity-100' : 'opacity-0'
                  )}
                />
                {label}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <Sheet open={filtersOpen} onOpenChange={setFiltersOpen}>
        <SheetContent
          side='right'
          className={sideDrawerContentClassName('sm:max-w-md')}
        >
          <SheetHeader className={sideDrawerHeaderClassName()}>
            <SheetTitle>{t('Filter')}</SheetTitle>
            <SheetDescription>
              {t('Filter models by provider, group, type, endpoint, and tags.')}
            </SheetDescription>
          </SheetHeader>
          <div className={sideDrawerFormClassName('gap-0')}>
            <PricingSidebar
              quotaTypeFilter={props.quotaTypeFilter}
              endpointTypeFilter={props.endpointTypeFilter}
              vendorFilter={props.vendorFilter}
              groupFilter={props.groupFilter}
              tagFilter={props.tagFilter}
              onQuotaTypeChange={props.onQuotaTypeChange}
              onEndpointTypeChange={props.onEndpointTypeChange}
              onVendorChange={props.onVendorChange}
              onGroupChange={props.onGroupChange}
              onTagChange={props.onTagChange}
              vendors={props.vendors}
              groups={props.groups}
              groupRatios={props.groupRatios}
              tags={props.tags}
              models={props.models}
              hasActiveFilters={props.hasActiveFilters}
              onClearFilters={props.onClearFilters}
              className='border-0 bg-transparent p-0 shadow-none'
            />
          </div>
        </SheetContent>
      </Sheet>
    </div>
  )
}
