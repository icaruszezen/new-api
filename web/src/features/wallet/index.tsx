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
import { useCallback, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'

import { AffiliateRewardsCard } from './components/affiliate-rewards-card'
import { BillingHistoryDialog } from './components/dialogs/billing-history-dialog'
import { CreemConfirmDialog } from './components/dialogs/creem-confirm-dialog'
import { PaymentConfirmDialog } from './components/dialogs/payment-confirm-dialog'
import { TransferDialog } from './components/dialogs/transfer-dialog'
import { RechargeFormCard } from './components/recharge-form-card'
import { SubscriptionPlansCard } from './components/subscription-plans-card'
import { WalletStatsCard } from './components/wallet-stats-card'
import { useWalletPage } from './hooks'

interface WalletProps {
  initialShowHistory?: boolean
}

export function Wallet(props: WalletProps) {
  const { t } = useTranslation()
  const [showSubscriptionPanel, setShowSubscriptionPanel] = useState(true)
  const page = useWalletPage({
    initialShowHistory: props.initialShowHistory,
  })

  const handleSubscriptionAvailabilityChange = useCallback(
    (available: boolean) => {
      setShowSubscriptionPanel(available)
    },
    []
  )

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Wallet')}</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <div className='mx-auto flex w-full max-w-7xl flex-col gap-4 sm:gap-5'>
            <WalletStatsCard user={page.user} loading={page.userLoading} />

            <div
              className={
                showSubscriptionPanel
                  ? 'grid gap-4 xl:grid-cols-[minmax(0,1.05fr)_minmax(360px,0.95fr)] xl:items-start'
                  : 'grid gap-4'
              }
            >
              <div id='wallet-add-funds' className='scroll-mt-4'>
                <RechargeFormCard
                  topupInfo={page.topupInfo}
                  presetAmounts={page.presetAmounts}
                  selectedPreset={page.selectedPreset}
                  onSelectPreset={page.handleSelectPreset}
                  topupAmount={page.topupAmount}
                  onTopupAmountChange={page.handleTopupAmountChange}
                  paymentAmount={page.paymentAmount}
                  calculating={page.calculating}
                  onPaymentMethodSelect={page.handlePaymentMethodSelect}
                  paymentLoading={page.paymentLoading}
                  redemptionCode={page.redemptionCode}
                  onRedemptionCodeChange={page.setRedemptionCode}
                  onRedeem={page.handleRedeem}
                  redeeming={page.redeeming}
                  topupLink={page.topupInfo?.topup_link}
                  loading={page.topupLoading}
                  priceRatio={page.priceRatio}
                  usdExchangeRate={page.usdExchangeRate}
                  onOpenBilling={() => page.setBillingDialogOpen(true)}
                  creemProducts={page.topupInfo?.creem_products}
                  enableCreemTopup={page.topupInfo?.enable_creem_topup}
                  onCreemProductSelect={page.handleCreemProductSelect}
                  enableWaffoTopup={page.topupInfo?.enable_waffo_topup}
                  waffoPayMethods={page.topupInfo?.waffo_pay_methods}
                  waffoMinTopup={page.topupInfo?.waffo_min_topup}
                  onWaffoMethodSelect={page.handleWaffoMethodSelect}
                  enableWaffoPancakeTopup={
                    page.topupInfo?.enable_waffo_pancake_topup
                  }
                />
              </div>

              <SubscriptionPlansCard
                topupInfo={page.topupInfo}
                onAvailabilityChange={handleSubscriptionAvailabilityChange}
                userQuota={page.user?.quota}
                onPurchaseSuccess={page.fetchUser}
              />
            </div>

            <AffiliateRewardsCard
              user={page.user}
              affiliateLink={page.affiliateLink}
              onTransfer={() => page.setTransferDialogOpen(true)}
              complianceConfirmed={
                page.topupInfo?.payment_compliance_confirmed !== false
              }
              loading={page.affiliateLoading}
            />
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

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

      <TransferDialog
        open={page.transferDialogOpen}
        onOpenChange={page.setTransferDialogOpen}
        onConfirm={page.handleTransfer}
        availableQuota={page.user?.aff_quota ?? 0}
        transferring={page.transferring}
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
    </>
  )
}
