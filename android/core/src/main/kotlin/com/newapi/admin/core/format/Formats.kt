package com.newapi.admin.core.format

import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale
import kotlin.math.abs
import kotlin.math.floor

/**
 * Number and time formatting shared by all screens, matching the web helpers in `web/src/lib/format.ts`.
 * Only [count] is locale dependent (grouping separators of the device default locale, like `toLocaleString`).
 */
object Formats {
    private val dateTimeFormatter = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss", Locale.ROOT)
    private val shortDateTimeFormatter = DateTimeFormatter.ofPattern("MM-dd HH:mm:ss", Locale.ROOT)

    /** `yyyy-MM-dd HH:mm:ss` in [zone]; "-" for 0 or negative timestamps. */
    fun dateTime(epochSeconds: Long, zone: ZoneId = ZoneId.systemDefault()): String =
        if (epochSeconds <= 0) "-" else dateTimeFormatter.format(Instant.ofEpochSecond(epochSeconds).atZone(zone))

    /** `MM-dd HH:mm:ss` for compact list rows; "-" for 0 or negative. */
    fun shortDateTime(epochSeconds: Long, zone: ZoneId = ZoneId.systemDefault()): String =
        if (epochSeconds <= 0) "-" else shortDateTimeFormatter.format(Instant.ofEpochSecond(epochSeconds).atZone(zone))

    /** Web `formatUseTime`: under 60 s "3.0s", otherwise "2m 5s". */
    fun duration(seconds: Double): String {
        if (seconds < 60) return JsNumbers.toFixed(seconds, 1) + "s"
        val minutes = floor(seconds / 60).toLong()
        return "${minutes}m ${JsNumbers.toFixed(seconds % 60, 0)}s"
    }

    /** Grouped integer in the device default locale, e.g. 1234567 -> "1,234,567" (en). */
    fun count(value: Long): String = NumberFormats.integer(Locale.getDefault()).format(value)

    /** Web `formatTokens`: 0 -> "-", <1000 raw, <1e6 "12.3K", else "1.23M". */
    fun compactTokens(value: Long): String = when {
        value == 0L -> "-"
        value < 1000 -> value.toString()
        value < 1_000_000 -> JsNumbers.toFixed(value / 1000.0, 1) + "K"
        else -> JsNumbers.toFixed(value / 1_000_000.0, 2) + "M"
    }

    /** Download sizes, 1024-based: "512 B", "12.3 KB", "2.0 MB". */
    fun fileSize(bytes: Long): String {
        if (bytes < 1024) return "${bytes.coerceAtLeast(0)} B"
        val units = listOf("KB", "MB", "GB")
        var value = bytes / 1024.0
        var unit = 0
        while (value >= 1024 && unit < units.lastIndex) {
            value /= 1024
            unit++
        }
        return JsNumbers.toFixed(value, 1) + " " + units[unit]
    }

    /** Ratio like the web (`1.5x`), trailing zeros removed, at most 4 decimals; "-" when not finite. */
    fun ratio(value: Double): String {
        if (!value.isFinite()) return "-"
        val text = if (value % 1.0 == 0.0 && abs(value) < 1e15) {
            value.toLong().toString()
        } else {
            JsNumbers.toFixed(value, 4).let { if (it.contains('.')) it.trimEnd('0').trimEnd('.') else it }
        }
        return text + "x"
    }
}
