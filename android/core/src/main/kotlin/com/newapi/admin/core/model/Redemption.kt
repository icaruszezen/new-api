package com.newapi.admin.core.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** `model.Redemption`, a code users redeem for quota (`/api/redemption/...`). */
@Serializable
data class Redemption(
    val id: Int = 0,
    /** The admin who created the code. */
    @SerialName("user_id") val creatorId: Int = 0,
    /** The code itself: 32 lowercase hex characters. */
    val key: String = "",
    /** [RedemptionStatus]; an expired code keeps [RedemptionStatus.ENABLED]. */
    val status: Int = 0,
    val name: String = "",
    /** Raw quota units credited when the code is redeemed. */
    val quota: Long = 0,
    /** Unix seconds. */
    @SerialName("created_time") val createdTime: Long = 0,
    /** Unix seconds; 0 until redeemed. */
    @SerialName("redeemed_time") val redeemedTime: Long = 0,
    /** The user who redeemed the code; 0 until redeemed. */
    @SerialName("used_user_id") val usedUserId: Int = 0,
    /** Unix seconds; 0 means never. */
    @SerialName("expired_time") val expiredTime: Long = 0,
) {
    /** Enabled but past its expiry, so it can no longer be redeemed (the server's "expired" filter). */
    fun isExpired(nowSeconds: Long): Boolean = status == RedemptionStatus.ENABLED && expiredTime != 0L && expiredTime < nowSeconds
}

/** The status filter of `GET /api/redemption/search`; [wire] null sends none. */
enum class RedemptionStatusFilter(val wire: String?) {
    ALL(null),

    /** Enabled and not expired. */
    UNUSED("1"),
    DISABLED("2"),
    USED("3"),
    EXPIRED("expired"),
}
