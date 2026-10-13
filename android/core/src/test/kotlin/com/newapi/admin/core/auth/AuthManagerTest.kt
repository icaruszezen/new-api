package com.newapi.admin.core.auth

import com.newapi.admin.core.ServerUrl
import com.newapi.admin.core.model.Roles
import com.newapi.admin.core.net.ApiError
import com.newapi.admin.core.net.AuthInterceptor
import com.newapi.admin.core.net.NewApiClient
import com.newapi.admin.core.net.RefreshFailedException
import com.newapi.admin.core.testing.FakeServer
import com.newapi.admin.core.testing.SERVER_NOW_SECONDS
import com.newapi.admin.core.testing.TestClock
import com.newapi.admin.core.testing.authError
import com.newapi.admin.core.testing.jsonResponse
import com.newapi.admin.core.testing.okData
import com.newapi.admin.core.testing.selfResponse
import com.newapi.admin.core.testing.sessionAccount
import com.newapi.admin.core.testing.sessionBundleResponse
import com.newapi.admin.core.testing.statusResponse
import com.newapi.admin.core.testing.storeWith
import java.io.IOException
import java.util.concurrent.atomic.AtomicInteger
import java.util.concurrent.atomic.AtomicReference
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.SocketPolicy
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Test

class AuthManagerTest {
    private val server = FakeServer()
    private val base get() = server.baseUrl

    /** Device clock equal to the fixtures' server time, so expiries need no skew correction. */
    private val clock = TestClock(SERVER_NOW_SECONDS * 1000)

    @After
    fun tearDown() {
        server.close()
    }

    private fun client(store: SessionStore, testClock: TestClock = clock) =
        NewApiClient(store, userAgent = "NewApiAdmin/test", acceptLanguage = { "zh-CN" }, clock = testClock)

    private fun groupsOk(): MockResponse = okData("""["vip","default"]""")

    // ---- Login ----

    @Test
    fun passwordLoginStoresCookieSessionIdSkewCorrectedExpiryAndProfile() = runBlocking {
        val skewedClock = TestClock(SERVER_NOW_SECONDS * 1000 + 120_000) // device clock 2 minutes ahead
        server.handler = { request ->
            when (request.path) {
                "/api/status" -> statusResponse()
                "/api/user/login" -> sessionBundleResponse("access-1", "sid-1.secret-1", secureCookie = true)
                "/api/user/self" -> selfResponse()
                else -> MockResponse().setResponseCode(404)
            }
        }
        val store = InMemorySessionStore()
        val client = client(store, skewedClock)

        val result = client.auth.loginWithPassword(base, "root", "secret-pass")

        val stored = store.load()!!
        assertTrue(result is LoginResult.Success)
        assertEquals(base, stored.baseUrl)
        assertEquals(
            Credentials.Session(
                accessToken = "access-1",
                accessExpiresAtMillis = (SERVER_NOW_SECONDS + 900) * 1000 + 120_000,
                refreshToken = "sid-1.secret-1",
                sessionId = "sid-1",
                origin = ServerUrl.origin(base),
            ),
            stored.credentials,
        )
        assertTrue(stored.user.can("channel", "sensitive_write"))
        assertEquals(AuthState.SignedIn(stored), client.auth.state.value)

        val login = server.requestsTo("/api/user/login").single()
        assertEquals(buildJsonObject { put("username", "root"); put("password", "secret-pass") }, login.json())
        assertEquals("NewApiAdmin/test", login.header("User-Agent"))
        assertEquals("zh-CN", login.header("Accept-Language"))
        assertEquals("Bearer access-1", server.requestsTo("/api/user/self").single().header("Authorization"))
    }

    @Test
    fun turnstileOrDisabledPasswordLoginFailsBeforeSpendingALoginAttempt() {
        server.handler = { statusResponse(turnstile = true) }
        assertThrows(ApiError.TurnstileRequired::class.java) {
            runBlocking { client(InMemorySessionStore()).auth.loginWithPassword(base, "root", "pw") }
        }
        server.handler = { statusResponse(passwordLogin = false) }
        assertThrows(ApiError.PasswordLoginDisabled::class.java) {
            runBlocking { client(InMemorySessionStore()).auth.loginWithPassword(base, "root", "pw") }
        }
        assertTrue(server.requestsTo("/api/user/login").isEmpty())
    }

    @Test
    fun twoFactorLoginPostsCodeWithFlowTokenThenStoresSession() = runBlocking {
        server.handler = { request ->
            when (request.path) {
                "/api/status" -> statusResponse()
                "/api/user/login" -> jsonResponse(
                    """{"success":true,"message":"Please enter two-factor authentication code",
                       "data":{"require_2fa":true,"flow_token":"flow-abc","expires_at":${SERVER_NOW_SECONDS + 300}}}""",
                )
                "/api/user/login/2fa" -> sessionBundleResponse("access-2fa", "sid-9.secret-9", sid = "sid-9")
                "/api/user/self" -> selfResponse()
                else -> MockResponse().setResponseCode(404)
            }
        }
        val store = InMemorySessionStore()
        val client = client(store)

        val first = client.auth.loginWithPassword(base, "root", "secret-pass")

        assertEquals(LoginResult.TwoFactorRequired(base, "flow-abc", SERVER_NOW_SECONDS + 300), first)
        assertNull(store.load())
        assertEquals(AuthState.SignedOut(), client.auth.state.value)

        client.auth.verifyTwoFactor(base, "flow-abc", " 123456 ")

        assertEquals(
            buildJsonObject { put("code", "123456"); put("flow_token", "flow-abc") },
            server.requestsTo("/api/user/login/2fa").single().json(),
        )
        val session = store.load()!!.credentials as Credentials.Session
        assertEquals("access-2fa", session.accessToken)
        assertEquals("sid-9.secret-9", session.refreshToken)
        assertEquals("sid-9", session.sessionId)
    }

    @Test
    fun nonAdminLoginRevokesTheNewSessionAndThrowsNotAdmin() {
        server.handler = { request ->
            when (request.path) {
                "/api/status" -> statusResponse()
                "/api/user/login" -> sessionBundleResponse("access-user", "sid-2.secret-2", sid = "sid-2", role = Roles.COMMON)
                "/api/user/auth/logout" -> okData("""{"revoked_sid":"sid-2","cookie_cleared":true}""")
                else -> MockResponse().setResponseCode(404)
            }
        }
        val store = InMemorySessionStore()
        val client = client(store)

        assertThrows(ApiError.NotAdmin::class.java) {
            runBlocking { client.auth.loginWithPassword(base, "alice", "secret-pass") }
        }

        val logout = server.requestsTo("/api/user/auth/logout").single()
        assertEquals("Bearer access-user", logout.header("Authorization"))
        assertEquals("new_api_refresh=sid-2.secret-2", logout.header("Cookie"))
        assertEquals("sid-2", logout.header("X-Auth-Session"))
        assertEquals(ServerUrl.origin(base), logout.header("Origin"))
        assertNull(store.load())
        assertEquals(AuthState.SignedOut(), client.auth.state.value)
    }

    @Test
    fun accessTokenLoginStoresThePatAndNeverRefreshes() = runBlocking {
        server.handler = { request ->
            val authorized = request.header("Authorization") == "Bearer pat-123"
            when {
                request.path == "/api/user/self" && authorized -> selfResponse()
                request.path == "/api/group/" && authorized -> groupsOk()
                else -> authError(401, "AUTH_UNAUTHORIZED")
            }
        }
        val store = InMemorySessionStore()
        val client = client(store)

        client.auth.loginWithAccessToken(base, "  pat-123 ")
        clock.advance(24 * 3_600_000L)
        client.admin.groups()

        assertEquals(Credentials.AccessToken("pat-123"), store.load()!!.credentials)
        assertTrue(server.requestsTo("/api/user/auth/refresh").isEmpty())
    }

    @Test
    fun rejectedAccessTokenSignsOutWithoutRefreshing() {
        server.handler = { authError(401, "AUTH_UNAUTHORIZED") }
        val store = storeWith(sessionAccount(base, expiresAtMillis = 0).copy(credentials = Credentials.AccessToken("pat-old")))
        val client = client(store)

        assertThrows(ApiError.Unauthorized::class.java) { runBlocking { client.admin.groups() } }

        assertEquals(AuthState.SignedOut(SignOutReason.SESSION_EXPIRED), client.auth.state.value)
        assertNull(store.load())
        assertTrue(server.requestsTo("/api/user/auth/refresh").isEmpty())
    }

    // ---- Refresh ----

    @Test
    fun tokenExpiringWithinAMinuteIsRefreshedOnceAndTheRotationIsPersistedBeforeUse() = runBlocking {
        val store = storeWith(sessionAccount(base, expiresAtMillis = clock() + 30_000))
        val storedWhenUsed = AtomicReference<Credentials?>()
        server.handler = { request ->
            when (request.path) {
                "/api/user/auth/refresh" -> sessionBundleResponse("access-new", "sid-1.secret-new")
                "/api/group/" -> {
                    storedWhenUsed.set(store.load()?.credentials)
                    if (request.header("Authorization") == "Bearer access-new") groupsOk() else authError(401, "AUTH_TOKEN_EXPIRED")
                }
                else -> MockResponse().setResponseCode(404)
            }
        }
        val client = client(store)

        assertEquals(listOf("default", "vip"), client.admin.groups())

        val refresh = server.requestsTo("/api/user/auth/refresh").single()
        assertEquals("new_api_refresh=sid-1.secret-old", refresh.header("Cookie"))
        assertEquals("sid-1", refresh.header("X-Auth-Session"))
        assertEquals(ServerUrl.origin(base), refresh.header("Origin"))
        assertEquals(null, refresh.header("Authorization"))
        val expected = Credentials.Session(
            accessToken = "access-new",
            accessExpiresAtMillis = (SERVER_NOW_SECONDS + 900) * 1000,
            refreshToken = "sid-1.secret-new",
            sessionId = "sid-1",
            origin = ServerUrl.origin(base),
        )
        assertEquals(expected, store.load()!!.credentials)
        assertEquals(expected, storedWhenUsed.get())
        assertEquals(listOf("Bearer access-new"), server.requestsTo("/api/group/").map { it.header("Authorization") })
    }

    @Test
    fun concurrentRequestsShareASingleRefresh() = runBlocking {
        val store = storeWith(sessionAccount(base, expiresAtMillis = clock() - 1_000))
        server.handler = { request ->
            when (request.path) {
                // Slow refresh so the other requests arrive while it is in flight.
                "/api/user/auth/refresh" -> sessionBundleResponse("access-new", "sid-1.secret-new").setHeadersDelay(300, java.util.concurrent.TimeUnit.MILLISECONDS)
                "/api/group/" -> if (request.header("Authorization") == "Bearer access-new") groupsOk() else authError(401, "AUTH_TOKEN_EXPIRED")
                else -> MockResponse().setResponseCode(404)
            }
        }
        val client = client(store)

        val results = (1..8).map { async(Dispatchers.IO) { client.admin.groups() } }.awaitAll()

        assertEquals(8, results.size)
        assertEquals(1, server.requestsTo("/api/user/auth/refresh").size)
        assertEquals(List(8) { "Bearer access-new" }, server.requestsTo("/api/group/").map { it.header("Authorization") })
    }

    @Test
    fun unauthorizedResponseRefreshesAndReplaysOnce() = runBlocking {
        val store = storeWith(sessionAccount(base, expiresAtMillis = clock() + 600_000))
        server.handler = { request ->
            when (request.path) {
                "/api/user/auth/refresh" -> sessionBundleResponse("access-new", "sid-1.secret-new")
                // The session version moved on (e.g. a password change elsewhere) while the token is unexpired.
                "/api/group/" -> if (request.header("Authorization") == "Bearer access-new") groupsOk() else authError(401, "AUTH_SESSION_REVOKED")
                else -> MockResponse().setResponseCode(404)
            }
        }
        val client = client(store)

        assertEquals(listOf("default", "vip"), client.admin.groups())

        assertEquals(
            listOf("/api/group/ Bearer access-old", "/api/user/auth/refresh null", "/api/group/ Bearer access-new"),
            server.requests.map { "${it.path} ${it.header("Authorization")}" },
        )
        assertTrue(client.auth.state.value is AuthState.SignedIn)
    }

    @Test
    fun revokedSessionOnRefreshSignsOutWithSessionExpired() {
        val store = storeWith(sessionAccount(base, expiresAtMillis = clock() - 1_000))
        server.handler = { request ->
            when (request.path) {
                "/api/user/auth/refresh" -> authError(401, "AUTH_SESSION_REVOKED")
                else -> authError(401, "AUTH_UNAUTHORIZED")
            }
        }
        val client = client(store)

        assertThrows(ApiError.Unauthorized::class.java) { runBlocking { client.admin.groups() } }

        assertEquals(1, server.requestsTo("/api/user/auth/refresh").size)
        assertEquals(AuthState.SignedOut(SignOutReason.SESSION_EXPIRED), client.auth.state.value)
        assertNull(store.load())
    }

    @Test
    fun refreshServerErrorKeepsTheSessionAndSurfacesAnIOException() {
        val account = sessionAccount(base, expiresAtMillis = clock() - 1_000)
        val store = storeWith(account)
        server.handler = { request ->
            if (request.path == "/api/user/auth/refresh") authError(503, "AUTH_INTERNAL_ERROR") else groupsOk()
        }
        val client = client(store)

        val error = assertThrows(ApiError.Network::class.java) { runBlocking { client.admin.groups() } }
        // (Coroutine stack-trace recovery may wrap the original exception, so walk the cause chain.)
        val refreshFailure = generateSequence(error.cause) { it.cause }.filterIsInstance<RefreshFailedException>().first()
        assertEquals(503, refreshFailure.status)
        assertThrows(IOException::class.java) { client.auth.accessTokenForRequest(AuthManager.endpoint(base, "/api/group/")) }

        assertEquals(AuthState.SignedIn(account), client.auth.state.value)
        assertEquals(account, store.load())
        assertTrue(server.requestsTo("/api/group/").isEmpty())
    }

    @Test
    fun refreshWhoseResponseIsLostIsRetriedWithTheSameTokenAndTheSessionSurvives() = runBlocking {
        val store = storeWith(sessionAccount(base, expiresAtMillis = clock() - 1_000))
        val refreshes = AtomicInteger()
        server.handler = { request ->
            when (request.path) {
                // The server rotated but the response never arrived. Re-presenting the old token inside the
                // 30 s replay window returns the same rotation; presenting it later would revoke the session.
                "/api/user/auth/refresh" ->
                    if (refreshes.incrementAndGet() == 1) MockResponse().setSocketPolicy(SocketPolicy.DISCONNECT_AFTER_REQUEST)
                    else sessionBundleResponse("access-new", "sid-1.secret-new")
                "/api/group/" -> if (request.header("Authorization") == "Bearer access-new") groupsOk() else authError(401, "AUTH_TOKEN_EXPIRED")
                else -> MockResponse().setResponseCode(404)
            }
        }
        val client = client(store)
        client.auth.sleeper = { clock.advance(it) }

        assertEquals(listOf("default", "vip"), client.admin.groups())

        assertEquals(List(2) { "new_api_refresh=sid-1.secret-old" }, server.requestsTo("/api/user/auth/refresh").map { it.header("Cookie") })
        assertEquals("sid-1.secret-new", (store.load()!!.credentials as Credentials.Session).refreshToken)
        assertTrue(client.auth.state.value is AuthState.SignedIn)
    }

    @Test
    fun rateLimitedProactiveRefreshKeepsUsingTheStillValidTokenWithoutHammeringRefresh() = runBlocking {
        val account = sessionAccount(base, expiresAtMillis = clock() + 30_000)
        val store = storeWith(account)
        server.handler = { request ->
            when (request.path) {
                "/api/user/auth/refresh" -> MockResponse().setResponseCode(429).setHeader("Retry-After", "20")
                "/api/group/" -> groupsOk()
                else -> MockResponse().setResponseCode(404)
            }
        }
        val client = client(store)

        client.admin.groups()
        clock.advance(5_000)
        client.admin.groups()

        assertEquals(1, server.requestsTo("/api/user/auth/refresh").size)
        assertEquals(List(2) { "Bearer access-old" }, server.requestsTo("/api/group/").map { it.header("Authorization") })
        assertEquals(account, store.load())
    }

    @Test
    fun forbiddenOriginFallsBackToHttpHostAndKeepsIt() = runBlocking {
        val hostOrigin = ServerUrl.origin(base) // the server sees http://<Host>
        val store = storeWith(sessionAccount(base, expiresAtMillis = clock() - 1_000, origin = "https://admin.example.com"))
        var rotation = 0
        server.handler = { request ->
            when (request.path) {
                "/api/user/auth/refresh" ->
                    if (request.header("Origin") != hostOrigin) {
                        authError(403, "AUTH_ORIGIN_FORBIDDEN")
                    } else {
                        rotation++
                        sessionBundleResponse("access-$rotation", "sid-1.secret-$rotation")
                    }
                "/api/group/" -> groupsOk()
                else -> MockResponse().setResponseCode(404)
            }
        }
        val client = client(store)

        client.admin.groups()
        clock.advance(20 * 60_000L)
        client.admin.groups()

        assertEquals(
            listOf("https://admin.example.com", hostOrigin, hostOrigin),
            server.requestsTo("/api/user/auth/refresh").map { it.header("Origin") },
        )
        assertEquals(hostOrigin, (store.load()!!.credentials as Credentials.Session).origin)
    }

    @Test
    fun sessionMismatchRetriesOnceWithoutTheSessionHeader() = runBlocking {
        val store = storeWith(sessionAccount(base, expiresAtMillis = clock() - 1_000, sid = "sid-stale"))
        server.handler = { request ->
            when (request.path) {
                "/api/user/auth/refresh" ->
                    if (request.header("X-Auth-Session") != null) authError(409, "AUTH_SESSION_MISMATCH")
                    else sessionBundleResponse("access-new", "sid-1.secret-new", sid = "sid-1")
                "/api/group/" -> groupsOk()
                else -> MockResponse().setResponseCode(404)
            }
        }
        val client = client(store)

        client.admin.groups()

        assertEquals(listOf("sid-stale", null), server.requestsTo("/api/user/auth/refresh").map { it.header("X-Auth-Session") })
        assertEquals("sid-1", (store.load()!!.credentials as Credentials.Session).sessionId)
    }

    // ---- Account binding ----

    @Test
    fun requestsNotUnderTheSignedInServerGetNoTokenAndTheir401DoesNotSignOut() {
        val previous = FakeServer()
        try {
            val store = storeWith(sessionAccount("$base/newapi", expiresAtMillis = clock() + 600_000))
            server.handler = { request -> if (request.path == "/newapi/api/group/") groupsOk() else authError(401, "AUTH_UNAUTHORIZED") }
            previous.handler = { authError(401, "AUTH_UNAUTHORIZED") }
            val auth = AuthManager(store, OkHttpClient(), clock)
            val http = OkHttpClient.Builder().addInterceptor(AuthInterceptor(auth)).build()

            // Queued before the admin signed in to this server: another port, a sibling path, the host root.
            for (url in listOf("${previous.baseUrl}/newapi/api/group/", "$base/newapi2/api/group/", "$base/api/group/", "$base/newapi/api/group/")) {
                http.newCall(Request.Builder().url(url).build()).execute().close()
            }

            assertEquals(listOf<String?>(null), previous.requests.map { it.header("Authorization") })
            assertEquals(
                listOf("/newapi2/api/group/ null", "/api/group/ null", "/newapi/api/group/ Bearer access-old"),
                server.requests.map { "${it.path} ${it.header("Authorization")}" },
            )
            assertTrue(auth.state.value is AuthState.SignedIn)
        } finally {
            previous.close()
        }
    }

    @Test
    fun unauthorizedReplayDoesNotSignOutANewerSignIn() {
        val store = storeWith(sessionAccount(base, expiresAtMillis = clock() + 600_000))
        val client = client(store)
        server.handler = { request ->
            val bearer = request.header("Authorization")
            when {
                request.path == "/api/user/auth/refresh" -> sessionBundleResponse("access-new", "sid-1.secret-new")
                request.path == "/api/user/self" && bearer == "Bearer pat-2" -> selfResponse()
                request.path == "/api/group/" && bearer == "Bearer access-new" -> {
                    // The admin signs in again while the replay is in flight, so this 401 is about a token that is gone.
                    runBlocking { client.auth.loginWithAccessToken(base, "pat-2") }
                    authError(401, "AUTH_SESSION_REVOKED")
                }
                else -> authError(401, "AUTH_SESSION_REVOKED")
            }
        }

        assertThrows(ApiError.Unauthorized::class.java) { runBlocking { client.admin.groups() } }

        assertEquals(Credentials.AccessToken("pat-2"), store.load()!!.credentials)
        assertEquals(Credentials.AccessToken("pat-2"), (client.auth.state.value as AuthState.SignedIn).account.credentials)
    }

    // ---- Logout, password, profile ----

    @Test
    fun logoutRevokesTheSessionAndClearsTheStore() = runBlocking {
        val store = storeWith(sessionAccount(base, expiresAtMillis = clock() + 600_000))
        server.handler = { okData("""{"revoked_sid":"sid-1","cookie_cleared":true}""") }
        val client = client(store)

        client.auth.logout()

        val logout = server.requestsTo("/api/user/auth/logout").single()
        assertEquals("Bearer access-old", logout.header("Authorization"))
        assertEquals("new_api_refresh=sid-1.secret-old", logout.header("Cookie"))
        assertEquals("sid-1", logout.header("X-Auth-Session"))
        assertEquals(ServerUrl.origin(base), logout.header("Origin"))
        assertNull(store.load())
        assertEquals(AuthState.SignedOut(SignOutReason.USER), client.auth.state.value)
    }

    @Test
    fun logoutClearsTheStoreEvenWhenTheServerIsUnreachable() = runBlocking {
        val store = storeWith(sessionAccount(base, expiresAtMillis = clock() + 600_000))
        val client = client(store)
        server.close()

        client.auth.logout()

        assertNull(store.load())
        assertEquals(AuthState.SignedOut(SignOutReason.USER), client.auth.state.value)
    }

    @Test
    fun changeOwnPasswordSendsOnlyThePasswordsAndAppliesTheRotatedAccessToken() = runBlocking {
        val store = storeWith(sessionAccount(base, expiresAtMillis = clock() + 600_000))
        server.handler = { request ->
            jsonResponse(
                """{"success":true,"message":"","data":{"access_token":"access-after-change","token_type":"Bearer",
                   "access_expires_at":${SERVER_NOW_SECONDS + 900},"session":{"sid":"sid-1"}}}""",
            ).setHeader("Date", com.newapi.admin.core.testing.httpDate(SERVER_NOW_SECONDS))
        }
        val client = client(store)

        client.auth.changeOwnPassword("old-pass", "new-pass-123")

        val put = server.requests.single()
        assertEquals("PUT /api/user/self", "${put.method} ${put.path}")
        assertEquals(buildJsonObject { put("original_password", "old-pass"); put("password", "new-pass-123") }, put.json())
        val session = store.load()!!.credentials as Credentials.Session
        assertEquals("access-after-change", session.accessToken)
        assertEquals("sid-1.secret-old", session.refreshToken)
        assertEquals((SERVER_NOW_SECONDS + 900) * 1000, session.accessExpiresAtMillis)
    }

    @Test
    fun changeOwnPasswordWithAnAccessTokenFailsWithoutCallingTheServer() {
        val store = storeWith(sessionAccount(base, expiresAtMillis = 0).copy(credentials = Credentials.AccessToken("pat-1")))
        val client = client(store)

        assertThrows(ApiError.Business::class.java) { runBlocking { client.auth.changeOwnPassword("a", "bbbbbbbb") } }
        assertTrue(server.requests.isEmpty())
    }

    @Test
    fun reloadProfileThatLostAdminSignsOutWithNotAdmin() {
        val store = storeWith(sessionAccount(base, expiresAtMillis = clock() + 600_000))
        server.handler = { request ->
            when (request.path) {
                "/api/user/self" -> selfResponse(role = Roles.COMMON)
                else -> okData()
            }
        }
        val client = client(store)

        assertThrows(ApiError.NotAdmin::class.java) { runBlocking { client.auth.reloadProfile() } }

        assertEquals(1, server.requestsTo("/api/user/auth/logout").size)
        assertEquals(AuthState.SignedOut(SignOutReason.NOT_ADMIN), client.auth.state.value)
        assertNull(store.load())
    }
}
