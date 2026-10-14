package com.newapi.admin.core.format

import java.time.ZoneId
import java.time.ZoneOffset
import java.util.Locale
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test

class FormatsTest {
    private val originalLocale = Locale.getDefault()
    private val utc: ZoneId = ZoneOffset.UTC

    @Before
    fun setUp() {
        Locale.setDefault(Locale.US)
    }

    @After
    fun tearDown() {
        Locale.setDefault(originalLocale)
    }

    @Test
    fun timestampsAreUnixSecondsAndZeroMeansNever() {
        assertEquals("2025-10-09 08:53:20", Formats.dateTime(1_760_000_000, utc))
        assertEquals("2025-10-09 16:53:20", Formats.dateTime(1_760_000_000, ZoneId.of("Asia/Shanghai")))
        assertEquals("10-09 08:53:20", Formats.shortDateTime(1_760_000_000, utc))
        assertEquals("-", Formats.dateTime(0, utc))
        assertEquals("-", Formats.shortDateTime(-1, utc))
    }

    @Test
    fun durationsMatchFormatUseTime() {
        assertEquals("3.0s", Formats.duration(3.0))
        assertEquals("0.3s", Formats.duration(0.35)) // toFixed rounds the exact binary value 0.34999...
        assertEquals("60.0s", Formats.duration(59.96))
        assertEquals("1m 0s", Formats.duration(60.0))
        assertEquals("2m 5s", Formats.duration(125.0))
    }

    @Test
    fun tokenCountsMatchFormatTokensAndToLocaleString() {
        assertEquals("-", Formats.compactTokens(0))
        assertEquals("999", Formats.compactTokens(999))
        assertEquals("12.3K", Formats.compactTokens(12_345))
        assertEquals("1.23M", Formats.compactTokens(1_234_567))
        assertEquals("1,234,567", Formats.count(1_234_567))
        assertEquals("56", Formats.count(56))
    }

    @Test
    fun fileSizesAre1024BasedWithOneDecimal() {
        assertEquals("0 B", Formats.fileSize(0))
        assertEquals("1023 B", Formats.fileSize(1023))
        assertEquals("1.0 KB", Formats.fileSize(1024))
        assertEquals("12.3 KB", Formats.fileSize(12_595))
        assertEquals("2.0 MB", Formats.fileSize(2_059_243))
        assertEquals("1.5 GB", Formats.fileSize(1_610_612_736))
    }

    @Test
    fun ratiosDropTrailingZerosAndKeepAtMostFourDecimals() {
        assertEquals("1.5x", Formats.ratio(1.5))
        assertEquals("2x", Formats.ratio(2.0))
        assertEquals("0.1235x", Formats.ratio(0.123456))
        assertEquals("1x", Formats.ratio(1.00001))
        assertEquals("-", Formats.ratio(Double.NaN))
    }
}
