package com.newapi.admin.ui.tickets

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.gestures.detectTransformGestures
import androidx.compose.foundation.gestures.scrollBy
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.ime
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.union
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.automirrored.outlined.Send
import androidx.compose.material.icons.outlined.AddPhotoAlternate
import androidx.compose.material.icons.outlined.BrokenImage
import androidx.compose.material.icons.outlined.Check
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.KeyboardArrowDown
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material.icons.outlined.SwapHoriz
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.IconButtonDefaults
import androidx.compose.material3.ListItem
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.format.Formats
import com.newapi.admin.core.model.Ticket
import com.newapi.admin.core.model.TicketAttachment
import com.newapi.admin.core.model.TicketDetail
import com.newapi.admin.core.model.TicketLimits
import com.newapi.admin.core.model.TicketMessage
import com.newapi.admin.core.model.TicketStatus
import com.newapi.admin.core.model.TicketUser
import com.newapi.admin.ui.common.UiMessagesEffect
import com.newapi.admin.ui.components.AppBottomSheet
import com.newapi.admin.ui.components.AppButton
import com.newapi.admin.ui.components.AppIconButton
import com.newapi.admin.ui.components.AppSnackbarHost
import com.newapi.admin.ui.components.AppTextField
import com.newapi.admin.ui.components.AppTopBar
import com.newapi.admin.ui.components.ButtonSize
import com.newapi.admin.ui.components.ButtonVariant
import com.newapi.admin.ui.components.ConfirmDialog
import com.newapi.admin.ui.components.FullScreenError
import com.newapi.admin.ui.components.FullScreenLoading
import com.newapi.admin.ui.components.InfoRow
import com.newapi.admin.ui.components.SectionCard
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.theme.Tone
import kotlinx.coroutines.CancellationException

/** Header and submitter come before the messages in the list. */
private const val ITEMS_BEFORE_MESSAGES = 2

/** The reply counter shows from here on, so short replies stay uncluttered. */
private const val COUNTER_FROM = 4000

/** One ticket: its status, who sent it, every message with its images, and the reply box. */
@Composable
fun TicketDetailScreen(ticketId: Int, onBack: () -> Unit, onOpenUser: (Int) -> Unit) {
    val container = LocalAppContainer.current
    val viewModel = viewModel(key = "ticket-$ticketId") { TicketDetailViewModel(container, ticketId) }
    val state by viewModel.state.collectAsStateWithLifecycle()
    val snackbarHostState = remember { SnackbarHostState() }
    val listState = rememberLazyListState()
    var confirmDiscard by rememberSaveable { mutableStateOf(false) }
    var showStatusSheet by rememberSaveable { mutableStateOf(false) }
    // The attachment open in the viewer, by id so it survives a rotation.
    var viewingId by rememberSaveable { mutableStateOf<Int?>(null) }
    // Messages already scrolled to, so a rotation does not jump back to the newest one.
    var scrolledToMessages by rememberSaveable { mutableIntStateOf(0) }

    val guardBack = viewModel.hasDraft
    val requestBack: () -> Unit = {
        if (guardBack) confirmDiscard = true else onBack()
    }
    UiMessagesEffect(viewModel.messages, snackbarHostState)
    // Leaving during a send or a status change would cancel it before the answer arrives, although the server may
    // already apply it.
    val busy = state.isSending || state.isChangingStatus
    BackHandler(enabled = guardBack || busy) {
        if (!busy) confirmDiscard = true
    }

    val detail = state.detail
    val messageCount = detail?.messages?.size ?: 0
    LaunchedEffect(messageCount) {
        // The newest message first: on opening, after a reply, and when a reload brings new ones.
        if (messageCount > scrolledToMessages) {
            listState.scrollToItem(ITEMS_BEFORE_MESSAGES + messageCount - 1)
            scrolledToMessages = messageCount
        }
    }
    // When the keyboard or a growing reply box shortens the thread, its end stays in view if it was.
    LaunchedEffect(listState) {
        var lastHeight = 0
        var wasAtEnd = false
        snapshotFlow { listState.layoutInfo.viewportSize.height to !listState.canScrollForward }
            .collect { (height, atEnd) ->
                if (wasAtEnd && lastHeight > 0 && height < lastHeight) listState.scrollBy((lastHeight - height).toFloat())
                lastHeight = height
                wasAtEnd = atEnd
            }
    }

    Box(Modifier.fillMaxSize()) {
        Scaffold(
            // The image viewer covers the screen; what is under it should not be read out either.
            modifier = if (viewingId != null) Modifier.clearAndSetSemantics {} else Modifier,
            topBar = {
                AppTopBar(
                    title = detail?.ticket?.subject ?: stringResource(R.string.tickets_detail_title, ticketId),
                    subtitle = detail?.let { listOfNotNull("#${it.ticket.id}", it.ticket.username.ifEmpty { null }).joinToString(" · ") },
                    navigationIcon = {
                        IconButton(onClick = requestBack, enabled = !busy) {
                            Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = stringResource(R.string.common_back))
                        }
                    },
                )
            },
            bottomBar = {
                if (detail != null) ReplyComposer(viewModel, detail, state.isSending, state.isChangingStatus)
            },
            snackbarHost = { AppSnackbarHost(snackbarHostState) },
        ) { padding ->
            PullToRefreshBox(
                isRefreshing = state.isRefreshing,
                onRefresh = viewModel::refresh,
                modifier = Modifier.fillMaxSize().padding(padding),
            ) {
                val error = state.error
                when {
                    detail != null -> LazyColumn(
                        state = listState,
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(16.dp),
                        verticalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        item(key = "header") {
                            TicketHeader(detail.ticket, state.isChangingStatus, enabled = !state.isSending, onChangeStatus = { showStatusSheet = true })
                        }
                        item(key = "user") {
                            SubmitterCard(detail.user, onOpenUser)
                        }
                        items(detail.messages, key = { it.id }) { message ->
                            MessageBubble(
                                message,
                                thumbnails = viewModel.thumbnails,
                                onLoadThumbnail = viewModel::loadThumbnail,
                                onView = { viewingId = it.id },
                            )
                        }
                    }
                    state.isLoading -> FullScreenLoading()
                    error != null -> FullScreenError(error, onRetry = viewModel::refresh)
                }
            }
        }
        viewingId?.let { id ->
            ImageViewer(
                attachmentId = id,
                load = { viewModel.fullImage(id) },
                onDismiss = { viewingId = null },
            )
        }
    }

    if (showStatusSheet && detail != null) {
        StatusSheet(
            current = detail.ticket.status,
            onSelect = {
                showStatusSheet = false
                viewModel.setStatus(it)
            },
            onDismiss = { showStatusSheet = false },
        )
    }

    if (confirmDiscard) {
        ConfirmDialog(
            title = stringResource(R.string.tickets_discard_title),
            text = stringResource(R.string.tickets_discard_message),
            confirmText = stringResource(R.string.tickets_discard_confirm),
            destructive = true,
            onConfirm = {
                confirmDiscard = false
                onBack()
            },
            onDismiss = { confirmDiscard = false },
        )
    }
}

@Composable
private fun TicketHeader(ticket: Ticket, isChangingStatus: Boolean, enabled: Boolean, onChangeStatus: () -> Unit) {
    val colors = AppTheme.colors
    SectionCard(title = null) {
        Row(verticalAlignment = Alignment.Top) {
            SelectionContainer(Modifier.weight(1f)) {
                Text(ticket.subject, style = MaterialTheme.typography.titleMedium)
            }
            Spacer(Modifier.width(8.dp))
            TicketStatusPill(ticket.status)
        }
        Spacer(Modifier.height(8.dp))
        val times = listOfNotNull(
            stringResource(R.string.tickets_created_at, Formats.dateTime(ticket.createdTime)),
            stringResource(R.string.tickets_updated_at, Formats.dateTime(ticket.updatedTime)),
            if (ticket.status == TicketStatus.CLOSED && ticket.closedTime > 0) {
                stringResource(R.string.tickets_closed_at, Formats.dateTime(ticket.closedTime))
            } else {
                null
            },
        )
        times.forEach { line ->
            Text(
                line,
                style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                color = colors.mutedForeground,
            )
        }
        Spacer(Modifier.height(12.dp))
        // loading = isChangingStatus also keeps the button disabled while the change runs.
        AppButton(
            stringResource(R.string.tickets_change_status),
            onClick = onChangeStatus,
            variant = ButtonVariant.Outline,
            size = ButtonSize.Small,
            enabled = enabled,
            loading = isChangingStatus,
            leadingIcon = Icons.Outlined.SwapHoriz,
        )
    }
}

@Composable
private fun SubmitterCard(user: TicketUser, onOpenUser: (Int) -> Unit) {
    SectionCard(stringResource(R.string.tickets_section_user)) {
        if (user.username.isEmpty()) {
            // The account was removed for good; only its id is left.
            Text(
                stringResource(R.string.tickets_user_deleted),
                style = MaterialTheme.typography.bodyMedium,
                color = AppTheme.colors.mutedForeground,
                modifier = Modifier.padding(vertical = 4.dp),
            )
            InfoRow(stringResource(R.string.tickets_field_user_id), user.id.toString(), copyable = true)
            return@SectionCard
        }
        InfoRow(stringResource(R.string.tickets_field_username), user.username, copyable = true)
        InfoRow(stringResource(R.string.tickets_field_display_name), user.displayName)
        InfoRow(stringResource(R.string.tickets_field_email), user.email, copyable = true)
        InfoRow(stringResource(R.string.tickets_field_group), user.group)
        InfoRow(stringResource(R.string.tickets_field_user_id), user.id.toString(), copyable = true)
        Spacer(Modifier.height(8.dp))
        AppButton(
            stringResource(R.string.tickets_open_user),
            onClick = { onOpenUser(user.id) },
            variant = ButtonVariant.Outline,
            size = ButtonSize.Small,
            leadingIcon = Icons.Outlined.Person,
        )
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun MessageBubble(
    message: TicketMessage,
    thumbnails: Map<Int, Thumbnail>,
    onLoadThumbnail: (Int) -> Unit,
    onView: (TicketAttachment) -> Unit,
) {
    val colors = AppTheme.colors
    val isSupport = message.isFromAdmin
    val shape = MaterialTheme.shapes.large
    val accent = colors.primary.copy(alpha = 0.5f)
    // Like the web: support replies sit inset from the left with an accent edge, the user's from the right.
    Column(
        Modifier
            .fillMaxWidth()
            .padding(start = if (isSupport) 24.dp else 0.dp, end = if (isSupport) 0.dp else 24.dp)
            .clip(shape)
            .background(if (isSupport) colors.muted else colors.card)
            .border(1.dp, colors.border, shape)
            .drawBehind {
                if (isSupport) drawRect(accent, size = size.copy(width = 4.dp.toPx()))
            }
            .padding(start = if (isSupport) 18.dp else 14.dp, end = 14.dp, top = 12.dp, bottom = 12.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            // A long name shortens before the support pill or the date do.
            Row(Modifier.weight(1f), verticalAlignment = Alignment.CenterVertically) {
                val author = message.authorName.ifEmpty {
                    stringResource(if (isSupport) R.string.tickets_author_admin else R.string.tickets_author_user)
                }
                Text(
                    author,
                    style = MaterialTheme.typography.labelLarge,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f, fill = false),
                )
                if (isSupport) {
                    Spacer(Modifier.width(6.dp))
                    StatusPill(stringResource(R.string.tickets_author_support), Tone.Neutral, outlined = true)
                }
            }
            Spacer(Modifier.width(8.dp))
            Text(
                Formats.dateTime(message.createdTime),
                style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                color = colors.mutedForeground,
                maxLines = 1,
                softWrap = false,
            )
        }
        if (message.content.isNotEmpty()) {
            Spacer(Modifier.height(6.dp))
            // Plain text only, as the server stores it.
            SelectionContainer {
                Text(message.content, style = MaterialTheme.typography.bodyMedium)
            }
        }
        if (message.attachments.isNotEmpty()) {
            Spacer(Modifier.height(10.dp))
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                message.attachments.forEachIndexed { index, attachment ->
                    AttachmentThumbnail(
                        index = index + 1,
                        thumbnail = thumbnails[attachment.id],
                        onLoad = { onLoadThumbnail(attachment.id) },
                        onView = { onView(attachment) },
                    )
                }
            }
        }
    }
}

@Composable
private fun AttachmentThumbnail(index: Int, thumbnail: Thumbnail?, onLoad: () -> Unit, onView: () -> Unit) {
    val colors = AppTheme.colors
    LaunchedEffect(Unit) { onLoad() }
    val shape = MaterialTheme.shapes.medium
    val box = Modifier
        .size(96.dp)
        .clip(shape)
        .background(colors.background)
        .border(1.dp, colors.border, shape)
    when (thumbnail) {
        // Top-aligned: most attachments are screenshots, whose top says what they show.
        is Thumbnail.Loaded -> Image(
            thumbnail.image,
            contentDescription = stringResource(R.string.tickets_view_attachment, index),
            contentScale = ContentScale.Crop,
            alignment = Alignment.TopCenter,
            modifier = box.clickable(role = Role.Button, onClick = onView),
        )
        Thumbnail.Failed -> Box(box.clickable(role = Role.Button, onClick = onLoad), contentAlignment = Alignment.Center) {
            Icon(
                Icons.Outlined.BrokenImage,
                contentDescription = stringResource(R.string.tickets_attachment_failed, index),
                tint = colors.mutedForeground,
                modifier = Modifier.size(24.dp),
            )
        }
        else -> Box(box, contentAlignment = Alignment.Center) {
            CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp, color = colors.mutedForeground)
        }
    }
}

@Composable
private fun ReplyComposer(viewModel: TicketDetailViewModel, detail: TicketDetail, isSending: Boolean, isChangingStatus: Boolean) {
    val colors = AppTheme.colors
    val context = LocalContext.current
    val room = viewModel.imageRoom(detail)
    val pickImages = rememberTicketImagePicker(room) { uris -> viewModel.addImages(context, uris) }
    Surface(color = colors.background, contentColor = colors.foreground) {
        BoxWithConstraints(
            Modifier
                .fillMaxWidth()
                .drawBehind { drawLine(colors.border, Offset(0f, 0f), Offset(size.width, 0f), strokeWidth = 1.dp.toPx()) }
                .windowInsetsPadding(WindowInsets.navigationBars.union(WindowInsets.ime)),
        ) {
            // At most half the space above the keyboard, so the thread stays in view; the buttons always show.
            Column(
                Modifier.fillMaxWidth().heightIn(max = maxHeight / 2).padding(horizontal = 16.dp, vertical = 10.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                if (!viewModel.acceptsReplies(detail)) {
                    ComposerNote(stringResource(R.string.tickets_message_limit, TicketLimits.MAX_MESSAGES_PER_TICKET))
                    return@Column
                }
                val length = viewModel.replyLength
                val tooLong = length > TicketLimits.MAX_CONTENT_RUNES
                Column(
                    Modifier.weight(1f, fill = false).verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    if (detail.ticket.status == TicketStatus.CLOSED && viewModel.replyStatus == null) {
                        ComposerNote(stringResource(R.string.tickets_closed_note))
                    }
                    if (detail.attachmentCount >= TicketLimits.MAX_ATTACHMENTS_PER_TICKET) {
                        ComposerNote(stringResource(R.string.tickets_attachment_limit, TicketLimits.MAX_ATTACHMENTS_PER_TICKET))
                    }
                    if (viewModel.pendingImages.isNotEmpty() || viewModel.preparingImages > 0) {
                        PendingImagesRow(viewModel, enabled = !isSending)
                    }
                    AppTextField(
                        value = viewModel.reply,
                        onValueChange = viewModel::onReplyChange,
                        modifier = Modifier.fillMaxWidth(),
                        placeholder = stringResource(R.string.tickets_reply_placeholder),
                        readOnly = isSending,
                        singleLine = false,
                        minLines = 1,
                        maxLines = 5,
                        isError = tooLong,
                        supportingText = if (length >= COUNTER_FROM) "${Formats.count(length.toLong())}/${Formats.count(TicketLimits.MAX_CONTENT_RUNES.toLong())}" else null,
                        keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences),
                    )
                }
                Row(verticalAlignment = Alignment.CenterVertically) {
                    AppIconButton(
                        icon = Icons.Outlined.AddPhotoAlternate,
                        contentDescription = stringResource(R.string.tickets_add_images),
                        onClick = pickImages,
                        enabled = room > 0 && !isSending,
                        variant = ButtonVariant.Outline,
                    )
                    Spacer(Modifier.width(8.dp))
                    ReplyStatusMenu(
                        viewModel.replyStatus,
                        enabled = !isSending,
                        onSelect = viewModel::onReplyStatusChange,
                        modifier = Modifier.weight(1f),
                    )
                    Spacer(Modifier.width(8.dp))
                    val canSend = viewModel.hasDraft && !tooLong && viewModel.preparingImages == 0 && !isChangingStatus
                    AppButton(
                        stringResource(R.string.tickets_send_reply),
                        onClick = viewModel::send,
                        enabled = canSend,
                        loading = isSending,
                        size = ButtonSize.Small,
                        leadingIcon = Icons.AutoMirrored.Outlined.Send,
                    )
                }
            }
        }
    }
}

@Composable
private fun ComposerNote(text: String) {
    Text(text, style = MaterialTheme.typography.bodySmall, color = AppTheme.colors.mutedForeground)
}

@Composable
private fun PendingImagesRow(viewModel: TicketDetailViewModel, enabled: Boolean) {
    val colors = AppTheme.colors
    val shape = MaterialTheme.shapes.medium
    Row(
        Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        viewModel.pendingImages.forEach { pending ->
            Box(Modifier.size(64.dp)) {
                val tile = Modifier.fillMaxSize().clip(shape).background(colors.muted).border(1.dp, colors.border, shape)
                val preview = pending.preview
                if (preview != null) {
                    Image(preview, contentDescription = null, contentScale = ContentScale.Crop, alignment = Alignment.TopCenter, modifier = tile)
                } else {
                    Box(tile, contentAlignment = Alignment.Center) {
                        Icon(Icons.Outlined.BrokenImage, contentDescription = null, tint = colors.mutedForeground, modifier = Modifier.size(20.dp))
                    }
                }
                IconButton(
                    onClick = { viewModel.removeImage(pending.key) },
                    enabled = enabled,
                    modifier = Modifier.align(Alignment.TopEnd).padding(2.dp).size(22.dp),
                    colors = IconButtonDefaults.iconButtonColors(
                        containerColor = colors.foreground.copy(alpha = 0.7f),
                        contentColor = colors.background,
                    ),
                ) {
                    Icon(Icons.Outlined.Close, contentDescription = stringResource(R.string.tickets_remove_image), modifier = Modifier.size(14.dp))
                }
            }
        }
        repeat(viewModel.preparingImages) {
            Box(
                Modifier.size(64.dp).clip(shape).background(colors.muted).border(1.dp, colors.border, shape),
                contentAlignment = Alignment.Center,
            ) {
                CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp, color = colors.mutedForeground)
            }
        }
        Text(
            stringResource(
                R.string.tickets_image_count,
                viewModel.pendingImages.size + viewModel.preparingImages,
                TicketLimits.MAX_IMAGES_PER_MESSAGE,
            ),
            style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
            color = colors.mutedForeground,
        )
    }
}

@Composable
private fun ReplyStatusMenu(selected: Int?, enabled: Boolean, onSelect: (Int?) -> Unit, modifier: Modifier = Modifier) {
    val colors = AppTheme.colors
    var expanded by remember { mutableStateOf(false) }
    val current = if (selected == null) stringResource(R.string.tickets_reply_status_auto) else ticketStatusLabel(selected)
    Box(modifier) {
        Row(
            Modifier
                .clip(MaterialTheme.shapes.small)
                .border(1.dp, colors.input, MaterialTheme.shapes.small)
                .clickable(enabled = enabled, role = Role.DropdownList) { expanded = true }
                .padding(start = 10.dp, end = 6.dp)
                .height(36.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                stringResource(R.string.tickets_reply_status_value, current),
                style = MaterialTheme.typography.labelMedium,
                color = if (enabled) colors.foreground else colors.mutedForeground,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f, fill = false),
            )
            Icon(Icons.Outlined.KeyboardArrowDown, contentDescription = null, tint = colors.mutedForeground, modifier = Modifier.size(18.dp))
        }
        DropdownMenu(
            expanded = expanded && enabled,
            onDismissRequest = { expanded = false },
            shape = MaterialTheme.shapes.medium,
            containerColor = colors.popover,
            border = BorderStroke(1.dp, colors.border),
        ) {
            Text(
                stringResource(R.string.tickets_reply_status),
                style = MaterialTheme.typography.labelMedium,
                color = colors.mutedForeground,
                modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp),
            )
            val check: @Composable () -> Unit = {
                Icon(Icons.Outlined.Check, contentDescription = null, modifier = Modifier.size(16.dp))
            }
            DropdownMenuItem(
                text = {
                    Column {
                        Text(stringResource(R.string.tickets_reply_status_auto))
                        Text(
                            stringResource(R.string.tickets_reply_status_auto_desc),
                            style = MaterialTheme.typography.bodySmall,
                            color = colors.mutedForeground,
                        )
                    }
                },
                onClick = {
                    onSelect(null)
                    expanded = false
                },
                trailingIcon = if (selected == null) check else null,
            )
            REPLY_STATUSES.forEach { status ->
                DropdownMenuItem(
                    text = { Text(ticketStatusLabel(status)) },
                    onClick = {
                        onSelect(status)
                        expanded = false
                    },
                    trailingIcon = if (selected == status) check else null,
                )
            }
        }
    }
}

@Composable
private fun StatusSheet(current: Int, onSelect: (Int) -> Unit, onDismiss: () -> Unit) {
    val colors = AppTheme.colors
    AppBottomSheet(onDismissRequest = onDismiss, sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)) {
        Column(Modifier.padding(horizontal = 20.dp), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(stringResource(R.string.tickets_status_sheet_title), style = MaterialTheme.typography.titleLarge)
            Text(stringResource(R.string.tickets_status_sheet_desc), style = MaterialTheme.typography.bodyMedium, color = colors.mutedForeground)
        }
        Spacer(Modifier.height(8.dp))
        Column(Modifier.fillMaxWidth().padding(bottom = 16.dp)) {
            TicketStatus.all.forEach { status ->
                val isSelected = status == current
                ListItem(
                    headlineContent = { TicketStatusPill(status) },
                    trailingContent = {
                        if (isSelected) Icon(Icons.Outlined.Check, contentDescription = null, tint = colors.foreground, modifier = Modifier.size(18.dp))
                    },
                    colors = ListItemDefaults.colors(containerColor = if (isSelected) colors.accent else Color.Transparent),
                    modifier = Modifier
                        .padding(horizontal = 8.dp)
                        .clip(MaterialTheme.shapes.small)
                        .clickable { if (isSelected) onDismiss() else onSelect(status) },
                )
            }
        }
    }
}

private sealed interface FullImage {
    data object Loading : FullImage

    data object Failed : FullImage

    class Loaded(val image: ImageBitmap) : FullImage
}

/**
 * Full-screen view of one attachment over the screen: pinch or double-tap to zoom, drag to pan. Drawn in the
 * screen's own window, edge to edge, so the black reaches behind the system bars.
 */
@Composable
private fun ImageViewer(attachmentId: Int, load: suspend () -> ImageBitmap?, onDismiss: () -> Unit) {
    var attempt by remember { mutableIntStateOf(0) }
    val image by produceState<FullImage>(FullImage.Loading, attachmentId, attempt) {
        value = FullImage.Loading
        value = try {
            load()?.let { FullImage.Loaded(it) } ?: FullImage.Failed
        } catch (e: CancellationException) {
            throw e
        } catch (e: Throwable) {
            FullImage.Failed
        }
    }
    BackHandler(onBack = onDismiss)
    Box(
        Modifier
            .fillMaxSize()
            .background(Color.Black)
            // Touches stop here instead of reaching the thread underneath.
            .pointerInput(Unit) {},
    ) {
        when (val current = image) {
            is FullImage.Loaded -> ZoomableImage(current.image, Modifier.fillMaxSize())
            FullImage.Failed -> Column(
                Modifier.align(Alignment.Center).padding(24.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text(stringResource(R.string.tickets_image_failed), color = Color.White, style = MaterialTheme.typography.bodyMedium)
                AppButton(stringResource(R.string.common_retry), onClick = { attempt++ }, variant = ButtonVariant.Secondary, size = ButtonSize.Small)
            }
            FullImage.Loading -> CircularProgressIndicator(Modifier.align(Alignment.Center).size(28.dp), strokeWidth = 2.dp, color = Color.White)
        }
        IconButton(
            onClick = onDismiss,
            modifier = Modifier.align(Alignment.TopEnd).statusBarsPadding().padding(8.dp),
            colors = IconButtonDefaults.iconButtonColors(containerColor = Color.Black.copy(alpha = 0.5f), contentColor = Color.White),
        ) {
            Icon(Icons.Outlined.Close, contentDescription = stringResource(R.string.common_close))
        }
    }
}

@Composable
private fun ZoomableImage(image: ImageBitmap, modifier: Modifier = Modifier) {
    var scale by remember { mutableFloatStateOf(1f) }
    var offset by remember { mutableStateOf(Offset.Zero) }
    var size by remember { mutableStateOf(IntSize.Zero) }
    Image(
        image,
        contentDescription = null,
        contentScale = ContentScale.Fit,
        modifier = modifier
            .onSizeChanged { size = it }
            .pointerInput(Unit) {
                detectTapGestures(
                    onDoubleTap = {
                        scale = if (scale > 1f) 1f else 2.5f
                        offset = Offset.Zero
                    },
                )
            }
            .pointerInput(Unit) {
                detectTransformGestures { _, pan, zoom, _ ->
                    val next = (scale * zoom).coerceIn(1f, 5f)
                    // Panning stops where the zoomed image's edges reach the screen's; the fitted image may not fill it.
                    val fit = minOf(size.width / image.width.toFloat(), size.height / image.height.toFloat())
                    val maxX = maxOf(0f, (image.width * fit * next - size.width) / 2f)
                    val maxY = maxOf(0f, (image.height * fit * next - size.height) / 2f)
                    val moved = offset + pan
                    scale = next
                    offset = Offset(moved.x.coerceIn(-maxX, maxX), moved.y.coerceIn(-maxY, maxY))
                }
            }
            .graphicsLayer {
                scaleX = scale
                scaleY = scale
                translationX = offset.x
                translationY = offset.y
            },
    )
}
