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
import { render, screen } from '@testing-library/react'
import { describe, expect, test, vi } from 'vitest'

import type { PaymentMethod, TopupInfo } from '@/features/wallet/types'

const topupInfo: TopupInfo = {
  enable_online_topup: true,
  enable_stripe_topup: false,
  pay_methods: [
    { name: 'Alipay', type: 'alipay' },
    { name: 'WeChat Pay', type: 'wxpay' },
  ],
  min_topup: 1,
  stripe_min_topup: 1,
  amount_options: [50, 100],
  discount: {},
  enable_redemption: true,
}

const pageState: {
  selectedPaymentMethod?: PaymentMethod
  canSubmitPayment: boolean
  paymentAmount: number
} = {
  selectedPaymentMethod: undefined,
  canSubmitPayment: false,
  paymentAmount: 100,
}

vi.mock('@/features/wallet/hooks', () => ({
  useWalletPage: () => ({
    user: null,
    userLoading: false,
    topupInfo,
    presetAmounts: [{ value: 50 }, { value: 100 }],
    topupLoading: false,
    topupAmount: 100,
    selectedPreset: 100,
    selectedPaymentMethod: pageState.selectedPaymentMethod,
    selectedWaffoMethodIndex: null,
    paymentLoading: null,
    paymentAmount: pageState.paymentAmount,
    calculating: false,
    processing: false,
    confirmDialogOpen: false,
    setConfirmDialogOpen: vi.fn(),
    transferDialogOpen: false,
    setTransferDialogOpen: vi.fn(),
    billingDialogOpen: false,
    setBillingDialogOpen: vi.fn(),
    redemptionCode: '',
    setRedemptionCode: vi.fn(),
    redeeming: false,
    creemDialogOpen: false,
    setCreemDialogOpen: vi.fn(),
    selectedCreemProduct: null,
    creemProcessing: false,
    affiliateLink: 'https://example.com/?aff=1',
    affiliateLoading: false,
    transferring: false,
    priceRatio: 1,
    usdExchangeRate: 1,
    discountRate: 1,
    canSubmitPayment: pageState.canSubmitPayment,
    fetchUser: vi.fn(),
    handleSelectPreset: vi.fn(),
    handleTopupAmountChange: vi.fn(),
    handlePaymentMethodSelect: vi.fn(),
    handlePaymentConfirm: vi.fn(),
    handleRedeem: vi.fn(),
    handleTransfer: vi.fn(),
    handleCreemProductSelect: vi.fn(),
    handleCreemConfirm: vi.fn(),
    handleWaffoMethodSelect: vi.fn(),
    handlePayNow: vi.fn(),
  }),
}))

vi.mock('@/features/wallet/components/dialogs/payment-confirm-dialog', () => ({
  PaymentConfirmDialog: () => null,
}))

vi.mock('@/features/wallet/components/dialogs/billing-history-dialog', () => ({
  BillingHistoryDialog: () => null,
}))

vi.mock('@/features/wallet/components/dialogs/creem-confirm-dialog', () => ({
  CreemConfirmDialog: () => null,
}))

const { NextWallet } = await import('../index')

describe('next wallet layout', () => {
  test('renders the split recharge page without stats, affiliate, or plans', () => {
    pageState.selectedPaymentMethod = undefined
    pageState.canSubmitPayment = false
    pageState.paymentAmount = 100

    render(<NextWallet />)

    const page = document.querySelector('[data-slot="next-wallet"]')
    expect(page).toBeInTheDocument()
    expect(page).toHaveClass('flex')
    expect(page).toHaveClass('flex-col')

    const columns = document.querySelector('[data-slot="next-wallet-columns"]')
    expect(columns).toHaveClass(
      'lg:grid-cols-[minmax(0,0.9fr)_minmax(380px,1.1fr)]'
    )

    const hero = document.querySelector('[data-slot="next-wallet-hero"]')
    expect(hero).toHaveClass('hidden')
    expect(hero).toHaveClass('lg:flex')
    expect(hero).toHaveClass('lg:self-center')
    expect(hero).not.toHaveClass('lg:sticky')
    expect(hero).not.toHaveClass('lg:h-[calc(100svh-8rem)]')
    expect(
      hero?.querySelectorAll('.landing-animate-fade-up').length
    ).toBeGreaterThanOrEqual(3)
    expect(hero).toHaveTextContent('Add Funds')
    expect(hero).not.toHaveTextContent('Wallet')
    expect(hero).toHaveTextContent(
      'Choose an amount and payment method. Quota arrives instantly.'
    )

    const funds = document.querySelector('[data-slot="next-wallet-add-funds"]')
    expect(funds).toHaveClass('rounded-xl')
    expect(funds).toHaveClass('border')
    expect(screen.getByRole('button', { name: 'Pay Now' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Order History' })).toBeVisible()

    const redeem = document.querySelector('[data-slot="next-wallet-redeem"]')
    expect(redeem).toBeInTheDocument()
    expect(redeem).not.toBe(funds)
    expect(redeem).toHaveClass('lg:col-start-2')
    expect(
      screen.getByRole('heading', { name: 'Redemption Code' })
    ).toBeVisible()
    expect(hero?.parentElement).toBe(columns)
    expect(funds?.parentElement).toBe(columns)
    expect(redeem?.parentElement).toBe(columns)

    expect(screen.queryByText('Current Balance')).toBeNull()
    expect(screen.queryByText('Total Usage')).toBeNull()
    expect(screen.queryByText('API Requests')).toBeNull()
    expect(screen.queryByText('Referral Program')).toBeNull()
    expect(screen.queryByText('Subscription Plans')).toBeNull()
    expect(document.querySelector('main')).toBeNull()
  })

  test('enables pay now after a payment method is selected', () => {
    pageState.selectedPaymentMethod = { name: 'Alipay', type: 'alipay' }
    pageState.canSubmitPayment = true
    pageState.paymentAmount = 100

    render(<NextWallet />)

    expect(screen.getByRole('button', { name: 'Pay Now' })).toBeEnabled()
  })

  test('lays out payment methods two per row', () => {
    pageState.selectedPaymentMethod = undefined
    pageState.canSubmitPayment = false
    pageState.paymentAmount = 100

    render(<NextWallet />)

    const methods = document.querySelector(
      '[data-slot="next-wallet-pay-methods"]'
    )
    expect(methods).toHaveClass('grid')
    expect(methods).toHaveClass('grid-cols-2')
    expect(screen.getByRole('button', { name: 'Alipay' }).parentElement).toBe(
      methods
    )
    expect(
      screen.getByRole('button', { name: 'WeChat Pay' }).parentElement
    ).toBe(methods)
  })

  test('shows a local pay estimate when the server quote is zero', () => {
    pageState.selectedPaymentMethod = undefined
    pageState.canSubmitPayment = false
    pageState.paymentAmount = 0

    render(<NextWallet />)

    expect(
      document.querySelector('[data-slot="next-wallet-payable"]')
    ).toHaveTextContent('100')
  })
})
