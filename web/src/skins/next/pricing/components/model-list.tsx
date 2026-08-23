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
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Accordion } from '@/components/ui/accordion'
import { Button } from '@/components/ui/button'
import { DEFAULT_PRICING_PAGE_SIZE } from '@/features/pricing/constants'
import type { PricingModel, TokenUnit } from '@/features/pricing/types'

import { MODEL_LIST_GRID_CLASS } from '../layout'
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

export function ModelList(props: ModelListProps) {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const pageSize = DEFAULT_PRICING_PAGE_SIZE
  const totalPages = Math.max(1, Math.ceil(props.models.length / pageSize))
  const currentPage = Math.min(page, totalPages)
  const tokenUnitLabel = props.tokenUnit === 'K' ? '1K' : '1M'

  const pagedModels = useMemo(() => {
    const start = (currentPage - 1) * pageSize
    return props.models.slice(start, start + pageSize)
  }, [currentPage, pageSize, props.models])

  if (props.models.length === 0) {
    return null
  }

  return (
    <div className='border-border overflow-hidden rounded-xl border'>
      <div
        className={`${MODEL_LIST_GRID_CLASS} border-border text-muted-foreground border-b px-3 py-2.5 text-[11px] font-medium tracking-wide sm:px-4`}
      >
        <span />
        <span>{t('Model')}</span>
        <span className='text-right'>
          {t('Input (per {{unit}} tokens)', { unit: tokenUnitLabel })}
        </span>
        <span className='text-right'>
          {t('Output (per {{unit}} tokens)', { unit: tokenUnitLabel })}
        </span>
        <span className='text-right'>
          {t('Cached input (per {{unit}} tokens)', { unit: tokenUnitLabel })}
        </span>
        <span />
      </div>

      <Accordion className='w-full' multiple={false}>
        {pagedModels.map((model) => (
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

      {totalPages > 1 ? (
        <div className='text-muted-foreground flex flex-col items-center justify-between gap-3 border-t px-4 py-3 text-sm sm:flex-row'>
          <p>
            {t('Page {{current}} of {{total}}', {
              current: currentPage,
              total: totalPages,
            })}
          </p>
          <div className='flex items-center gap-2'>
            <Button
              type='button'
              variant='ghost'
              size='sm'
              onClick={() => setPage((current) => Math.max(1, current - 1))}
              disabled={currentPage <= 1}
              className='gap-1.5'
            >
              <ChevronLeft className='size-4' />
              {t('Previous page')}
            </Button>
            <Button
              type='button'
              variant='ghost'
              size='sm'
              onClick={() =>
                setPage((current) => Math.min(totalPages, current + 1))
              }
              disabled={currentPage >= totalPages}
              className='gap-1.5'
            >
              {t('Next page')}
              <ChevronRight className='size-4' />
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  )
}
