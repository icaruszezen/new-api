package com.newapi.admin.core.net

import com.newapi.admin.core.api.AdminApi
import com.newapi.admin.core.auth.AuthManager
import com.newapi.admin.core.auth.SessionStore
import okhttp3.OkHttpClient

/**
 * Wires the HTTP stack: a base client that adds `User-Agent` and `Accept-Language`, the [AuthManager]
 * on top of it, and an authenticated client (base + [AuthInterceptor]) for [AdminApi].
 */
class NewApiClient(
    store: SessionStore,
    userAgent: String,
    acceptLanguage: () -> String,
    okHttpBuilder: OkHttpClient.Builder = OkHttpClient.Builder(),
    clock: () -> Long = System::currentTimeMillis,
) {
    val auth: AuthManager = TODO()
    val admin: AdminApi = TODO()
}
