package com.newapi.admin.core

import com.newapi.admin.core.auth.AuthState
import com.newapi.admin.core.auth.Credentials
import com.newapi.admin.core.auth.InMemorySessionStore
import com.newapi.admin.core.auth.LoginResult
import com.newapi.admin.core.auth.SignOutReason
import com.newapi.admin.core.model.ChannelDraft
import com.newapi.admin.core.model.ChannelPatch
import com.newapi.admin.core.model.ChannelQuery
import com.newapi.admin.core.model.ChannelStatus
import com.newapi.admin.core.model.LogQuery
import com.newapi.admin.core.model.LogType
import com.newapi.admin.core.model.QuotaMode
import com.newapi.admin.core.model.RedemptionStatus
import com.newapi.admin.core.model.RedemptionStatusFilter
import com.newapi.admin.core.model.TicketImage
import com.newapi.admin.core.model.TicketStatus
import com.newapi.admin.core.model.UserAction
import com.newapi.admin.core.model.UserStatus
import com.newapi.admin.core.model.UserUpdate
import com.newapi.admin.core.net.ApiError
import com.newapi.admin.core.net.NewApiClient
import java.awt.image.BufferedImage
import java.io.ByteArrayOutputStream
import java.util.UUID
import java.util.concurrent.atomic.AtomicLong
import javax.imageio.ImageIO
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.junit.AfterClass
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Assume.assumeTrue
import org.junit.BeforeClass
import org.junit.FixMethodOrder
import org.junit.Test
import org.junit.runners.MethodSorters

/**
 * Contract tests against a real new-api server. Skipped unless the `newapi.it.baseUrl` system property
 * is set (the Gradle test task copies `NEWAPI_IT_BASE_URL`, `NEWAPI_IT_USERNAME`, `NEWAPI_IT_PASSWORD`).
 *
 * The account must be root (channel create/delete need `sensitive_write`). The class logs in once,
 * because login shares a 20-per-20-minutes bucket per IP; every test removes what it creates.
 */
@FixMethodOrder(MethodSorters.NAME_ASCENDING)
class LiveServerContractTest {

    companion object {
        private val baseUrlProperty = System.getProperty("newapi.it.baseUrl").orEmpty()
        private val clockOffset = AtomicLong(0)
        private val store = InMemorySessionStore()
        private lateinit var client: NewApiClient
        private val suffix = UUID.randomUUID().toString().take(6)

        @BeforeClass
        @JvmStatic
        fun logIn() {
            assumeTrue("set NEWAPI_IT_BASE_URL to run live contract tests", baseUrlProperty.isNotBlank())
            client = NewApiClient(
                store = store,
                userAgent = "NewApiAdmin-ContractTest/1.0",
                acceptLanguage = { "en" },
                clock = { System.currentTimeMillis() + clockOffset.get() },
            )
            val result = runBlocking {
                client.auth.loginWithPassword(
                    ServerUrl.normalize(baseUrlProperty),
                    System.getProperty("newapi.it.username").orEmpty(),
                    System.getProperty("newapi.it.password").orEmpty(),
                )
            }
            check(result is LoginResult.Success) { "the contract test account must not use 2FA" }
        }

        @AfterClass
        @JvmStatic
        fun logOutIfStillSignedIn() {
            if (baseUrlProperty.isNotBlank() && ::client.isInitialized && client.auth.account != null) {
                runBlocking { client.auth.logout() }
            }
        }
    }

    private val api get() = client.admin

    @Test
    fun a_loginStoredARootSessionWithRefreshCookie() {
        val account = store.load()!!
        val session = account.credentials as Credentials.Session
        assertTrue(account.user.isRoot)
        assertTrue(account.user.can("channel", "sensitive_write"))
        assertTrue(session.refreshToken.startsWith(session.sessionId + "."))
        assertTrue(session.accessExpiresAtMillis > System.currentTimeMillis() + 10 * 60_000)
        assertEquals(ServerUrl.origin(account.baseUrl), session.origin)
    }

    @Test
    fun b_statusDescribesTheDisplayCurrency() = runBlocking<Unit> {
        val status = api.status()
        assertTrue(status.quotaPerUnit > 0)
        assertTrue(status.quotaDisplayType in setOf("USD", "CNY", "TOKENS", "CUSTOM"))
        assertTrue(status.passwordLoginEnabled)
    }

    @Test
    fun c_logsAndStatsForTheDefaultRangeIncludeOurLogin() = runBlocking<Unit> {
        val logins = api.logs(LogQuery(type = LogType.LOGIN, username = store.load()!!.user.username), page = 1, pageSize = 10)
        val stat = api.logStat(LogQuery())
        val all = api.logs(LogQuery(), page = 1, pageSize = 5)

        assertEquals(1, logins.page)
        assertEquals(10, logins.pageSize)
        val newest = logins.items.orEmpty().first()
        assertEquals(LogType.LOGIN, newest.type)
        assertTrue(System.currentTimeMillis() / 1000 - newest.createdAt < 600)
        assertTrue(newest.ip.isNotEmpty())
        assertTrue(stat.quota >= 0)
        assertTrue(all.items.orEmpty().size <= 5)
    }

    @Test
    fun d_groupsAreSortedAndContainDefault() = runBlocking<Unit> {
        val groups = api.groups()
        assertTrue(groups.contains("default"))
        assertEquals(groups.sorted(), groups)
    }

    @Test
    fun e_userLifecycle() = runBlocking<Unit> {
        val username = "it_$suffix"
        api.createUser(username, "ItPass-12345", displayName = "", role = 1)
        var id = 0
        try {
            id = api.users(page = 1, pageSize = 10, keyword = username).items.orEmpty().single { it.username == username }.id
            assertTrue(api.users(page = 1, pageSize = 100).total >= 2)

            val created = api.user(id)
            assertEquals(username, created.displayName)
            assertEquals(UserStatus.ENABLED, created.status)
            assertTrue(created.adminPermissions != null)

            api.updateUser(UserUpdate(id, username, displayName = "IT Temp", group = "default", remark = "contract test"))
            val updated = api.user(id)
            assertEquals("IT Temp", updated.displayName)
            assertEquals("contract test", updated.remark)

            val start = updated.quota
            api.adjustQuota(id, QuotaMode.ADD, 1_000)
            assertEquals(start + 1_000, api.user(id).quota)
            api.adjustQuota(id, QuotaMode.SUBTRACT, 400)
            assertEquals(start + 600, api.user(id).quota)
            api.adjustQuota(id, QuotaMode.OVERRIDE, 12_345)
            assertEquals(12_345L, api.user(id).quota)

            api.manageUser(id, UserAction.DISABLE)
            assertEquals(UserStatus.DISABLED, api.user(id).status)
            api.manageUser(id, UserAction.ENABLE)
            assertEquals(UserStatus.ENABLED, api.user(id).status)

            api.updateUser(UserUpdate(id, username, "IT Temp", "default", "contract test", password = "ItPass-67890"))
            assertEquals("contract test", api.user(id).remark)

            val tooShort = assertThrows(ApiError.Business::class.java) {
                runBlocking { api.updateUser(UserUpdate(id, username, "IT Temp", "default", "x", password = "short")) }
            }
            assertTrue(tooShort.message.orEmpty().isNotBlank())
        } finally {
            if (id > 0) api.deleteUser(id)
        }
        assertThrows(ApiError.Business::class.java) { runBlocking { api.user(id) } }
        assertTrue(api.users(page = 1, pageSize = 10, keyword = username).items.orEmpty().none { it.username == username })
    }

    @Test
    fun f2_redemptionLifecycle() = runBlocking<Unit> {
        val name = "it-code-$suffix"
        val nowSeconds = System.currentTimeMillis() / 1000
        val keys = api.createRedemptions(name, quota = 500_000, count = 2, expiredTime = nowSeconds + 3600)
        var ids = emptyList<Int>()
        try {
            assertEquals(2, keys.size)
            assertTrue(keys.all { it.matches(Regex("[0-9a-f]{32}")) })
            val found = api.redemptions(page = 1, pageSize = 20, keyword = name).items.orEmpty().filter { it.name == name }
            ids = found.map { it.id }
            assertEquals(keys.toSet(), found.map { it.key }.toSet())

            val id = ids.first()
            val created = api.redemption(id)
            assertEquals(RedemptionStatus.ENABLED, created.status)
            assertEquals(500_000L, created.quota)
            assertEquals(nowSeconds + 3600, created.expiredTime)
            assertEquals(store.load()!!.user.id, created.creatorId)
            assertEquals(0, created.usedUserId)

            val updated = api.updateRedemption(id, "$name-x", quota = 250_000, expiredTime = 0)
            assertEquals(250_000L, updated?.quota)
            assertEquals("$name-x", api.redemption(id).name)
            assertEquals(0L, api.redemption(id).expiredTime)

            assertEquals(RedemptionStatus.DISABLED, api.setRedemptionEnabled(id, enabled = false)?.status)
            val disabled = api.redemptions(page = 1, pageSize = 20, keyword = name, status = RedemptionStatusFilter.DISABLED)
            assertEquals(listOf(id), disabled.items.orEmpty().map { it.id })
            api.setRedemptionEnabled(id, enabled = true)
            assertEquals(RedemptionStatus.ENABLED, api.redemption(id).status)
            val unused = api.redemptions(page = 1, pageSize = 20, keyword = name, status = RedemptionStatusFilter.UNUSED)
            assertEquals(ids.toSet(), unused.items.orEmpty().map { it.id }.toSet())

            for (invalid in listOf<suspend () -> Any?>(
                { api.createRedemptions("x".repeat(21), 1, 1, 0) },
                { api.createRedemptions(name, 1, 101, 0) },
                { api.createRedemptions(name, 1, 1, nowSeconds - 3600) },
                { api.updateRedemption(id, name, 1, nowSeconds - 3600) },
            )) {
                val error = assertThrows(ApiError.Business::class.java) { runBlocking { invalid() } }
                assertTrue(error.message.orEmpty().isNotBlank())
            }

            val second = ids.last()
            api.setRedemptionEnabled(second, enabled = false)
            assertTrue(api.deleteInvalidRedemptions() >= 1)
            ids = listOf(id)
            assertThrows(ApiError.Business::class.java) { runBlocking { api.redemption(second) } }
        } finally {
            ids.forEach { api.deleteRedemption(it) }
        }
        assertThrows(ApiError.Business::class.java) { runBlocking { api.redemption(ids.single()) } }
        assertTrue(api.redemptions(page = 1, pageSize = 20, keyword = name).items.orEmpty().isEmpty())
    }

    @Test
    fun f3_ticketLifecycle() = runBlocking<Unit> {
        val username = "it_tk_$suffix"
        val subject = "it-ticket-$suffix"
        val baseUrl = store.load()!!.baseUrl
        val admin = store.load()!!.user.username
        api.createUser(username, "ItPass-12345", displayName = "", role = 1)
        var userId = 0
        try {
            userId = api.users(page = 1, pageSize = 10, keyword = username).items.orEmpty().single { it.username == username }.id
            val ticketId = openTicketAs(baseUrl, username, "ItPass-12345", subject, "Cannot top up", png(4, 3))

            val listed = api.tickets(page = 1, pageSize = 20, keyword = subject).items.orEmpty().single()
            assertEquals(ticketId, listed.id)
            assertEquals(TicketStatus.PENDING, listed.status)
            assertEquals(username, listed.username)
            assertTrue(listed.adminUnread)
            assertEquals(listOf(ticketId), api.tickets(page = 1, pageSize = 20, keyword = "#$ticketId").items.orEmpty().map { it.id })
            assertTrue(api.tickets(page = 1, pageSize = 20, keyword = subject, status = TicketStatus.CLOSED).items.orEmpty().isEmpty())
            assertTrue(api.ticketStats().unread >= 1)

            val detail = api.ticket(ticketId)
            assertEquals(userId, detail.user.id)
            assertEquals(username, detail.user.username)
            val first = detail.messages.single()
            assertFalse(first.isFromAdmin)
            assertEquals("Cannot top up", first.content)
            val attachment = first.attachments.single()
            assertEquals(4 to 3, attachment.width to attachment.height)
            assertFalse(api.tickets(page = 1, pageSize = 20, keyword = subject).items.orEmpty().single().adminUnread)
            val stored = api.ticketAttachment(ticketId, attachment.id)
            assertEquals(attachment.size, stored.size.toLong())
            assertTrue(stored.isPng() || stored.isJpeg())

            val reply = api.replyTicket(ticketId, " Looking into it ", listOf(TicketImage(png(2, 2), "image/png")))
            assertTrue(reply.isFromAdmin)
            assertEquals(admin, reply.authorName)
            assertEquals("Looking into it", reply.content)
            assertEquals(1, reply.attachments.size)
            assertEquals(TicketStatus.PROCESSING, api.ticket(ticketId).ticket.status)

            val imageOnly = api.replyTicket(ticketId, "", listOf(TicketImage(png(3, 3), "image/png")), status = TicketStatus.RESOLVED)
            assertEquals("", imageOnly.content)
            assertEquals(TicketStatus.RESOLVED, api.ticket(ticketId).ticket.status)

            val closed = api.setTicketStatus(ticketId, TicketStatus.CLOSED)
            assertEquals(TicketStatus.CLOSED, closed.status)
            assertTrue(closed.closedTime > 0)
            api.replyTicket(ticketId, "Closing note")
            val final = api.ticket(ticketId)
            assertEquals(TicketStatus.CLOSED, final.ticket.status)
            assertEquals(4, final.messages.size)
            assertEquals(3, final.attachmentCount)

            for (invalid in listOf<suspend () -> Any?>(
                { api.replyTicket(ticketId, "   ") },
                { api.replyTicket(ticketId, "x".repeat(5001)) },
                { api.replyTicket(ticketId, "", listOf(TicketImage("not an image".toByteArray(), "image/png"))) },
                { api.ticket(Int.MAX_VALUE) },
            )) {
                val error = assertThrows(ApiError.Business::class.java) { runBlocking { invalid() } }
                assertTrue(error.message.orEmpty().isNotBlank())
            }
            val missing = assertThrows(ApiError.Http::class.java) { runBlocking { api.ticketAttachment(ticketId, Int.MAX_VALUE) } }
            assertEquals(404, missing.status)
        } finally {
            // Tickets cannot be deleted; deleting the owner leaves this one behind with its user id only.
            if (userId > 0) api.deleteUser(userId)
        }
    }

    /** Logs in as a normal user without touching [store], and opens a ticket with one image. */
    private fun openTicketAs(baseUrl: String, username: String, password: String, subject: String, content: String, image: ByteArray): Int {
        val http = OkHttpClient()
        val login = Request.Builder()
            .url("$baseUrl/api/user/login")
            .post("""{"username":"$username","password":"$password"}""".toRequestBody("application/json".toMediaType()))
            .build()
        val token = http.newCall(login).execute().use { response ->
            Json.parseToJsonElement(response.body!!.string()).jsonObject["data"]!!.jsonObject["access_token"]!!.jsonPrimitive.content
        }
        val form = MultipartBody.Builder().setType(MultipartBody.FORM)
            .addFormDataPart("subject", subject)
            .addFormDataPart("content", content)
            .addFormDataPart("files", "shot.png", image.toRequestBody("image/png".toMediaType()))
            .build()
        val create = Request.Builder().url("$baseUrl/api/ticket/self").header("Authorization", "Bearer $token").post(form).build()
        val created = http.newCall(create).execute().use { Json.parseToJsonElement(it.body!!.string()).jsonObject }
        assertEquals(created.toString(), "true", created["success"]!!.jsonPrimitive.content)
        return created["data"]!!.jsonObject["id"]!!.jsonPrimitive.int
    }

    private fun png(width: Int, height: Int): ByteArray {
        val image = BufferedImage(width, height, BufferedImage.TYPE_INT_RGB)
        return ByteArrayOutputStream().also { ImageIO.write(image, "png", it) }.toByteArray()
    }

    private fun ByteArray.isPng() = size > 4 && this[0] == 0x89.toByte() && this[1] == 'P'.code.toByte()

    private fun ByteArray.isJpeg() = size > 3 && this[0] == 0xFF.toByte() && this[1] == 0xD8.toByte()

    @Test
    fun f_channelLifecycle() = runBlocking<Unit> {
        val name = "it-channel-$suffix"
        api.createChannel(
            ChannelDraft(
                type = 1,
                name = name,
                key = "sk-contract-test-dummy",
                baseUrl = "http://127.0.0.1:9",
                models = "gpt-4o-mini, gpt-4o",
                group = "default",
                priority = 3,
            ),
        )
        var id = 0
        try {
            val found = api.channels(ChannelQuery(keyword = name), page = 1, pageSize = 20)
            id = found.items.orEmpty().single { it.name == name }.id
            assertEquals(1L, found.total)
            assertTrue(api.channels(ChannelQuery(), page = 1, pageSize = 20).items.orEmpty().isNotEmpty())

            val channel = api.channel(id)
            assertEquals(listOf("gpt-4o-mini", "gpt-4o"), channel.modelList)
            assertEquals(3L, channel.priority)
            assertEquals("http://127.0.0.1:9", channel.baseUrl)
            assertEquals(ChannelStatus.ENABLED, channel.status)

            val reloaded = api.updateChannel(ChannelPatch(id = id, priority = 0, remark = "contract test", tag = "it"))
            assertEquals(0L, reloaded?.priority)
            assertEquals("contract test", api.channel(id).remark)

            assertTrue(api.setChannelStatus(id, enabled = false))
            val disabled = api.channel(id)
            assertEquals(ChannelStatus.MANUALLY_DISABLED, disabled.status)
            assertEquals("manual operation", disabled.statusReason)
            assertFalse(api.setChannelStatus(id, enabled = false))
            assertTrue(api.setChannelStatus(id, enabled = true))
            assertEquals(ChannelStatus.ENABLED, api.channel(id).status)

            val test = api.testChannel(id)
            assertFalse(test.success)
            assertTrue(test.message.isNotBlank())
            assertEquals(0.0, test.timeSeconds, 0.0)

            // Upstream is unreachable: both report a business failure rather than a transport error.
            assertThrows(ApiError.Business::class.java) { runBlocking { api.fetchUpstreamModels(id) } }
            assertThrows(ApiError.Business::class.java) { runBlocking { api.updateChannelBalance(id) } }
        } finally {
            if (id > 0) api.deleteChannel(id)
        }
        assertThrows(ApiError.Business::class.java) { runBlocking { api.channel(id) } }
    }

    @Test
    fun g_expiredAccessTokenIsRefreshedWithRotation() = runBlocking<Unit> {
        val before = store.load()!!.credentials as Credentials.Session
        clockOffset.addAndGet(20 * 60_000L) // the stored token now looks expired

        api.groups()

        val after = store.load()!!.credentials as Credentials.Session
        assertEquals(before.sessionId, after.sessionId)
        assertNotEquals(before.accessToken, after.accessToken)
        assertNotEquals(before.refreshToken, after.refreshToken)
        assertTrue(after.accessExpiresAtMillis > System.currentTimeMillis() + clockOffset.get() + 10 * 60_000)
        assertEquals(client.auth.account, store.load())
        api.groups() // the fresh token works without another refresh
    }

    @Test
    fun h_logoutRevokesTheSessionOnTheServer() = runBlocking<Unit> {
        val account = store.load()!!
        val session = account.credentials as Credentials.Session

        client.auth.logout()

        assertNull(store.load())
        assertEquals(AuthState.SignedOut(SignOutReason.USER), client.auth.state.value)
        val probe = Request.Builder()
            .url(account.baseUrl + "/api/user/self")
            .header("Authorization", "Bearer ${session.accessToken}")
            .build()
        val code = OkHttpClient().newCall(probe).execute().use { it.code }
        assertEquals(401, code)
    }
}
