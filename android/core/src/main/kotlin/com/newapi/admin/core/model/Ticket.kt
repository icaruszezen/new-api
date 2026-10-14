package com.newapi.admin.core.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** `model.Ticket`, a user's support ticket (`/api/ticket/admin/...`). */
@Serializable
data class Ticket(
    val id: Int = 0,
    /** The user who opened the ticket. */
    @SerialName("user_id") val userId: Int = 0,
    val subject: String = "",
    /** [TicketStatus]. */
    val status: Int = 0,
    /** [TicketRole] of the latest message. */
    @SerialName("last_reply_role") val lastReplyRole: String = "",
    @SerialName("message_count") val messageCount: Int = 0,
    /** The user has not read the latest admin reply or status change. */
    @SerialName("user_unread") val userUnread: Boolean = false,
    /** No admin has opened the ticket since the user wrote; opening its detail clears it for every admin. */
    @SerialName("admin_unread") val adminUnread: Boolean = false,
    /** Unix seconds. */
    @SerialName("created_time") val createdTime: Long = 0,
    /** Unix seconds; the list is sorted by it, newest first. */
    @SerialName("updated_time") val updatedTime: Long = 0,
    /** Unix seconds; 0 unless closed. */
    @SerialName("closed_time") val closedTime: Long = 0,
    /** The owner's username; empty in a status change response or for a deleted user. */
    val username: String = "",
)

/** `model.TicketMessage`. [content] is plain text (the server strips control characters). */
@Serializable
data class TicketMessage(
    val id: Int = 0,
    @SerialName("ticket_id") val ticketId: Int = 0,
    /** [TicketRole]. */
    @SerialName("author_role") val authorRole: String = "",
    val content: String = "",
    /** Unix seconds. */
    @SerialName("created_time") val createdTime: Long = 0,
    /** The author's username; empty for a deleted user. */
    @SerialName("author_name") val authorName: String = "",
    val attachments: List<TicketAttachment> = emptyList(),
) {
    val isFromAdmin: Boolean get() = authorRole == TicketRole.ADMIN
}

/** Metadata of an image the server re-encoded; the bytes come from [com.newapi.admin.core.api.AdminApi.ticketAttachment]. */
@Serializable
data class TicketAttachment(
    val id: Int = 0,
    @SerialName("ticket_id") val ticketId: Int = 0,
    @SerialName("message_id") val messageId: Int = 0,
    /** `image/png` or `image/jpeg`. */
    @SerialName("mime_type") val mimeType: String = "",
    /** Stored bytes. */
    val size: Long = 0,
    val width: Int = 0,
    val height: Int = 0,
    @SerialName("created_time") val createdTime: Long = 0,
)

/** `model.TicketUser`, the owner shown on the admin detail; only [id] is set when the user row is gone. */
@Serializable
data class TicketUser(
    val id: Int = 0,
    val username: String = "",
    @SerialName("display_name") val displayName: String = "",
    val email: String = "",
    val group: String = "",
)

/** `model.AdminTicketDetail`: every message (at most [TicketLimits.MAX_MESSAGES_PER_TICKET]), oldest first. */
@Serializable
data class TicketDetail(
    val ticket: Ticket = Ticket(),
    val messages: List<TicketMessage> = emptyList(),
    val user: TicketUser = TicketUser(),
) {
    val attachmentCount: Int get() = messages.sumOf { it.attachments.size }
}

/** `model.TicketStats`: tickets per status, and [unread] tickets no admin has opened since the user wrote. */
@Serializable
data class TicketStats(
    val pending: Long = 0,
    val processing: Long = 0,
    val resolved: Long = 0,
    val closed: Long = 0,
    val unread: Long = 0,
) {
    val total: Long get() = pending + processing + resolved + closed

    fun count(status: Int): Long = when (status) {
        TicketStatus.PENDING -> pending
        TicketStatus.PROCESSING -> processing
        TicketStatus.RESOLVED -> resolved
        TicketStatus.CLOSED -> closed
        else -> 0
    }
}

/** An image for a reply: bytes the server accepts (PNG, JPEG, GIF or WebP, at most [TicketLimits.MAX_IMAGE_BYTES]). */
class TicketImage(val bytes: ByteArray, val mimeType: String) {
    companion object {
        /**
         * The upload type the server detects in [bytes] (Go's `http.DetectContentType` signatures), or null for a
         * format it rejects, such as HEIC, AVIF or BMP. The server ignores the declared type and file name.
         */
        fun detectMimeType(bytes: ByteArray): String? = when {
            bytes.matches(0, "\u0089PNG\r\n\u001A\n") -> "image/png"
            bytes.matches(0, "\u00FF\u00D8\u00FF") -> "image/jpeg"
            bytes.matches(0, "GIF87a") || bytes.matches(0, "GIF89a") -> "image/gif"
            bytes.matches(0, "RIFF") && bytes.matches(8, "WEBPVP") -> "image/webp"
            else -> null
        }

        /** Whether [signature], one byte per Latin-1 character, appears at [offset]. */
        private fun ByteArray.matches(offset: Int, signature: String): Boolean =
            size >= offset + signature.length && signature.indices.all { this[offset + it] == signature[it].code.toByte() }
    }
}

object TicketStatus {
    /** New, or the user wrote last. */
    const val PENDING = 1
    const val PROCESSING = 2
    const val RESOLVED = 3

    /** The user can no longer reply; an admin can still reply or reopen it. */
    const val CLOSED = 4

    val all: List<Int> = listOf(PENDING, PROCESSING, RESOLVED, CLOSED)
}

object TicketRole {
    const val USER = "user"
    const val ADMIN = "admin"
}

/** `model/ticket.go` and `service/ticket_image.go` limits that apply to admin replies. */
object TicketLimits {
    /** Runes, counted after the server strips control characters and surrounding whitespace. */
    const val MAX_CONTENT_RUNES = 5000
    const val MAX_IMAGES_PER_MESSAGE = 4
    const val MAX_MESSAGES_PER_TICKET = 200
    const val MAX_ATTACHMENTS_PER_TICKET = 40

    /** Per uploaded file, before the server re-encodes it. */
    const val MAX_IMAGE_BYTES = 8 * 1024 * 1024

    /** Checked before decoding: each side and the pixel count. */
    const val MAX_IMAGE_SIDE = 12_000
    const val MAX_IMAGE_PIXELS = 25_000_000L

    /** The server scales stored images down to this long side, so larger uploads only cost bandwidth. */
    const val STORED_IMAGE_SIDE = 3072

    /** A stored image is at most 3 MiB. */
    const val MAX_STORED_IMAGE_BYTES = 3 * 1024 * 1024

    val UPLOAD_MIME_TYPES: Set<String> = setOf("image/png", "image/jpeg", "image/gif", "image/webp")
}
