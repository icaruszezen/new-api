package com.newapi.admin.core.testing

import com.newapi.admin.core.ServerUrl
import com.newapi.admin.core.auth.Credentials
import com.newapi.admin.core.auth.InMemorySessionStore
import com.newapi.admin.core.auth.StoredAccount
import com.newapi.admin.core.model.Roles
import com.newapi.admin.core.model.SelfUser
import java.time.Instant
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter
import java.util.concurrent.CopyOnWriteArrayList
import java.util.concurrent.atomic.AtomicLong
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonObject
import okhttp3.mockwebserver.Dispatcher
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.RecordedRequest

/**
 * A MockWebServer whose responses are chosen per request by [handler]; every request is recorded
 * (with its body read once) so tests can assert on paths, headers and JSON bodies.
 */
class FakeServer : AutoCloseable {
    val server = MockWebServer()
    val requests = CopyOnWriteArrayList<Recorded>()

    @Volatile
    var handler: (Recorded) -> MockResponse = { MockResponse().setResponseCode(404) }

    class Recorded(val raw: RecordedRequest, val body: String) {
        val method: String get() = raw.method.orEmpty()
        val path: String get() = raw.requestUrl!!.encodedPath
        val target: String get() = raw.path.orEmpty()
        fun header(name: String): String? = raw.getHeader(name)
        fun query(name: String): String? = raw.requestUrl!!.queryParameter(name)
        fun json(): JsonObject = Json.parseToJsonElement(body).jsonObject
    }

    init {
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                val recorded = Recorded(request, request.body.readUtf8())
                requests += recorded
                return handler(recorded)
            }
        }
        server.start()
    }

    val baseUrl: String get() = ServerUrl.normalize(server.url("/").toString())

    fun requestsTo(path: String): List<Recorded> = requests.filter { it.path == path }

    override fun close() {
        server.shutdown()
    }
}

/** A device clock tests can move. */
class TestClock(start: Long) : () -> Long {
    private val now = AtomicLong(start)
    override fun invoke(): Long = now.get()
    fun advance(millis: Long) {
        now.addAndGet(millis)
    }
}

/** 2026-10-13T10:00:00Z, the server time used by the fixtures. */
const val SERVER_NOW_SECONDS = 1_791_885_600L

fun httpDate(epochSeconds: Long): String =
    DateTimeFormatter.RFC_1123_DATE_TIME.format(Instant.ofEpochSecond(epochSeconds).atOffset(ZoneOffset.UTC))

fun jsonResponse(body: String, code: Int = 200): MockResponse =
    MockResponse().setResponseCode(code).setHeader("Content-Type", "application/json").setBody(body)

fun statusResponse(turnstile: Boolean = false, passwordLogin: Boolean = true): MockResponse = jsonResponse(
    """{"success":true,"message":"","data":{"version":"v1.0.0","system_name":"New API","quota_per_unit":500000,
       "quota_display_type":"USD","turnstile_check":$turnstile,"password_login_enabled":$passwordLogin}}""",
)

fun userJson(id: Int = 1, username: String = "root", role: Int = Roles.ROOT): String =
    """{"id":$id,"username":"$username","display_name":"$username","role":$role,"status":1,"email":"","group":"default",
       "quota":1000,"used_quota":0,"request_count":0,"setting":"","sidebar_modules":"",
       "permissions":{"sidebar_settings":false,"sidebar_modules":{},"admin_permissions":{"channel":{"read":true,"operate":true,"write":true,"sensitive_write":${role >= Roles.ROOT},"secret_view":false}}}}"""

fun selfResponse(id: Int = 1, username: String = "root", role: Int = Roles.ROOT): MockResponse =
    jsonResponse("""{"success":true,"message":"","data":${userJson(id, username, role)}}""")

fun refreshCookieHeader(token: String, secure: Boolean = false): String =
    "new_api_refresh=$token; Path=/api/user/auth; Expires=Thu, 12 Nov 2026 10:00:00 GMT; Max-Age=2592000; HttpOnly" +
        (if (secure) "; Secure" else "") + "; SameSite=Strict"

/** The bundle login, 2FA and refresh return, with the rotated cookie and a `Date` header. */
fun sessionBundleResponse(
    accessToken: String,
    refreshToken: String,
    sid: String = "sid-1",
    expiresAtSeconds: Long = SERVER_NOW_SECONDS + 900,
    serverNowSeconds: Long = SERVER_NOW_SECONDS,
    role: Int = Roles.ROOT,
    secureCookie: Boolean = false,
): MockResponse = jsonResponse(
    """{"success":true,"message":"","data":{"access_token":"$accessToken","token_type":"Bearer",
       "access_expires_at":$expiresAtSeconds,
       "session":{"sid":"$sid","current":true,"login_method":"password","ip":"127.0.0.1","user_agent":"test",
                  "created_at":$serverNowSeconds,"last_active_at":$serverNowSeconds,"expires_at":${serverNowSeconds + 2_592_000}},
       "user":${userJson(role = role)}}}""",
).setHeader("Date", httpDate(serverNowSeconds)).addHeader("Set-Cookie", refreshCookieHeader(refreshToken, secureCookie))

fun authError(code: Int, errorCode: String): MockResponse =
    jsonResponse("""{"success":false,"code":"$errorCode","message":"error"}""", code)

fun okData(data: String = "null"): MockResponse = jsonResponse("""{"success":true,"message":"","data":$data}""")

fun adminUser(): SelfUser = SelfUser(id = 1, username = "root", displayName = "root", role = Roles.ROOT)

fun sessionAccount(
    baseUrl: String,
    accessToken: String = "access-old",
    expiresAtMillis: Long,
    refreshToken: String = "sid-1.secret-old",
    sid: String = "sid-1",
    origin: String = ServerUrl.origin(baseUrl),
): StoredAccount = StoredAccount(
    baseUrl = baseUrl,
    credentials = Credentials.Session(accessToken, expiresAtMillis, refreshToken, sid, origin),
    user = adminUser(),
)

fun storeWith(account: StoredAccount?): InMemorySessionStore = InMemorySessionStore(account)
