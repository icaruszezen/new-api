package com.newapi.admin.core.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** One row of `GET /api/log/` (admin view, all users). */
@Serializable
data class LogEntry(
    val id: Long = 0,
    @SerialName("user_id") val userId: Int = 0,
    /** Unix seconds. */
    @SerialName("created_at") val createdAt: Long = 0,
    val type: Int = LogType.ALL,
    val content: String = "",
    val username: String = "",
    @SerialName("token_name") val tokenName: String = "",
    @SerialName("model_name") val modelName: String = "",
    /** Raw quota units. */
    val quota: Long = 0,
    @SerialName("prompt_tokens") val promptTokens: Long = 0,
    @SerialName("completion_tokens") val completionTokens: Long = 0,
    /** Whole seconds. */
    @SerialName("use_time") val useTime: Int = 0,
    @SerialName("is_stream") val isStream: Boolean = false,
    /** Channel id (the JSON key is `channel`). */
    val channel: Int = 0,
    @SerialName("channel_name") val channelName: String = "",
    @SerialName("token_id") val tokenId: Int = 0,
    val group: String = "",
    val ip: String = "",
    @SerialName("request_id") val requestId: String = "",
    @SerialName("upstream_request_id") val upstreamRequestId: String = "",
    /** A JSON object encoded as a string; may be "", "null" or an object. Use [LogOther.parse]. */
    val other: String = "",
)

/** Totals for the current filter: `quota` honours the time range, `rpm`/`tpm` always cover the last 60 s. */
@Serializable
data class LogStat(
    val quota: Long = 0,
    val rpm: Long = 0,
    val tpm: Long = 0,
)

/** Filters shared by `GET /api/log/` and `GET /api/log/stat`. Blank strings and nulls are not sent. */
data class LogQuery(
    val type: Int = LogType.ALL,
    /** Unix seconds, inclusive. */
    val startTimestamp: Long? = null,
    /** Unix seconds, inclusive. */
    val endTimestamp: Long? = null,
    val username: String = "",
    val tokenName: String = "",
    val modelName: String = "",
    val channel: Int? = null,
    val group: String = "",
    val requestId: String = "",
)

/** Parsed view of [LogEntry.other]. Every accessor tolerates missing or mistyped keys. */
data class StreamStatus(val status: String, val endReason: String, val errorCount: Int)

data class AuditOp(val action: String, val params: Map<String, String>)
