package com.newapi.admin.ui.tickets

import android.content.Context
import android.net.Uri
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.graphics.ImageBitmap
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.newapi.admin.AppContainer
import com.newapi.admin.R
import com.newapi.admin.core.model.TicketDetail
import com.newapi.admin.core.model.TicketImage
import com.newapi.admin.core.model.TicketLimits
import com.newapi.admin.core.net.ApiError
import com.newapi.admin.ui.common.UiMessage
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

data class TicketDetailState(
    val detail: TicketDetail? = null,
    val isLoading: Boolean = true,
    /** Pull-to-refresh with the ticket already shown. */
    val isRefreshing: Boolean = false,
    /** Failure of the last load; shown full screen when no ticket is loaded yet. */
    val error: Throwable? = null,
    val isSending: Boolean = false,
    val isChangingStatus: Boolean = false,
)

/** A picked image waiting to be sent, with a small [preview] (null if it could not be decoded for display). */
class PendingImage(val key: Long, val image: TicketImage, val preview: ImageBitmap?)

/** A message attachment's thumbnail. */
sealed interface Thumbnail {
    data object Loading : Thumbnail

    data object Failed : Thumbnail

    class Loaded(val image: ImageBitmap) : Thumbnail
}

class TicketDetailViewModel(private val container: AppContainer, private val ticketId: Int) : ViewModel() {
    private val api = container.api

    private val mutableState = MutableStateFlow(TicketDetailState())
    val state: StateFlow<TicketDetailState> = mutableState.asStateFlow()

    private val messageChannel = Channel<UiMessage>(Channel.BUFFERED)
    val messages: Flow<UiMessage> = messageChannel.receiveAsFlow()

    // The reply draft is Compose state so typing stays synchronous with the text field.
    var reply by mutableStateOf("")
        private set

    /** The status the reply sets; null lets the server decide (awaiting support becomes in progress). */
    var replyStatus by mutableStateOf<Int?>(null)
        private set

    val pendingImages = mutableStateListOf<PendingImage>()

    /** Picked images still being read or converted. */
    var preparingImages by mutableIntStateOf(0)
        private set

    /** By attachment id; filled as messages scroll into view. */
    val thumbnails = mutableStateMapOf<Int, Thumbnail>()

    private var nextImageKey = 0L
    private var loadJob: Job? = null

    // Decoding takes memory and CPU; a few at a time is enough for a screenful of thumbnails.
    private val decodeDispatcher = Dispatchers.Default.limitedParallelism(2)

    val hasDraft: Boolean get() = reply.isNotBlank() || pendingImages.isNotEmpty()

    /** The trimmed reply in Unicode code points, as the server counts its limit. */
    val replyLength: Int get() = reply.trim().let { it.codePointCount(0, it.length) }

    init {
        load(showIndicator = false)
    }

    /** Pull-to-refresh; skipped while a reply or status change runs, whose own answer updates the ticket. */
    fun refresh() {
        val current = mutableState.value
        if (current.isSending || current.isChangingStatus) return
        load(showIndicator = true)
    }

    private fun load(showIndicator: Boolean) {
        loadJob?.cancel()
        mutableState.update {
            it.copy(isLoading = it.detail == null, isRefreshing = showIndicator && it.detail != null, error = null)
        }
        loadJob = viewModelScope.launch {
            try {
                val detail = api.ticket(ticketId)
                mutableState.update { it.copy(detail = detail, isLoading = false, isRefreshing = false) }
                // Opening the ticket marked it read for every admin.
                container.events.notifyTicketsChanged(detail.ticket)
                container.refreshTicketStats()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isLoading = false, isRefreshing = false, error = e) }
                if (mutableState.value.detail != null) messageChannel.send(UiMessage.Error(e))
            }
        }
    }

    /** False once the ticket holds as many messages as the server allows. */
    fun acceptsReplies(detail: TicketDetail): Boolean =
        maxOf(detail.ticket.messageCount, detail.messages.size) < TicketLimits.MAX_MESSAGES_PER_TICKET

    /** How many more images this reply can carry: at most four per message and forty per ticket. */
    fun imageRoom(detail: TicketDetail): Int {
        val used = pendingImages.size + preparingImages
        val perMessage = TicketLimits.MAX_IMAGES_PER_MESSAGE - used
        val perTicket = TicketLimits.MAX_ATTACHMENTS_PER_TICKET - detail.attachmentCount - used
        return minOf(perMessage, perTicket).coerceAtLeast(0)
    }

    fun onReplyChange(value: String) {
        reply = value
    }

    fun onReplyStatusChange(status: Int?) {
        replyStatus = status
    }

    /** Reads and, where needed, converts picked images; the ones beyond [imageRoom] are dropped with a warning. */
    fun addImages(context: Context, uris: List<Uri>) {
        val detail = mutableState.value.detail ?: return
        val room = imageRoom(detail)
        if (uris.size > room) {
            val limits = listOf(TicketLimits.MAX_IMAGES_PER_MESSAGE, TicketLimits.MAX_ATTACHMENTS_PER_TICKET)
            messageChannel.trySend(UiMessage.Res(R.string.tickets_error_too_many_images, limits))
        }
        val accepted = uris.take(room)
        if (accepted.isEmpty()) return
        val appContext = context.applicationContext
        preparingImages += accepted.size
        viewModelScope.launch {
            for (uri in accepted) {
                try {
                    val image = prepareTicketUpload(appContext, uri)
                    val preview = withContext(decodeDispatcher) { decodeTicketThumbnail(image.bytes, PREVIEW_SIDE) }
                    pendingImages.add(PendingImage(nextImageKey++, image, preview))
                } catch (e: CancellationException) {
                    throw e
                } catch (e: TicketUploadException) {
                    val text = when (e.problem) {
                        UploadProblem.UNREADABLE -> R.string.tickets_error_image_unreadable
                        UploadProblem.UNSUPPORTED -> R.string.tickets_error_image_unsupported
                        UploadProblem.TOO_LARGE -> R.string.tickets_error_image_too_large
                    }
                    messageChannel.send(UiMessage.Res(text))
                } catch (e: Throwable) {
                    messageChannel.send(UiMessage.Res(R.string.tickets_error_image_unreadable))
                } finally {
                    preparingImages -= 1
                }
            }
        }
    }

    fun removeImage(key: Long) {
        if (mutableState.value.isSending) return
        pendingImages.removeAll { it.key == key }
    }

    fun send() {
        val current = mutableState.value
        if (current.detail == null || current.isSending || current.isChangingStatus || preparingImages > 0 || !hasDraft) return
        if (replyLength > TicketLimits.MAX_CONTENT_RUNES) {
            messageChannel.trySend(UiMessage.Res(R.string.tickets_error_too_long, listOf(TicketLimits.MAX_CONTENT_RUNES)))
            return
        }
        val content = reply.trim()
        val images = pendingImages.map { it.image }
        val status = replyStatus
        mutableState.update { it.copy(isSending = true) }
        viewModelScope.launch {
            try {
                val message = api.replyTicket(ticketId, content, images, status)
                reply = ""
                pendingImages.clear()
                replyStatus = null
                // The answer is only the new message: show it now and reload for the status the server chose. A load
                // that finished during the send may already hold it.
                mutableState.update { state ->
                    val shown = state.detail
                    if (shown == null || shown.messages.any { it.id == message.id }) return@update state
                    state.copy(
                        detail = shown.copy(
                            ticket = shown.ticket.copy(messageCount = shown.ticket.messageCount + 1),
                            messages = shown.messages + message,
                        ),
                    )
                }
                messageChannel.send(UiMessage.Res(R.string.tickets_msg_reply_sent))
                load(showIndicator = false)
            } catch (e: CancellationException) {
                throw e
            } catch (e: ApiError.Http) {
                // A reverse proxy in front of the server rejects a large multipart body with 413.
                messageChannel.send(if (e.status == 413) UiMessage.Res(R.string.tickets_error_upload_too_large) else UiMessage.Error(e))
            } catch (e: Throwable) {
                messageChannel.send(UiMessage.Error(e))
            } finally {
                mutableState.update { it.copy(isSending = false) }
            }
        }
    }

    fun setStatus(status: Int) {
        val current = mutableState.value
        val detail = current.detail ?: return
        if (current.isChangingStatus || current.isSending || status == detail.ticket.status) return
        // A load still running could answer with the old status after this change.
        loadJob?.cancel()
        mutableState.update { it.copy(isChangingStatus = true, isRefreshing = false) }
        viewModelScope.launch {
            try {
                val saved = api.setTicketStatus(ticketId, status)
                var changed = saved
                mutableState.update { state ->
                    val shown = state.detail ?: return@update state
                    // The answer has no username.
                    changed = saved.copy(username = saved.username.ifEmpty { shown.ticket.username })
                    state.copy(detail = shown.copy(ticket = changed))
                }
                messageChannel.send(UiMessage.Res(R.string.tickets_msg_status_updated))
                container.events.notifyTicketsChanged(changed)
                container.refreshTicketStats()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                messageChannel.send(UiMessage.Error(e))
            } finally {
                mutableState.update { it.copy(isChangingStatus = false) }
            }
        }
    }

    /** Loads an attachment's thumbnail once, or again after it failed. */
    fun loadThumbnail(attachmentId: Int) {
        val current = thumbnails[attachmentId]
        if (current != null && current != Thumbnail.Failed) return
        thumbnails[attachmentId] = Thumbnail.Loading
        viewModelScope.launch {
            val result = try {
                val bytes = api.ticketAttachment(ticketId, attachmentId)
                withContext(decodeDispatcher) { decodeTicketThumbnail(bytes, THUMBNAIL_SIDE) }?.let { Thumbnail.Loaded(it) } ?: Thumbnail.Failed
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                Thumbnail.Failed
            }
            thumbnails[attachmentId] = result
        }
    }

    /** The attachment at full size for the viewer; downloaded again each time, since full images are large. */
    suspend fun fullImage(attachmentId: Int): ImageBitmap? {
        val bytes = api.ticketAttachment(ticketId, attachmentId)
        return withContext(decodeDispatcher) { decodeTicketImage(bytes, FULL_SIDE) }
    }

    private companion object {
        /** Square sides for 96dp thumbnails and 64dp previews on dense screens; stored images show whole. */
        const val THUMBNAIL_SIDE = 320
        const val PREVIEW_SIDE = 224
        const val FULL_SIDE = TicketLimits.STORED_IMAGE_SIDE
    }
}
