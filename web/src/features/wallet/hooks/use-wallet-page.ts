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
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { useStatus } from '@/hooks/use-status'
import { useSystemConfig } from '@/hooks/use-system-config'
import { getSelf } from '@/lib/api'

import { DEFAULT_DISCOUNT_RATE, PAYMENT_TYPES } from '../constants'
import {
  calculatePresetPricing,
  dispatchSelectedPayment,
  getDefaultPaymentType,
  getDisplayPaymentAmount,
  getMinTopupAmount,
  getPaymentMethodKey,
} from '../lib'
import type {
  CreemProduct,
  PaymentMethod,
  PresetAmount,
  UserWalletData,
  WaffoPayMethod,
} from '../types'
import { useAffiliate } from './use-affiliate'
import { useCreemPayment } from './use-creem-payment'
import { usePayment } from './use-payment'
import { useRedemption } from './use-redemption'
import { useTopupInfo } from './use-topup-info'
import { useWaffoPancakePayment } from './use-waffo-pancake-payment'
import { useWaffoPayment } from './use-waffo-payment'

export type UseWalletPageOptions = {
  initialShowHistory?: boolean
  confirmOnMethodSelect?: boolean
}

export function useWalletPage(options: UseWalletPageOptions = {}) {
  const confirmOnMethodSelect = options.confirmOnMethodSelect !== false
  const [user, setUser] = useState<UserWalletData | null>(null)
  const [userLoading, setUserLoading] = useState(true)
  const [topupAmount, setTopupAmount] = useState(0)
  const [selectedPreset, setSelectedPreset] = useState<number | null>(null)
  const [selectedPaymentMethod, setSelectedPaymentMethod] =
    useState<PaymentMethod>()
  const [selectedWaffoMethodIndex, setSelectedWaffoMethodIndex] = useState<
    number | null
  >(null)
  const [paymentLoading, setPaymentLoading] = useState<string | null>(null)
  const [confirmDialogOpen, setConfirmDialogOpen] = useState(false)
  const [transferDialogOpen, setTransferDialogOpen] = useState(false)
  const [billingDialogOpen, setBillingDialogOpen] = useState(false)
  const [redemptionCode, setRedemptionCode] = useState('')
  const [creemDialogOpen, setCreemDialogOpen] = useState(false)
  const [selectedCreemProduct, setSelectedCreemProduct] =
    useState<CreemProduct | null>(null)

  const { status } = useStatus()
  const { currency } = useSystemConfig()
  const { topupInfo, presetAmounts, loading: topupLoading } = useTopupInfo()
  const effectiveUsdExchangeRate = useMemo(() => {
    return currency?.quotaDisplayType === 'USD'
      ? 1
      : currency?.usdExchangeRate || 1
  }, [currency?.quotaDisplayType, currency?.usdExchangeRate])
  const priceRatio = (status?.price as number) || 1
  const {
    amount: paymentAmount,
    calculating,
    processing,
    calculatePaymentAmount,
    processPayment,
    setAmount: setPaymentAmount,
  } = usePayment()
  const {
    affiliateLink,
    loading: affiliateLoading,
    transferQuota,
    transferring,
  } = useAffiliate()
  const { redeeming, redeemCode } = useRedemption()
  const { processing: creemProcessing, processCreemPayment } = useCreemPayment()
  const { processing: waffoProcessing, processWaffoPayment } = useWaffoPayment()
  const { processing: pancakeProcessing, processWaffoPancakePayment } =
    useWaffoPancakePayment()

  const fetchUser = useCallback(async () => {
    try {
      setUserLoading(true)
      const response = await getSelf()
      if (response.success && response.data) {
        setUser(response.data as UserWalletData)
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to fetch user data:', error)
    } finally {
      setUserLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchUser()
  }, [fetchUser])

  useEffect(() => {
    if (options.initialShowHistory) {
      setBillingDialogOpen(true)
      window.history.replaceState({}, '', window.location.pathname)
    }
  }, [options.initialShowHistory])

  const topupAmountInitializedRef = useRef(false)
  useEffect(() => {
    if (topupInfo && !topupAmountInitializedRef.current) {
      topupAmountInitializedRef.current = true
      const minTopup = getMinTopupAmount(topupInfo)
      const discount = topupInfo.discount?.[minTopup] || DEFAULT_DISCOUNT_RATE
      setTopupAmount(minTopup)
      setSelectedPreset(minTopup)
      setPaymentAmount(
        calculatePresetPricing(minTopup, priceRatio, discount, 1).actualPrice
      )
      calculatePaymentAmount(minTopup, getDefaultPaymentType(topupInfo))
    }
  }, [topupInfo, calculatePaymentAmount, priceRatio, setPaymentAmount])

  const getCurrentPaymentType = useCallback(() => {
    return selectedPaymentMethod?.type || getDefaultPaymentType(topupInfo)
  }, [selectedPaymentMethod, topupInfo])

  const handleSelectPreset = (preset: PresetAmount) => {
    const discount =
      preset.discount ||
      topupInfo?.discount?.[preset.value] ||
      DEFAULT_DISCOUNT_RATE
    setTopupAmount(preset.value)
    setSelectedPreset(preset.value)
    setPaymentAmount(
      calculatePresetPricing(preset.value, priceRatio, discount, 1).actualPrice
    )
    calculatePaymentAmount(preset.value, getCurrentPaymentType())
  }

  const handleTopupAmountChange = (amount: number) => {
    const discount = topupInfo?.discount?.[amount] || DEFAULT_DISCOUNT_RATE
    setTopupAmount(amount)
    setSelectedPreset(null)
    setPaymentAmount(
      calculatePresetPricing(amount, priceRatio, discount, 1).actualPrice
    )
    calculatePaymentAmount(amount, getCurrentPaymentType())
  }

  const handlePaymentMethodSelect = async (method: PaymentMethod) => {
    setSelectedPaymentMethod(method)
    setSelectedWaffoMethodIndex(null)

    if (!confirmOnMethodSelect) {
      calculatePaymentAmount(topupAmount, method.type)
      return
    }

    setPaymentLoading(getPaymentMethodKey(method))
    try {
      const minTopup = getMinTopupAmount(topupInfo)
      if (topupAmount < minTopup) {
        return
      }
      await calculatePaymentAmount(topupAmount, method.type)
      setConfirmDialogOpen(true)
    } finally {
      setPaymentLoading(null)
    }
  }

  const handlePaymentConfirm = async () => {
    if (!selectedPaymentMethod) return

    const success = await dispatchSelectedPayment(
      selectedPaymentMethod,
      topupAmount,
      selectedWaffoMethodIndex,
      {
        regular: processPayment,
        waffo: processWaffoPayment,
        waffoPancake: processWaffoPancakePayment,
      }
    )

    if (success) {
      setConfirmDialogOpen(false)
      await fetchUser()
    }
  }

  const handleRedeem = async () => {
    if (!redemptionCode) return

    const success = await redeemCode(redemptionCode)
    if (success) {
      setRedemptionCode('')
      await fetchUser()
    }
  }

  const handleTransfer = async (amount: number) => {
    const success = await transferQuota(amount)
    if (success) {
      await fetchUser()
    }
    return success
  }

  const handleCreemProductSelect = (product: CreemProduct) => {
    setSelectedCreemProduct(product)
    setCreemDialogOpen(true)
  }

  const handleCreemConfirm = async () => {
    if (!selectedCreemProduct) return

    const success = await processCreemPayment(selectedCreemProduct.productId)
    if (success) {
      setCreemDialogOpen(false)
      setSelectedCreemProduct(null)
      await fetchUser()
    }
  }

  const handleWaffoMethodSelect = async (
    method: WaffoPayMethod,
    index: number
  ) => {
    setSelectedPaymentMethod({
      name: method.name,
      type: PAYMENT_TYPES.WAFFO,
      icon: method.icon,
    })
    setSelectedWaffoMethodIndex(index)

    if (!confirmOnMethodSelect) {
      calculatePaymentAmount(topupAmount, PAYMENT_TYPES.WAFFO)
      return
    }

    const loadingKey = `waffo-${index}`
    setPaymentLoading(loadingKey)
    try {
      await calculatePaymentAmount(topupAmount, PAYMENT_TYPES.WAFFO)
      setConfirmDialogOpen(true)
    } finally {
      setPaymentLoading(null)
    }
  }

  const handlePayNow = async () => {
    if (!selectedPaymentMethod) return

    const minTopup = getMinTopupAmount(topupInfo)
    if (topupAmount < minTopup) return

    setPaymentLoading(getPaymentMethodKey(selectedPaymentMethod))
    try {
      await calculatePaymentAmount(topupAmount, selectedPaymentMethod.type)
      setConfirmDialogOpen(true)
    } finally {
      setPaymentLoading(null)
    }
  }

  const getDiscountRate = useCallback(() => {
    return topupInfo?.discount?.[topupAmount] || DEFAULT_DISCOUNT_RATE
  }, [topupInfo, topupAmount])

  const minTopupAmount = getMinTopupAmount(topupInfo)
  const methodMinTopup = Math.max(
    selectedPaymentMethod?.min_topup || 0,
    minTopupAmount
  )
  const waffoMinTopup = topupInfo?.waffo_min_topup || 0
  const belowWaffoMin =
    selectedWaffoMethodIndex !== null && waffoMinTopup > topupAmount
  const canSubmitPayment =
    !!selectedPaymentMethod &&
    topupAmount >= methodMinTopup &&
    !belowWaffoMin &&
    !paymentLoading &&
    !calculating

  return {
    user,
    userLoading,
    topupInfo,
    presetAmounts,
    topupLoading,
    topupAmount,
    selectedPreset,
    selectedPaymentMethod,
    selectedWaffoMethodIndex,
    paymentLoading,
    paymentAmount: getDisplayPaymentAmount(
      paymentAmount,
      topupAmount,
      priceRatio,
      topupInfo?.discount?.[topupAmount] || DEFAULT_DISCOUNT_RATE
    ),
    calculating,
    processing: processing || waffoProcessing || pancakeProcessing,
    confirmDialogOpen,
    setConfirmDialogOpen,
    transferDialogOpen,
    setTransferDialogOpen,
    billingDialogOpen,
    setBillingDialogOpen,
    redemptionCode,
    setRedemptionCode,
    redeeming,
    creemDialogOpen,
    setCreemDialogOpen,
    selectedCreemProduct,
    creemProcessing,
    affiliateLink,
    affiliateLoading,
    transferring,
    priceRatio,
    usdExchangeRate: effectiveUsdExchangeRate,
    discountRate: getDiscountRate(),
    canSubmitPayment,
    fetchUser,
    handleSelectPreset,
    handleTopupAmountChange,
    handlePaymentMethodSelect,
    handlePaymentConfirm,
    handleRedeem,
    handleTransfer,
    handleCreemProductSelect,
    handleCreemConfirm,
    handleWaffoMethodSelect,
    handlePayNow,
  }
}
