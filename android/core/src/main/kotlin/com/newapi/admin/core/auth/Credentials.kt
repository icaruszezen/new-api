package com.newapi.admin.core.auth

import com.newapi.admin.core.model.SelfUser
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** How the app authenticates against one server. Persisted as JSON by a [SessionStore]. */
@Serializable
sealed interface Credentials {

    /**
     * A dashboard login session (password or 2FA login).
     *
     * The access token lives 15 minutes; the refresh token (`new_api_refresh` cookie value, rotated on
     * every refresh) keeps the session alive for 30 days from login.
     */
    @Serializable
    @SerialName("session")
    data class Session(
        val accessToken: String,
        /** Device-clock epoch millis at which the access token expires (server expiry corrected for clock skew). */
        val accessExpiresAtMillis: Long,
        val refreshToken: String,
        val sessionId: String,
        /** Origin header value sent on refresh/logout; see `SessionCookieOriginGuard`. */
        val origin: String,
    ) : Credentials

    /** A personal access token generated in the web profile. Never expires, never refreshes. */
    @Serializable
    @SerialName("access_token")
    data class AccessToken(val token: String) : Credentials
}

/** Everything persisted for the signed-in account. */
@Serializable
data class StoredAccount(
    /** Normalised base URL, see [com.newapi.admin.core.ServerUrl.normalize]. */
    val baseUrl: String,
    val credentials: Credentials,
    /** Last known profile, refreshed on every login/refresh. */
    val user: SelfUser,
)

/**
 * Synchronous persistence for the signed-in account. Called from OkHttp threads while refreshing,
 * so implementations must write durably before returning (the refresh token rotates and the old one
 * is revoked 30 s later).
 */
interface SessionStore {
    fun load(): StoredAccount?
    fun save(account: StoredAccount?)
}

class InMemorySessionStore(private var account: StoredAccount? = null) : SessionStore {
    @Synchronized override fun load(): StoredAccount? = account
    @Synchronized override fun save(account: StoredAccount?) {
        this.account = account
    }
}

enum class SignOutReason {
    /** The user tapped "log out". */
    USER,
    /** Refresh failed definitively or a PAT was rejected. */
    SESSION_EXPIRED,
    /** The account lost admin rights or was disabled. */
    NOT_ADMIN,
}

sealed interface AuthState {
    data class SignedOut(val reason: SignOutReason? = null) : AuthState
    data class SignedIn(val account: StoredAccount) : AuthState
}

sealed interface LoginResult {
    data class Success(val user: SelfUser) : LoginResult

    /**
     * Password was right but the account has 2FA; call `verifyTwoFactor` within 5 minutes.
     * [baseUrl] is normalised; [expiresAt] is the flow's expiry in unix seconds of the server clock.
     * A wrong code does not consume the flow; five wrong codes lock 2FA for five minutes.
     */
    data class TwoFactorRequired(val baseUrl: String, val flowToken: String, val expiresAt: Long) : LoginResult
}
