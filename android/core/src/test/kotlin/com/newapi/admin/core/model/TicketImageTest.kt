package com.newapi.admin.core.model

import org.junit.Assert.assertEquals
import org.junit.Test

class TicketImageTest {

    private fun bytes(vararg values: Int) = ByteArray(values.size) { values[it].toByte() }

    private fun latin1(text: String) = text.toByteArray(Charsets.ISO_8859_1)

    /** SOI, a 4-byte APP0 segment, then a 16x16 frame header of type [sof] with [components]. */
    private fun jpeg(sof: Int, components: Int) =
        bytes(0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x04, 0x00, 0x00, 0xFF, sof, 0x00, 8 + 3 * components, 0x08, 0x00, 0x10, 0x00, 0x10, components)

    @Test
    fun detectsOnlyTheFormatsTheServerAccepts() {
        val cases = listOf(
            bytes(0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0) to "image/png",
            bytes(0xFF, 0xD8, 0xFF, 0xE0) to "image/jpeg",
            latin1("GIF89a....") to "image/gif",
            latin1("GIF87a") to "image/gif",
            latin1("RIFF\u0010\u0000\u0000\u0000WEBPVP8 ") to "image/webp",
            // Extended WebP: alpha (bit 4) is a still image, animation (bit 1) is not decoded by the server.
            latin1("RIFF\u0010\u0000\u0000\u0000WEBPVP8X\u000A\u0000\u0000\u0000\u0010") to "image/webp",
            latin1("RIFF\u0010\u0000\u0000\u0000WEBPVP8X\u000A\u0000\u0000\u0000\u0012") to null,
            // JPEG frame headers after an APP0 segment: 3 components (YCbCr) is accepted, 4 (CMYK) is rejected,
            // for baseline (SOF0) and progressive (SOF2) alike.
            jpeg(0xC0, components = 3) to "image/jpeg",
            jpeg(0xC2, components = 3) to "image/jpeg",
            jpeg(0xC0, components = 4) to null,
            jpeg(0xC2, components = 4) to null,
            // RIFF without the WebP marker is a WAV or AVI file.
            latin1("RIFF\u0010\u0000\u0000\u0000WAVEfmt ") to null,
            // HEIC and AVIF (ISO base media), BMP and a truncated PNG signature are rejected.
            latin1("\u0000\u0000\u0000\u0018ftypheic") to null,
            latin1("\u0000\u0000\u0000\u001CftypavifX") to null,
            latin1("BM6\u0000") to null,
            bytes(0x89, 0x50, 0x4E, 0x47) to null,
            ByteArray(0) to null,
        )
        for ((input, expected) in cases) {
            assertEquals(input.joinToString(" ") { "%02x".format(it) }, expected, TicketImage.detectMimeType(input))
        }
    }
}
