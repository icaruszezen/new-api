package com.newapi.admin.core.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.longOrNull

/** A channel as returned by list/search/get. `key` is never returned and is omitted here on purpose. */
@Serializable
data class Channel(
    val id: Int = 0,
    val type: Int = 0,
    val name: String = "",
    val status: Int = ChannelStatus.UNKNOWN,
    /** Empty or null means the type's default base URL. */
    @SerialName("base_url") val baseUrl: String? = null,
    /** Comma-separated. */
    val models: String = "",
    /** Comma-separated. */
    val group: String = "",
    val priority: Long? = null,
    val weight: Long? = null,
    /** Upstream balance, already in USD (not quota units). */
    val balance: Double = 0.0,
    @SerialName("balance_updated_time") val balanceUpdatedTime: Long = 0,
    /** Milliseconds; 0 means never tested. */
    @SerialName("response_time") val responseTime: Int = 0,
    /** Unix seconds; 0 means never. */
    @SerialName("test_time") val testTime: Long = 0,
    @SerialName("created_time") val createdTime: Long = 0,
    /** Raw quota units. */
    @SerialName("used_quota") val usedQuota: Long = 0,
    val tag: String? = null,
    val remark: String? = null,
    @SerialName("test_model") val testModel: String? = null,
    @SerialName("model_mapping") val modelMapping: String? = null,
    val other: String = "",
    /** JSON string `{"status_reason": "...", "status_time": <unix s>}`. */
    @SerialName("other_info") val otherInfo: String = "",
    @SerialName("auto_ban") val autoBan: Int? = null,
    @SerialName("channel_info") val channelInfo: ChannelInfo = ChannelInfo(),
) {
    val modelList: List<String> get() = splitCsv(models)
    val groupList: List<String> get() = splitCsv(group)

    /** Why the channel was last enabled/disabled, from `other_info`. */
    val statusReason: String? get() = otherInfoJson()?.let { (it["status_reason"] as? JsonPrimitive)?.contentOrNull }?.takeIf { it.isNotBlank() }

    /** Unix seconds of the last status change, from `other_info`. */
    val statusTime: Long? get() = otherInfoJson()?.let { (it["status_time"] as? JsonPrimitive)?.longOrNull }?.takeIf { it > 0 }

    private fun otherInfoJson(): JsonObject? {
        if (otherInfo.isBlank()) return null
        return runCatching { Json.parseToJsonElement(otherInfo) as? JsonObject }.getOrNull()
    }

    companion object {
        fun splitCsv(value: String): List<String> = value.split(',').map { it.trim() }.filter { it.isNotEmpty() }
    }
}

@Serializable
data class ChannelInfo(
    @SerialName("is_multi_key") val isMultiKey: Boolean = false,
    @SerialName("multi_key_size") val multiKeySize: Int = 0,
    /** Key index -> status (1 enabled, 2 manually disabled, 3 auto disabled). Missing index means enabled. */
    @SerialName("multi_key_status_list") val multiKeyStatusList: Map<String, Int>? = null,
    @SerialName("multi_key_mode") val multiKeyMode: String = "",
) {
    val enabledKeyCount: Int
        get() = (0 until multiKeySize).count { index -> (multiKeyStatusList?.get(index.toString()) ?: ChannelStatus.ENABLED) == ChannelStatus.ENABLED }
}

/** `GET /api/channel/` and `/api/channel/search` responses (search omits page fields). */
@Serializable
data class ChannelPage(
    val items: List<Channel>? = null,
    val total: Long = 0,
    val page: Int = 0,
    @SerialName("page_size") val pageSize: Int = 0,
    /** Channel type id (as a string) -> count, computed without the type filter. */
    @SerialName("type_counts") val typeCounts: Map<String, Long>? = null,
)

enum class ChannelStatusFilter { ALL, ENABLED, DISABLED }

data class ChannelQuery(
    /** Matches id, name, exact key or base URL; uses `/api/channel/search` when not blank. */
    val keyword: String = "",
    /** Model substring; uses `/api/channel/search` when not blank. */
    val model: String = "",
    val status: ChannelStatusFilter = ChannelStatusFilter.ALL,
    val type: Int? = null,
    val group: String = "",
)

/** Result of `GET /api/channel/test/:id`; fields are top-level in the response. */
data class ChannelTestResult(
    val success: Boolean,
    val message: String,
    /** Seconds; 0 on failure. */
    val timeSeconds: Double,
    val errorCode: String? = null,
)

/** Fields for `POST /api/channel/` (mode `single`, or `batch` with one key per line). Requires `channel.sensitive_write`. */
data class ChannelDraft(
    val type: Int,
    val name: String,
    val key: String,
    val baseUrl: String = "",
    val models: String = "",
    val group: String = "default",
    val priority: Long? = null,
    val weight: Long? = null,
    val tag: String = "",
    val remark: String = "",
    val testModel: String = "",
    val modelMapping: String = "",
    val autoBan: Boolean = true,
    /** One channel per non-empty key line. */
    val batch: Boolean = false,
)

/**
 * Fields for `PUT /api/channel/`. Only non-null fields are sent, together with `id`.
 *
 * Never send `status` (use [com.newapi.admin.core.api.AdminApi.setChannelStatus]). `type`, a non-empty
 * `key` and `baseUrl` need `channel.sensitive_write`. Blank `name`/`models`/`group` are ignored by the server.
 */
data class ChannelPatch(
    val id: Int,
    val name: String? = null,
    val models: String? = null,
    val group: String? = null,
    val priority: Long? = null,
    val weight: Long? = null,
    val tag: String? = null,
    val remark: String? = null,
    val testModel: String? = null,
    val modelMapping: String? = null,
    val autoBan: Boolean? = null,
    val type: Int? = null,
    /** Blank keeps the current key. */
    val key: String? = null,
    val baseUrl: String? = null,
)
