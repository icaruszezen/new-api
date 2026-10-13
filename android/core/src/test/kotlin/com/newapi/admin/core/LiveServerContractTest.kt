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
import com.newapi.admin.core.model.UserAction
import com.newapi.admin.core.model.UserStatus
import com.newapi.admin.core.model.UserUpdate
import com.newapi.admin.core.net.ApiError
import com.newapi.admin.core.net.NewApiClient
import java.util.UUID
import java.util.concurrent.atomic.AtomicLong
import kotlinx.coroutines.runBlocking
import okhttp3.OkHttpClient
import okhttp3.Request
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
