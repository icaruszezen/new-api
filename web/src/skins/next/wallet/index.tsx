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
import { BillingHistoryDialog } from '@/features/wallet/components/dialogs/billing-history-dialog'
import { CreemConfirmDialog } from '@/features/wallet/components/dialogs/creem-confirm-dialog'
import { PaymentConfirmDialog } from '@/features/wallet/components/dialogs/payment-confirm-dialog'
import { useWalletPage } from '@/features/wallet/hooks'

import { NextWalletAddFundsCard } from './components/add-funds-card'
import { NextWalletHero } from './components/hero'
import { NextWalletRedeemCard } from './components/redeem-card'

type NextWalletProps = {
  initialShowHistory?: boolean
}

export function NextWallet(props: NextWalletProps) {
  const page = useWalletPage({
    initialShowHistory: props.initialShowHistory,
    confirmOnMethodSelect: false,
  })

  return (
    <div data-slot='next-wallet' className='flex flex-col gap-5 pb-10'>
      <div
        data-slot='next-wallet-columns'
        className='grid items-start gap-5 lg:grid-cols-[minmax(0,0.9fr)_minmax(380px,1.1fr)]'
      >
        <NextWalletHero amount={page.topupAmount} loading={page.topupLoading} />

        <NextWalletAddFundsCard
          topupInfo={page.topupInfo}
          presetAmounts={page.presetAmounts}
          selectedPreset={page.selectedPreset}
          onSelectPreset={page.handleSelectPreset}
          topupAmount={page.topupAmount}
          onTopupAmountChange={page.handleTopupAmountChange}
          paymentAmount={page.paymentAmount}
          calculating={page.calculating}
          selectedPaymentMethod={page.selectedPaymentMethod}
          selectedWaffoMethodIndex={page.selectedWaffoMethodIndex}
          onPaymentMethodSelect={page.handlePaymentMethodSelect}
          onWaffoMethodSelect={page.handleWaffoMethodSelect}
          paymentLoading={page.paymentLoading}
          loading={page.topupLoading}
          priceRatio={page.priceRatio}
          usdExchangeRate={page.usdExchangeRate}
          canSubmitPayment={page.canSubmitPayment}
          onPayNow={page.handlePayNow}
          onOpenBilling={() => page.setBillingDialogOpen(true)}
          onCreemProductSelect={page.handleCreemProductSelect}
        />
        <NextWalletRedeemCard
          code={page.redemptionCode}
          onCodeChange={page.setRedemptionCode}
          onRedeem={page.handleRedeem}
          redeeming={page.redeeming}
          topupLink={page.topupInfo?.topup_link}
          enabled={page.topupInfo?.enable_redemption !== false}
          loading={page.topupLoading}
        />
      </div>

      <PaymentConfirmDialog
        open={page.confirmDialogOpen}
        onOpenChange={page.setConfirmDialogOpen}
        onConfirm={page.handlePaymentConfirm}
        topupAmount={page.topupAmount}
        paymentAmount={page.paymentAmount}
        paymentMethod={page.selectedPaymentMethod}
        calculating={page.calculating}
        processing={page.processing}
        discountRate={page.discountRate}
        usdExchangeRate={page.usdExchangeRate}
      />

      <BillingHistoryDialog
        open={page.billingDialogOpen}
        onOpenChange={page.setBillingDialogOpen}
      />

      <CreemConfirmDialog
        open={page.creemDialogOpen}
        onOpenChange={page.setCreemDialogOpen}
        onConfirm={page.handleCreemConfirm}
        product={page.selectedCreemProduct}
        processing={page.creemProcessing}
      />
    </div>
  )
}
