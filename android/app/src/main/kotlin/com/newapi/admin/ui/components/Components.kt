package com.newapi.admin.ui.components

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Clear
import androidx.compose.material.icons.outlined.ContentCopy
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.Inbox
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.core.model.SelfUser
import com.newapi.admin.ui.common.errorMessage
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.theme.Tone
import com.newapi.admin.ui.theme.colors
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.filter

/** Display settings of the signed-in server. */
@Composable
fun rememberQuotaFormatter(): QuotaFormatter {
    val formatter by LocalAppContainer.current.quotaFormatter.collectAsStateWithLifecycle()
    return formatter
}

/** The signed-in admin (null only briefly while signing out). */
@Composable
fun rememberCurrentUser(): SelfUser? {
    val user by LocalAppContainer.current.currentUser.collectAsStateWithLifecycle()
    return user
}

@Composable
fun FullScreenLoading(modifier: Modifier = Modifier) {
    Box(modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        CircularProgressIndicator(Modifier.size(28.dp), strokeWidth = 2.5.dp, color = AppTheme.colors.mutedForeground)
    }
}

@Composable
fun FullScreenError(error: Throwable, onRetry: () -> Unit, modifier: Modifier = Modifier) {
    StateMessage(
        icon = Icons.Outlined.ErrorOutline,
        iconTint = AppTheme.colors.destructive,
        text = errorMessage(error),
        modifier = modifier,
        action = { AppButton(stringResource(R.string.common_retry), onClick = onRetry, variant = ButtonVariant.Outline, size = ButtonSize.Small) },
    )
}

@Composable
fun EmptyState(text: String, modifier: Modifier = Modifier, icon: ImageVector = Icons.Outlined.Inbox) {
    StateMessage(icon = icon, iconTint = AppTheme.colors.mutedForeground, text = text, modifier = modifier)
}

/** shadcn "empty" block: an icon in a muted tile above a centred message, with an optional action. */
@Composable
private fun StateMessage(
    icon: ImageVector,
    iconTint: Color,
    text: String,
    modifier: Modifier,
    action: (@Composable () -> Unit)? = null,
) {
    Column(
        modifier = modifier.fillMaxSize().padding(32.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier.size(48.dp).background(AppTheme.colors.muted, MaterialTheme.shapes.medium),
            contentAlignment = Alignment.Center,
        ) {
            Icon(icon, contentDescription = null, tint = iconTint, modifier = Modifier.size(24.dp))
        }
        Spacer(Modifier.height(16.dp))
        Text(text, style = MaterialTheme.typography.bodyMedium, color = AppTheme.colors.mutedForeground, textAlign = TextAlign.Center)
        if (action != null) {
            Spacer(Modifier.height(16.dp))
            action()
        }
    }
}

/**
 * Calls [onLoadMore] when the last visible item is within [buffer] items of the end, and again
 * whenever the item count grows while still near the end (a page that fits on screen keeps loading).
 */
@Composable
fun LoadMoreEffect(listState: LazyListState, onLoadMore: () -> Unit, buffer: Int = 5) {
    val currentOnLoadMore by rememberUpdatedState(onLoadMore)
    LaunchedEffect(listState) {
        snapshotFlow {
            val info = listState.layoutInfo
            val last = info.visibleItemsInfo.lastOrNull()?.index ?: -1
            val nearEnd = info.totalItemsCount > 0 && last >= info.totalItemsCount - 1 - buffer
            nearEnd to info.totalItemsCount
        }
            .distinctUntilChanged()
            .filter { (nearEnd, _) -> nearEnd }
            .collect { currentOnLoadMore() }
    }
}

/** Footer row for paged lists: spinner while loading, retry on failure, "no more" at the end. */
@Composable
fun ListFooter(isLoadingMore: Boolean, error: Throwable?, endReached: Boolean, onRetry: () -> Unit) {
    val colors = AppTheme.colors
    Box(Modifier.fillMaxWidth().padding(16.dp), contentAlignment = Alignment.Center) {
        when {
            isLoadingMore -> CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp, color = colors.mutedForeground)
            error != null -> Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(errorMessage(error), style = MaterialTheme.typography.bodySmall, color = colors.destructive, textAlign = TextAlign.Center)
                Spacer(Modifier.height(4.dp))
                AppButton(stringResource(R.string.common_retry), onClick = onRetry, variant = ButtonVariant.Ghost, size = ButtonSize.Small)
            }
            endReached -> Text(stringResource(R.string.common_no_more), style = MaterialTheme.typography.bodySmall, color = colors.mutedForeground)
        }
    }
}

@Composable
fun ConfirmDialog(
    title: String,
    text: String,
    confirmText: String,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
    destructive: Boolean = false,
) {
    AppDialog(
        onDismissRequest = onDismiss,
        title = title,
        description = text,
        confirmButton = {
            AppButton(confirmText, onClick = onConfirm, variant = if (destructive) ButtonVariant.Destructive else ButtonVariant.Default)
        },
        dismissButton = { AppButton(stringResource(R.string.common_cancel), onClick = onDismiss, variant = ButtonVariant.Outline) },
    )
}

/**
 * shadcn badge, e.g. a status or role: tinted by [tone], or a bordered neutral badge when [outlined].
 */
@Composable
fun StatusPill(text: String, tone: Tone, modifier: Modifier = Modifier, outlined: Boolean = false) {
    val colors = tone.colors()
    val shape = MaterialTheme.shapes.extraSmall
    Surface(
        color = if (outlined) Color.Transparent else colors.container,
        contentColor = if (outlined) AppTheme.colors.foreground else colors.content,
        shape = shape,
        border = if (outlined) BorderStroke(1.dp, AppTheme.colors.border) else null,
        modifier = modifier,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelMedium,
            maxLines = 1,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp),
        )
    }
}

/**
 * Titled card grouping [InfoRow]s or actions. [action] sits at the end of the title row, for example
 * an icon button.
 */
@Composable
fun SectionCard(
    title: String?,
    modifier: Modifier = Modifier,
    description: String? = null,
    action: (@Composable RowScope.() -> Unit)? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    AppCard(modifier = modifier.fillMaxWidth()) {
        if (title != null || action != null) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    if (title != null) Text(title, style = MaterialTheme.typography.titleSmall)
                    if (description != null) {
                        Text(description, style = MaterialTheme.typography.bodySmall, color = AppTheme.colors.mutedForeground)
                    }
                }
                action?.invoke(this)
            }
            Spacer(Modifier.height(8.dp))
        }
        content()
    }
}

/** Label/value line; long values wrap. With [copyable] a copy button puts the value on the clipboard. */
@Composable
fun InfoRow(label: String, value: String, modifier: Modifier = Modifier, copyable: Boolean = false) {
    val clipboard = LocalClipboardManager.current
    Row(modifier.fillMaxWidth().heightIn(min = 36.dp).padding(vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
        Text(
            label,
            style = MaterialTheme.typography.bodyMedium,
            color = AppTheme.colors.mutedForeground,
            modifier = Modifier.width(112.dp),
        )
        Text(
            value.ifEmpty { "-" },
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.weight(1f),
            overflow = TextOverflow.Ellipsis,
        )
        if (copyable && value.isNotEmpty()) {
            AppIconButton(
                icon = Icons.Outlined.ContentCopy,
                contentDescription = stringResource(R.string.common_copy),
                onClick = { clipboard.setText(AnnotatedString(value)) },
                size = 28.dp,
                iconSize = 14.dp,
                modifier = Modifier.padding(start = 8.dp),
            )
        }
    }
}

/** Single-line search box; [onSearch] fires on the keyboard action and on clear. */
@Composable
fun SearchField(
    query: String,
    onQueryChange: (String) -> Unit,
    onSearch: () -> Unit,
    placeholder: String,
    modifier: Modifier = Modifier,
) {
    AppTextField(
        value = query,
        onValueChange = onQueryChange,
        modifier = modifier.fillMaxWidth(),
        placeholder = placeholder,
        leadingIcon = Icons.Outlined.Search,
        trailing = if (query.isEmpty()) null else {
            {
                IconButton(onClick = { onQueryChange(""); onSearch() }) {
                    Icon(Icons.Outlined.Clear, contentDescription = stringResource(R.string.common_clear), modifier = Modifier.size(18.dp))
                }
            }
        },
        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
        keyboardActions = KeyboardActions(onSearch = { onSearch() }),
    )
}
