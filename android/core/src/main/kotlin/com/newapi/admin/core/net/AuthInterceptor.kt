package com.newapi.admin.core.net

import com.newapi.admin.core.auth.AuthManager
import java.io.IOException
import okhttp3.Interceptor
import okhttp3.Request
import okhttp3.Response

/**
 * Adds `Authorization: Bearer <token>` from [AuthManager.accessTokenForRequest]. On a 401 it asks
 * [AuthManager.tokenAfterUnauthorized] for a fresh token and replays the request once; a second 401
 * signs out via [AuthManager.signOutExpired]. Transient refresh failures surface as IOException.
 *
 * Requests to `/api/user/auth/...`, `/api/user/login...` and `/api/status`, and requests that already
 * carry an `Authorization` header, pass through untouched.
 */
class AuthInterceptor(private val auth: AuthManager) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val request = chain.request()
        if (request.header("Authorization") != null || isAuthExempt(request)) return chain.proceed(request)

        val token = guarded { auth.accessTokenForRequest() } ?: return chain.proceed(request)
        val response = chain.proceed(request.withBearer(token))
        if (response.code != 401) return response

        // Keep the 401 body so it can still be returned after the original response is closed.
        val unauthorizedBody = response.peekBody(MAX_ERROR_BODY_BYTES)
        val unauthorized = response.newBuilder().body(unauthorizedBody).build()
        response.close()

        val fresh = guarded { auth.tokenAfterUnauthorized(token) } ?: return unauthorized
        val replay = chain.proceed(request.withBearer(fresh))
        if (replay.code == 401) guarded { auth.signOutExpired() }
        return replay
    }

    private fun Request.withBearer(token: String): Request = newBuilder().header("Authorization", "Bearer $token").build()

    /** OkHttp only tolerates IOExceptions from interceptors; anything else would crash a dispatcher thread. */
    private inline fun <T> guarded(block: () -> T): T = try {
        block()
    } catch (e: IOException) {
        throw e
    } catch (e: RuntimeException) {
        throw IOException("authentication failed: ${e.message}", e)
    }

    companion object {
        private const val MAX_ERROR_BODY_BYTES = 64L * 1024

        internal fun isAuthExempt(request: Request): Boolean {
            val path = request.url.encodedPath
            return path.contains("/api/user/auth/") ||
                path.endsWith("/api/user/login") ||
                path.contains("/api/user/login/") ||
                path.endsWith("/api/status")
        }
    }
}
