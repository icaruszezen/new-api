package com.newapi.admin.core.model

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.longOrNull

/**
 * Typed accessors over the `other` JSON string of a log row.
 *
 * Numbers are read as doubles because the server writes whole floats as `2`, and some
 * keys (for example `admin_info.use_channel`) hold strings where the web types say numbers.
 */
class LogOther private constructor(val json: JsonObject) {

    val modelRatio: Double? get() = double("model_ratio")
    val completionRatio: Double? get() = double("completion_ratio")
    val groupRatio: Double? get() = double("group_ratio")

    /** -1 means the user has no specific group ratio. */
    val userGroupRatio: Double? get() = double("user_group_ratio")

    /** -1 in per-token mode; a positive value is a per-call price in USD. */
    val modelPrice: Double? get() = double("model_price")
    val cacheTokens: Long get() = double("cache_tokens")?.toLong() ?: 0
    val cacheRatio: Double? get() = double("cache_ratio")

    /** Cache writes: 5m + 1h when either is set, otherwise `cache_creation_tokens` (web rule). */
    val cacheWriteTokens: Long
        get() {
            val split = (double("cache_creation_tokens_5m") ?: 0.0) + (double("cache_creation_tokens_1h") ?: 0.0)
            return if (split > 0) split.toLong() else double("cache_creation_tokens")?.toLong() ?: 0
        }

    /** Time to first response in milliseconds; null when missing or not positive (the server writes -1000 when nothing was sent). */
    val firstResponseMillis: Double? get() = double("frt")?.takeIf { it > 0 }
    val reasoningEffort: String? get() = string("reasoning_effort")
    val isModelMapped: Boolean get() = bool("is_model_mapped") == true
    val upstreamModelName: String? get() = string("upstream_model_name")
    val requestPath: String? get() = string("request_path")
    val billingSource: String? get() = string("billing_source")
    val billingCorrected: Boolean get() = bool("billing_corrected") == true
    val rejectReason: String? get() = string("reject_reason")

    /** Error logs (type 5). */
    val errorType: String? get() = string("error_type")
    val errorCode: String? get() = string("error_code")
    val statusCode: Long? get() = double("status_code")?.toLong()

    val streamStatus: StreamStatus?
        get() {
            val obj = json["stream_status"] as? JsonObject ?: return null
            return StreamStatus(
                status = obj.string("status").orEmpty(),
                endReason = obj.string("end_reason").orEmpty(),
                errorCount = obj.double("error_count")?.toInt() ?: 0,
            )
        }

    /** Channel ids tried in order (`admin_info.use_channel`). */
    val retryChannels: List<String>
        get() {
            val adminInfo = json["admin_info"] as? JsonObject ?: return emptyList()
            val list = adminInfo["use_channel"] as? JsonArray ?: return emptyList()
            return list.mapNotNull { (it as? JsonPrimitive)?.contentOrNull }
        }

    /** Audit operation of manage (type 3) and login (type 7) logs. */
    val auditOp: AuditOp?
        get() {
            val op = json["op"] as? JsonObject ?: return null
            val action = op.string("action") ?: return null
            val params = (op["params"] as? JsonObject)?.mapValues { (_, value) -> value.displayText() } ?: emptyMap()
            return AuditOp(action, params)
        }

    /** Pretty-printed JSON for a raw view. */
    fun prettyJson(): String = prettyPrinter.encodeToString(JsonObject.serializer(), json)

    private fun double(key: String): Double? = json.double(key)
    private fun string(key: String): String? = json.string(key)
    private fun bool(key: String): Boolean? = (json[key] as? JsonPrimitive)?.booleanOrNull

    companion object {
        private val parser = Json { ignoreUnknownKeys = true; isLenient = true }
        private val prettyPrinter = Json { prettyPrint = true }

        /** Returns null for "", "null", invalid JSON or a non-object value. */
        fun parse(raw: String): LogOther? {
            if (raw.isBlank()) return null
            val element = runCatching { parser.parseToJsonElement(raw) }.getOrNull()
            return (element as? JsonObject)?.let(::LogOther)
        }

        private fun JsonObject.double(key: String): Double? {
            val primitive = this[key] as? JsonPrimitive ?: return null
            return primitive.doubleOrNull ?: primitive.longOrNull?.toDouble()
        }

        private fun JsonObject.string(key: String): String? {
            val primitive = this[key] as? JsonPrimitive ?: return null
            if (primitive is JsonNull) return null
            return primitive.contentOrNull?.takeIf { it.isNotEmpty() }
        }

        private fun JsonElement.displayText(): String = when (this) {
            is JsonPrimitive -> contentOrNull.orEmpty()
            else -> toString()
        }
    }
}

fun LogEntry.parsedOther(): LogOther? = LogOther.parse(other)
