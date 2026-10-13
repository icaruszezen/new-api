package com.newapi.admin.core.net

/** Every failure the API layer surfaces. The app maps these to localised text. */
sealed class ApiError(message: String, cause: Throwable? = null) : Exception(message, cause) {

    /** HTTP 200 with `success:false`; [message] is the server's (localised or Chinese) text. */
    class Business(message: String) : ApiError(message)

    /** 401 after any refresh attempt: the stored credentials no longer work and the app has signed out. */
    class Unauthorized(val code: String?, message: String) : ApiError(message)

    /** 403: role or channel permission missing (`code` is absent for channel permission checks). */
    class Forbidden(val code: String?, message: String) : ApiError(message)

    /** 429 (empty body). */
    class RateLimited(val retryAfterSeconds: Long?) : ApiError("rate limited")

    /** Any other non-2xx status. */
    class Http(val status: Int, val code: String?, message: String) : ApiError(message)

    /** Connection, TLS or timeout failure, or a transient failure while refreshing the session. */
    class Network(cause: Throwable) : ApiError(cause.message ?: cause.javaClass.simpleName, cause)

    /** The body was not the JSON we expected (wrong URL, proxy page, incompatible server). */
    class InvalidResponse(detail: String, cause: Throwable? = null) : ApiError(detail, cause)

    /** Login worked but the account is not an administrator (role < 10). The new session was logged out again. */
    class NotAdmin : ApiError("not an administrator")

    /** Password login needs a Cloudflare Turnstile token, which the app cannot obtain; use an access token. */
    class TurnstileRequired : ApiError("turnstile required")

    /** Password login is disabled on this server. */
    class PasswordLoginDisabled : ApiError("password login disabled")
}
