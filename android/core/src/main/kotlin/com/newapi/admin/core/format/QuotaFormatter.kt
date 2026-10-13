package com.newapi.admin.core.format

import com.newapi.admin.core.model.SystemStatus
import java.math.BigDecimal
import java.math.RoundingMode
import java.text.DecimalFormat
import java.text.NumberFormat
import java.util.Currency
import java.util.Locale
import kotlin.math.abs
import kotlin.math.pow

/**
 * Converts raw quota units to display text exactly like the web dashboard
 * (`web/src/lib/currency.ts`, `web/src/lib/format.ts`), using the `/api/status` display settings.
 *
 * Grouping and symbol placement follow the device default locale at the time of each call (the web
 * uses the browser locale); USD and CNY always use the narrow symbols `$` and `¥`.
 */
class QuotaFormatter(status: SystemStatus) {

    enum class DisplayType { USD, CNY, TOKENS, CUSTOM }

    /** Unknown strings fall back to USD, like the web. */
    val displayType: DisplayType = when (status.quotaDisplayType) {
        "CNY" -> DisplayType.CNY
        "TOKENS" -> DisplayType.TOKENS
        "CUSTOM" -> DisplayType.CUSTOM
        else -> DisplayType.USD
    }

    /** `quota_per_unit`, 500000 when missing or <= 0. */
    val quotaPerUnit: Double = status.quotaPerUnit.takeIf { it.isFinite() && it > 0 } ?: DEFAULT_QUOTA_PER_UNIT

    /** USD -> display currency rate: 1 (USD), usd_exchange_rate (CNY), custom rate (CUSTOM); <= 0 falls back to 1. */
    val rate: Double = when (displayType) {
        DisplayType.CNY -> status.usdExchangeRate.positiveOr(1.0)
        DisplayType.CUSTOM -> status.customCurrencyExchangeRate.positiveOr(1.0)
        DisplayType.USD, DisplayType.TOKENS -> 1.0
    }

    private val customSymbol: String = status.customCurrencySymbol.trim().ifEmpty { DEFAULT_CUSTOM_SYMBOL }

    /** Label for amount inputs: "USD", "CNY", the custom symbol, or "Tokens". */
    val unitLabel: String = when (displayType) {
        DisplayType.USD -> "USD"
        DisplayType.CNY -> "CNY"
        DisplayType.CUSTOM -> customSymbol
        DisplayType.TOKENS -> "Tokens"
    }

    val isTokens: Boolean get() = displayType == DisplayType.TOKENS

    /** Balances and totals (`formatQuota`: 2/4 decimals, TOKENS abbreviated with "k"). */
    fun format(quota: Long): String = formatQuotaWithCurrency(quota, digitsLarge = 2, digitsSmall = 4, abbreviate = true)

    /** Log costs (`formatLogQuota`: 4/6 decimals, no abbreviation). */
    fun formatLog(quota: Long): String = formatQuotaWithCurrency(quota, digitsLarge = 4, digitsSmall = 6, abbreviate = false)

    /**
     * A USD amount (channel balance, unit prices) in the display currency; TOKENS mode shows USD
     * (`formatBillingCurrencyFromUSD`, 2/4 decimals). NaN gives "-".
     */
    fun formatUsd(usd: Double): String {
        if (usd.isNaN()) return "-"
        return when (displayType) {
            DisplayType.TOKENS, DisplayType.USD -> formatCurrency(usd, 2, 4, "USD", "$")
            DisplayType.CNY -> formatCurrency(usd * rate, 2, 4, "CNY", "¥")
            DisplayType.CUSTOM -> formatCustom(usd * rate, 2, 4)
        }
    }

    /** Amount typed by an admin -> raw quota units (`parseQuotaFromDollars`, JS Math.round semantics). */
    fun toQuota(amount: Double): Long {
        if (!amount.isFinite()) return 0
        if (isTokens) return Math.round(amount)
        val usd = if (rate > 0) amount / rate else amount
        return Math.round(usd * quotaPerUnit)
    }

    /** Raw quota -> amount in the display unit (inverse of [toQuota], unrounded). */
    fun toAmount(quota: Long): Double = if (isTokens) quota.toDouble() else quota / quotaPerUnit * rate

    private fun formatQuotaWithCurrency(quota: Long, digitsLarge: Int, digitsSmall: Int, abbreviate: Boolean): String {
        val usd = quota / quotaPerUnit
        return when (displayType) {
            // The web goes through USD and back, so keep the same floating-point steps.
            DisplayType.TOKENS -> formatWithSuffix(usd * quotaPerUnit, digitsLarge = 0, digitsSmall = digitsSmall, abbreviate = abbreviate)
            DisplayType.USD -> formatCurrency(usd, digitsLarge, digitsSmall, "USD", "$")
            DisplayType.CNY -> formatCurrency(usd * rate, digitsLarge, digitsSmall, "CNY", "¥")
            DisplayType.CUSTOM -> formatCustom(usd * rate, digitsLarge, digitsSmall)
        }
    }

    /** `Intl.NumberFormat(undefined, {style:'currency', currencyDisplay:'narrowSymbol', minimumFractionDigits:0, ...})`. */
    private fun formatCurrency(value: Double, digitsLarge: Int, digitsSmall: Int, currencyCode: String, narrowSymbol: String): String {
        val digits = fractionDigits(value, digitsLarge, digitsSmall)
        val adjusted = clampTiny(value, digits)
        val format = NumberFormats.currency(Locale.getDefault(), currencyCode, narrowSymbol, digits)
        return format.format(BigDecimal(adjusted).setScale(digits, RoundingMode.HALF_UP))
    }

    /** CUSTOM: `` `${symbol} ${Intl decimal}` `` with a space. */
    private fun formatCustom(value: Double, digitsLarge: Int, digitsSmall: Int): String {
        val digits = fractionDigits(value, digitsLarge, digitsSmall)
        val adjusted = clampTiny(value, digits)
        val format = NumberFormats.decimal(Locale.getDefault(), digits)
        return "$customSymbol ${format.format(BigDecimal(adjusted).setScale(digits, RoundingMode.HALF_UP))}"
    }

    companion object {
        private const val DEFAULT_QUOTA_PER_UNIT = 500_000.0
        private const val DEFAULT_CUSTOM_SYMBOL = "¤"

        val Default = QuotaFormatter(SystemStatus())

        private fun Double.positiveOr(fallback: Double): Double = if (isFinite() && this > 0) this else fallback

        private fun fractionDigits(value: Double, digitsLarge: Int, digitsSmall: Int): Int =
            if (abs(value) >= 1) digitsLarge else digitsSmall

        /** `adjustForMinimum`: tiny non-zero amounts show as ±10^-digits instead of 0. */
        private fun clampTiny(value: Double, digits: Int): Double {
            if (value == 0.0) return value
            val threshold = 10.0.pow(-digits)
            return if (abs(value) < threshold) (if (value > 0) threshold else -threshold) else value
        }

        /** `formatNumberWithSuffix`: "k" above 1000 when abbreviating, no grouping. */
        internal fun formatWithSuffix(value: Double, digitsLarge: Int, digitsSmall: Int, abbreviate: Boolean): String {
            if (abbreviate && abs(value) >= 1000) return JsNumbers.stripTrailingZeros(JsNumbers.toFixed(value / 1000, 1)) + "k"
            return JsNumbers.stripTrailingZeros(JsNumbers.toFixed(value, fractionDigits(value, digitsLarge, digitsSmall)))
        }
    }
}

/**
 * Per-thread cache of locale formats (creating them is slow on Android and they are not thread-safe).
 * Formats are configured once and never mutated afterwards.
 */
internal object NumberFormats {
    private val cache: ThreadLocal<HashMap<String, NumberFormat>> = ThreadLocal.withInitial { HashMap() }

    /** Currency style with [symbol] in place of the locale's symbol, 0..[digits] fraction digits, ties away from zero. */
    fun currency(locale: Locale, currencyCode: String, symbol: String, digits: Int): NumberFormat =
        cache.get().getOrPut("currency|$locale|$currencyCode|$symbol|$digits") {
            NumberFormat.getCurrencyInstance(locale).apply {
                if (this is DecimalFormat) {
                    currency = Currency.getInstance(currencyCode)
                    decimalFormatSymbols = decimalFormatSymbols.apply { currencySymbol = symbol }
                }
                minimumFractionDigits = 0
                maximumFractionDigits = digits
                roundingMode = RoundingMode.HALF_UP
            }
        }

    /** Grouped decimal with 0..[digits] fraction digits, ties away from zero. */
    fun decimal(locale: Locale, digits: Int): NumberFormat =
        cache.get().getOrPut("decimal|$locale|$digits") {
            NumberFormat.getNumberInstance(locale).apply {
                minimumFractionDigits = 0
                maximumFractionDigits = digits
                roundingMode = RoundingMode.HALF_UP
            }
        }

    /** Grouped integer. */
    fun integer(locale: Locale): NumberFormat =
        cache.get().getOrPut("integer|$locale") { NumberFormat.getIntegerInstance(locale) }
}

/** JavaScript number-to-string helpers the web formatters rely on. */
internal object JsNumbers {
    /** `Number.prototype.toFixed`: exact binary value, ties away from zero, "-" kept for negative values. */
    fun toFixed(value: Double, digits: Int): String {
        if (!value.isFinite()) return value.toString()
        val text = BigDecimal(abs(value)).setScale(digits, RoundingMode.HALF_UP).toPlainString()
        return if (value < 0) "-$text" else text
    }

    /** `removeTrailingZeros`: "1.50" -> "1.5", "2.00" -> "2"; strings without a dot are unchanged. */
    fun stripTrailingZeros(text: String): String {
        if (!text.contains('.')) return text
        return text.trimEnd('0').trimEnd('.')
    }
}
