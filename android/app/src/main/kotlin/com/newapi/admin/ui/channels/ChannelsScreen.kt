package com.newapi.admin.ui.channels

import androidx.compose.foundation.BorderStroke
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
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.AccountBalanceWallet
import androidx.compose.material.icons.outlined.Add
import androidx.compose.material.icons.outlined.FilterList
import androidx.compose.material.icons.outlined.Hub
import androidx.compose.material.icons.outlined.MoreHoriz
import androidx.compose.material.icons.outlined.NetworkCheck
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
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
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.core.model.Channel
import com.newapi.admin.core.model.ChannelQuery
import com.newapi.admin.core.model.ChannelStatus
import com.newapi.admin.core.model.ChannelStatusFilter
import com.newapi.admin.core.model.ChannelTypes
import com.newapi.admin.ui.common.UiMessagesEffect
import com.newapi.admin.ui.components.AppBottomSheet
import com.newapi.admin.ui.components.AppButton
import com.newapi.admin.ui.components.AppCard
import com.newapi.admin.ui.components.AppFab
import com.newapi.admin.ui.components.AppFilterChip
import com.newapi.admin.ui.components.AppIconButton
import com.newapi.admin.ui.components.AppRemovableChip
import com.newapi.admin.ui.components.AppSnackbarHost
import com.newapi.admin.ui.components.AppSwitch
import com.newapi.admin.ui.components.AppTextField
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

/** The Channels tab: searchable, filterable, infinitely scrolling channel list. */
@Composable
fun ChannelsScreen(onOpenChannel: (Int) -> Unit, onCreateChannel: () -> Unit) {
    val container = LocalAppContainer.current
    val viewModel = viewModel { ChannelsViewModel(container) }
    val state by viewModel.state.collectAsStateWithLifecycle()
    val filters by viewModel.filters.collectAsStateWithLifecycle()
    val typeCounts by viewModel.typeCounts.collectAsStateWithLifecycle()
    val groups by viewModel.groups.collectAsStateWithLifecycle()
    val activity by viewModel.activity.collectAsStateWithLifecycle()
    val permissions = rememberChannelPermissions()
    val formatter = rememberQuotaFormatter()
    val listState = rememberLazyListState()
    val snackbarHostState = remember { SnackbarHostState() }
    var showFilters by rememberSaveable { mutableStateOf(false) }
    val fabExpanded by remember { derivedStateOf { listState.firstVisibleItemIndex == 0 } }

    UiMessagesEffect(viewModel.messages, snackbarHostState)
    LoadMoreEffect(listState, onLoadMore = viewModel::loadMore)

    Scaffold(
        topBar = {
            AppTopBar(
                title = stringResource(R.string.channels_title),
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
            if (permissions.sensitiveWrite) {
                AppFab(
                    text = stringResource(R.string.channels_add),
                    icon = Icons.Outlined.Add,
                    onClick = onCreateChannel,
                    expanded = fabExpanded,
                )
            }
        },
        snackbarHost = { AppSnackbarHost(snackbarHostState) },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding)) {
            SearchField(
                query = viewModel.query,
                onQueryChange = viewModel::onQueryChange,
                onSearch = viewModel::search,
                placeholder = stringResource(R.string.channels_search_placeholder),
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
                            val filtered = filters.hasColumnFilters || filters.keyword.isNotEmpty()
                            EmptyState(
                                text = stringResource(if (filtered) R.string.channels_empty_filtered else R.string.channels_empty),
                                icon = Icons.Outlined.Hub,
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
                                pluralStringResource(R.plurals.channels_count, total, Formats.count(state.total)),
                                style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                                color = AppTheme.colors.mutedForeground,
                                modifier = Modifier.padding(horizontal = 2.dp),
                            )
                        }
                        items(state.items, key = { it.id }) { channel ->
                            ChannelRow(
                                channel = channel,
                                formatter = formatter,
                                permissions = permissions,
                                isToggling = channel.id in activity.toggling,
                                isBusy = channel.id in activity.testing || channel.id in activity.updatingBalance,
                                onClick = { onOpenChannel(channel.id) },
                                onToggle = { enabled -> viewModel.setEnabled(channel, enabled) },
                                onTest = { viewModel.test(channel) },
                                onUpdateBalance = { viewModel.updateBalance(channel) },
                            )
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
        ChannelFilterSheet(
            filters = filters,
            typeCounts = typeCounts,
            groups = groups,
            onRetryGroups = viewModel::loadGroups,
            onApply = { newFilters ->
                showFilters = false
                viewModel.applyFilters(newFilters)
            },
            onDismiss = { showFilters = false },
        )
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun ChannelRow(
    channel: Channel,
    formatter: QuotaFormatter,
    permissions: ChannelPermissions,
    isToggling: Boolean,
    isBusy: Boolean,
    onClick: () -> Unit,
    onToggle: (Boolean) -> Unit,
    onTest: () -> Unit,
    onUpdateBalance: () -> Unit,
) {
    val colors = AppTheme.colors
    AppCard(onClick = onClick, modifier = Modifier.fillMaxWidth(), contentPadding = PaddingValues(0.dp)) {
        Row(
            Modifier.padding(start = 16.dp, top = 12.dp, end = 8.dp, bottom = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(
                    channel.displayName,
                    style = MaterialTheme.typography.titleSmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                FlowRow(
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                    verticalArrangement = Arrangement.spacedBy(4.dp),
                ) {
                    Text(
                        "#${channel.id} · ${channelTypeLabel(channel.type)}",
                        style = MaterialTheme.typography.bodySmall,
                        color = colors.mutedForeground,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.align(Alignment.CenterVertically),
                    )
                    ChannelStatusPill(channel.status, Modifier.align(Alignment.CenterVertically))
                    val tag = channel.tag?.trim().orEmpty()
                    if (tag.isNotEmpty()) StatusPill(tag, Tone.Info, Modifier.align(Alignment.CenterVertically))
                }
            }
            if (permissions.operate) {
                val switchLabel = stringResource(R.string.channels_switch_description, channel.displayName)
                AppSwitch(
                    checked = channel.status == ChannelStatus.ENABLED,
                    onCheckedChange = onToggle,
                    enabled = !isToggling,
                    modifier = Modifier.semantics { contentDescription = switchLabel },
                )
            }
            RowActionsMenu(channel, permissions, isBusy, onTest, onUpdateBalance)
        }
        HorizontalDivider(color = colors.border)
        FlowRow(
            Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(14.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            // Health and spend first; routing numbers after them.
            Metric(
                stringResource(R.string.channels_label_response),
                responseTimeText(channel.responseTime),
                toneTextColor(responseTimeTone(channel.responseTime)),
            )
            if (channel.hasBalance) {
                Metric(
                    stringResource(R.string.channels_label_balance),
                    formatter.formatUsd(channel.balance),
                    toneTextColor(balanceTone(channel.balance)),
                )
            }
            Metric(stringResource(R.string.channels_label_used), formatter.format(channel.usedQuota))
            Metric(stringResource(R.string.channels_label_priority), channel.priority?.toString() ?: "-")
            Metric(stringResource(R.string.channels_label_weight), channel.weight?.toString() ?: "-")
        }
    }
}

@Composable
private fun RowActionsMenu(
    channel: Channel,
    permissions: ChannelPermissions,
    isBusy: Boolean,
    onTest: () -> Unit,
    onUpdateBalance: () -> Unit,
) {
    val canTest = permissions.operate && channel.isTestable
    val canUpdateBalance = permissions.operate && channel.hasBalance
    if (!canTest && !canUpdateBalance) return
    // Same footprint as the menu button, so the row does not shift while busy.
    if (isBusy) {
        Box(Modifier.padding(start = 4.dp).size(36.dp), contentAlignment = Alignment.Center) {
            CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp, color = AppTheme.colors.mutedForeground)
        }
        return
    }
    var expanded by remember { mutableStateOf(false) }
    Box(Modifier.padding(start = 4.dp)) {
        AppIconButton(
            icon = Icons.Outlined.MoreHoriz,
            contentDescription = stringResource(R.string.common_more),
            onClick = { expanded = true },
        )
        DropdownMenu(
            expanded = expanded,
            onDismissRequest = { expanded = false },
            shape = MaterialTheme.shapes.medium,
            containerColor = AppTheme.colors.popover,
            border = BorderStroke(1.dp, AppTheme.colors.border),
        ) {
            if (canTest) {
                DropdownMenuItem(
                    text = { Text(stringResource(R.string.channels_action_test), style = MaterialTheme.typography.bodyMedium) },
                    leadingIcon = { Icon(Icons.Outlined.NetworkCheck, contentDescription = null, modifier = Modifier.size(18.dp)) },
                    onClick = {
                        expanded = false
                        onTest()
                    },
                )
            }
            if (canUpdateBalance) {
                DropdownMenuItem(
                    text = { Text(stringResource(R.string.channels_action_update_balance), style = MaterialTheme.typography.bodyMedium) },
                    leadingIcon = { Icon(Icons.Outlined.AccountBalanceWallet, contentDescription = null, modifier = Modifier.size(18.dp)) },
                    onClick = {
                        expanded = false
                        onUpdateBalance()
                    },
                )
            }
        }
    }
}

@Composable
private fun Metric(label: String, value: String, valueColor: Color = Color.Unspecified) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Text(label, style = MaterialTheme.typography.bodySmall, color = AppTheme.colors.mutedForeground, maxLines = 1)
        Spacer(Modifier.width(4.dp))
        Text(
            value,
            style = MaterialTheme.typography.labelMedium.copy(fontFeatureSettings = "tnum"),
            color = valueColor,
            maxLines = 1,
        )
    }
}

/** Removable chips for the filters set in the sheet. */
@Composable
private fun ActiveFilters(filters: ChannelQuery, onChange: (ChannelQuery) -> Unit) {
    val removeDescription = stringResource(R.string.channels_remove_filter)
    Row(
        Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()).padding(horizontal = 16.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        if (filters.status != ChannelStatusFilter.ALL) {
            val label = if (filters.status == ChannelStatusFilter.ENABLED) R.string.common_enabled else R.string.common_disabled
            AppRemovableChip(
                label = stringResource(label),
                onRemove = { onChange(filters.copy(status = ChannelStatusFilter.ALL)) },
                removeDescription = removeDescription,
            )
        }
        filters.type?.let { type ->
            AppRemovableChip(
                label = channelTypeLabel(type),
                onRemove = { onChange(filters.copy(type = null)) },
                removeDescription = removeDescription,
            )
        }
        if (filters.group.isNotEmpty()) {
            AppRemovableChip(
                label = stringResource(R.string.channels_filter_chip_group, filters.group),
                onRemove = { onChange(filters.copy(group = "")) },
                removeDescription = removeDescription,
            )
        }
        if (filters.model.isNotEmpty()) {
            AppRemovableChip(
                label = stringResource(R.string.channels_filter_chip_model, filters.model),
                onRemove = { onChange(filters.copy(model = "")) },
                removeDescription = removeDescription,
            )
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun ChannelFilterSheet(
    filters: ChannelQuery,
    typeCounts: Map<String, Long>?,
    groups: GroupOptions,
    onRetryGroups: () -> Unit,
    onApply: (ChannelQuery) -> Unit,
    onDismiss: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    var status by rememberSaveable { mutableStateOf(filters.status) }
    var type by rememberSaveable { mutableStateOf(filters.type) }
    var group by rememberSaveable { mutableStateOf(filters.group) }
    var model by rememberSaveable { mutableStateOf(filters.model) }

    // With counts: only types that have channels (hidden legacy types included), most used first, then by label.
    val typeOptions: List<Triple<Int, String, Long?>> = typeCounts
        ?.mapNotNull { (key, count) -> key.toIntOrNull()?.takeIf { count > 0 }?.let { Triple(it, channelTypeLabel(it), count) } }
        ?.sortedWith(compareByDescending<Triple<Int, String, Long>> { it.third }.thenBy { it.second })
        ?: ChannelTypes.selectable.map { Triple(it.id, channelTypeLabel(it.id), null) }
    val selectedType = type
    val shownTypes = if (selectedType != null && typeOptions.none { it.first == selectedType }) {
        listOf(Triple(selectedType, channelTypeLabel(selectedType), typeCounts?.let { 0L })) + typeOptions
    } else {
        typeOptions
    }
    val groupNames = if (group.isNotEmpty() && group !in groups.names) listOf(group) + groups.names else groups.names

    AppBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(start = 20.dp, end = 20.dp, bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(20.dp),
        ) {
            Text(stringResource(R.string.common_filter), style = MaterialTheme.typography.titleLarge)

            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                SheetLabel(stringResource(R.string.channels_filter_status))
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    ChannelStatusFilter.entries.forEach { option ->
                        val label = when (option) {
                            ChannelStatusFilter.ALL -> R.string.common_all
                            ChannelStatusFilter.ENABLED -> R.string.common_enabled
                            ChannelStatusFilter.DISABLED -> R.string.common_disabled
                        }
                        AppFilterChip(selected = status == option, onClick = { status = option }, label = stringResource(label))
                    }
                }
            }

            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                SheetLabel(stringResource(R.string.channels_filter_type))
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    AppFilterChip(
                        selected = type == null,
                        onClick = { type = null },
                        label = stringResource(R.string.channels_filter_all_types),
                    )
                    shownTypes.forEach { (id, name, count) ->
                        AppFilterChip(
                            selected = type == id,
                            onClick = { type = id },
                            label = if (count != null) stringResource(R.string.channels_type_with_count, name, Formats.count(count)) else name,
                        )
                    }
                }
            }

            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                SheetLabel(stringResource(R.string.channels_filter_group))
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    AppFilterChip(
                        selected = group.isEmpty(),
                        onClick = { group = "" },
                        label = stringResource(R.string.channels_filter_all_groups),
                    )
                    groupNames.forEach { name ->
                        AppFilterChip(selected = group == name, onClick = { group = name }, label = name)
                    }
                }
                when {
                    groups.isLoading -> CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp, color = AppTheme.colors.mutedForeground)
                    groups.failed -> Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(
                            stringResource(R.string.channels_groups_load_failed),
                            style = MaterialTheme.typography.bodySmall,
                            color = AppTheme.colors.destructive,
                            modifier = Modifier.weight(1f),
                        )
                        AppButton(
                            stringResource(R.string.common_retry),
                            onClick = onRetryGroups,
                            variant = ButtonVariant.Outline,
                            size = ButtonSize.Small,
                        )
                    }
                }
            }

            AppTextField(
                value = model,
                onValueChange = { model = it },
                modifier = Modifier.fillMaxWidth(),
                label = stringResource(R.string.channels_filter_model),
                placeholder = stringResource(R.string.channels_filter_model_placeholder),
                singleLine = true,
            )

            Row(Modifier.fillMaxWidth().padding(top = 4.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                AppButton(
                    stringResource(R.string.common_reset),
                    onClick = {
                        status = ChannelStatusFilter.ALL
                        type = null
                        group = ""
                        model = ""
                    },
                    modifier = Modifier.weight(1f),
                    variant = ButtonVariant.Outline,
                )
                AppButton(
                    stringResource(R.string.common_apply),
                    onClick = { onApply(filters.copy(status = status, type = type, group = group.trim(), model = model.trim())) },
                    modifier = Modifier.weight(1f),
                )
            }
        }
    }
}

@Composable
private fun SheetLabel(text: String) {
    Text(text, style = MaterialTheme.typography.labelLarge)
}
