package com.newapi.admin.ui.redemptions

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Add
import androidx.compose.material.icons.outlined.ContentCopy
import androidx.compose.material.icons.outlined.DeleteSweep
import androidx.compose.material.icons.outlined.Redeem
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.format.Formats
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.core.model.Redemption
import com.newapi.admin.core.model.RedemptionStatusFilter
import com.newapi.admin.ui.common.UiMessagesEffect
import com.newapi.admin.ui.components.AppCard
import com.newapi.admin.ui.components.AppFab
import com.newapi.admin.ui.components.AppFilterChip
import com.newapi.admin.ui.components.AppIconButton
import com.newapi.admin.ui.components.AppSnackbarHost
import com.newapi.admin.ui.components.AppTopBar
import com.newapi.admin.ui.components.ConfirmDialog
import com.newapi.admin.ui.components.EmptyState
import com.newapi.admin.ui.components.FullScreenError
import com.newapi.admin.ui.components.FullScreenLoading
import com.newapi.admin.ui.components.ListFooter
import com.newapi.admin.ui.components.LoadMoreEffect
import com.newapi.admin.ui.components.SearchField
import com.newapi.admin.ui.components.rememberQuotaFormatter
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.theme.GeistMono
import com.newapi.admin.ui.theme.Tone
import com.newapi.admin.ui.theme.colors

/** The Codes tab: redemption codes filtered by name, id and status, newest first. */
@Composable
fun RedemptionsScreen(onOpenCode: (Int) -> Unit, onCreateCodes: () -> Unit) {
    val container = LocalAppContainer.current
    val viewModel = viewModel { RedemptionsViewModel(container) }
    val state by viewModel.state.collectAsStateWithLifecycle()
    val filters by viewModel.filters.collectAsStateWithLifecycle()
    val isDeletingInvalid by viewModel.isDeletingInvalid.collectAsStateWithLifecycle()
    val formatter = rememberQuotaFormatter()
    val listState = rememberLazyListState()
    val snackbarHostState = remember { SnackbarHostState() }
    var confirmDeleteInvalid by rememberSaveable { mutableStateOf(false) }
    val fabExpanded by remember { derivedStateOf { listState.firstVisibleItemIndex == 0 } }

    UiMessagesEffect(viewModel.messages, snackbarHostState)
    LoadMoreEffect(listState, onLoadMore = viewModel::loadMore)

    Scaffold(
        topBar = {
            AppTopBar(
                title = stringResource(R.string.redemptions_title),
                actions = {
                    if (isDeletingInvalid) {
                        Box(Modifier.size(48.dp), contentAlignment = Alignment.Center) {
                            CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp, color = AppTheme.colors.mutedForeground)
                        }
                    } else {
                        IconButton(onClick = { confirmDeleteInvalid = true }) {
                            Icon(Icons.Outlined.DeleteSweep, contentDescription = stringResource(R.string.redemptions_delete_invalid))
                        }
                    }
                },
            )
        },
        floatingActionButton = {
            AppFab(
                text = stringResource(R.string.redemptions_add),
                icon = Icons.Outlined.Add,
                onClick = onCreateCodes,
                expanded = fabExpanded,
            )
        },
        snackbarHost = { AppSnackbarHost(snackbarHostState) },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding)) {
            SearchField(
                query = viewModel.query,
                onQueryChange = viewModel::onQueryChange,
                onSearch = {
                    viewModel.search()
                    listState.requestScrollToItem(0)
                },
                placeholder = stringResource(R.string.redemptions_search_placeholder),
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
            )
            StatusFilterRow(
                selected = filters.status,
                onSelect = {
                    viewModel.setStatus(it)
                    listState.requestScrollToItem(0)
                },
            )
            PullToRefreshBox(
                isRefreshing = state.isRefreshing,
                onRefresh = viewModel::refresh,
                modifier = Modifier.fillMaxSize(),
            ) {
                val error = state.error
                when {
                    state.isInitialLoading -> FullScreenLoading()
                    error != null && state.items.isEmpty() -> FullScreenError(error, onRetry = viewModel::refresh)
                    state.items.isEmpty() -> LazyColumn(Modifier.fillMaxSize()) {
                        // A scrollable child so pull-to-refresh also works on the empty state.
                        item(key = "empty") {
                            EmptyState(
                                text = stringResource(if (filters.isActive) R.string.redemptions_empty_filtered else R.string.redemptions_empty),
                                icon = Icons.Outlined.Redeem,
                                modifier = Modifier.fillParentMaxSize(),
                            )
                        }
                    }
                    else -> LazyColumn(
                        state = listState,
                        modifier = Modifier.fillMaxSize(),
                        // Room below the last card for the floating button.
                        contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 4.dp, bottom = 96.dp),
                        verticalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        item(key = "count") {
                            val total = state.total.coerceIn(0, Int.MAX_VALUE.toLong()).toInt()
                            Text(
                                pluralStringResource(R.plurals.redemptions_count, total, Formats.count(state.total)),
                                style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                                color = AppTheme.colors.mutedForeground,
                                modifier = Modifier.padding(horizontal = 2.dp),
                            )
                        }
                        items(state.items, key = { it.id }) { code ->
                            CodeRow(code, formatter, onClick = { onOpenCode(code.id) })
                        }
                        item(key = "footer") {
                            // A failed refresh keeps the old rows; offer the retry here too (paging waits for it).
                            ListFooter(
                                isLoadingMore = state.isLoadingMore,
                                error = state.loadMoreError ?: error,
                                endReached = state.endReached,
                                onRetry = if (state.loadMoreError != null) viewModel::loadMore else viewModel::refresh,
                            )
                        }
                    }
                }
            }
        }
    }

    if (confirmDeleteInvalid) {
        ConfirmDialog(
            title = stringResource(R.string.redemptions_delete_invalid),
            text = stringResource(R.string.redemptions_delete_invalid_message),
            confirmText = stringResource(R.string.common_delete),
            destructive = true,
            onConfirm = {
                confirmDeleteInvalid = false
                viewModel.deleteInvalid()
            },
            onDismiss = { confirmDeleteInvalid = false },
        )
    }
}

@Composable
private fun StatusFilterRow(selected: RedemptionStatusFilter, onSelect: (RedemptionStatusFilter) -> Unit) {
    Row(
        Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()).padding(horizontal = 16.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        RedemptionStatusFilter.entries.forEach { status ->
            val label = when (status) {
                RedemptionStatusFilter.ALL -> R.string.common_all
                RedemptionStatusFilter.UNUSED -> R.string.redemptions_status_unused
                RedemptionStatusFilter.DISABLED -> R.string.redemptions_status_disabled
                RedemptionStatusFilter.USED -> R.string.redemptions_status_used
                RedemptionStatusFilter.EXPIRED -> R.string.redemptions_status_expired
            }
            AppFilterChip(selected = status == selected, onClick = { onSelect(status) }, label = stringResource(label))
        }
    }
}

@Composable
private fun CodeRow(code: Redemption, formatter: QuotaFormatter, onClick: () -> Unit) {
    val colors = AppTheme.colors
    val clipboard = LocalClipboardManager.current
    val nowSeconds = System.currentTimeMillis() / 1000
    val codeState = code.state(nowSeconds)
    AppCard(onClick = onClick, modifier = Modifier.fillMaxWidth()) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(code.displayName, style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text("#${code.id}", style = MaterialTheme.typography.bodySmall, color = colors.mutedForeground, maxLines = 1)
            }
            Spacer(Modifier.width(8.dp))
            CodeStatusPill(codeState)
        }
        Spacer(Modifier.height(10.dp))
        Surface(color = colors.muted, contentColor = colors.foreground, shape = MaterialTheme.shapes.small, modifier = Modifier.fillMaxWidth()) {
            Row(Modifier.padding(start = 10.dp, end = 2.dp), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    maskCode(code.key),
                    style = MaterialTheme.typography.bodySmall,
                    fontFamily = GeistMono,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                AppIconButton(
                    icon = Icons.Outlined.ContentCopy,
                    contentDescription = stringResource(R.string.redemptions_copy_code),
                    onClick = { clipboard.setText(AnnotatedString(code.key)) },
                    size = 32.dp,
                    iconSize = 14.dp,
                )
            }
        }
        Spacer(Modifier.height(10.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                formatter.format(code.quota),
                style = MaterialTheme.typography.labelLarge.copy(fontFeatureSettings = "tnum"),
                maxLines = 1,
            )
            Spacer(Modifier.width(12.dp))
            val (note, noteColor) = when {
                codeState == CodeState.USED ->
                    stringResource(R.string.redemptions_redeemed_by_at, code.usedUserId, Formats.dateTime(code.redeemedTime)) to colors.mutedForeground
                code.expiredTime == 0L -> stringResource(R.string.redemptions_never_expires) to colors.mutedForeground
                // A disabled code can be past its expiry too; only an enabled one is shown as expired.
                code.expiredTime < nowSeconds -> stringResource(R.string.redemptions_expired_at, Formats.dateTime(code.expiredTime)) to
                    if (codeState == CodeState.EXPIRED) Tone.Warning.colors().content else colors.mutedForeground
                else -> stringResource(R.string.redemptions_expires_at, Formats.dateTime(code.expiredTime)) to colors.mutedForeground
            }
            Text(
                note,
                style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                color = noteColor,
                textAlign = TextAlign.End,
                maxLines = 2,
                modifier = Modifier.weight(1f),
            )
        }
    }
}
