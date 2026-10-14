package com.newapi.admin.ui.redemptions

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import com.newapi.admin.R
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.core.model.Redemption
import com.newapi.admin.core.model.RedemptionStatus
import com.newapi.admin.ui.common.UiMessage
import com.newapi.admin.ui.common.parseTypedDecimal
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.theme.Tone
import com.newapi.admin.ui.users.MAX_WALLET_QUOTA

/** `controller.AddRedemption` limits: a name of 1..20 runes and 1..100 codes per request. */
internal const val MAX_CODE_NAME_LENGTH = 20
internal val CODE_COUNT_RANGE = 1..100

/** How a code reads to an admin. The server stores an expired code as enabled; expiry is derived from the time. */
enum class CodeState { UNUSED, DISABLED, USED, EXPIRED }

internal fun Redemption.state(nowSeconds: Long): CodeState = when (status) {
    RedemptionStatus.USED -> CodeState.USED
    RedemptionStatus.DISABLED -> CodeState.DISABLED
    else -> if (isExpired(nowSeconds)) CodeState.EXPIRED else CodeState.UNUSED
}

/** Like the web: only an unused code can be edited, and a used or expired one cannot be switched on or off. */
internal val CodeState.canEdit: Boolean get() = this == CodeState.UNUSED
internal val CodeState.canToggle: Boolean get() = this == CodeState.UNUSED || this == CodeState.DISABLED

internal val Redemption.displayName: String get() = name.ifBlank { "#$id" }

/** The first and last 8 characters, as the web's mobile list shows a code. */
internal fun maskCode(key: String): String = if (key.length <= 16) key else key.take(8) + "******" + key.takeLast(8)

/** The amount per code, typed in the display currency (raw units when the display type is TOKENS). */
sealed interface CodeAmount {
    data object Empty : CodeAmount

    data class Invalid(val message: UiMessage.Res) : CodeAmount

    data class Valid(val quota: Long) : CodeAmount

    companion object {
        /**
         * Converted like the web's `parseQuotaFromDollars`. The server checks nothing here and stores 0 as its
         * column default (100), so the bounds are the app's: above 0 and at most `common.MaxWalletQuota`,
         * since redeeming adds the amount to a wallet.
         */
        fun parse(text: String, formatter: QuotaFormatter): CodeAmount {
            if (text.isBlank()) return Empty
            val amount = parseTypedDecimal(text) ?: return Invalid(UiMessage.Res(R.string.redemptions_error_amount_invalid))
            if (amount <= 0) return Invalid(UiMessage.Res(R.string.redemptions_error_amount_positive))
            val quota = formatter.toQuota(amount)
            if (quota <= 0) return Invalid(UiMessage.Res(R.string.redemptions_error_amount_too_small))
            if (quota > MAX_WALLET_QUOTA) {
                return Invalid(UiMessage.Res(R.string.redemptions_error_amount_too_large, listOf(formatter.format(MAX_WALLET_QUOTA))))
            }
            return Valid(quota)
        }
    }
}

@Composable
internal fun CodeStatusPill(state: CodeState, modifier: Modifier = Modifier) {
    val (label, tone) = when (state) {
        CodeState.UNUSED -> R.string.redemptions_status_unused to Tone.Success
        CodeState.DISABLED -> R.string.redemptions_status_disabled to Tone.Neutral
        CodeState.USED -> R.string.redemptions_status_used to Tone.Neutral
        CodeState.EXPIRED -> R.string.redemptions_status_expired to Tone.Warning
    }
    StatusPill(stringResource(label), tone, modifier)
}
