package com.newapi.admin.core

import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

object ServerUrl {
    /**
     * Turns user input into a base URL without a trailing slash: adds `https://` when no scheme is given,
     * lowercases the host, drops a default port, keeps a sub-path (for reverse proxies), removes a
     * trailing `/api` or `/`. Throws IllegalArgumentException when the input is not an http(s) URL.
     *
     * Examples: "example.com/" -> "https://example.com", "http://10.0.0.2:3000" -> "http://10.0.0.2:3000",
     * "https://Example.com:443/newapi/" -> "https://example.com/newapi".
     */
    fun normalize(input: String): String {
        val trimmed = input.trim()
        require(trimmed.isNotEmpty()) { "server URL is empty" }
        require(trimmed.none { it.isWhitespace() }) { "server URL contains whitespace" }
        val schemeEnd = trimmed.indexOf("://")
        val withScheme = when {
            schemeEnd < 0 -> "https://$trimmed"
            else -> {
                val scheme = trimmed.substring(0, schemeEnd).lowercase()
                require(scheme == "http" || scheme == "https") { "only http and https URLs are supported" }
                scheme + trimmed.substring(schemeEnd)
            }
        }
        val url = withScheme.toHttpUrlOrNull() ?: throw IllegalArgumentException("not a valid URL: $input")
        require(url.username.isEmpty() && url.password.isEmpty()) { "server URL must not contain credentials" }
        var path = url.encodedPath.trimEnd('/')
        if (path.endsWith("/api", ignoreCase = true)) path = path.dropLast(4).trimEnd('/')
        return hostRoot(url) + path
    }

    /** `scheme://host[:port]` of a normalised base URL, as browsers send it in `Origin`. */
    fun origin(baseUrl: String): String = hostRoot(parse(baseUrl))

    /** True for `http://` URLs (tokens travel unencrypted). */
    fun isCleartext(baseUrl: String): Boolean = baseUrl.startsWith("http://", ignoreCase = true)

    /**
     * `http://` + the `Host` header OkHttp sends for [baseUrl]. The server's origin guard accepts this
     * value because it builds the request origin from `"http://" + Host` (it never sees TLS).
     */
    internal fun hostHeaderOrigin(baseUrl: String): String {
        val url = parse(baseUrl)
        val host = if (url.host.contains(':')) "[${url.host}]" else url.host
        val port = if (url.port == HttpUrl.defaultPort(url.scheme)) "" else ":${url.port}"
        val origin = "http://$host$port"
        return if (origin.endsWith(":80")) origin.dropLast(3) else origin
    }

    private fun parse(baseUrl: String): HttpUrl =
        baseUrl.toHttpUrlOrNull() ?: throw IllegalArgumentException("not a valid URL: $baseUrl")

    private fun hostRoot(url: HttpUrl): String =
        HttpUrl.Builder().scheme(url.scheme).host(url.host).port(url.port).build().toString().trimEnd('/')
}
