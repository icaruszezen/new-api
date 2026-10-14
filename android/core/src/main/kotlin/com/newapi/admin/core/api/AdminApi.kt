package com.newapi.admin.core.api

import com.newapi.admin.core.auth.AuthManager
import com.newapi.admin.core.model.AdminUser
import com.newapi.admin.core.model.Channel
import com.newapi.admin.core.model.ChannelDraft
import com.newapi.admin.core.model.ChannelPage
import com.newapi.admin.core.model.ChannelPatch
import com.newapi.admin.core.model.ChannelQuery
import com.newapi.admin.core.model.ChannelStatus
import com.newapi.admin.core.model.ChannelStatusFilter
import com.newapi.admin.core.model.ChannelTestResult
import com.newapi.admin.core.model.LogEntry
import com.newapi.admin.core.model.LogQuery
import com.newapi.admin.core.model.LogStat
import com.newapi.admin.core.model.PageData
import com.newapi.admin.core.model.QuotaMode
import com.newapi.admin.core.model.Redemption
import com.newapi.admin.core.model.RedemptionStatus
import com.newapi.admin.core.model.RedemptionStatusFilter
import com.newapi.admin.core.model.SystemStatus
import com.newapi.admin.core.model.UserAction
import com.newapi.admin.core.model.UserUpdate
import com.newapi.admin.core.net.ApiError
import com.newapi.admin.core.net.HttpResult
import com.newapi.admin.core.net.await
import com.newapi.admin.core.net.bool
import com.newapi.admin.core.net.decodeData
import com.newapi.admin.core.net.double
import com.newapi.admin.core.net.long
import com.newapi.admin.core.net.string
import com.newapi.admin.core.net.toApiError
import com.newapi.admin.core.net.toRequestBody
import java.io.IOException
import java.time.Instant
import java.time.ZoneId
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonObjectBuilder
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonObject
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.OkHttpClient
import okhttp3.Request

/**
 * Admin endpoints of the signed-in server. Every call throws [com.newapi.admin.core.net.ApiError] on failure
 * (and IllegalArgumentException for an id <= 0, which `/api/user/manage` would apply to the lowest id, i.e. root).
 * Pages are 1-based; `pageSize` must be 1..100 (values outside are clamped).
 *
 * @param client authenticated client (base client + AuthInterceptor).
 * @param clock device clock in epoch millis, used for the default log time range.
 */
class AdminApi(
    private val auth: AuthManager,
    private val client: OkHttpClient,
    private val clock: () -> Long = System::currentTimeMillis,
) {
    /** Channel test, balance and model fetch wait on the upstream provider. */
    private val slowClient: OkHttpClient by lazy {
        client.newBuilder()
            .readTimeout(SLOW_READ_TIMEOUT_SECONDS, TimeUnit.SECONDS)
            .callTimeout(SLOW_CALL_TIMEOUT_SECONDS, TimeUnit.SECONDS)
            .build()
    }

    // ---- System ----

    /** `GET /api/status`. */
    suspend fun status(): SystemStatus = auth.fetchStatus(baseUrl())

    // ---- Usage logs ----

    /**
     * `GET /api/log/` (with the trailing slash). A missing start defaults to local midnight today and a
     * missing end to now + 1 hour (the web's default range), so the server's `COUNT(*)` stays bounded.
     */
    suspend fun logs(query: LogQuery, page: Int, pageSize: Int): PageData<LogEntry> {
        val params = pageParams(page, pageSize) + logFilterParams(query, includeRequestId = true)
        val envelope = get("/api/log/", params)
        return envelope.decodeData(PageData.serializer(LogEntry.serializer()), "log page") { PageData() }
    }

    /** `GET /api/log/stat`; `type` is ignored by the server (always consume logs). */
    suspend fun logStat(query: LogQuery): LogStat {
        val envelope = get("/api/log/stat", logFilterParams(query, includeRequestId = false))
        return envelope.decodeData(LogStat.serializer(), "log stat") { LogStat() }
    }

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
    ): PageData<AdminUser> {
        val filtered = keyword.isNotBlank() || group.isNotBlank() || role != null || status != null
        val envelope = if (!filtered) {
            get("/api/user/", pageParams(page, pageSize))
        } else {
            // Like the web: keyword and group are always sent (possibly empty), role/status only when chosen.
            val params = listOf(
                "keyword" to keyword.trim(),
                "group" to group.trim(),
                "role" to role?.toString(),
                "status" to status?.toString(),
            ) + pageParams(page, pageSize)
            get("/api/user/search", params)
        }
        return envelope.decodeData(PageData.serializer(AdminUser.serializer()), "user page") { PageData() }
    }

    /** `GET /api/user/{id}` (includes `admin_permissions`). */
    suspend fun user(id: Int): AdminUser = get("/api/user/${requireId(id)}").decodeData(AdminUser.serializer(), "user")

    /** `POST /api/user/`. [role] must be lower than the caller's (1, or 10 when the caller is root). */
    suspend fun createUser(username: String, password: String, displayName: String, role: Int) {
        val body = buildJsonObject {
            put("username", username.trim())
            put("password", password)
            put("display_name", displayName.trim().ifEmpty { username.trim() })
            put("role", role)
        }
        send("POST", "/api/user/", body = body)
    }

    /**
     * `PUT /api/user/`; a password or group change logs the target out everywhere.
     *
     * Always sends `id`, `username`, `display_name` (the username when blank, like the web), `group`
     * and `remark`, because the server overwrites all four; `password` only when set.
     */
    suspend fun updateUser(update: UserUpdate) {
        val password = update.password?.takeIf { it.isNotEmpty() }
        // The server silently treats this exact value as "keep the password".
        if (password == UNCHANGED_PASSWORD_SENTINEL) throw ApiError.Business("This password cannot be used.")
        val body = buildJsonObject {
            put("id", requireId(update.id))
            put("username", update.username.trim())
            put("display_name", update.displayName.trim().ifEmpty { update.username.trim() })
            put("group", update.group.trim())
            put("remark", update.remark)
            if (password != null) put("password", password)
        }
        send("PUT", "/api/user/", body = body)
    }

    /** `POST /api/user/manage {action:"add_quota"}`; [value] is in raw quota units (positive for add/subtract). */
    suspend fun adjustQuota(id: Int, mode: QuotaMode, value: Long) {
        val body = buildJsonObject {
            put("id", requireId(id))
            put("action", "add_quota")
            put("mode", mode.wire)
            put("value", value)
        }
        send("POST", "/api/user/manage", body = body)
    }

    /** `POST /api/user/manage` with enable/disable/promote/demote. */
    suspend fun manageUser(id: Int, action: UserAction) {
        val body = buildJsonObject {
            put("id", requireId(id))
            put("action", action.wire)
        }
        send("POST", "/api/user/manage", body = body)
    }

    /** `DELETE /api/user/{id}`: permanent; needs a strictly higher role than the target. */
    suspend fun deleteUser(id: Int) {
        send("DELETE", "/api/user/${requireId(id)}")
    }

    /** `DELETE /api/user/{id}/2fa`; returns the server message. */
    suspend fun resetUserTwoFactor(id: Int): String =
        send("DELETE", "/api/user/${requireId(id)}/2fa").string("message").orEmpty()

    /** `DELETE /api/user/{id}/reset_passkey`; returns the server message. */
    suspend fun resetUserPasskey(id: Int): String =
        send("DELETE", "/api/user/${requireId(id)}/reset_passkey").string("message").orEmpty()

    /** `GET /api/group/`, sorted. */
    suspend fun groups(): List<String> =
        get("/api/group/").decodeData(ListSerializer(String.serializer()), "groups") { emptyList() }.distinct().sorted()

    // ---- Channels ----

    /** `GET /api/channel/` or, when keyword/model is set, `GET /api/channel/search`. */
    suspend fun channels(query: ChannelQuery, page: Int, pageSize: Int): ChannelPage {
        val search = query.keyword.isNotBlank() || query.model.isNotBlank()
        val filters = listOf(
            "status" to when (query.status) {
                ChannelStatusFilter.ALL -> null
                ChannelStatusFilter.ENABLED -> "enabled"
                ChannelStatusFilter.DISABLED -> "disabled"
            },
            "type" to query.type?.toString(),
            "group" to query.group.trim().takeIf { it.isNotEmpty() },
        )
        val (p, size) = clampPage(page, pageSize)
        val envelope = if (search) {
            val params = listOf("keyword" to query.keyword.trim(), "model" to query.model.trim()) + filters + pageParams(p, size)
            get("/api/channel/search", params)
        } else {
            get("/api/channel/", filters + pageParams(p, size))
        }
        val result = envelope.decodeData(ChannelPage.serializer(), "channel page") { ChannelPage() }
        // Search responses carry no page fields.
        return if (result.page == 0) result.copy(page = p, pageSize = size) else result
    }

    /** `GET /api/channel/{id}`. */
    suspend fun channel(id: Int): Channel = get("/api/channel/${requireId(id)}").decodeData(Channel.serializer(), "channel")

    /**
     * `POST /api/channel/` (mode single or batch). The server returns no id.
     *
     * Batch keys are split on line breaks, trimmed and emptied lines dropped (the server does not trim).
     * Model and group lists are trimmed and de-duplicated; a blank group becomes `default`.
     */
    suspend fun createChannel(draft: ChannelDraft) {
        val key = if (draft.batch) {
            draft.key.replace("\r\n", "\n").replace('\r', '\n').split('\n').map { it.trim() }.filter { it.isNotEmpty() }.joinToString("\n")
        } else {
            draft.key.trim()
        }
        val body = buildJsonObject {
            put("mode", if (draft.batch) "batch" else "single")
            putJsonObject("channel") {
                put("type", draft.type)
                put("name", draft.name.trim())
                put("key", key)
                putIfNotBlank("base_url", draft.baseUrl)
                put("models", normalizeCsv(draft.models))
                put("group", normalizeCsv(draft.group).ifEmpty { "default" })
                draft.priority?.let { put("priority", it) }
                draft.weight?.let { put("weight", it) }
                putIfNotBlank("tag", draft.tag)
                putIfNotBlank("remark", draft.remark)
                putIfNotBlank("test_model", draft.testModel)
                putIfNotBlank("model_mapping", draft.modelMapping)
                put("auto_ban", if (draft.autoBan) 1 else 0)
            }
        }
        send("POST", "/api/channel/", body = body)
    }

    /** `PUT /api/channel/` with `id` plus the non-null fields of [patch]; returns the reloaded channel when the server sends it. */
    suspend fun updateChannel(patch: ChannelPatch): Channel? {
        val body = channelPatchBody(patch)
        val envelope = send("PUT", "/api/channel/", body = body)
        return envelope.decodeData(Channel.serializer(), "channel") { null }
    }

    /** `POST /api/channel/{id}/status {status: 1|2}`; returns whether anything changed. */
    suspend fun setChannelStatus(id: Int, enabled: Boolean): Boolean {
        val body = buildJsonObject { put("status", if (enabled) ChannelStatus.ENABLED else ChannelStatus.MANUALLY_DISABLED) }
        return send("POST", "/api/channel/${requireId(id)}/status", body = body).bool("data") ?: false
    }

    /** `DELETE /api/channel/{id}`. */
    suspend fun deleteChannel(id: Int) {
        send("DELETE", "/api/channel/${requireId(id)}")
    }

    /** `GET /api/channel/test/{id}?model=`; slow (long timeout). A failed test is returned, not thrown. */
    suspend fun testChannel(id: Int, model: String? = null): ChannelTestResult {
        val params = listOf("model" to model?.trim()?.takeIf { it.isNotEmpty() })
        val result = execute(request("GET", "/api/channel/test/${requireId(id)}", params), slow = true)
        if (!result.isSuccessful) throw result.toApiError()
        val obj = result.json ?: throw ApiError.InvalidResponse("channel test response is not JSON")
        val success = obj.bool("success") ?: throw ApiError.InvalidResponse("channel test response has no success flag")
        return ChannelTestResult(
            success = success,
            message = obj.string("message").orEmpty(),
            timeSeconds = if (success) obj.double("time") ?: 0.0 else 0.0,
            errorCode = obj.string("error_code")?.takeIf { it.isNotEmpty() },
        )
    }

    /**
     * `GET /api/channel/update_balance/{id}`; returns the balance in USD. Slow (long timeout).
     * Returns NaN when the server answers with `raw_response` instead of a number (type 58).
     */
    suspend fun updateChannelBalance(id: Int): Double {
        val envelope = execute(request("GET", "/api/channel/update_balance/${requireId(id)}"), slow = true).envelope()
        return envelope.double("balance") ?: Double.NaN
    }

    /** `GET /api/channel/fetch_models/{id}`: model ids offered by the saved channel's upstream. Slow. */
    suspend fun fetchUpstreamModels(id: Int): List<String> {
        val envelope = execute(request("GET", "/api/channel/fetch_models/${requireId(id)}"), slow = true).envelope()
        return envelope.decodeData(ListSerializer(String.serializer()), "model list") { emptyList() }
    }

    // ---- Redemption codes ----

    /** `GET /api/redemption/`, or `GET /api/redemption/search` when [keyword] or [status] is set. Newest first. */
    suspend fun redemptions(
        page: Int,
        pageSize: Int,
        keyword: String = "",
        status: RedemptionStatusFilter = RedemptionStatusFilter.ALL,
    ): PageData<Redemption> {
        val envelope = if (keyword.isBlank() && status.wire == null) {
            get("/api/redemption/", pageParams(page, pageSize))
        } else {
            // Like the web: keyword is always sent (possibly empty), status only when chosen. The keyword
            // matches the id exactly or the start of the name, never the code itself.
            get("/api/redemption/search", listOf("keyword" to keyword.trim(), "status" to status.wire) + pageParams(page, pageSize))
        }
        return envelope.decodeData(PageData.serializer(Redemption.serializer()), "redemption page") { PageData() }
    }

    /** `GET /api/redemption/{id}`. */
    suspend fun redemption(id: Int): Redemption =
        get("/api/redemption/${requireId(id)}").decodeData(Redemption.serializer(), "redemption")

    /**
     * `POST /api/redemption/`: creates [count] codes (1..100) and returns their keys; the server returns no ids.
     * [name] must be 1..20 characters and [expiredTime] (unix seconds) 0 for never or in the future. The server
     * does not check [quota] and stores 0 as 100, so callers keep it positive.
     */
    suspend fun createRedemptions(name: String, quota: Long, count: Int, expiredTime: Long): List<String> {
        val body = buildJsonObject {
            put("name", name.trim())
            put("quota", quota)
            put("count", count)
            put("expired_time", expiredTime)
        }
        return send("POST", "/api/redemption/", body = body).decodeData(ListSerializer(String.serializer()), "redemption keys") { emptyList() }
    }

    /**
     * `PUT /api/redemption/`: sets name, quota and expiry together (the server writes all three, so an omitted
     * one would become 0). Returns the saved code when the server sends it.
     */
    suspend fun updateRedemption(id: Int, name: String, quota: Long, expiredTime: Long): Redemption? {
        val body = buildJsonObject {
            put("id", requireId(id))
            put("name", name.trim())
            put("quota", quota)
            put("expired_time", expiredTime)
        }
        return send("PUT", "/api/redemption/", body = body).decodeData(Redemption.serializer(), "redemption") { null }
    }

    /** `PUT /api/redemption/?status_only=true {id, status: 1|2}`; returns the saved code when the server sends it. */
    suspend fun setRedemptionEnabled(id: Int, enabled: Boolean): Redemption? {
        val body = buildJsonObject {
            put("id", requireId(id))
            put("status", if (enabled) RedemptionStatus.ENABLED else RedemptionStatus.DISABLED)
        }
        return send("PUT", "/api/redemption/", listOf("status_only" to "true"), body).decodeData(Redemption.serializer(), "redemption") { null }
    }

    /** `DELETE /api/redemption/{id}`. */
    suspend fun deleteRedemption(id: Int) {
        send("DELETE", "/api/redemption/${requireId(id)}")
    }

    /** `DELETE /api/redemption/invalid`: deletes every used, disabled and expired code; returns how many. */
    suspend fun deleteInvalidRedemptions(): Long = send("DELETE", "/api/redemption/invalid").long("data") ?: 0

    // ---- Internals ----

    private fun baseUrl(): String = auth.account?.baseUrl ?: throw ApiError.Unauthorized(null, "signed out")

    private suspend fun get(path: String, params: List<Pair<String, String?>> = emptyList()): JsonObject =
        execute(request("GET", path, params)).envelope()

    private suspend fun send(method: String, path: String, params: List<Pair<String, String?>> = emptyList(), body: JsonObject? = null): JsonObject =
        execute(request(method, path, params, body)).envelope()

    private fun request(method: String, path: String, params: List<Pair<String, String?>> = emptyList(), body: JsonObject? = null): Request {
        val url = url(path, params)
        return Request.Builder().url(url).method(method, body?.toRequestBody()).build()
    }

    private fun url(path: String, params: List<Pair<String, String?>>): HttpUrl {
        val base = (baseUrl() + path).toHttpUrlOrNull() ?: throw ApiError.InvalidResponse("invalid server URL")
        return base.newBuilder().apply {
            for ((name, value) in params) if (value != null) addQueryParameter(name, value)
        }.build()
    }

    private suspend fun execute(request: Request, slow: Boolean = false): HttpResult = withContext(Dispatchers.IO) {
        try {
            (if (slow) slowClient else client).await(request, clock)
        } catch (e: IOException) {
            throw e.toApiError()
        }
    }

    private fun pageParams(page: Int, pageSize: Int): List<Pair<String, String?>> {
        val (p, size) = clampPage(page, pageSize)
        return listOf("p" to p.toString(), "page_size" to size.toString())
    }

    private fun clampPage(page: Int, pageSize: Int): Pair<Int, Int> = page.coerceAtLeast(1) to pageSize.coerceIn(1, MAX_PAGE_SIZE)

    private fun logFilterParams(query: LogQuery, includeRequestId: Boolean): List<Pair<String, String?>> {
        val nowSeconds = clock() / 1000
        val end = query.endTimestamp ?: (nowSeconds + DEFAULT_LOG_RANGE_AHEAD_SECONDS)
        val start = query.startTimestamp ?: startOfLocalDay(minOf(nowSeconds, end))
        return listOfNotNull(
            "type" to query.type.toString(),
            "start_timestamp" to start.toString(),
            "end_timestamp" to end.toString(),
            "username" to query.username.trim().takeIf { it.isNotEmpty() },
            "token_name" to query.tokenName.trim().takeIf { it.isNotEmpty() },
            "model_name" to query.modelName.trim().takeIf { it.isNotEmpty() },
            "channel" to query.channel?.toString(),
            "group" to query.group.trim().takeIf { it.isNotEmpty() },
            if (includeRequestId) "request_id" to query.requestId.trim().takeIf { it.isNotEmpty() } else null,
        )
    }

    private fun startOfLocalDay(epochSeconds: Long): Long {
        val zone = ZoneId.systemDefault()
        return Instant.ofEpochSecond(epochSeconds).atZone(zone).toLocalDate().atStartOfDay(zone).toEpochSecond()
    }

    private fun requireId(id: Int): Int {
        // `/api/user/manage` with id 0 targets the lowest id (normally root); never send it.
        if (id <= 0) throw IllegalArgumentException("invalid id $id")
        return id
    }

    internal companion object {
        const val MAX_PAGE_SIZE = 100
        const val DEFAULT_LOG_RANGE_AHEAD_SECONDS = 3600L
        const val SLOW_READ_TIMEOUT_SECONDS = 180L
        const val SLOW_CALL_TIMEOUT_SECONDS = 200L
        const val UNCHANGED_PASSWORD_SENTINEL = "\$I_LOVE_U"

        /** Trims every comma-separated entry, drops empty ones and duplicates, keeps the order. */
        fun normalizeCsv(value: String): String = value.split(',').map { it.trim() }.filter { it.isNotEmpty() }.distinct().joinToString(",")

        /** `PUT /api/channel/` body: `id` plus the non-null fields; never `status`; a blank key is left out (keeps the key). */
        fun channelPatchBody(patch: ChannelPatch): JsonObject = buildJsonObject {
            put("id", patch.id)
            patch.name?.let { put("name", it.trim()) }
            patch.models?.let { put("models", normalizeCsv(it)) }
            patch.group?.let { put("group", normalizeCsv(it)) }
            patch.priority?.let { put("priority", it) }
            patch.weight?.let { put("weight", it) }
            patch.tag?.let { put("tag", it.trim()) }
            patch.remark?.let { put("remark", it) }
            patch.testModel?.let { put("test_model", it.trim()) }
            patch.modelMapping?.let { put("model_mapping", it) }
            patch.autoBan?.let { put("auto_ban", if (it) 1 else 0) }
            patch.type?.let { put("type", it) }
            patch.key?.takeIf { it.isNotBlank() }?.let { put("key", it.trim()) }
            patch.baseUrl?.let { put("base_url", it.trim()) }
        }

        private fun JsonObjectBuilder.putIfNotBlank(key: String, value: String) {
            val trimmed = value.trim()
            if (trimmed.isNotEmpty()) put(key, trimmed)
        }
    }
}
