package com.newapi.admin.core.net

import com.newapi.admin.core.auth.AuthManager
import okhttp3.Interceptor
import okhttp3.Response

/**
 * Adds `Authorization: Bearer <token>` from [AuthManager.accessTokenForRequest]. On a 401 it asks
 * [AuthManager.tokenAfterUnauthorized] for a fresh token and replays the request once; a second 401
 * signs out via [AuthManager.signOutExpired]. Transient refresh failures surface as IOException.
 */
class AuthInterceptor(private val auth: AuthManager) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response = TODO()
}
