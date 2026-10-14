package com.newapi.admin.core.model

import org.junit.Assert.assertEquals
import org.junit.Test

class TicketImageTest {

    private fun bytes(vararg values: Int) = ByteArray(values.size) { values[it].toByte() }

    private fun latin1(text: String) = text.toByteArray(Charsets.ISO_8859_1)

    @Test
    fun detectsOnlyTheFormatsTheServerAccepts() {
        val cases = listOf(
            bytes(0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0) to "image/png",
            bytes(0xFF, 0xD8, 0xFF, 0xE0) to "image/jpeg",
            latin1("GIF89a....") to "image/gif",
            latin1("GIF87a") to "image/gif",
            latin1("RIFF\u0010\u0000\u0000\u0000WEBPVP8 ") to "image/webp",
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
