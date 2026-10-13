package com.newapi.admin.core.auth

import com.newapi.admin.core.model.SelfUser
import com.newapi.admin.core.model.SystemStatus
import kotlinx.coroutines.flow.StateFlow
import okhttp3.OkHttpClient

/**
 * Owns the signed-in account: login (password, 2FA, access token), single-flight refresh with
 * rotation, logout and own-password change. Thread-safe; refresh blocks the calling OkHttp thread.
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
    /** SignedIn as long as usable credentials are stored. */
    val state: StateFlow<AuthState> get() = TODO()

    /** The current account, or null when signed out. */
    val account: StoredAccount? get() = TODO()

    /** `GET /api/status` without credentials. [baseUrl] must already be normalised. */
    suspend fun fetchStatus(baseUrl: String): SystemStatus = TODO()

    /**
     * `POST /api/user/login`. Throws [com.newapi.admin.core.net.ApiError] (including `TurnstileRequired`,
     * `PasswordLoginDisabled`, `NotAdmin`, `Business` for wrong credentials).
     */
    suspend fun loginWithPassword(baseUrl: String, username: String, password: String): LoginResult = TODO()

    /** `POST /api/user/login/2fa` with the flow token from [LoginResult.TwoFactorRequired]. */
    suspend fun verifyTwoFactor(baseUrl: String, flowToken: String, code: String): SelfUser = TODO()

    /** Validates a personal access token with `GET /api/user/self` and stores it. */
    suspend fun loginWithAccessToken(baseUrl: String, token: String): SelfUser = TODO()

    /** Best-effort `POST /api/user/auth/logout`, then clears local credentials ([SignOutReason.USER]). */
    suspend fun logout(): Unit = TODO()

    /**
     * `PUT /api/user/self {original_password, password}` for the signed-in admin (session login only;
     * a PAT gets a Business error). Applies the rotated access token from the response.
     */
    suspend fun changeOwnPassword(originalPassword: String, newPassword: String): Unit = TODO()

    /** Re-reads `GET /api/user/self` and updates the stored profile (signs out with NOT_ADMIN if no longer admin). */
    suspend fun reloadProfile(): SelfUser = TODO()

    // ---- Used by the auth interceptor (blocking, called on OkHttp threads) ----

    /** Bearer value to send now; refreshes first when a session token expires within 60 s. Null when signed out. */
    fun accessTokenForRequest(): String? = TODO()

    /**
     * Called after a 401 that was sent with [staleToken]. Returns a token to retry with, or null when the
     * credentials are definitively invalid (in which case the manager has already signed out with
     * SESSION_EXPIRED). Throws [java.io.IOException] for transient refresh failures (network, 429, 5xx).
     */
    fun tokenAfterUnauthorized(staleToken: String): String? = TODO()

    /** Clears credentials after a 401 that a fresh token could not fix. */
    fun signOutExpired(): Unit = TODO()
}
