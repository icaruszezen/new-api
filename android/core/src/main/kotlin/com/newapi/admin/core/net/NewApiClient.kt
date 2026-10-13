package com.newapi.admin.core.net

import com.newapi.admin.core.api.AdminApi
import com.newapi.admin.core.auth.AuthManager
import com.newapi.admin.core.auth.SessionStore
import java.util.concurrent.TimeUnit
import okhttp3.OkHttpClient

/**
 * Wires the HTTP stack: a base client that adds `User-Agent` and `Accept-Language`, the [AuthManager]
 * on top of it, and an authenticated client (base + [AuthInterceptor]) for [AdminApi].
 *
 * Timeouts set on [okHttpBuilder] are replaced by the defaults below; slow channel operations get
 * longer per-call timeouts inside [AdminApi].
 */
class NewApiClient(
    store: SessionStore,
    userAgent: String,
    acceptLanguage: () -> String,
    okHttpBuilder: OkHttpClient.Builder = OkHttpClient.Builder(),
    clock: () -> Long = System::currentTimeMillis,
) {
    private val baseClient: OkHttpClient = okHttpBuilder
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .writeTimeout(30, TimeUnit.SECONDS)
        .callTimeout(90, TimeUnit.SECONDS)
        .addInterceptor { chain ->
            val builder = chain.request().newBuilder().header("User-Agent", userAgent)
            val language = runCatching(acceptLanguage).getOrNull()?.trim().orEmpty()
            if (language.isNotEmpty()) builder.header("Accept-Language", language)
            chain.proceed(builder.build())
        }
        .build()
        .also { client ->
            // Refreshes block inside interceptors and channel tests are slow; keep other calls flowing.
            if (client.dispatcher.maxRequestsPerHost < 16) client.dispatcher.maxRequestsPerHost = 16
        }

    val auth: AuthManager = AuthManager(store, baseClient, clock)

    private val authenticatedClient: OkHttpClient = baseClient.newBuilder()
        .addInterceptor(AuthInterceptor(auth))
        .build()

    val admin: AdminApi = AdminApi(auth, authenticatedClient, clock)
}
