package com.newapi.admin.core.format

import com.newapi.admin.core.model.SystemStatus

/**
 * Converts raw quota units to display text exactly like the web dashboard
 * (`web/src/lib/currency.ts`, `web/src/lib/format.ts`), using the `/api/status` display settings.
 */
class QuotaFormatter(status: SystemStatus) {

    enum class DisplayType { USD, CNY, TOKENS, CUSTOM }

    /** Unknown strings fall back to USD, like the web. */
    val displayType: DisplayType = TODO()

    /** `quota_per_unit`, 500000 when missing or <= 0. */
    val quotaPerUnit: Double = TODO()

    /** USD -> display currency rate: 1 (USD), usd_exchange_rate (CNY), custom rate (CUSTOM); <= 0 falls back to 1. */
    val rate: Double = TODO()

    /** Label for amount inputs: "USD", "CNY", the custom symbol, or "Tokens". */
    val unitLabel: String = TODO()

    val isTokens: Boolean get() = displayType == DisplayType.TOKENS

    /** Balances and totals (`formatQuota`: 2/4 decimals, TOKENS abbreviated with "k"). */
    fun format(quota: Long): String = TODO()

    /** Log costs (`formatLogQuota`: 4/6 decimals, no abbreviation). */
    fun formatLog(quota: Long): String = TODO()

    /** A USD amount (channel balance, unit prices) in the display currency; TOKENS mode shows USD. */
    fun formatUsd(usd: Double): String = TODO()

    /** Amount typed by an admin -> raw quota units (`parseQuotaFromDollars`, JS Math.round semantics). */
    fun toQuota(amount: Double): Long = TODO()

    /** Raw quota -> amount in the display unit (inverse of [toQuota], unrounded). */
    fun toAmount(quota: Long): Double = TODO()

    companion object {
        val Default = QuotaFormatter(SystemStatus())
    }
}
