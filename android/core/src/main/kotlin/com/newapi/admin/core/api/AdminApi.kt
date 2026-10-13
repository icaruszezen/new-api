package com.newapi.admin.core.api

import com.newapi.admin.core.auth.AuthManager
import com.newapi.admin.core.model.AdminUser
import com.newapi.admin.core.model.Channel
import com.newapi.admin.core.model.ChannelDraft
import com.newapi.admin.core.model.ChannelPage
import com.newapi.admin.core.model.ChannelPatch
import com.newapi.admin.core.model.ChannelQuery
import com.newapi.admin.core.model.ChannelTestResult
import com.newapi.admin.core.model.LogEntry
import com.newapi.admin.core.model.LogQuery
import com.newapi.admin.core.model.LogStat
import com.newapi.admin.core.model.PageData
import com.newapi.admin.core.model.QuotaMode
import com.newapi.admin.core.model.SystemStatus
import com.newapi.admin.core.model.UserAction
import com.newapi.admin.core.model.UserUpdate
import okhttp3.OkHttpClient

/**
 * Admin endpoints of the signed-in server. Every call throws [com.newapi.admin.core.net.ApiError] on failure.
 * Pages are 1-based; `pageSize` must be 1..100.
 *
 * @param client authenticated client (base client + AuthInterceptor).
 */
class AdminApi(
    private val auth: AuthManager,
    private val client: OkHttpClient,
) {
    // ---- System ----

    /** `GET /api/status`. */
    suspend fun status(): SystemStatus = TODO()

    // ---- Usage logs ----

    /** `GET /api/log/` (with the trailing slash). */
    suspend fun logs(query: LogQuery, page: Int, pageSize: Int): PageData<LogEntry> = TODO()

    /** `GET /api/log/stat`; `type` is ignored by the server (always consume logs). */
    suspend fun logStat(query: LogQuery): LogStat = TODO()

    // ---- Users ----

    /**
     * `GET /api/user/` when no filter is set, otherwise `GET /api/user/search`.
     * @param status 1 enabled, 2 disabled, -1 soft-deleted only, null any.
     */
    suspend fun users(
        page: Int,
        pageSize: Int,
        keyword: String = "",
        group: String = "",
        role: Int? = null,
        status: Int? = null,
    ): PageData<AdminUser> = TODO()

    /** `GET /api/user/{id}` (includes `admin_permissions`). */
    suspend fun user(id: Int): AdminUser = TODO()

    /** `POST /api/user/`. [role] must be lower than the caller's (1, or 10 when the caller is root). */
    suspend fun createUser(username: String, password: String, displayName: String, role: Int): Unit = TODO()

    /** `PUT /api/user/`; a password or group change logs the target out everywhere. */
    suspend fun updateUser(update: UserUpdate): Unit = TODO()

    /** `POST /api/user/manage {action:"add_quota"}`; [value] is in raw quota units (positive for add/subtract). */
    suspend fun adjustQuota(id: Int, mode: QuotaMode, value: Long): Unit = TODO()

    /** `POST /api/user/manage` with enable/disable/promote/demote. */
    suspend fun manageUser(id: Int, action: UserAction): Unit = TODO()

    /** `DELETE /api/user/{id}`: permanent; needs a strictly higher role than the target. */
    suspend fun deleteUser(id: Int): Unit = TODO()

    /** `DELETE /api/user/{id}/2fa`; returns the server message. */
    suspend fun resetUserTwoFactor(id: Int): String = TODO()

    /** `DELETE /api/user/{id}/reset_passkey`; returns the server message. */
    suspend fun resetUserPasskey(id: Int): String = TODO()

    /** `GET /api/group/`, sorted. */
    suspend fun groups(): List<String> = TODO()

    // ---- Channels ----

    /** `GET /api/channel/` or, when keyword/model is set, `GET /api/channel/search`. */
    suspend fun channels(query: ChannelQuery, page: Int, pageSize: Int): ChannelPage = TODO()

    /** `GET /api/channel/{id}`. */
    suspend fun channel(id: Int): Channel = TODO()

    /** `POST /api/channel/` (mode single or batch). The server returns no id. */
    suspend fun createChannel(draft: ChannelDraft): Unit = TODO()

    /** `PUT /api/channel/` with `id` plus the non-null fields of [patch]; returns the reloaded channel when the server sends it. */
    suspend fun updateChannel(patch: ChannelPatch): Channel? = TODO()

    /** `POST /api/channel/{id}/status {status: 1|2}`; returns whether anything changed. */
    suspend fun setChannelStatus(id: Int, enabled: Boolean): Boolean = TODO()

    /** `DELETE /api/channel/{id}`. */
    suspend fun deleteChannel(id: Int): Unit = TODO()

    /** `GET /api/channel/test/{id}?model=`; slow (long timeout). A failed test is returned, not thrown. */
    suspend fun testChannel(id: Int, model: String? = null): ChannelTestResult = TODO()

    /** `GET /api/channel/update_balance/{id}`; returns the balance in USD. Slow (long timeout). */
    suspend fun updateChannelBalance(id: Int): Double = TODO()

    /** `GET /api/channel/fetch_models/{id}`: model ids offered by the saved channel's upstream. Slow. */
    suspend fun fetchUpstreamModels(id: Int): List<String> = TODO()
}
