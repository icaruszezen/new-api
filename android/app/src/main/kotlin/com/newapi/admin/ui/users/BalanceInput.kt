package com.newapi.admin.ui.users

import com.newapi.admin.R
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.core.model.QuotaMode
import com.newapi.admin.ui.common.UiMessage
import java.text.DecimalFormatSymbols

/** The amount typed into the adjust-balance dialog, converted to raw quota units and checked. */
sealed interface BalanceInput {
    data object Empty : BalanceInput

    data class Invalid(val message: UiMessage.Res) : BalanceInput

    /** [value] is what `add_quota` gets (positive for add/subtract); [resultingBalance] is the preview. */
    data class Valid(val value: Long, val resultingBalance: Long) : BalanceInput

    companion object {
        /**
         * [text] is in the display currency (raw units when the display type is TOKENS), converted like the
         * web's `parseQuotaFromDollars`. Applies the server's wallet bounds (`common.MaxWalletQuota`):
         * add must keep the balance at or below the maximum; subtract may go negative (the server has no
         * floor) but not past the negative bound; "set to" accepts 0 up to the maximum.
         */
        fun parse(text: String, mode: QuotaMode, currentQuota: Long, formatter: QuotaFormatter): BalanceInput {
            // A comma is the decimal mark only where the locale writes it that way (and no '.' is typed);
            // elsewhere it is a grouping separator, so "1,000" is rejected instead of read as 1.
            val trimmed = text.trim()
            val decimalComma = DecimalFormatSymbols.getInstance().decimalSeparator == ','
            val normalized = if (decimalComma && '.' !in trimmed) trimmed.replace(',', '.') else trimmed
            if (normalized.isEmpty()) return Empty
            val amount = normalized.toDoubleOrNull()?.takeIf { it.isFinite() }
                ?: return Invalid(UiMessage.Res(R.string.users_quota_error_invalid))
            val tooLarge = Invalid(UiMessage.Res(R.string.users_quota_error_too_large, listOf(formatter.format(MAX_WALLET_QUOTA))))

            if (mode == QuotaMode.OVERRIDE) {
                if (amount < 0) return Invalid(UiMessage.Res(R.string.users_quota_error_negative))
                val value = formatter.toQuota(amount)
                if (value > MAX_WALLET_QUOTA) return tooLarge
                return Valid(value, value)
            }

            if (amount <= 0) return Invalid(UiMessage.Res(R.string.users_quota_error_positive))
            val value = formatter.toQuota(amount)
            if (value <= 0) return Invalid(UiMessage.Res(R.string.users_quota_error_too_small))
            return when (mode) {
                QuotaMode.ADD ->
                    if (value > MAX_WALLET_QUOTA - currentQuota) tooLarge else Valid(value, currentQuota + value)
                QuotaMode.SUBTRACT ->
                    if (value > MAX_WALLET_QUOTA || currentQuota - value < -MAX_WALLET_QUOTA) {
                        Invalid(UiMessage.Res(R.string.users_quota_error_range))
                    } else {
                        Valid(value, currentQuota - value)
                    }
                QuotaMode.OVERRIDE -> Valid(value, value)
            }
        }
    }
}
