package com.newapi.admin.core.net

import java.io.IOException
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.serialization.DeserializationStrategy
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.longOrNull
import okhttp3.Call
import okhttp3.Callback
import okhttp3.Headers
import okhttp3.MediaType
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import okio.Buffer
import okio.BufferedSink

/** Parser for every server response: tolerant of unknown keys, nulls, missing fields and quoted numbers. */
internal val ApiJson: Json = Json {
    ignoreUnknownKeys = true
    coerceInputValues = true
    explicitNulls = false
    isLenient = true
}

private val JsonMediaType = "application/json; charset=utf-8".toMediaType()

internal fun JsonElement.toRequestBody(): RequestBody = toString().toRequestBody(JsonMediaType)

internal val EmptyBody: RequestBody = ByteArray(0).toRequestBody(null)

/**
 * A fully read HTTP response. [receivedAtMillis] is the device clock when the headers arrived and
 * is used with the `Date` header to correct for clock skew.
 */
internal class HttpResult(
    val code: Int,
    val headers: Headers,
    val body: String,
    val receivedAtMillis: Long,
) {
    val isSuccessful: Boolean get() = code in 200..299

    /** The body as a JSON object, or null when it is empty, not JSON, or not an object. */
    val json: JsonObject? by lazy(LazyThreadSafetyMode.NONE) {
        if (body.isBlank()) null else runCatching { ApiJson.parseToJsonElement(body) as? JsonObject }.getOrNull()
    }

    /** The `code` field of an error body (`AUTH_...`), if any. */
    val errorCode: String? get() = json?.string("code")

    /** Server time from the `Date` header in epoch millis. */
    val serverDateMillis: Long?
        get() = headers["Date"]?.let { value ->
            runCatching { ZonedDateTime.parse(value, DateTimeFormatter.RFC_1123_DATE_TIME).toInstant().toEpochMilli() }.getOrNull()
        }

    /** Device-clock minus server-clock in millis; 0 without a usable `Date` header. */
    val clockSkewMillis: Long get() = serverDateMillis?.let { receivedAtMillis - it } ?: 0L

    /** `Retry-After` in seconds (delta-seconds or an HTTP date). */
    val retryAfterSeconds: Long?
        get() {
            val raw = headers["Retry-After"]?.trim() ?: return null
            raw.toLongOrNull()?.let { return it.coerceAtLeast(0) }
            val date = runCatching { ZonedDateTime.parse(raw, DateTimeFormatter.RFC_1123_DATE_TIME).toInstant().toEpochMilli() }.getOrNull()
                ?: return null
            val reference = serverDateMillis ?: receivedAtMillis
            return ((date - reference) / 1000).coerceAtLeast(0)
        }

    /** The [ApiError] for a non-2xx status. */
    fun toApiError(): ApiError {
        val message = json?.string("message") ?: "HTTP $code"
        return when (code) {
            401 -> ApiError.Unauthorized(errorCode, message)
            403 -> ApiError.Forbidden(errorCode, message)
            429 -> ApiError.RateLimited(retryAfterSeconds)
            else -> ApiError.Http(code, errorCode, message)
        }
    }

    /**
     * Checks the HTTP status and the `{success, message, data}` envelope and returns the envelope.
     * Throws [ApiError.Business] for `success:false` and [ApiError.InvalidResponse] for anything that is not an envelope.
     */
    fun envelope(): JsonObject {
        if (!isSuccessful) throw toApiError()
        val obj = json ?: throw ApiError.InvalidResponse("HTTP $code: response is not a JSON object")
        val success = (obj["success"] as? JsonPrimitive)?.booleanOrNull
            ?: throw ApiError.InvalidResponse("HTTP $code: response has no success flag")
        if (!success) throw ApiError.Business(obj.string("message").orEmpty())
        return obj
    }
}

/** Decodes [element] with [ApiJson], turning decoding failures into [ApiError.InvalidResponse]. */
internal fun <T> decode(deserializer: DeserializationStrategy<T>, element: JsonElement, what: String): T = try {
    ApiJson.decodeFromJsonElement(deserializer, element)
} catch (e: SerializationException) {
    throw ApiError.InvalidResponse("unexpected $what: ${e.message}", e)
} catch (e: IllegalArgumentException) {
    throw ApiError.InvalidResponse("unexpected $what: ${e.message}", e)
}

/** Decodes the envelope's `data`; a missing or null `data` uses [whenMissing] or fails. */
internal fun <T> JsonObject.decodeData(deserializer: DeserializationStrategy<T>, what: String, whenMissing: (() -> T)? = null): T {
    val data = this["data"]
    if (data == null || data is JsonNull) {
        if (whenMissing != null) return whenMissing()
        throw ApiError.InvalidResponse("$what: response has no data")
    }
    return decode(deserializer, data, what)
}

internal fun JsonObject.string(key: String): String? {
    val primitive = this[key] as? JsonPrimitive ?: return null
    if (primitive is JsonNull) return null
    return primitive.contentOrNull
}

internal fun JsonObject.long(key: String): Long? {
    val primitive = this[key] as? JsonPrimitive ?: return null
    if (primitive is JsonNull) return null
    return primitive.longOrNull ?: primitive.doubleOrNull?.toLong()
}

internal fun JsonObject.double(key: String): Double? {
    val primitive = this[key] as? JsonPrimitive ?: return null
    if (primitive is JsonNull) return null
    return primitive.doubleOrNull
}

internal fun JsonObject.bool(key: String): Boolean? {
    val primitive = this[key] as? JsonPrimitive ?: return null
    if (primitive is JsonNull) return null
    return primitive.booleanOrNull
}

internal fun JsonObject.obj(key: String): JsonObject? = this[key] as? JsonObject

private fun Response.toResult(receivedAt: Long): HttpResult = use {
    HttpResult(code, headers, body?.string().orEmpty(), receivedAt)
}

/**
 * OkHttp silently sends a request again when its connection fails, even after the request was written.
 * Only reads may be repeated: the server may already have applied a write (add_quota, create, delete,
 * login). Marking a write's body one-shot makes OkHttp give up once sending started, while it still
 * tries the next address or a fresh connection when connecting fails. Writes that are safe to re-send,
 * like the session refresh, retry explicitly.
 */
private fun OkHttpClient.newCallWithoutResendingWrites(request: Request): Call {
    if (request.method == "GET") return newCall(request)
    val body = request.body ?: ByteArray(0).toRequestBody()
    val oneShot = object : RequestBody() {
        override fun contentType(): MediaType? = body.contentType()
        override fun contentLength(): Long = body.contentLength()
        override fun writeTo(sink: BufferedSink) = body.writeTo(sink)
        override fun isOneShot(): Boolean = true
    }
    return newCall(request.newBuilder().method(request.method, oneShot).build())
}

/** Blocking execution, for code that already runs on an OkHttp thread (refresh, logout from the interceptor). */
internal fun OkHttpClient.executeBlocking(request: Request, clock: () -> Long): HttpResult {
    val response = newCallWithoutResendingWrites(request).execute()
    return response.toResult(clock())
}

/** Suspending execution with cancellation; the body is read on the OkHttp dispatcher thread. */
internal suspend fun OkHttpClient.await(request: Request, clock: () -> Long): HttpResult =
    suspendCancellableCoroutine { continuation ->
        val call = newCallWithoutResendingWrites(request)
        continuation.invokeOnCancellation { call.cancel() }
        call.enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                continuation.resumeWithException(e)
            }

            override fun onResponse(call: Call, response: Response) {
                val result = try {
                    response.toResult(clock())
                } catch (e: IOException) {
                    continuation.resumeWithException(e)
                    return
                }
                continuation.resume(result)
            }
        })
    }

/**
 * Like [await], for an image download of at most [maxBytes]. Any response that is not a 2xx image is read as
 * text and fails like an envelope does (a 404, or a 200 `success:false` the server sends for a database error).
 */
internal suspend fun OkHttpClient.awaitImage(request: Request, maxBytes: Long, clock: () -> Long): ByteArray =
    suspendCancellableCoroutine { continuation ->
        val call = newCall(request)
        continuation.invokeOnCancellation { call.cancel() }
        call.enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                continuation.resumeWithException(e)
            }

            override fun onResponse(call: Call, response: Response) {
                val bytes = try {
                    response.use { it.toImage(maxBytes, clock) }
                } catch (e: ApiError) {
                    continuation.resumeWithException(e)
                    return
                } catch (e: IOException) {
                    continuation.resumeWithException(e)
                    return
                }
                continuation.resume(bytes)
            }
        })
    }

private fun Response.toImage(maxBytes: Long, clock: () -> Long): ByteArray {
    val body = this.body
    if (body == null || !isSuccessful || body.contentType()?.type != "image") {
        val result = HttpResult(code, headers, body?.string().orEmpty(), clock())
        if (!result.isSuccessful) throw result.toApiError()
        result.envelope()
        throw ApiError.InvalidResponse("HTTP $code: response is not an image")
    }
    val tooLarge = ApiError.InvalidResponse("image larger than $maxBytes bytes")
    if (body.contentLength() > maxBytes) throw tooLarge
    val buffer = Buffer()
    body.source().use { source ->
        while (source.read(buffer, 64 * 1024L) != -1L) {
            if (buffer.size > maxBytes) throw tooLarge
        }
    }
    return buffer.readByteArray()
}

/**
 * Thrown on OkHttp threads when a session refresh failed for a transient reason (network, 429, 5xx).
 * The account stays signed in. It is an IOException because interceptors may only throw those.
 */
class RefreshFailedException(
    message: String,
    /** HTTP status of the refresh response, null for a network failure. */
    val status: Int? = null,
    val retryAfterSeconds: Long? = null,
    cause: Throwable? = null,
) : IOException(message, cause)

/** Maps a transport failure to the [ApiError] the app shows. */
internal fun IOException.toApiError(): ApiError =
    if (this is RefreshFailedException && status == 429) ApiError.RateLimited(retryAfterSeconds) else ApiError.Network(this)
