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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.FilterList
import androidx.compose.material.icons.outlined.PersonAdd
import androidx.compose.material.icons.outlined.PersonSearch
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
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
import androidx.compose.ui.semantics.clearAndSetSemantics
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
import com.newapi.admin.ui.components.AppAvatar
import com.newapi.admin.ui.components.AppBottomSheet
import com.newapi.admin.ui.components.AppButton
import com.newapi.admin.ui.components.AppCard
import com.newapi.admin.ui.components.AppFab
import com.newapi.admin.ui.components.AppFilterChip
import com.newapi.admin.ui.components.AppRemovableChip
import com.newapi.admin.ui.components.AppSnackbarHost
import com.newapi.admin.ui.components.AppTopBar
import com.newapi.admin.ui.components.ButtonSize
import com.newapi.admin.ui.components.ButtonVariant
import com.newapi.admin.ui.components.EmptyState
import com.newapi.admin.ui.components.FullScreenError
import com.newapi.admin.ui.components.FullScreenLoading
import com.newapi.admin.ui.components.ListFooter
import com.newapi.admin.ui.components.LoadMoreEffect
import com.newapi.admin.ui.components.SearchField
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.components.rememberQuotaFormatter
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.theme.Tone

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

    // A new search or filter starts at the top; pull-to-refresh, paging and row updates keep the position.
    val search = {
        viewModel.search()
        listState.requestScrollToItem(0)
    }
    val applyFilters: (Int?, Int?, String) -> Unit = { role, status, group ->
        viewModel.applyFilters(role, status, group)
        listState.requestScrollToItem(0)
    }
    LoadMoreEffect(listState, onLoadMore = viewModel::loadMore)

    Scaffold(
        topBar = {
            AppTopBar(
                title = stringResource(R.string.users_title),
                actions = {
                    IconButton(onClick = {
                        viewModel.loadGroups()
                        showFilters = true
                    }) {
                        BadgedBox(
                            badge = {
                                if (filters.hasColumnFilters) {
                                    Badge(containerColor = AppTheme.colors.primary, contentColor = AppTheme.colors.primaryForeground)
                                }
                            },
                        ) {
                            Icon(Icons.Outlined.FilterList, contentDescription = stringResource(R.string.common_filter))
                        }
                    }
                },
            )
        },
        floatingActionButton = {
            AppFab(
                text = stringResource(R.string.users_add),
                icon = Icons.Outlined.PersonAdd,
                onClick = onCreateUser,
                expanded = fabExpanded,
            )
        },
        snackbarHost = { AppSnackbarHost(snackbarHostState) },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding)) {
            SearchField(
                query = viewModel.query,
                onQueryChange = viewModel::onQueryChange,
                onSearch = search,
                placeholder = stringResource(R.string.users_search_placeholder),
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
            )
            if (filters.hasColumnFilters) {
                ActiveFilters(filters, onChange = applyFilters)
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
                                icon = Icons.Outlined.PersonSearch,
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
                                pluralStringResource(R.plurals.users_count, total, Formats.count(state.total)),
                                style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                                color = AppTheme.colors.mutedForeground,
                                modifier = Modifier.padding(horizontal = 2.dp),
                            )
                        }
                        items(state.items, key = { it.id }) { user ->
                            UserRow(user, formatter, onClick = { onOpenUser(user.id) })
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

    if (showFilters) {
        UserFilterSheet(
            filters = filters,
            groups = groups,
            onRetryGroups = viewModel::loadGroups,
            onApply = { role, status, group ->
                showFilters = false
                applyFilters(role, status, group)
            },
            onDismiss = { showFilters = false },
        )
    }
}

@Composable
private fun UserRow(user: AdminUser, formatter: QuotaFormatter, onClick: () -> Unit) {
    val colors = AppTheme.colors
    val name = user.displayName.ifBlank { user.username }
    AppCard(onClick = onClick, modifier = Modifier.fillMaxWidth()) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            // Decorative: the name next to it already says who this is.
            AppAvatar(avatarInitials(name), Modifier.clearAndSetSemantics {})
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    name,
                    style = MaterialTheme.typography.titleSmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    "@${user.username} · #${user.id}",
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.mutedForeground,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Spacer(Modifier.width(8.dp))
            UserStatusPill(user)
        }
        Spacer(Modifier.height(12.dp))
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            RolePill(user.role)
            Box(Modifier.weight(1f)) {
                if (user.group.isNotEmpty()) StatusPill(user.group, Tone.Neutral, outlined = true)
            }
            Text(
                stringResource(R.string.users_requests_value, Formats.count(user.requestCount)),
                style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                color = colors.mutedForeground,
                maxLines = 1,
            )
        }
        Spacer(Modifier.height(12.dp))
        if (user.quota == 0L && user.usedQuota == 0L) {
            Text(
                stringResource(R.string.users_no_quota),
                style = MaterialTheme.typography.bodySmall,
                color = colors.mutedForeground,
            )
        } else {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    stringResource(R.string.users_remaining_value, formatter.format(user.quota)),
                    style = MaterialTheme.typography.labelLarge.copy(fontFeatureSettings = "tnum"),
                    color = if (user.quota < 0) colors.destructive else colors.foreground,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                Text(
                    stringResource(R.string.users_used_value, formatter.format(user.usedQuota)),
                    style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                    color = colors.mutedForeground,
                    maxLines = 1,
                )
            }
        }
        Spacer(Modifier.height(8.dp))
        QuotaBar(user.quota, user.usedQuota)
    }
}

/** Removable chips for the applied role, status and group filters. */
@Composable
private fun ActiveFilters(filters: UserFilters, onChange: (role: Int?, status: Int?, group: String) -> Unit) {
    val removeDescription = stringResource(R.string.users_remove_filter)
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 16.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        filters.role?.let { role ->
            AppRemovableChip(
                label = roleLabel(role),
                onRemove = { onChange(null, filters.status, filters.group) },
                removeDescription = removeDescription,
            )
        }
        filters.status?.let { status ->
            AppRemovableChip(
                label = statusFilterLabel(status),
                onRemove = { onChange(filters.role, null, filters.group) },
                removeDescription = removeDescription,
            )
        }
        if (filters.group.isNotEmpty()) {
            AppRemovableChip(
                label = filters.group,
                onRemove = { onChange(filters.role, filters.status, "") },
                removeDescription = removeDescription,
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

    AppBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(start = 20.dp, end = 20.dp, bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(20.dp),
        ) {
            Text(stringResource(R.string.users_filters_title), style = MaterialTheme.typography.titleLarge)

            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(stringResource(R.string.users_filter_role), style = MaterialTheme.typography.labelLarge)
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    AppFilterChip(selected = role == null, onClick = { role = null }, label = stringResource(R.string.common_all))
                    ROLE_FILTERS.forEach { value ->
                        AppFilterChip(selected = role == value, onClick = { role = value }, label = roleLabel(value))
                    }
                }
            }

            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(stringResource(R.string.users_filter_status), style = MaterialTheme.typography.labelLarge)
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    AppFilterChip(selected = status == null, onClick = { status = null }, label = stringResource(R.string.common_all))
                    STATUS_FILTERS.forEach { value ->
                        AppFilterChip(selected = status == value, onClick = { status = value }, label = statusFilterLabel(value))
                    }
                }
            }

            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                GroupDropdown(
                    value = group,
                    options = groups.names,
                    onSelect = { group = it },
                    label = stringResource(R.string.users_filter_group),
                    allLabel = stringResource(R.string.users_all_groups),
                    supportingText = if (groups.failed) stringResource(R.string.users_groups_load_failed) else null,
                )
                if (groups.failed) {
                    AppButton(
                        stringResource(R.string.common_retry),
                        onClick = onRetryGroups,
                        variant = ButtonVariant.Outline,
                        size = ButtonSize.Small,
                    )
                }
            }

            Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth().padding(top = 4.dp)) {
                AppButton(
                    stringResource(R.string.common_reset),
                    onClick = {
                        role = null
                        status = null
                        group = ""
                    },
                    modifier = Modifier.weight(1f),
                    variant = ButtonVariant.Outline,
                )
                AppButton(
                    stringResource(R.string.common_apply),
                    onClick = { onApply(role, status, group) },
                    modifier = Modifier.weight(1f),
                )
            }
        }
    }
}
