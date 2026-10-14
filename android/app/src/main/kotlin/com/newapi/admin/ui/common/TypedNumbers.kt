package com.newapi.admin.ui.common

import java.text.DecimalFormatSymbols

/**
 * A decimal number typed by an admin, or null when [text] is not a finite number. A comma is the decimal
 * mark only where the locale writes it that way (and no '.' is typed); elsewhere it is a grouping
 * separator, so "1,000" is rejected instead of read as 1.
 */
fun parseTypedDecimal(text: String): Double? {
    val trimmed = text.trim()
    val decimalComma = DecimalFormatSymbols.getInstance().decimalSeparator == ','
    val normalized = if (decimalComma && '.' !in trimmed) trimmed.replace(',', '.') else trimmed
    return normalized.toDoubleOrNull()?.takeIf { it.isFinite() }
}
