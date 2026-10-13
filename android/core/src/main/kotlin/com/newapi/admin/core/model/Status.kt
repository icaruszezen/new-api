package com.newapi.admin.core.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** Subset of `GET /api/status` the app needs. The endpoint is public. */
@Serializable
data class SystemStatus(
    @SerialName("system_name") val systemName: String = "",
    val version: String = "",
    val logo: String = "",
    @SerialName("server_address") val serverAddress: String = "",
    val setup: Boolean = true,
    @SerialName("quota_per_unit") val quotaPerUnit: Double = 500000.0,
    @SerialName("quota_display_type") val quotaDisplayType: String = "USD",
    @SerialName("usd_exchange_rate") val usdExchangeRate: Double = 1.0,
    @SerialName("custom_currency_symbol") val customCurrencySymbol: String = "¤",
    @SerialName("custom_currency_exchange_rate") val customCurrencyExchangeRate: Double = 1.0,
    @SerialName("turnstile_check") val turnstileCheck: Boolean = false,
    @SerialName("password_login_enabled") val passwordLoginEnabled: Boolean = true,
)
