package com.newapi.admin.core.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** The `{success, message, data}` envelope every dashboard endpoint returns. */
@Serializable
data class Envelope<T>(
    val success: Boolean = false,
    val message: String = "",
    val data: T? = null,
    val code: String? = null,
)

/** `common.PageInfo`: `p` is 1-based, `page_size` is capped at 100 by the server. */
@Serializable
data class PageData<T>(
    val page: Int = 0,
    @SerialName("page_size") val pageSize: Int = 0,
    val total: Long = 0,
    val items: List<T>? = null,
)

object Roles {
    const val GUEST = 0
    const val COMMON = 1
    const val ADMIN = 10
    const val ROOT = 100
}

object UserStatus {
    const val ENABLED = 1
    const val DISABLED = 2
}

object ChannelStatus {
    const val UNKNOWN = 0
    const val ENABLED = 1
    const val MANUALLY_DISABLED = 2
    const val AUTO_DISABLED = 3
}

object LogType {
    /** As a filter value 0 means "all types". */
    const val ALL = 0
    const val TOPUP = 1
    const val CONSUME = 2
    const val MANAGE = 3
    const val SYSTEM = 4
    const val ERROR = 5
    const val REFUND = 6
    const val LOGIN = 7
}
