package com.newapi.admin.ui.tickets

import androidx.compose.foundation.background
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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.SupportAgent
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.format.Formats
import com.newapi.admin.core.model.Ticket
import com.newapi.admin.core.model.TicketStats
import com.newapi.admin.core.model.TicketStatus
import com.newapi.admin.core.net.ApiError
import com.newapi.admin.ui.common.UiMessagesEffect
import com.newapi.admin.ui.components.AppCard
import com.newapi.admin.ui.components.AppFilterChip
import com.newapi.admin.ui.components.AppSnackbarHost
import com.newapi.admin.ui.components.AppTopBar
import com.newapi.admin.ui.components.EmptyState
import com.newapi.admin.ui.components.FullScreenError
import com.newapi.admin.ui.components.FullScreenLoading
import com.newapi.admin.ui.components.ListFooter
import com.newapi.admin.ui.components.LoadMoreEffect
import com.newapi.admin.ui.components.SearchField
import com.newapi.admin.ui.theme.AppTheme

/** Every user's tickets, latest activity first, filtered by status and by id, subject or username. */
@Composable
fun TicketsScreen(onBack: () -> Unit, onOpenTicket: (Int) -> Unit) {
    val container = LocalAppContainer.current
    val viewModel = viewModel { TicketsViewModel(container) }
    val state by viewModel.state.collectAsStateWithLifecycle()
    val filters by viewModel.filters.collectAsStateWithLifecycle()
    val stats by container.ticketStats.collectAsStateWithLifecycle()
    val listState = rememberLazyListState()
    val snackbarHostState = remember { SnackbarHostState() }

    UiMessagesEffect(viewModel.messages, snackbarHostState)
    LoadMoreEffect(listState, onLoadMore = viewModel::loadMore)

    val unread = stats?.unread ?: 0
    Scaffold(
        topBar = {
            AppTopBar(
                title = stringResource(R.string.tickets_title),
                subtitle = if (unread > 0) {
                    pluralStringResource(R.plurals.tickets_unread_count, unread.coerceAtMost(Int.MAX_VALUE.toLong()).toInt(), Formats.count(unread))
                } else {
                    null
                },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = stringResource(R.string.common_back))
                    }
                },
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
                placeholder = stringResource(R.string.tickets_search_placeholder),
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
            )
            StatusFilterRow(
                selected = filters.status,
                stats = stats,
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
                    // A server without the ticket system answers 404.
                    error is ApiError.Http && error.status == 404 && state.items.isEmpty() -> LazyColumn(Modifier.fillMaxSize()) {
                        item(key = "unsupported") {
                            EmptyState(
                                text = stringResource(R.string.tickets_unsupported),
                                icon = Icons.Outlined.SupportAgent,
                                modifier = Modifier.fillParentMaxSize(),
                            )
                        }
                    }
                    error != null && state.items.isEmpty() -> FullScreenError(error, onRetry = viewModel::refresh)
                    state.items.isEmpty() -> LazyColumn(Modifier.fillMaxSize()) {
                        // A scrollable child so pull-to-refresh also works on the empty state.
                        item(key = "empty") {
                            EmptyState(
                                text = stringResource(if (filters.isActive) R.string.tickets_empty_filtered else R.string.tickets_empty),
                                icon = Icons.Outlined.SupportAgent,
                                modifier = Modifier.fillParentMaxSize(),
                            )
                        }
                    }
                    else -> LazyColumn(
                        state = listState,
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 4.dp, bottom = 16.dp),
                        verticalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        item(key = "count") {
                            val total = state.total.coerceIn(0, Int.MAX_VALUE.toLong()).toInt()
                            Text(
                                pluralStringResource(R.plurals.tickets_count, total, Formats.count(state.total)),
                                style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                                color = AppTheme.colors.mutedForeground,
                                modifier = Modifier.padding(horizontal = 2.dp),
                            )
                        }
                        items(state.items, key = { it.id }) { ticket ->
                            TicketRow(ticket, onClick = { onOpenTicket(ticket.id) })
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
}

@Composable
private fun StatusFilterRow(selected: Int?, stats: TicketStats?, onSelect: (Int?) -> Unit) {
    Row(
        Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()).padding(horizontal = 16.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        val all = stringResource(R.string.common_all)
        AppFilterChip(
            selected = selected == null,
            onClick = { onSelect(null) },
            label = if (stats != null) "$all ${Formats.count(stats.total)}" else all,
        )
        TicketStatus.all.forEach { status ->
            val label = ticketStatusLabel(status)
            AppFilterChip(
                selected = status == selected,
                onClick = { onSelect(status) },
                label = if (stats != null) "$label ${Formats.count(stats.count(status))}" else label,
            )
        }
    }
}

@Composable
private fun TicketRow(ticket: Ticket, onClick: () -> Unit) {
    val colors = AppTheme.colors
    AppCard(onClick = onClick, modifier = Modifier.fillMaxWidth()) {
        Row(verticalAlignment = Alignment.Top) {
            if (ticket.adminUnread) {
                val unread = stringResource(R.string.tickets_unread)
                Box(
                    Modifier
                        .padding(top = 6.dp, end = 8.dp)
                        .size(8.dp)
                        .background(colors.primary, CircleShape)
                        .semantics { contentDescription = unread },
                )
            }
            Text(
                ticket.subject,
                style = MaterialTheme.typography.titleSmall,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(8.dp))
            TicketStatusPill(ticket.status)
        }
        Spacer(Modifier.height(8.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            val parts = listOfNotNull(
                "#${ticket.id}",
                ticket.username.ifEmpty { null },
                pluralStringResource(R.plurals.tickets_message_count, ticket.messageCount, Formats.count(ticket.messageCount.toLong())),
            )
            Text(
                parts.joinToString(" · "),
                style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                color = colors.mutedForeground,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(12.dp))
            Text(
                Formats.shortDateTime(ticket.updatedTime),
                style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                color = colors.mutedForeground,
                maxLines = 1,
            )
        }
    }
}
