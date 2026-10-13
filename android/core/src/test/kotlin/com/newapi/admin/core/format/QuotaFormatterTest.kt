package com.newapi.admin.core.format

import com.newapi.admin.core.model.SystemStatus
import java.util.Locale
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test

/** Expected strings are what `formatQuota` / `formatLogQuota` in web/src/lib produce in an en-US browser. */
class QuotaFormatterTest {
    private val originalLocale = Locale.getDefault()

    @Before
    fun setUp() {
        Locale.setDefault(Locale.US)
    }

    @After
    fun tearDown() {
        Locale.setDefault(originalLocale)
    }

    private fun formatter(
        type: String,
        quotaPerUnit: Double = 500_000.0,
        usdRate: Double = 7.3,
        symbol: String = "¤",
        customRate: Double = 1.0,
    ) = QuotaFormatter(
        SystemStatus(
            quotaPerUnit = quotaPerUnit,
            quotaDisplayType = type,
            usdExchangeRate = usdRate,
            customCurrencySymbol = symbol,
            customCurrencyExchangeRate = customRate,
        ),
    )

    @Test
    fun usdUsesTwoOrFourDecimalsDropsTrailingZerosAndClampsTinyAmounts() {
        val usd = formatter("USD")
        val cases = listOf(
            750_000L to "$1.5",
            123L to "$0.0002",
            20L to "$0.0001",
            617_283_950L to "$1,234.57",
            -125_000L to "-$0.25",
            0L to "$0",
            5_000_000L to "$10",
        )
        for ((quota, expected) in cases) assertEquals("format($quota)", expected, usd.format(quota))
    }

    @Test
    fun logCostsUseFourOrSixDecimals() {
        val usd = formatter("USD")
        assertEquals("$0.00004", usd.formatLog(20))
        assertEquals("$0.000246", usd.formatLog(123))
        assertEquals("$1,234.5679", usd.formatLog(617_283_950))
        assertEquals("$0.000002", usd.formatLog(1))
        assertEquals("$0", usd.formatLog(0))
    }

    @Test
    fun cnyAndCustomApplyTheirRates() {
        assertEquals("¥70", formatter("CNY", usdRate = 7.0).format(5_000_000))
        assertEquals("¥73", formatter("CNY").format(5_000_000))
        assertEquals("€ 9.5", formatter("CUSTOM", symbol = "€", customRate = 0.95).format(5_000_000))
        assertEquals("€ -9.5", formatter("CUSTOM", symbol = "€", customRate = 0.95).format(-5_000_000))
    }

    @Test
    fun invalidSettingsFallBackLikeTheWeb() {
        // Rate <= 0 -> 1, blank symbol -> "¤", quota_per_unit <= 0 -> 500000, unknown type -> USD.
        assertEquals("¥10", formatter("CNY", usdRate = 0.0).format(5_000_000))
        assertEquals("¤ 10", formatter("CUSTOM", symbol = "  ", customRate = -2.0).format(5_000_000))
        assertEquals("$10", formatter("USD", quotaPerUnit = 0.0).format(5_000_000))
        assertEquals(QuotaFormatter.DisplayType.USD, formatter("EUR").displayType)
        assertEquals("$10", formatter("EUR").format(5_000_000))
    }

    @Test
    fun tokensAbbreviateWithKInBalancesButNotInLogs() {
        val tokens = formatter("TOKENS")
        val cases = listOf(1_500_000L to "1500k", 1_234L to "1.2k", 1_000L to "1k", 999L to "999", 0L to "0", -1_500L to "-1.5k")
        for ((quota, expected) in cases) assertEquals("format($quota)", expected, tokens.format(quota))
        assertEquals("1500000", tokens.formatLog(1_500_000))
        assertEquals("Tokens", tokens.unitLabel)
    }

    @Test
    fun usdAmountsAreShownInTheDisplayCurrencyAndAsUsdInTokensMode() {
        assertEquals("$12.5", formatter("TOKENS").formatUsd(12.5))
        assertEquals("¥7.3", formatter("CNY").formatUsd(1.0))
        assertEquals("$0.0001", formatter("USD").formatUsd(0.00001))
        assertEquals("-", formatter("USD").formatUsd(Double.NaN))
    }

    @Test
    fun typedAmountsConvertToRawQuotaWithJsRounding() {
        assertEquals(750_000L, formatter("USD").toQuota(1.5))
        assertEquals(5_000_000L, formatter("CNY", usdRate = 7.0).toQuota(70.0))
        assertEquals(3L, formatter("TOKENS").toQuota(2.5))
        assertEquals(-2L, formatter("TOKENS").toQuota(-2.5)) // Math.round ties toward +infinity
        assertEquals(1L, formatter("USD").toQuota(0.000001)) // 0.5 units rounds up
        assertEquals(0L, formatter("USD").toQuota(Double.NaN))
    }

    @Test
    fun toAmountIsTheInverseOfToQuota() {
        for (type in listOf("USD", "CNY", "CUSTOM", "TOKENS")) {
            val formatter = formatter(type, customRate = 0.9)
            for (quota in listOf(0L, 1L, 123L, 750_000L, 617_283_950L, -125_000L)) {
                assertEquals("$type $quota", quota, formatter.toQuota(formatter.toAmount(quota)))
            }
        }
        assertEquals(10.0, formatter("CNY", usdRate = 7.0).toAmount(5_000_000) / 7.0, 1e-9)
    }

    @Test
    fun unitLabelsNameTheInputUnit() {
        assertEquals("USD", formatter("USD").unitLabel)
        assertEquals("CNY", formatter("CNY").unitLabel)
        assertEquals("€", formatter("CUSTOM", symbol = " € ").unitLabel)
    }
}
