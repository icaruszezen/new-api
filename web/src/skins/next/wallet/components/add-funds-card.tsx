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
import { Check, Loader2, Receipt } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { CreemProductsSection } from '@/features/wallet/components/creem-products-section'
import {
  DEFAULT_DISCOUNT_RATE,
  PAYMENT_TYPES,
} from '@/features/wallet/constants'
import {
  calculatePresetPricing,
  formatCurrency,
  getDiscountLabel,
  getDisplayPaymentAmount,
  getMinTopupAmount,
  getPaymentIcon,
} from '@/features/wallet/lib'
import type {
  CreemProduct,
  PaymentMethod,
  PresetAmount,
  TopupInfo,
  WaffoPayMethod,
} from '@/features/wallet/types'
import { formatLocalCurrencyAmount } from '@/lib/currency'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'

type NextWalletAddFundsCardProps = {
  topupInfo: TopupInfo | null
  presetAmounts: PresetAmount[]
  selectedPreset: number | null
  onSelectPreset: (preset: PresetAmount) => void
  topupAmount: number
  onTopupAmountChange: (amount: number) => void
  paymentAmount: number
  calculating: boolean
  selectedPaymentMethod?: PaymentMethod
  selectedWaffoMethodIndex: number | null
  onPaymentMethodSelect: (method: PaymentMethod) => void
  onWaffoMethodSelect: (method: WaffoPayMethod, index: number) => void
  paymentLoading: string | null
  loading: boolean
  priceRatio: number
  usdExchangeRate: number
  canSubmitPayment: boolean
  onPayNow: () => void
  onOpenBilling: () => void
  onCreemProductSelect: (product: CreemProduct) => void
}

export function NextWalletAddFundsCard(props: NextWalletAddFundsCardProps) {
  const { t } = useTranslation()
  const [localAmount, setLocalAmount] = useState(props.topupAmount.toString())

  useEffect(() => {
    setLocalAmount((prev) =>
      prev === '' && props.topupAmount === 0
        ? prev
        : props.topupAmount.toString()
    )
  }, [props.topupAmount])

  const topupInfo = props.topupInfo
  const hasConfigurableTopup =
    !!topupInfo?.enable_online_topup ||
    !!topupInfo?.enable_stripe_topup ||
    !!topupInfo?.enable_waffo_topup ||
    !!topupInfo?.enable_waffo_pancake_topup
  const enableCreemTopup = !!topupInfo?.enable_creem_topup
  const creemProducts = topupInfo?.creem_products
  const hasAnyTopup = hasConfigurableTopup || enableCreemTopup
  const payMethods = topupInfo?.pay_methods
  const hasStandardPaymentMethods =
    Array.isArray(payMethods) && payMethods.length > 0
  const waffoPayMethods = topupInfo?.waffo_pay_methods
  const hasWaffoPaymentMethods =
    Array.isArray(waffoPayMethods) && waffoPayMethods.length > 0
  const minTopup = getMinTopupAmount(topupInfo)
  const payableAmount = getDisplayPaymentAmount(
    props.paymentAmount,
    props.topupAmount,
    props.priceRatio,
    topupInfo?.discount?.[props.topupAmount] || DEFAULT_DISCOUNT_RATE
  )

  if (props.loading) {
    return (
      <section
        data-slot='next-wallet-add-funds'
        className='bg-card overflow-hidden rounded-xl border'
      >
        <div className='space-y-2 px-5 py-5'>
          <Skeleton className='h-6 w-28' />
          <Skeleton className='h-4 w-52' />
        </div>
        <div className='space-y-4 border-t px-5 py-5'>
          <Skeleton className='h-24 w-full' />
          <Skeleton className='h-16 w-full' />
          <Skeleton className='h-10 w-full' />
        </div>
      </section>
    )
  }

  return (
    <section
      data-slot='next-wallet-add-funds'
      className='bg-card overflow-hidden rounded-xl border'
    >
      <div className='flex items-start justify-between gap-3 px-5 py-5'>
        <div>
          <h2 className='text-lg font-medium tracking-tight'>
            {t('Add Funds')}
          </h2>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Choose an amount and payment method')}
          </p>
        </div>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={props.onOpenBilling}
          className='shrink-0 gap-2'
        >
          <Receipt className='size-4' aria-hidden='true' />
          {t('Order History')}
        </Button>
      </div>

      <div className='space-y-6 border-t px-5 py-5'>
        {hasAnyTopup ? (
          <>
            {hasConfigurableTopup ? (
              <>
                <div className='space-y-3'>
                  <p className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                    {t('Amount')}
                  </p>
                  {props.presetAmounts.length > 0 ? (
                    <div className='grid grid-cols-2 gap-2 sm:grid-cols-3'>
                      {props.presetAmounts.map((preset) => {
                        const discount =
                          preset.discount ||
                          topupInfo?.discount?.[preset.value] ||
                          1
                        const pricing = calculatePresetPricing(
                          preset.value,
                          props.priceRatio,
                          discount,
                          props.usdExchangeRate
                        )
                        const selected = props.selectedPreset === preset.value
                        return (
                          <button
                            key={preset.value}
                            type='button'
                            onClick={() => props.onSelectPreset(preset)}
                            className={cn(
                              'min-h-16 rounded-xl border px-3 py-3 text-left transition-colors',
                              selected
                                ? 'border-foreground bg-foreground text-background'
                                : 'border-border hover:bg-muted/40'
                            )}
                          >
                            <span className='block text-lg font-medium tabular-nums'>
                              {formatNumber(pricing.displayValue)}
                            </span>
                            <span
                              className={cn(
                                'mt-1 block text-xs',
                                selected
                                  ? 'text-background/70'
                                  : 'text-muted-foreground'
                              )}
                            >
                              Pay {formatCurrency(pricing.actualPrice)}
                              {pricing.hasDiscount
                                ? ` · ${getDiscountLabel(discount)}`
                                : ''}
                            </span>
                          </button>
                        )
                      })}
                    </div>
                  ) : null}
                  <div className='space-y-2'>
                    <label
                      htmlFor='next-wallet-custom-amount'
                      className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
                    >
                      {t('Custom Amount')}
                    </label>
                    <div className='grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2'>
                      <Input
                        id='next-wallet-custom-amount'
                        type='number'
                        value={localAmount}
                        min={minTopup}
                        onChange={(event) => {
                          const value = event.target.value
                          setLocalAmount(value)
                          const numValue = Number.parseInt(value) || 0
                          if (numValue >= 0) {
                            props.onTopupAmountChange(numValue)
                          }
                        }}
                        placeholder={`${t('Minimum:')} ${minTopup}`}
                        className='h-10'
                      />
                      <div className='text-muted-foreground px-1 text-xs tabular-nums'>
                        {t('Amount to pay:')}{' '}
                        <span
                          data-slot='next-wallet-payable'
                          className='text-foreground font-medium'
                        >
                          {props.calculating && payableAmount <= 0
                            ? '…'
                            : formatLocalCurrencyAmount(payableAmount, {
                                abbreviate: false,
                              })}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                <div className='space-y-3'>
                  <p className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                    {t('Payment Method')}
                  </p>
                  {hasStandardPaymentMethods ? (
                    <div className='space-y-2'>
                      {payMethods.map((method) => {
                        const methodMin = Math.max(
                          method.min_topup || 0,
                          minTopup
                        )
                        const disabled = methodMin > props.topupAmount
                        const selected =
                          props.selectedPaymentMethod?.type === method.type &&
                          props.selectedWaffoMethodIndex === null
                        let methodAccessory = (
                          <span className='border-border size-4 shrink-0 rounded-full border' />
                        )
                        if (disabled) {
                          methodAccessory = (
                            <span className='text-muted-foreground text-xs'>
                              {t('Minimum:')} {methodMin}
                            </span>
                          )
                        } else if (selected) {
                          methodAccessory = (
                            <Check
                              className='size-4 shrink-0'
                              aria-hidden='true'
                            />
                          )
                        }
                        const button = (
                          <button
                            key={method.type}
                            type='button'
                            disabled={disabled || !!props.paymentLoading}
                            onClick={() => props.onPaymentMethodSelect(method)}
                            className={cn(
                              'flex w-full items-center gap-3 rounded-xl border px-4 py-3 text-left transition-colors',
                              selected
                                ? 'border-foreground bg-foreground/5'
                                : 'border-border hover:bg-muted/40',
                              disabled && 'opacity-50'
                            )}
                          >
                            <span className='flex size-8 items-center justify-center'>
                              {props.paymentLoading === method.type ? (
                                <Loader2 className='size-4 animate-spin' />
                              ) : (
                                getPaymentIcon(
                                  method.type,
                                  'h-5 w-5',
                                  method.icon,
                                  method.name
                                )
                              )}
                            </span>
                            <span className='min-w-0 flex-1 truncate text-sm font-medium'>
                              {method.name}
                            </span>
                            {methodAccessory}
                          </button>
                        )

                        if (!disabled) return button

                        return (
                          <TooltipProvider key={method.type}>
                            <Tooltip>
                              <TooltipTrigger render={button} />
                              <TooltipContent>
                                {t('Minimum topup amount: {{amount}}', {
                                  amount: methodMin,
                                })}
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        )
                      })}
                    </div>
                  ) : null}

                  {topupInfo?.enable_waffo_topup &&
                  hasWaffoPaymentMethods &&
                  waffoPayMethods ? (
                    <div className='space-y-2'>
                      <p className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                        {t('Waffo Payment')}
                      </p>
                      {waffoPayMethods.map((method, index) => {
                        const loadingKey = `waffo-${index}`
                        const methodKey = `${method.payMethodType ?? 'unknown'}-${method.payMethodName ?? method.name}`
                        const waffoMin = topupInfo.waffo_min_topup || 0
                        const belowMin = waffoMin > props.topupAmount
                        const selected =
                          props.selectedWaffoMethodIndex === index
                        let methodIcon = getPaymentIcon(
                          PAYMENT_TYPES.WAFFO,
                          'h-5 w-5',
                          method.icon,
                          method.name
                        )
                        if (props.paymentLoading === loadingKey) {
                          methodIcon = (
                            <Loader2 className='size-4 animate-spin' />
                          )
                        }

                        const button = (
                          <button
                            key={methodKey}
                            type='button'
                            disabled={belowMin || !!props.paymentLoading}
                            onClick={() =>
                              props.onWaffoMethodSelect(method, index)
                            }
                            className={cn(
                              'flex w-full items-center gap-3 rounded-xl border px-4 py-3 text-left transition-colors',
                              selected
                                ? 'border-foreground bg-foreground/5'
                                : 'border-border hover:bg-muted/40',
                              belowMin && 'opacity-50'
                            )}
                          >
                            <span className='flex size-8 items-center justify-center'>
                              {methodIcon}
                            </span>
                            <span className='min-w-0 flex-1 truncate text-sm font-medium'>
                              {method.name}
                            </span>
                            {selected ? (
                              <Check
                                className='size-4 shrink-0'
                                aria-hidden='true'
                              />
                            ) : (
                              <span className='border-border size-4 shrink-0 rounded-full border' />
                            )}
                          </button>
                        )

                        if (!belowMin) return button

                        return (
                          <TooltipProvider key={methodKey}>
                            <Tooltip>
                              <TooltipTrigger render={button} />
                              <TooltipContent>
                                {t('Minimum topup amount: {{amount}}', {
                                  amount: waffoMin,
                                })}
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        )
                      })}
                    </div>
                  ) : null}

                  {!hasStandardPaymentMethods && !hasWaffoPaymentMethods ? (
                    <Alert>
                      <AlertDescription>
                        {t(
                          'No payment methods available. Please contact administrator.'
                        )}
                      </AlertDescription>
                    </Alert>
                  ) : null}
                </div>
              </>
            ) : null}

            {enableCreemTopup &&
            Array.isArray(creemProducts) &&
            creemProducts.length > 0 ? (
              <div className='space-y-3'>
                <p className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                  {t('Creem Payment')}
                </p>
                <CreemProductsSection
                  products={creemProducts}
                  onProductSelect={props.onCreemProductSelect}
                />
              </div>
            ) : null}

            {hasConfigurableTopup ? (
              <Button
                type='button'
                className='h-11 w-full'
                disabled={!props.canSubmitPayment}
                onClick={props.onPayNow}
              >
                {props.paymentLoading ? (
                  <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                ) : null}
                {t('Pay Now')}
              </Button>
            ) : null}
          </>
        ) : (
          <Alert>
            <AlertDescription>
              {t(
                'Online topup is not enabled. Please use redemption code or contact administrator.'
              )}
            </AlertDescription>
          </Alert>
        )}
      </div>
    </section>
  )
}
