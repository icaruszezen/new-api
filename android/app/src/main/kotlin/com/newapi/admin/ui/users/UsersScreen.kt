package com.newapi.admin.ui.users

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
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
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.FilterList
import androidx.compose.material.icons.filled.PersonAdd
import androidx.compose.material.icons.filled.PersonSearch
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.InputChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.format.Formats
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.core.model.AdminUser
import com.newapi.admin.core.model.Roles
import com.newapi.admin.core.model.UserStatus
import com.newapi.admin.ui.common.UiMessage
import com.newapi.admin.ui.common.text
import com.newapi.admin.ui.components.EmptyState
import com.newapi.admin.ui.components.FullScreenError
import com.newapi.admin.ui.components.FullScreenLoading
import com.newapi.admin.ui.components.ListFooter
import com.newapi.admin.ui.components.LoadMoreEffect
import com.newapi.admin.ui.components.SearchField
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.components.rememberQuotaFormatter
import com.newapi.admin.ui.theme.Tone

/** Status filter value for soft-deleted users (`/api/user/search?status=-1`). */
private const val STATUS_DELETED = -1

private val ROLE_FILTERS = listOf(Roles.COMMON, Roles.ADMIN, Roles.ROOT)
private val STATUS_FILTERS = listOf(UserStatus.ENABLED, UserStatus.DISABLED, STATUS_DELETED)

/** The Users tab: searchable, filterable, infinitely scrolling user list. */
@Composable
fun UsersScreen(onOpenUser: (Int) -> Unit, onCreateUser: () -> Unit) {
    val container = LocalAppContainer.current
    val viewModel = viewModel { UsersViewModel(container) }
    val state by viewModel.state.collectAsStateWithLifecycle()
    val filters by viewModel.filters.collectAsStateWithLifecycle()
    val groups by viewModel.groups.collectAsStateWithLifecycle()
    val formatter = rememberQuotaFormatter()
    val listState = rememberLazyListState()
    val snackbarHostState = remember { SnackbarHostState() }
    var showFilters by rememberSaveable { mutableStateOf(false) }
    val fabExpanded by remember { derivedStateOf { listState.firstVisibleItemIndex == 0 } }

    var snackbarMessage by remember { mutableStateOf<UiMessage?>(null) }
    LaunchedEffect(viewModel) {
        viewModel.messages.collect { snackbarMessage = it }
    }
    snackbarMessage?.let { message ->
        val text = message.text()
        LaunchedEffect(message) {
            snackbarHostState.showSnackbar(text)
            snackbarMessage = null
        }
    }

    LoadMoreEffect(listState, onLoadMore = viewModel::loadMore)

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.users_title)) },
                actions = {
                    IconButton(onClick = {
                        viewModel.loadGroups()
                        showFilters = true
                    }) {
                        BadgedBox(badge = { if (filters.hasColumnFilters) Badge() }) {
                            Icon(Icons.Filled.FilterList, contentDescription = stringResource(R.string.common_filter))
                        }
                    }
                },
            )
        },
        floatingActionButton = {
            ExtendedFloatingActionButton(
                onClick = onCreateUser,
                expanded = fabExpanded,
                icon = { Icon(Icons.Filled.PersonAdd, contentDescription = null) },
                text = { Text(stringResource(R.string.users_add)) },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding)) {
            SearchField(
                query = viewModel.query,
                onQueryChange = viewModel::onQueryChange,
                onSearch = viewModel::search,
                placeholder = stringResource(R.string.users_search_placeholder),
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
            )
            if (filters.hasColumnFilters) {
                ActiveFilters(filters, onChange = viewModel::applyFilters)
            }
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
                                text = stringResource(if (filters.isActive) R.string.users_empty_filtered else R.string.users_empty),
                                icon = Icons.Filled.PersonSearch,
                                modifier = Modifier.fillParentMaxSize(),
                            )
                        }
                    }
                    else -> LazyColumn(
                        state = listState,
                        modifier = Modifier.fillMaxSize(),
                        // Room below the last card for the floating button.
                        contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 4.dp, bottom = 96.dp),
                        verticalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        item(key = "count") {
                            val total = state.total.coerceIn(0, Int.MAX_VALUE.toLong()).toInt()
                            Text(
                                pluralStringResource(R.plurals.users_count, total, Formats.count(state.total)),
                                style = MaterialTheme.typography.labelMedium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                        items(state.items, key = { it.id }) { user ->
                            UserRow(user, formatter, onClick = { onOpenUser(user.id) })
                        }
                        item(key = "footer") {
                            ListFooter(
                                isLoadingMore = state.isLoadingMore,
                                error = state.loadMoreError,
                                endReached = state.endReached,
                                onRetry = viewModel::loadMore,
                            )
                        }
                    }
                }
            }
        }
    }

    if (showFilters) {
        UserFilterSheet(
            filters = filters,
            groups = groups,
            onRetryGroups = viewModel::loadGroups,
            onApply = { role, status, group ->
                showFilters = false
                viewModel.applyFilters(role, status, group)
            },
            onDismiss = { showFilters = false },
        )
    }
}

@Composable
private fun UserRow(user: AdminUser, formatter: QuotaFormatter, onClick: () -> Unit) {
    Card(
        onClick = onClick,
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow),
    ) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    user.displayName.ifBlank { user.username },
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                Spacer(Modifier.width(8.dp))
                UserStatusPill(user)
            }
            Text(
                "@${user.username} · #${user.id}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                RolePill(user.role)
                Box(Modifier.weight(1f)) {
                    if (user.group.isNotEmpty()) StatusPill(user.group, Tone.Neutral)
                }
                Text(
                    stringResource(R.string.users_requests_value, Formats.count(user.requestCount)),
                    style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                )
            }
            Spacer(Modifier.height(2.dp))
            if (user.quota == 0L && user.usedQuota == 0L) {
                Text(
                    stringResource(R.string.users_no_quota),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            } else {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(
                        stringResource(R.string.users_remaining_value, formatter.format(user.quota)),
                        style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                        fontWeight = FontWeight.Medium,
                        color = if (user.quota < 0) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurface,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f),
                    )
                    Text(
                        stringResource(R.string.users_used_value, formatter.format(user.usedQuota)),
                        style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 1,
                    )
                }
            }
            QuotaBar(user.quota, user.usedQuota)
        }
    }
}

/** Removable chips for the applied role, status and group filters. */
@Composable
private fun ActiveFilters(filters: UserFilters, onChange: (role: Int?, status: Int?, group: String) -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 16.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        val removeIcon: @Composable () -> Unit = {
            Icon(Icons.Filled.Close, contentDescription = stringResource(R.string.users_remove_filter), modifier = Modifier.size(16.dp))
        }
        filters.role?.let { role ->
            InputChip(
                selected = true,
                onClick = { onChange(null, filters.status, filters.group) },
                label = { Text(roleLabel(role)) },
                trailingIcon = removeIcon,
            )
        }
        filters.status?.let { status ->
            InputChip(
                selected = true,
                onClick = { onChange(filters.role, null, filters.group) },
                label = { Text(statusFilterLabel(status)) },
                trailingIcon = removeIcon,
            )
        }
        if (filters.group.isNotEmpty()) {
            InputChip(
                selected = true,
                onClick = { onChange(filters.role, filters.status, "") },
                label = { Text(filters.group, maxLines = 1, overflow = TextOverflow.Ellipsis) },
                trailingIcon = removeIcon,
            )
        }
    }
}

@Composable
private fun statusFilterLabel(status: Int): String = when (status) {
    UserStatus.ENABLED -> stringResource(R.string.common_enabled)
    UserStatus.DISABLED -> stringResource(R.string.common_disabled)
    STATUS_DELETED -> stringResource(R.string.users_status_deleted)
    else -> stringResource(R.string.users_status_other, status)
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun UserFilterSheet(
    filters: UserFilters,
    groups: GroupList,
    onRetryGroups: () -> Unit,
    onApply: (role: Int?, status: Int?, group: String) -> Unit,
    onDismiss: () -> Unit,
) {
    var role by rememberSaveable { mutableStateOf(filters.role) }
    var status by rememberSaveable { mutableStateOf(filters.status) }
    var group by rememberSaveable { mutableStateOf(filters.group) }
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(start = 24.dp, end = 24.dp, bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(stringResource(R.string.users_filters_title), style = MaterialTheme.typography.titleLarge)

            Text(stringResource(R.string.users_filter_role), style = MaterialTheme.typography.titleSmall)
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FilterChip(selected = role == null, onClick = { role = null }, label = { Text(stringResource(R.string.common_all)) })
                ROLE_FILTERS.forEach { value ->
                    FilterChip(selected = role == value, onClick = { role = value }, label = { Text(roleLabel(value)) })
                }
            }

            Text(stringResource(R.string.users_filter_status), style = MaterialTheme.typography.titleSmall)
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FilterChip(selected = status == null, onClick = { status = null }, label = { Text(stringResource(R.string.common_all)) })
                STATUS_FILTERS.forEach { value ->
                    FilterChip(selected = status == value, onClick = { status = value }, label = { Text(statusFilterLabel(value)) })
                }
            }

            GroupDropdown(
                value = group,
                options = groups.names,
                onSelect = { group = it },
                label = stringResource(R.string.users_filter_group),
                allLabel = stringResource(R.string.users_all_groups),
                supportingText = if (groups.failed) stringResource(R.string.users_groups_load_failed) else null,
            )
            if (groups.failed) {
                TextButton(onClick = onRetryGroups) { Text(stringResource(R.string.common_retry)) }
            }

            Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
                OutlinedButton(
                    onClick = {
                        role = null
                        status = null
                        group = ""
                    },
                    modifier = Modifier.weight(1f),
                ) { Text(stringResource(R.string.common_reset)) }
                Button(onClick = { onApply(role, status, group) }, modifier = Modifier.weight(1f)) {
                    Text(stringResource(R.string.common_apply))
                }
            }
        }
    }
}
