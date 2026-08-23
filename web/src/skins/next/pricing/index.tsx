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
import { useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/features/pricing/components/empty-state'
import { SearchBar } from '@/features/pricing/components/search-bar'
import { EXCLUDED_GROUPS } from '@/features/pricing/constants'
import { useFilters } from '@/features/pricing/hooks/use-filters'
import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'
import { groupModelsByName } from '@/features/pricing/lib/group-models'

import { NextPublicShell } from '../public-shell'
import { NextPricingSkeleton } from './components/loading-skeleton'
import { ModelList } from './components/model-list'
import { NextPricingToolbar } from './components/toolbar'

export function NextPricing() {
  const { t } = useTranslation()
  const {
    models,
    vendors,
    groupRatio,
    usableGroup,
    isLoading,
    priceRate,
    usdExchangeRate,
  } = usePricingData()

  const uniqueModels = useMemo(() => groupModelsByName(models || []), [models])

  const {
    searchInput,
    sortBy,
    vendorFilter,
    groupFilter,
    quotaTypeFilter,
    endpointTypeFilter,
    tagFilter,
    tokenUnit,
    showRechargePrice,
    setSearchInput,
    setSortBy,
    setVendorFilter,
    setGroupFilter,
    setQuotaTypeFilter,
    setEndpointTypeFilter,
    setTagFilter,
    setTokenUnit,
    setShowRechargePrice,
    filteredModels,
    hasActiveFilters,
    activeFilterCount,
    availableTags,
    clearFilters,
    clearSearch,
  } = useFilters(uniqueModels)

  const availableGroups = useMemo(
    () =>
      Object.keys(usableGroup || {}).filter(
        (group) => !EXCLUDED_GROUPS.includes(group)
      ),
    [usableGroup]
  )

  const handleClearAll = useCallback(() => {
    clearFilters()
    clearSearch()
  }, [clearFilters, clearSearch])

  if (isLoading) {
    return (
      <NextPublicShell>
        <NextPricingSkeleton />
      </NextPublicShell>
    )
  }

  return (
    <NextPublicShell>
      <div className='pt-16 pb-20'>
        <header className='mx-auto mb-10 max-w-2xl text-center'>
          <h1 className='text-[clamp(2.25rem,5vw,3.25rem)] font-semibold tracking-tight'>
            {t('Model Square')}
          </h1>
          <SearchBar
            value={searchInput}
            onChange={setSearchInput}
            onClear={clearSearch}
            placeholder={t('Search models...')}
            className='mt-8'
            inputClassName='rounded-xl border-border focus:border-foreground/40 focus:ring-0'
          />
        </header>

        <div className='space-y-4'>
          <NextPricingToolbar
            filteredCount={filteredModels.length}
            totalCount={uniqueModels.length}
            sortBy={sortBy}
            onSortChange={setSortBy}
            tokenUnit={tokenUnit}
            onTokenUnitChange={setTokenUnit}
            showRechargePrice={showRechargePrice}
            onRechargePriceChange={setShowRechargePrice}
            quotaTypeFilter={quotaTypeFilter}
            endpointTypeFilter={endpointTypeFilter}
            vendorFilter={vendorFilter}
            groupFilter={groupFilter}
            tagFilter={tagFilter}
            onQuotaTypeChange={setQuotaTypeFilter}
            onEndpointTypeChange={setEndpointTypeFilter}
            onVendorChange={setVendorFilter}
            onGroupChange={setGroupFilter}
            onTagChange={setTagFilter}
            vendors={vendors || []}
            groups={availableGroups}
            groupRatios={groupRatio}
            tags={availableTags}
            models={uniqueModels}
            hasActiveFilters={hasActiveFilters}
            activeFilterCount={activeFilterCount}
            onClearFilters={clearFilters}
          />

          {filteredModels.length === 0 ? (
            <EmptyState
              searchQuery={searchInput}
              hasActiveFilters={hasActiveFilters}
              onClearFilters={handleClearAll}
            />
          ) : (
            <ModelList
              models={filteredModels}
              usableGroup={usableGroup || {}}
              groupRatio={groupRatio || {}}
              tokenUnit={tokenUnit}
              showRechargePrice={showRechargePrice}
              priceRate={priceRate ?? 1}
              usdExchangeRate={usdExchangeRate ?? 1}
              selectedGroup={groupFilter}
            />
          )}
        </div>
      </div>
    </NextPublicShell>
  )
}
