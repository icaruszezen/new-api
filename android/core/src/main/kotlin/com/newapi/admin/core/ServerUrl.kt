package com.newapi.admin.core

object ServerUrl {
    /**
     * Turns user input into a base URL without a trailing slash: adds `https://` when no scheme is given,
     * lowercases the host, drops a default port, keeps a sub-path (for reverse proxies), removes a
     * trailing `/api` or `/`. Throws IllegalArgumentException when the input is not an http(s) URL.
     *
     * Examples: "example.com/" -> "https://example.com", "http://10.0.0.2:3000" -> "http://10.0.0.2:3000",
     * "https://Example.com:443/newapi/" -> "https://example.com/newapi".
     */
    fun normalize(input: String): String = TODO()

    /** `scheme://host[:port]` of a normalised base URL, as browsers send it in `Origin`. */
    fun origin(baseUrl: String): String = TODO()

    /** True for `http://` URLs (tokens travel unencrypted). */
    fun isCleartext(baseUrl: String): Boolean = baseUrl.startsWith("http://", ignoreCase = true)
}
