package com.newapi.admin.core.auth

import com.newapi.admin.core.ServerUrl
import com.newapi.admin.core.model.SelfUser
import com.newapi.admin.core.model.SystemStatus
import com.newapi.admin.core.net.ApiError
import com.newapi.admin.core.net.ApiJson
import com.newapi.admin.core.net.AuthInterceptor
import com.newapi.admin.core.net.EmptyBody
import com.newapi.admin.core.net.HttpResult
import com.newapi.admin.core.net.RefreshFailedException
import com.newapi.admin.core.net.await
import com.newapi.admin.core.net.bool
import com.newapi.admin.core.net.decode
import com.newapi.admin.core.net.decodeData
import com.newapi.admin.core.net.executeBlocking
import com.newapi.admin.core.net.long
import com.newapi.admin.core.net.obj
import com.newapi.admin.core.net.string
import com.newapi.admin.core.net.toApiError
import com.newapi.admin.core.net.toRequestBody
import java.io.IOException
import java.io.InterruptedIOException
import java.net.ConnectException
import java.net.UnknownHostException
import java.util.concurrent.CompletableFuture
import java.util.concurrent.ExecutionException
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.NonCancellable
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import okhttp3.Headers
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.OkHttpClient
import okhttp3.Request

/**
 * Owns the signed-in account: login (password, 2FA, access token), single-flight refresh with
 * rotation, logout and own-password change. Thread-safe; refresh blocks the calling OkHttp thread.
 *
 * Every credential change is written with [SessionStore.save] before the new value is used, because
 * the refresh token rotates on every refresh and the previous one dies 30 s later.
 *
 * @param baseClient an OkHttp client WITHOUT the auth interceptor (it already adds User-Agent and
 *   Accept-Language); used for status, login, refresh and logout.
 * @param clock device clock in epoch millis (injectable for tests).
 */
class AuthManager(
    private val store: SessionStore,
    private val baseClient: OkHttpClient,
    private val clock: () -> Long = System::currentTimeMillis,
) {
    /** Guards [current], [mutableState] and every [SessionStore.save]. */
    private val lock = Any()

    @Volatile
    private var current: StoredAccount? = runCatching { store.load() }.getOrNull()

    private val mutableState = MutableStateFlow<AuthState>(current?.let { AuthState.SignedIn(it) } ?: AuthState.SignedOut())

    /** SignedIn as long as usable credentials are stored. */
    val state: StateFlow<AuthState> = mutableState.asStateFlow()

    /** The current account, or null when signed out. */
    val account: StoredAccount? get() = current

    /** Waits between AUTH_REFRESH_RACE retries; replaced in tests. */
    internal var sleeper: (Long) -> Unit = { Thread.sleep(it) }

    private val refreshGate = Any()
    private var inFlightRefresh: CompletableFuture<String?>? = null

    /** After a transient failure of a proactive refresh, still-valid tokens are used without retrying until then. */
    @Volatile
    private var proactiveRefreshPausedUntil = 0L

    /** For `/api/user/self` calls made on behalf of the signed-in account (refresh + replay on 401). */
    private val authenticatedClient: OkHttpClient by lazy {
        baseClient.newBuilder().addInterceptor(AuthInterceptor(this)).build()
    }

    /** Logout is best effort and must not hold up signing out for long. */
    private val logoutClient: OkHttpClient by lazy {
        baseClient.newBuilder().callTimeout(10, TimeUnit.SECONDS).build()
    }

    /** One refresh attempt is short, so a lost response can still be re-presented inside the server's replay window. */
    private val refreshClient: OkHttpClient by lazy {
        baseClient.newBuilder().callTimeout(REFRESH_ATTEMPT_TIMEOUT_MS, TimeUnit.MILLISECONDS).build()
    }

    /**
     * `GET /api/status` without credentials. [baseUrl] should already be normalised (it is normalised
     * again, which throws IllegalArgumentException for unusable input). Throws
     * [ApiError.InvalidResponse] when the server is not a new-api server.
     */
    suspend fun fetchStatus(baseUrl: String): SystemStatus = withContext(Dispatchers.IO) {
        val url = endpoint(ServerUrl.normalize(baseUrl), "/api/status")
        val result = send(baseClient, Request.Builder().url(url).get().build())
        if (!result.isSuccessful) {
            if (result.code == 429 || result.code >= 500) throw result.toApiError()
            throw ApiError.InvalidResponse("HTTP ${result.code} from /api/status: not a new-api server")
        }
        val obj = result.json ?: throw ApiError.InvalidResponse("/api/status did not return JSON")
        val data = obj.obj("data")
        if (obj.bool("success") != true || data == null) throw ApiError.InvalidResponse("/api/status is not a new-api response")
        decode(SystemStatus.serializer(), data, "status")
    }

    /**
     * `POST /api/user/login`. Throws [com.newapi.admin.core.net.ApiError] (including `TurnstileRequired`,
     * `PasswordLoginDisabled`, `NotAdmin`, `Business` for wrong credentials).
     *
     * Reads `/api/status` first so that Turnstile and disabled password login fail without spending
     * an attempt of the server's login rate limit (20 per 20 minutes per IP).
     */
    suspend fun loginWithPassword(baseUrl: String, username: String, password: String): LoginResult =
        withContext(Dispatchers.IO) {
            val base = ServerUrl.normalize(baseUrl)
            val status = fetchStatus(base)
            if (!status.passwordLoginEnabled) throw ApiError.PasswordLoginDisabled()
            if (status.turnstileCheck) throw ApiError.TurnstileRequired()
            val body = buildJsonObject {
                put("username", username.trim())
                put("password", password)
            }
            val result = send(baseClient, Request.Builder().url(endpoint(base, "/api/user/login")).post(body.toRequestBody()).build())
            val envelope = try {
                result.envelope()
            } catch (e: ApiError.Business) {
                throw loginBusinessError(e)
            }
            val data = envelope.obj("data") ?: throw ApiError.InvalidResponse("login response has no data")
            if (data.bool("require_2fa") == true) {
                val flowToken = data.string("flow_token")?.takeIf { it.isNotBlank() }
                    ?: throw ApiError.InvalidResponse("2FA login response has no flow token")
                return@withContext LoginResult.TwoFactorRequired(base, flowToken, data.long("expires_at") ?: 0L)
            }
            LoginResult.Success(completeSessionLogin(base, result, data))
        }

    /** `POST /api/user/login/2fa` with the flow token from [LoginResult.TwoFactorRequired]. */
    suspend fun verifyTwoFactor(baseUrl: String, flowToken: String, code: String): SelfUser = withContext(Dispatchers.IO) {
        val base = ServerUrl.normalize(baseUrl)
        val body = buildJsonObject {
            put("code", code.trim())
            put("flow_token", flowToken)
        }
        val result = send(baseClient, Request.Builder().url(endpoint(base, "/api/user/login/2fa")).post(body.toRequestBody()).build())
        val data = result.envelope().obj("data") ?: throw ApiError.InvalidResponse("2FA response has no data")
        completeSessionLogin(base, result, data)
    }

    /** Validates a personal access token with `GET /api/user/self` and stores it. */
    suspend fun loginWithAccessToken(baseUrl: String, token: String): SelfUser = withContext(Dispatchers.IO) {
        val base = ServerUrl.normalize(baseUrl)
        val pat = token.trim().let { if (it.startsWith("Bearer ", ignoreCase = true)) it.substring(7).trim() else it }
        if (pat.isEmpty()) throw ApiError.Business("The access token is empty.")
        val self = fetchSelf(base, pat)
        if (!self.isAdmin) throw ApiError.NotAdmin()
        synchronized(lock) { commitLocked(StoredAccount(base, Credentials.AccessToken(pat), self)) }
        self
    }

    /** Best-effort `POST /api/user/auth/logout`, then clears local credentials ([SignOutReason.USER]). */
    suspend fun logout(): Unit = withContext(Dispatchers.IO + NonCancellable) {
        val account = current
        try {
            val credentials = account?.credentials
            if (account != null && credentials is Credentials.Session) revokeBlocking(account.baseUrl, credentials)
        } finally {
            synchronized(lock) { clearLocked(SignOutReason.USER) }
        }
    }

    /**
     * `PUT /api/user/self {original_password, password}` for the signed-in admin (session login only;
     * a PAT gets a Business error). Applies the rotated access token from the response.
     */
    suspend fun changeOwnPassword(originalPassword: String, newPassword: String): Unit = withContext(Dispatchers.IO) {
        val account = current ?: throw ApiError.Unauthorized(null, "signed out")
        val session = account.credentials as? Credentials.Session
            ?: throw ApiError.Business("Changing your own password needs a password login; an access token cannot do it.")
        // The server treats this exact value as "no password given" and would report success without changing anything.
        if (newPassword.isEmpty() || newPassword == UNCHANGED_PASSWORD_SENTINEL) throw ApiError.Business("This password cannot be used.")
        val body = buildJsonObject {
            put("original_password", originalPassword)
            put("password", newPassword)
        }
        val request = Request.Builder().url(endpoint(account.baseUrl, "/api/user/self")).put(body.toRequestBody()).build()
        val result = send(authenticatedClient, request)
        val data = result.envelope().obj("data") ?: return@withContext
        synchronized(lock) {
            val now = current ?: return@synchronized
            val nowSession = now.credentials as? Credentials.Session ?: return@synchronized
            if (now.baseUrl != account.baseUrl || nowSession.sessionId != session.sessionId) return@synchronized
            val rotated = sessionFromBundle(
                result,
                data,
                origin = nowSession.origin,
                fallbackRefreshToken = nowSession.refreshToken,
                fallbackSid = nowSession.sessionId,
            ) ?: return@synchronized
            commitLocked(now.copy(credentials = rotated))
        }
    }

    /** Re-reads `GET /api/user/self` and updates the stored profile (signs out with NOT_ADMIN if no longer admin). */
    suspend fun reloadProfile(): SelfUser = withContext(Dispatchers.IO) {
        val account = current ?: throw ApiError.Unauthorized(null, "signed out")
        val request = Request.Builder().url(endpoint(account.baseUrl, "/api/user/self")).get().build()
        val self = send(authenticatedClient, request).envelope().decodeData(SelfUser.serializer(), "profile")
        if (!self.isAdmin) {
            val now = current
            if (now != null && now.baseUrl == account.baseUrl && sameIdentity(now.credentials, account.credentials)) {
                val credentials = now.credentials
                if (credentials is Credentials.Session) revokeBlocking(now.baseUrl, credentials)
                synchronized(lock) { if (current?.credentials == now.credentials) clearLocked(SignOutReason.NOT_ADMIN) }
            }
            throw ApiError.NotAdmin()
        }
        synchronized(lock) {
            val now = current
            if (now != null && now.baseUrl == account.baseUrl && sameIdentity(now.credentials, account.credentials)) {
                commitLocked(now.copy(user = self))
            }
        }
        self
    }

    // ---- Used by the auth interceptor (blocking, called on OkHttp threads) ----

    /**
     * Bearer value to send to [url] now; refreshes first when a session token expires within 60 s. Null when
     * signed out or when [url] is not under the signed-in server (a request built before the account changed).
     */
    fun accessTokenForRequest(url: HttpUrl): String? {
        val account = accountFor(url) ?: return null
        return when (val credentials = account.credentials) {
            is Credentials.AccessToken -> credentials.token
            is Credentials.Session -> {
                val now = clock()
                if (credentials.accessExpiresAtMillis - now > REFRESH_MARGIN_MS) return credentials.accessToken
                val stillValid = credentials.accessExpiresAtMillis > now
                if (stillValid && now < proactiveRefreshPausedUntil) return credentials.accessToken
                try {
                    refreshSingleFlight(url, credentials.accessToken)
                } catch (e: IOException) {
                    if (!stillValid) throw e
                    // The token still works for a few more seconds; use it and retry the refresh later.
                    val retryAfterMillis = ((e as? RefreshFailedException)?.retryAfterSeconds ?: 0L) * 1000
                    proactiveRefreshPausedUntil = clock() + maxOf(PROACTIVE_RETRY_PAUSE_MS, retryAfterMillis)
                    credentials.accessToken
                }
            }
        }
    }

    /**
     * Called after a 401 from [url] that was sent with [staleToken]. Returns a token to retry with, or null when
     * the credentials are definitively invalid (in which case the manager has already signed out with
     * SESSION_EXPIRED) or [url] is no longer under the signed-in server. Throws [java.io.IOException] for
     * transient refresh failures (network, 429, 5xx).
     */
    fun tokenAfterUnauthorized(url: HttpUrl, staleToken: String): String? {
        val account = accountFor(url) ?: return null
        return when (val credentials = account.credentials) {
            is Credentials.AccessToken -> {
                if (credentials.token != staleToken) return credentials.token
                synchronized(lock) { if (current?.credentials == credentials) clearLocked(SignOutReason.SESSION_EXPIRED) }
                null
            }
            is Credentials.Session -> {
                if (credentials.accessToken != staleToken) credentials.accessToken else refreshSingleFlight(url, staleToken)
            }
        }
    }

    /**
     * Clears credentials after a 401 that the fresh [usedToken] could not fix. Does nothing when the account
     * has moved on since (signed in again, or refreshed by another request).
     */
    fun signOutExpired(usedToken: String) {
        synchronized(lock) {
            val currentToken = when (val credentials = current?.credentials) {
                is Credentials.Session -> credentials.accessToken
                is Credentials.AccessToken -> credentials.token
                null -> null
            }
            if (currentToken == usedToken) clearLocked(SignOutReason.SESSION_EXPIRED)
        }
    }

    /** The signed-in account, when [url] was built for its server. */
    private fun accountFor(url: HttpUrl): StoredAccount? =
        current?.takeIf { url.toString().startsWith(it.baseUrl.trimEnd('/') + "/") }

    // ---- Login internals ----

    private suspend fun completeSessionLogin(base: String, result: HttpResult, data: JsonObject): SelfUser {
        val session = sessionFromBundle(result, data, origin = ServerUrl.origin(base), fallbackRefreshToken = null, fallbackSid = null)
            ?: throw ApiError.InvalidResponse("login response lacks the access token, session id or new_api_refresh cookie")
        val loginUser = data.obj("user")?.let { runCatching { decode(SelfUser.serializer(), it, "user") }.getOrNull() }
        if (loginUser != null && !loginUser.isAdmin) {
            revokeBlocking(base, session)
            throw ApiError.NotAdmin()
        }
        val self = try {
            fetchSelf(base, session.accessToken)
        } catch (e: ApiError) {
            val transient = e is ApiError.Network || e is ApiError.Http || e is ApiError.RateLimited || e is ApiError.InvalidResponse
            if (loginUser == null || !transient) {
                revokeBlocking(base, session)
                throw e
            }
            loginUser
        }
        if (!self.isAdmin) {
            revokeBlocking(base, session)
            throw ApiError.NotAdmin()
        }
        synchronized(lock) { commitLocked(StoredAccount(base, session, self)) }
        return self
    }

    private suspend fun fetchSelf(base: String, bearer: String): SelfUser {
        val request = Request.Builder()
            .url(endpoint(base, "/api/user/self"))
            .header("Authorization", "Bearer $bearer")
            .get()
            .build()
        return send(baseClient, request).envelope().decodeData(SelfUser.serializer(), "profile")
    }

    private fun loginBusinessError(error: ApiError.Business): ApiError {
        val message = error.message.orEmpty()
        return when {
            message.contains("Turnstile", ignoreCase = true) -> ApiError.TurnstileRequired()
            PASSWORD_LOGIN_DISABLED_MESSAGES.any { message.trim() == it } -> ApiError.PasswordLoginDisabled()
            else -> error
        }
    }

    // ---- Refresh internals ----

    private fun refreshSingleFlight(url: HttpUrl, staleToken: String): String? {
        val account: StoredAccount
        val future: CompletableFuture<String?>
        val owner: Boolean
        synchronized(refreshGate) {
            account = accountFor(url) ?: return null
            val credentials = account.credentials as? Credentials.Session
                ?: return (account.credentials as Credentials.AccessToken).token
            if (credentials.accessToken != staleToken) return credentials.accessToken
            val running = inFlightRefresh
            if (running != null) {
                future = running
                owner = false
            } else {
                future = CompletableFuture()
                inFlightRefresh = future
                owner = true
            }
        }
        if (owner) {
            try {
                future.complete(performRefresh(account))
            } catch (e: Throwable) {
                future.completeExceptionally(e)
            } finally {
                synchronized(refreshGate) { inFlightRefresh = null }
            }
        }
        return try {
            future.get()
        } catch (e: ExecutionException) {
            when (val cause = e.cause) {
                is IOException -> throw cause
                else -> throw RefreshFailedException("session refresh failed: ${cause?.message}", cause = cause)
            }
        } catch (e: InterruptedException) {
            Thread.currentThread().interrupt()
            throw InterruptedIOException("interrupted while waiting for the session refresh")
        }
    }

    private fun performRefresh(account: StoredAccount): String? {
        val start = account.credentials as? Credentials.Session
            ?: return (account.credentials as Credentials.AccessToken).token
        val fallbackOrigin = runCatching { ServerUrl.hostHeaderOrigin(account.baseUrl) }.getOrNull()
        var origin = start.origin
        var sendSessionHeader = true
        var retriedUnauthorized = false
        var retriedOrigin = false
        var raceRetries = 0
        var lostResponseRetries = 0
        val firstSentAt = clock()
        while (true) {
            val request = refreshRequest(account.baseUrl, start, origin, sendSessionHeader)
            val result = try {
                refreshClient.executeBlocking(request, clock)
            } catch (e: IOException) {
                // The server may have rotated before the response was lost. Re-presenting the same token returns
                // that same rotation, but only inside the replay window, which opened after the first send; a token
                // presented later revokes the whole session. So every retry must be able to finish inside it.
                val neverSent = e is UnknownHostException || e is ConnectException
                val pause = LOST_RESPONSE_BACKOFF_MS.getOrNull(lostResponseRetries)
                if (neverSent || pause == null || clock() + pause + REFRESH_ATTEMPT_TIMEOUT_MS - firstSentAt > REFRESH_REPLAY_WINDOW_MS) {
                    throw RefreshFailedException("session refresh failed: ${e.message}", cause = e)
                }
                lostResponseRetries++
                backoff(pause)
                continue
            }
            val code = result.errorCode
            when {
                result.isSuccessful -> return applyRefresh(account, start, origin, result)
                result.code == 401 && code == "AUTH_SESSION_REVOKED" -> return signOutAfterRefresh(account)
                // AUTH_UNAUTHORIZED can come from a transient session lookup failure; re-presenting the token is safe.
                result.code == 401 && !retriedUnauthorized -> retriedUnauthorized = true
                result.code == 401 -> return signOutAfterRefresh(account)
                result.code == 409 && code == "AUTH_SESSION_MISMATCH" && sendSessionHeader -> sendSessionHeader = false
                result.code == 409 && code == "AUTH_REFRESH_RACE" && raceRetries < RACE_BACKOFF_MS.size ->
                    backoff(RACE_BACKOFF_MS[raceRetries++])
                result.code == 403 && code == "AUTH_ORIGIN_FORBIDDEN" && !retriedOrigin &&
                    fallbackOrigin != null && fallbackOrigin != origin -> {
                    origin = fallbackOrigin
                    retriedOrigin = true
                }
                result.code == 429 -> throw RefreshFailedException("session refresh rate limited", 429, result.retryAfterSeconds)
                result.code >= 500 -> throw RefreshFailedException("session refresh failed with HTTP ${result.code}", result.code)
                else -> return signOutAfterRefresh(account)
            }
        }
    }

    /** Waits with [sleeper] on an OkHttp thread, where only IOExceptions may escape. */
    private fun backoff(millis: Long) {
        try {
            sleeper(millis)
        } catch (e: InterruptedException) {
            Thread.currentThread().interrupt()
            throw InterruptedIOException("interrupted during refresh backoff")
        }
    }

    private fun refreshRequest(base: String, session: Credentials.Session, origin: String, sendSessionHeader: Boolean): Request =
        Request.Builder()
            .url(endpoint(base, "/api/user/auth/refresh"))
            .post(EmptyBody)
            .header("Cookie", "$REFRESH_COOKIE=${session.refreshToken}")
            .header("Origin", origin)
            .apply { if (sendSessionHeader) header("X-Auth-Session", session.sessionId) }
            .build()

    private fun applyRefresh(account: StoredAccount, start: Credentials.Session, origin: String, result: HttpResult): String? {
        val data = result.json?.takeIf { it.bool("success") == true }?.obj("data")
            ?: throw RefreshFailedException("refresh response is not a session bundle", result.code)
        val session = sessionFromBundle(result, data, origin, fallbackRefreshToken = null, fallbackSid = start.sessionId)
            ?: throw RefreshFailedException("refresh response lacks the access token or new_api_refresh cookie", result.code)
        val user = data.obj("user")
            ?.let { runCatching { ApiJson.decodeFromJsonElement(SelfUser.serializer(), it) }.getOrNull() }
            ?: account.user
        if (!user.isAdmin) {
            revokeBlocking(account.baseUrl, session)
            synchronized(lock) { if (isSameRefreshChain(current, account, start)) clearLocked(SignOutReason.NOT_ADMIN) }
            return null
        }
        synchronized(lock) {
            val now = current
            // Logged out or replaced while refreshing: drop the result.
            if (!isSameRefreshChain(now, account, start) || now == null) return null
            try {
                commitLocked(now.copy(credentials = session, user = user))
            } catch (e: RuntimeException) {
                // Not persisted, so not used. Re-presenting the old token within 30 s returns this same rotation.
                throw RefreshFailedException("could not store the refreshed session: ${e.message}", cause = e)
            }
        }
        proactiveRefreshPausedUntil = 0L
        return session.accessToken
    }

    private fun isSameRefreshChain(now: StoredAccount?, account: StoredAccount, start: Credentials.Session): Boolean {
        val session = now?.credentials as? Credentials.Session ?: return false
        return now.baseUrl == account.baseUrl && session.sessionId == start.sessionId && session.refreshToken == start.refreshToken
    }

    private fun signOutAfterRefresh(account: StoredAccount): String? {
        val start = account.credentials as Credentials.Session
        synchronized(lock) { if (isSameRefreshChain(current, account, start)) clearLocked(SignOutReason.SESSION_EXPIRED) }
        return null
    }

    // ---- Shared internals ----

    /**
     * Builds session credentials from a `{access_token, access_expires_at, session, ...}` bundle. The
     * expiry is moved onto the device clock using the response `Date` header.
     */
    private fun sessionFromBundle(
        result: HttpResult,
        data: JsonObject,
        origin: String,
        fallbackRefreshToken: String?,
        fallbackSid: String?,
    ): Credentials.Session? {
        val accessToken = data.string("access_token")?.takeIf { it.isNotBlank() } ?: return null
        val expiresAtSeconds = data.long("access_expires_at") ?: return null
        val sessionId = data.obj("session")?.string("sid")?.takeIf { it.isNotBlank() } ?: fallbackSid ?: return null
        val refreshToken = parseRefreshCookie(result.headers)?.takeIf { !it.cleared }?.value ?: fallbackRefreshToken ?: return null
        return Credentials.Session(
            accessToken = accessToken,
            accessExpiresAtMillis = expiresAtSeconds * 1000 + result.clockSkewMillis,
            refreshToken = refreshToken,
            sessionId = sessionId,
            origin = origin,
        )
    }

    /** Revokes a session server-side; never throws. */
    private fun revokeBlocking(base: String, session: Credentials.Session) {
        val fallbackOrigin = runCatching { ServerUrl.hostHeaderOrigin(base) }.getOrNull()
        var origin = session.origin
        var sendSessionHeader = true
        repeat(3) {
            val request = try {
                Request.Builder()
                    .url(endpoint(base, "/api/user/auth/logout"))
                    .post(EmptyBody)
                    .header("Authorization", "Bearer ${session.accessToken}")
                    .header("Cookie", "$REFRESH_COOKIE=${session.refreshToken}")
                    .header("Origin", origin)
                    .apply { if (sendSessionHeader) header("X-Auth-Session", session.sessionId) }
                    .build()
            } catch (e: IllegalArgumentException) {
                return
            }
            val result = try {
                logoutClient.executeBlocking(request, clock)
            } catch (e: IOException) {
                return
            }
            when {
                result.code == 403 && result.errorCode == "AUTH_ORIGIN_FORBIDDEN" && fallbackOrigin != null && fallbackOrigin != origin ->
                    origin = fallbackOrigin
                result.code == 409 && result.errorCode == "AUTH_SESSION_MISMATCH" && sendSessionHeader -> sendSessionHeader = false
                else -> return
            }
        }
    }

    private fun commitLocked(account: StoredAccount) {
        store.save(account)
        current = account
        mutableState.value = AuthState.SignedIn(account)
    }

    private fun clearLocked(reason: SignOutReason) {
        try {
            store.save(null)
        } finally {
            current = null
            mutableState.value = AuthState.SignedOut(reason)
        }
    }

    private suspend fun send(client: OkHttpClient, request: Request): HttpResult = try {
        client.await(request, clock)
    } catch (e: IOException) {
        throw e.toApiError()
    }

    private fun sameIdentity(a: Credentials, b: Credentials): Boolean = when {
        a is Credentials.Session && b is Credentials.Session -> a.sessionId == b.sessionId
        a is Credentials.AccessToken && b is Credentials.AccessToken -> a.token == b.token
        else -> false
    }

    internal companion object {
        const val REFRESH_COOKIE = "new_api_refresh"
        const val REFRESH_MARGIN_MS = 60_000L
        const val PROACTIVE_RETRY_PAUSE_MS = 10_000L
        val RACE_BACKOFF_MS = longArrayOf(80, 200, 500)

        /** The server accepts the previous refresh token for 30 s after rotating it (`RefreshReplayWindow`). */
        const val REFRESH_REPLAY_WINDOW_MS = 30_000L
        const val REFRESH_ATTEMPT_TIMEOUT_MS = 10_000L
        val LOST_RESPONSE_BACKOFF_MS = longArrayOf(500, 1_500, 3_000)
        const val UNCHANGED_PASSWORD_SENTINEL = "\$I_LOVE_U"

        /** `user.password_login_disabled` in en, zh-CN and zh-TW (i18n/locales). */
        val PASSWORD_LOGIN_DISABLED_MESSAGES = listOf(
            "Password login has been disabled by administrator",
            "管理员关闭了密码登录",
            "管理員關閉了密碼登錄",
        )

        fun endpoint(base: String, path: String): HttpUrl =
            (base.trimEnd('/') + path).toHttpUrlOrNull() ?: throw IllegalArgumentException("invalid server URL: $base")

        /** The `new_api_refresh` cookie from `Set-Cookie` headers (the last one wins). */
        fun parseRefreshCookie(headers: Headers): RefreshCookie? {
            var found: RefreshCookie? = null
            for (header in headers.values("Set-Cookie")) {
                val parts = header.split(';')
                val nameValue = parts.first()
                val separator = nameValue.indexOf('=')
                if (separator < 0 || nameValue.substring(0, separator).trim() != REFRESH_COOKIE) continue
                val value = nameValue.substring(separator + 1).trim().removeSurrounding("\"")
                var secure = false
                var cleared = value.isEmpty()
                for (attribute in parts.drop(1)) {
                    val name = attribute.substringBefore('=').trim().lowercase()
                    val attributeValue = attribute.substringAfter('=', "").trim()
                    when (name) {
                        "secure" -> secure = true
                        "max-age" -> if ((attributeValue.toLongOrNull() ?: 1L) <= 0L) cleared = true
                    }
                }
                found = RefreshCookie(value, secure, cleared)
            }
            return found
        }
    }

    /**
     * The refresh cookie. [secure] is set exactly when the server runs with `SESSION_COOKIE_SECURE=true`,
     * i.e. when refresh and logout are guarded by the Origin check (the client always sends `Origin`).
     */
    internal data class RefreshCookie(val value: String, val secure: Boolean, val cleared: Boolean)
}
