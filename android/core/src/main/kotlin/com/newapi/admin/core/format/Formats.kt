package com.newapi.admin.core.format

import java.time.ZoneId

/** Locale-independent number and time formatting shared by all screens. */
object Formats {
    /** `yyyy-MM-dd HH:mm:ss` in [zone]; "-" for 0 or negative timestamps. */
    fun dateTime(epochSeconds: Long, zone: ZoneId = ZoneId.systemDefault()): String = TODO()

    /** `MM-dd HH:mm:ss` for compact list rows; "-" for 0 or negative. */
    fun shortDateTime(epochSeconds: Long, zone: ZoneId = ZoneId.systemDefault()): String = TODO()

    /** Web `formatUseTime`: under 60 s "3.0s", otherwise "2m 5s". */
    fun duration(seconds: Double): String = TODO()

    /** Grouped integer, e.g. 1234567 -> "1,234,567". */
    fun count(value: Long): String = TODO()

    /** Web `formatTokens`: 0 -> "-", <1000 raw, <1e6 "12.3K", else "1.23M". */
    fun compactTokens(value: Long): String = TODO()

    /** Ratio like the web (`1.5x`), trailing zeros removed, at most 4 decimals. */
    fun ratio(value: Double): String = TODO()
}
