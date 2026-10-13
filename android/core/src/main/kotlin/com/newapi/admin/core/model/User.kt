package com.newapi.admin.core.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** Effective admin permission matrix, e.g. `{"channel": {"read": true, "sensitive_write": false}}`. */
typealias PermissionMatrix = Map<String, Map<String, Boolean>>

@Serializable
data class UserPermissions(
    @SerialName("admin_permissions") val adminPermissions: PermissionMatrix? = null,
)

/** `GET /api/user/self`, also returned as `data.user` by login and refresh. */
@Serializable
data class SelfUser(
    val id: Int = 0,
    val username: String = "",
    @SerialName("display_name") val displayName: String = "",
    val role: Int = Roles.COMMON,
    val status: Int = UserStatus.ENABLED,
    val email: String = "",
    val group: String = "",
    val quota: Long = 0,
    @SerialName("used_quota") val usedQuota: Long = 0,
    @SerialName("request_count") val requestCount: Long = 0,
    val permissions: UserPermissions? = null,
) {
    val isAdmin: Boolean get() = role >= Roles.ADMIN
    val isRoot: Boolean get() = role >= Roles.ROOT

    /** Root can do everything; admins follow `permissions.admin_permissions` (same rule as the web). */
    fun can(resource: String, action: String): Boolean =
        isRoot || permissions?.adminPermissions?.get(resource)?.get(action) == true

    /** Server rule `canManageTargetRole`: root manages anyone, others only strictly lower roles. */
    fun canManageRole(targetRole: Int): Boolean = isRoot || role > targetRole
}

/** A user as seen by an admin (`GET /api/user/`, `/api/user/search`, `/api/user/:id`). */
@Serializable
data class AdminUser(
    val id: Int = 0,
    val username: String = "",
    @SerialName("display_name") val displayName: String = "",
    val role: Int = Roles.COMMON,
    val status: Int = UserStatus.ENABLED,
    val email: String = "",
    @SerialName("github_id") val githubId: String = "",
    @SerialName("discord_id") val discordId: String = "",
    @SerialName("oidc_id") val oidcId: String = "",
    @SerialName("wechat_id") val wechatId: String = "",
    @SerialName("telegram_id") val telegramId: String = "",
    @SerialName("linux_do_id") val linuxDoId: String = "",
    /** Remaining balance in raw quota units; can be negative. */
    val quota: Long = 0,
    @SerialName("used_quota") val usedQuota: Long = 0,
    @SerialName("request_count") val requestCount: Long = 0,
    val group: String = "",
    @SerialName("aff_code") val affCode: String = "",
    @SerialName("aff_count") val affCount: Int = 0,
    @SerialName("aff_quota") val affQuota: Long = 0,
    @SerialName("aff_history_quota") val affHistoryQuota: Long = 0,
    @SerialName("inviter_id") val inviterId: Int = 0,
    /** Soft-delete marker. The server uses the capitalised key `DeletedAt`. */
    @SerialName("DeletedAt") val deletedAt: String? = null,
    /** Omitted by the server when empty. */
    val remark: String = "",
    @SerialName("created_at") val createdAt: Long = 0,
    @SerialName("last_login_at") val lastLoginAt: Long = 0,
    /** Only present on `GET /api/user/:id`. */
    @SerialName("admin_permissions") val adminPermissions: PermissionMatrix? = null,
) {
    val isDeleted: Boolean get() = deletedAt != null
}

/** Body of `PUT /api/user/`. The server overwrites all four text fields, so always send the current values. */
data class UserUpdate(
    val id: Int,
    val username: String,
    val displayName: String,
    val group: String,
    val remark: String,
    /** Null keeps the current password; otherwise 8-20 characters. */
    val password: String? = null,
)

enum class QuotaMode(val wire: String) { ADD("add"), SUBTRACT("subtract"), OVERRIDE("override") }

enum class UserAction(val wire: String) { ENABLE("enable"), DISABLE("disable"), PROMOTE("promote"), DEMOTE("demote") }
