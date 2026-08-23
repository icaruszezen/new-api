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
import { Filter } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  sideDrawerContentClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { PricingSidebar } from '@/features/pricing/components/pricing-sidebar'
import type { PricingModel, PricingVendor } from '@/features/pricing/types'

export type NextPricingToolbarProps = {
  filteredCount: number
  totalCount?: number
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

export function NextPricingToolbar(props: NextPricingToolbarProps) {
  const { t } = useTranslation()
  const [filtersOpen, setFiltersOpen] = useState(false)

  return (
    <div className='flex items-center'>
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
