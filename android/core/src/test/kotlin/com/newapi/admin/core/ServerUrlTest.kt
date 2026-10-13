package com.newapi.admin.core

import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test

class ServerUrlTest {
    @Test
    fun normalizeProducesTheBaseUrlTheClientAppendsApiPathsTo() {
        val cases = listOf(
            "example.com/" to "https://example.com",
            "  example.com  " to "https://example.com",
            "http://10.0.0.2:3000" to "http://10.0.0.2:3000",
            "https://Example.com:443/newapi/" to "https://example.com/newapi",
            "HTTP://EXAMPLE.COM:80/" to "http://example.com",
            "https://example.com:8443" to "https://example.com:8443",
            "https://example.com/api" to "https://example.com",
            "https://example.com/api/" to "https://example.com",
            "https://example.com/newapi/api" to "https://example.com/newapi",
            "https://example.com/apis" to "https://example.com/apis",
            "https://example.com/?tab=1#top" to "https://example.com",
            "localhost:3000" to "https://localhost:3000",
            "http://[::1]:3000/" to "http://[::1]:3000",
        )
        for ((input, expected) in cases) {
            assertEquals(input, expected, ServerUrl.normalize(input))
        }
    }

    @Test
    fun normalizeRejectsUnusableInput() {
        for (input in listOf("", "   ", "ftp://example.com", "http://", "exa mple.com", "https://user:pw@example.com")) {
            assertThrows(input, IllegalArgumentException::class.java) { ServerUrl.normalize(input) }
        }
    }

    @Test
    fun originIsSchemeHostAndNonDefaultPort() {
        assertEquals("https://example.com", ServerUrl.origin("https://example.com/newapi"))
        assertEquals("http://10.0.0.2:3000", ServerUrl.origin("http://10.0.0.2:3000"))
        assertEquals("http://[::1]:3000", ServerUrl.origin("http://[::1]:3000"))
    }

    @Test
    fun hostHeaderOriginIsWhatTheOriginGuardBuildsBehindATlsProxy() {
        assertEquals("http://example.com", ServerUrl.hostHeaderOrigin("https://example.com/newapi"))
        assertEquals("http://example.com:8443", ServerUrl.hostHeaderOrigin("https://example.com:8443"))
        assertEquals("http://10.0.0.2:3000", ServerUrl.hostHeaderOrigin("http://10.0.0.2:3000"))
    }
}
