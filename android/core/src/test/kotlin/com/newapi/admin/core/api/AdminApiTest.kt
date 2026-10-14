package com.newapi.admin.core.api

import com.newapi.admin.core.auth.Credentials
import com.newapi.admin.core.auth.StoredAccount
import com.newapi.admin.core.model.ChannelDraft
import com.newapi.admin.core.model.ChannelPatch
import com.newapi.admin.core.model.ChannelQuery
import com.newapi.admin.core.model.ChannelStatusFilter
import com.newapi.admin.core.model.ChannelTestResult
import com.newapi.admin.core.model.LogQuery
import com.newapi.admin.core.model.QuotaMode
import com.newapi.admin.core.model.RedemptionStatusFilter
import com.newapi.admin.core.model.UserUpdate
import com.newapi.admin.core.net.ApiError
import com.newapi.admin.core.net.NewApiClient
import com.newapi.admin.core.testing.FakeServer
import com.newapi.admin.core.testing.SERVER_NOW_SECONDS
import com.newapi.admin.core.testing.TestClock
import com.newapi.admin.core.testing.adminUser
import com.newapi.admin.core.testing.jsonResponse
import com.newapi.admin.core.testing.okData
import com.newapi.admin.core.testing.storeWith
import java.net.InetAddress
import java.util.TimeZone
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import okhttp3.Dns
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.SocketPolicy
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class AdminApiTest {
    private val server = FakeServer()
    private val clock = TestClock(SERVER_NOW_SECONDS * 1000)
    private lateinit var api: AdminApi
    private val originalZone: TimeZone = TimeZone.getDefault()

    private val emptyPage = """{"page":1,"page_size":20,"total":0,"items":[]}"""
    private val channelJson = """{"id":7,"type":1,"name":"main","status":1,"models":"gpt-4o","group":"default","priority":0,
        "remark":"r","channel_info":{"is_multi_key":false,"multi_key_size":0,"multi_key_status_list":null,"multi_key_mode":""}}"""

    @Before
    fun setUp() {
        TimeZone.setDefault(TimeZone.getTimeZone("UTC"))
        val account = StoredAccount(server.baseUrl, Credentials.AccessToken("pat-1"), adminUser())
        api = NewApiClient(storeWith(account), userAgent = "test", acceptLanguage = { "en" }, clock = clock).admin
    }

    @After
    fun tearDown() {
        TimeZone.setDefault(originalZone)
        server.close()
    }

    @Test
    fun successFalseBecomesBusinessErrorWithTheServerMessage() {
        server.handler = { jsonResponse("""{"success":false,"message":"No permission to update users of same or higher permission level"}""") }

        val error = assertThrows(ApiError.Business::class.java) { runBlocking { api.manageUser(5, com.newapi.admin.core.model.UserAction.DISABLE) } }

        assertEquals("No permission to update users of same or higher permission level", error.message)
    }

    @Test
    fun endpointsUseTheExactPathsTheServerRoutes() = runBlocking {
        server.handler = { request ->
            when {
                request.path in setOf("/api/user/", "/api/user/search", "/api/log/") && request.method == "GET" -> okData(emptyPage)
                request.path == "/api/user/5" -> okData("""{"id":5,"username":"alice","role":1,"status":1}""")
                request.path == "/api/log/stat" -> okData("""{"quota":10,"rpm":1,"tpm":2}""")
                request.path == "/api/group/" -> okData("""["default"]""")
                request.path in setOf("/api/channel/", "/api/channel/search") && request.method == "GET" -> okData("""{"items":[],"total":0}""")
                request.path == "/api/channel/3" -> okData(channelJson)
                else -> okData()
            }
        }

        api.users(page = 1, pageSize = 20)
        api.users(page = 1, pageSize = 20, keyword = "al")
        api.user(5)
        api.logs(LogQuery(), page = 1, pageSize = 20)
        api.logStat(LogQuery())
        api.groups()
        api.channels(ChannelQuery(), page = 1, pageSize = 20)
        api.channels(ChannelQuery(keyword = "main", status = ChannelStatusFilter.DISABLED), page = 2, pageSize = 20)
        api.channel(3)
        api.createUser("bob", "password1", "", role = 1)
        api.updateUser(UserUpdate(id = 5, username = "alice", displayName = "Alice", group = "default", remark = ""))
        api.adjustQuota(5, QuotaMode.ADD, 1000)
        api.deleteUser(5)
        api.createChannel(ChannelDraft(type = 1, name = "c", key = "sk"))
        api.updateChannel(ChannelPatch(id = 3, priority = 1))
        api.setChannelStatus(3, enabled = false)
        api.deleteChannel(3)

        assertEquals(
            listOf(
                "GET /api/user/", "GET /api/user/search", "GET /api/user/5", "GET /api/log/", "GET /api/log/stat",
                "GET /api/group/", "GET /api/channel/", "GET /api/channel/search", "GET /api/channel/3",
                "POST /api/user/", "PUT /api/user/", "POST /api/user/manage", "DELETE /api/user/5",
                "POST /api/channel/", "PUT /api/channel/", "POST /api/channel/3/status", "DELETE /api/channel/3",
            ),
            server.requests.map { "${it.method} ${it.path}" },
        )
        assertTrue(server.requests.all { it.header("Authorization") == "Bearer pat-1" })

        val search = server.requestsTo("/api/user/search").single()
        assertEquals("al", search.query("keyword"))
        assertEquals("", search.query("group"))
        assertNull(search.query("role"))
        val channelSearch = server.requestsTo("/api/channel/search").single()
        assertEquals("disabled", channelSearch.query("status"))
        assertEquals("2", channelSearch.query("p"))
        assertEquals(buildJsonObject { put("status", 2) }, server.requestsTo("/api/channel/3/status").single().json())
        assertEquals(
            buildJsonObject { put("id", 5); put("action", "add_quota"); put("mode", "add"); put("value", 1000) },
            server.requestsTo("/api/user/manage").single().json(),
        )
    }

    @Test
    fun logsAlwaysSendATimeRangeAndAClampedPage() = runBlocking {
        server.handler = { okData(emptyPage) }

        api.logs(LogQuery(username = " alice ", channel = 4), page = 0, pageSize = 500)

        val request = server.requests.single()
        // Clock is 2026-10-13T10:00:00Z in UTC: local midnight to now + 1 h, like the web default.
        assertEquals((SERVER_NOW_SECONDS - 10 * 3600).toString(), request.query("start_timestamp"))
        assertEquals((SERVER_NOW_SECONDS + 3600).toString(), request.query("end_timestamp"))
        assertEquals("1", request.query("p"))
        assertEquals("100", request.query("page_size"))
        assertEquals("0", request.query("type"))
        assertEquals("alice", request.query("username"))
        assertEquals("4", request.query("channel"))
        assertNull(request.query("model_name"))
    }

    @Test
    fun updateUserAlwaysSendsTheFourOverwrittenFieldsAndThePasswordOnlyWhenSet() = runBlocking {
        server.handler = { okData() }

        api.updateUser(UserUpdate(id = 5, username = "alice", displayName = "Alice", group = "vip", remark = ""))
        api.updateUser(UserUpdate(id = 5, username = "alice", displayName = "", group = "vip", remark = "note", password = "new-pass-1"))

        assertEquals(
            listOf(
                buildJsonObject { put("id", 5); put("username", "alice"); put("display_name", "Alice"); put("group", "vip"); put("remark", "") },
                buildJsonObject {
                    put("id", 5); put("username", "alice"); put("display_name", "alice"); put("group", "vip"); put("remark", "note")
                    put("password", "new-pass-1")
                },
            ),
            server.requests.map { it.json() },
        )
    }

    @Test
    fun updateChannelSendsOnlyTheNonNullFieldsAndNeverStatus() = runBlocking {
        server.handler = { okData(channelJson) }

        val reloaded = api.updateChannel(ChannelPatch(id = 7, priority = 0, tag = "", autoBan = false, key = "  ", models = "gpt-4o, gpt-4o,,o3 "))

        assertEquals(
            buildJsonObject { put("id", 7); put("models", "gpt-4o,o3"); put("priority", 0); put("tag", ""); put("auto_ban", 0) },
            server.requests.single().json(),
        )
        assertEquals("main", reloaded?.name)
    }

    @Test
    fun createChannelTrimsBatchKeysAndDefaultsTheGroup() = runBlocking {
        server.handler = { okData() }

        api.createChannel(ChannelDraft(type = 1, name = "c", key = "k1 \r\n\n k2\n", group = " ", batch = true, autoBan = false))

        val body = server.requests.single().json()
        assertEquals("\"batch\"", body["mode"].toString())
        assertEquals(
            buildJsonObject {
                put("type", 1); put("name", "c"); put("key", "k1\nk2"); put("models", ""); put("group", "default"); put("auto_ban", 0)
            },
            body["channel"],
        )
    }

    @Test
    fun redemptionEndpointsSendWhatTheServerReads() = runBlocking {
        val codeJson = """{"id":9,"user_id":1,"key":"0123456789abcdef0123456789abcdef","status":1,"name":"promo","quota":500000,
            "created_time":1,"redeemed_time":0,"count":0,"used_user_id":0,"DeletedAt":null,"expired_time":${SERVER_NOW_SECONDS - 1}}"""
        server.handler = { request ->
            when {
                request.method == "GET" && request.path in setOf("/api/redemption/", "/api/redemption/search") ->
                    okData("""{"page":1,"page_size":20,"total":1,"items":[$codeJson]}""")
                request.method == "GET" -> okData(codeJson)
                request.method == "POST" -> okData("""["k1","k2"]""")
                request.method == "PUT" -> okData(codeJson)
                request.path == "/api/redemption/invalid" -> okData("3")
                else -> okData()
            }
        }

        val page = api.redemptions(page = 1, pageSize = 20)
        api.redemptions(page = 2, pageSize = 20, status = RedemptionStatusFilter.EXPIRED)
        api.redemptions(page = 1, pageSize = 20, keyword = " promo ")
        api.redemption(9)
        val keys = api.createRedemptions(" promo ", quota = 500000, count = 2, expiredTime = 0)
        api.updateRedemption(9, "promo2", quota = 250000, expiredTime = SERVER_NOW_SECONDS + 60)
        api.setRedemptionEnabled(9, enabled = false)
        api.deleteRedemption(9)
        val deleted = api.deleteInvalidRedemptions()

        assertEquals(
            listOf(
                "GET /api/redemption/", "GET /api/redemption/search", "GET /api/redemption/search", "GET /api/redemption/9",
                "POST /api/redemption/", "PUT /api/redemption/", "PUT /api/redemption/", "DELETE /api/redemption/9",
                "DELETE /api/redemption/invalid",
            ),
            server.requests.map { "${it.method} ${it.path}" },
        )
        val code = page.items.orEmpty().single()
        assertEquals(500000L, code.quota)
        assertTrue(code.isExpired(SERVER_NOW_SECONDS))
        assertEquals(listOf("k1", "k2"), keys)
        assertEquals(3L, deleted)

        val (statusSearch, keywordSearch) = server.requestsTo("/api/redemption/search")
        assertEquals("", statusSearch.query("keyword"))
        assertEquals("expired", statusSearch.query("status"))
        assertEquals("2", statusSearch.query("p"))
        assertEquals("promo", keywordSearch.query("keyword"))
        assertNull(keywordSearch.query("status"))
        assertEquals(
            buildJsonObject { put("name", "promo"); put("quota", 500000); put("count", 2); put("expired_time", 0) },
            server.requests[4].json(),
        )
        val (update, toggle) = server.requestsTo("/api/redemption/").filter { it.method == "PUT" }
        assertNull(update.query("status_only"))
        assertEquals(
            buildJsonObject { put("id", 9); put("name", "promo2"); put("quota", 250000); put("expired_time", SERVER_NOW_SECONDS + 60) },
            update.json(),
        )
        assertEquals("true", toggle.query("status_only"))
        assertEquals(buildJsonObject { put("id", 9); put("status", 2) }, toggle.json())
    }

    @Test
    fun httpFailuresMapToTypedErrors() {
        val cases = listOf(
            jsonResponse("""{"success":false,"message":"Unauthorized, insufficient privileges"}""", 403) to ApiError.Forbidden::class.java,
            MockResponse().setResponseCode(429).setHeader("Retry-After", "30") to ApiError.RateLimited::class.java,
            jsonResponse("""{"success":false,"code":"AUTH_INTERNAL_ERROR","message":"Internal Server Error"}""", 500) to ApiError.Http::class.java,
            MockResponse().setResponseCode(200).setBody("<html>proxy</html>") to ApiError.InvalidResponse::class.java,
        )
        for ((response, expected) in cases) {
            server.handler = { response }
            val error = assertThrows(expected) { runBlocking { api.channel(1) } }
            when (error) {
                is ApiError.Forbidden -> assertNull(error.code)
                is ApiError.RateLimited -> assertEquals(30L, error.retryAfterSeconds)
                is ApiError.Http -> {
                    assertEquals(500, error.status)
                    assertEquals("AUTH_INTERNAL_ERROR", error.code)
                }
                else -> Unit
            }
        }
    }

    @Test
    fun writeWhoseConnectionDropsAfterItWasSentIsNotResent() {
        val adjustments = AtomicInteger()
        server.handler = { request ->
            when (request.path) {
                // The server may have applied it before the connection dropped; a resend would add the quota twice.
                "/api/user/manage" ->
                    if (adjustments.incrementAndGet() == 1) MockResponse().setSocketPolicy(SocketPolicy.DISCONNECT_AFTER_REQUEST) else okData()
                else -> okData("""["default"]""")
            }
        }
        // Leaves a pooled keep-alive connection, the case OkHttp silently retries on.
        runBlocking { api.groups() }

        assertThrows(ApiError.Network::class.java) { runBlocking { api.adjustQuota(5, QuotaMode.ADD, 500_000) } }

        assertEquals(1, server.requestsTo("/api/user/manage").size)
    }

    @Test
    fun writeStillFallsBackToTheNextAddressWhenConnectingFails() = runBlocking {
        server.handler = { okData() }
        // A dual-stack host whose first address refuses connections (CI runners resolve localhost to ::1 first).
        val reachable = InetAddress.getByName(server.server.hostName)
        val refused = InetAddress.getByAddress(server.server.hostName, byteArrayOf(127, 0, 0, 2))
        val dualStack = object : Dns {
            override fun lookup(hostname: String): List<InetAddress> = listOf(refused, reachable)
        }
        val account = StoredAccount(server.baseUrl, Credentials.AccessToken("pat-1"), adminUser())
        val dualStackApi = NewApiClient(
            storeWith(account),
            userAgent = "test",
            acceptLanguage = { "en" },
            okHttpBuilder = OkHttpClient.Builder().dns(dualStack).connectTimeout(2, TimeUnit.SECONDS),
            clock = clock,
        ).admin

        dualStackApi.adjustQuota(5, QuotaMode.ADD, 500_000)

        assertEquals(1, server.requestsTo("/api/user/manage").size)
    }

    @Test
    fun failedChannelTestIsReturnedWithItsTopLevelFields() = runBlocking {
        server.handler = { jsonResponse("""{"success":false,"message":"connection refused","time":1.5,"error_code":"do_request_failed"}""") }

        val result = api.testChannel(9, model = "gpt-4o-mini")

        assertEquals(ChannelTestResult(success = false, message = "connection refused", timeSeconds = 0.0, errorCode = "do_request_failed"), result)
        assertEquals("gpt-4o-mini", server.requests.single().query("model"))
        assertEquals("/api/channel/test/9", server.requests.single().path)
    }

    @Test
    fun successfulChannelTestAndBalanceReadTopLevelNumbers() = runBlocking {
        server.handler = { request ->
            if (request.path.startsWith("/api/channel/test/")) jsonResponse("""{"success":true,"message":"","time":0.42}""")
            else jsonResponse("""{"success":true,"message":"","balance":12.5}""")
        }

        assertEquals(ChannelTestResult(true, "", 0.42), api.testChannel(9))
        assertEquals(12.5, api.updateChannelBalance(9), 0.0)
    }

    @Test
    fun decodingToleratesNullsMissingFieldsAndQuotedNumbers() = runBlocking {
        server.handler = { request ->
            when (request.path) {
                "/api/status" -> okData("""{"quota_per_unit":"1000000","usd_exchange_rate":"7.2","quota_display_type":"CNY","system_name":null}""")
                "/api/channel/" -> okData(
                    """{"items":[{"id":1,"type":1,"name":"a","status":3,"base_url":null,"models":null,"priority":null,"tag":null,
                       "other_info":"{\"status_reason\":\"余额不足\",\"status_time\":1760000000}",
                       "channel_info":{"is_multi_key":true,"multi_key_size":3,"multi_key_status_list":{"1":2}}}],
                       "total":1,"page":1,"page_size":20,"type_counts":{"1":1}}""",
                )
                "/api/log/" -> okData("""{"page":1,"page_size":20,"total":0,"items":null}""")
                else -> okData()
            }
        }

        val status = api.status()
        val channel = api.channels(ChannelQuery(), 1, 20).items!!.single()
        val logs = api.logs(LogQuery(), 1, 20)

        assertEquals(1_000_000.0, status.quotaPerUnit, 0.0)
        assertEquals(7.2, status.usdExchangeRate, 0.0)
        assertEquals("", status.systemName)
        assertEquals("", channel.models)
        assertNull(channel.priority)
        assertEquals("余额不足", channel.statusReason)
        assertEquals(1_760_000_000L, channel.statusTime)
        assertEquals(2, channel.channelInfo.enabledKeyCount)
        assertNull(logs.items)
    }
}
