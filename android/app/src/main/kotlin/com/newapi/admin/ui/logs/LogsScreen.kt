package com.newapi.admin.ui.logs

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
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
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.DateRange
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.FilterList
import androidx.compose.material.icons.outlined.Refresh
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.format.Formats
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.core.model.LogEntry
import com.newapi.admin.core.model.LogType
import com.newapi.admin.core.model.parsedOther
import com.newapi.admin.ui.common.UiMessage
import com.newapi.admin.ui.common.text
import com.newapi.admin.ui.components.AppButton
import com.newapi.admin.ui.components.AppCard
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
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.components.rememberQuotaFormatter
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.theme.GeistMono
import com.newapi.admin.ui.theme.Tone
import com.newapi.admin.ui.theme.colors
import java.time.LocalDate
import kotlinx.coroutines.launch

/** Usage logs of all users with filters, consumption stats and a detail sheet per row. */
@Composable
fun LogsScreen(
    initialUsername: String? = null,
    initialChannelId: Int? = null,
    onBack: (() -> Unit)? = null,
    onOpenUser: (Int) -> Unit,
    onOpenChannel: (Int) -> Unit,
) {
    val container = LocalAppContainer.current
    val viewModel = viewModel(key = "logs:${initialUsername.orEmpty()}:${initialChannelId ?: 0}") {
        LogsViewModel(container, initialUsername, initialChannelId)
    }
    val state by viewModel.logs.collectAsStateWithLifecycle()
    val filter by viewModel.filter.collectAsStateWithLifecycle()
    val stats by viewModel.stats.collectAsStateWithLifecycle()
    val selected by viewModel.selected.collectAsStateWithLifecycle()
    val formatter = rememberQuotaFormatter()
    val snackbarHostState = remember { SnackbarHostState() }
    val listState = rememberLazyListState()
    val scope = rememberCoroutineScope()
    var showFilters by rememberSaveable { mutableStateOf(false) }

    // Messages arrive once from the ViewModel; the text is resolved in composition, then shown.
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

    // A new filter starts at the top; pull-to-refresh and paging keep the position.
    val applyFilter: (LogFilter) -> Unit = { newFilter ->
        viewModel.applyFilter(newFilter)
        scope.launch { listState.scrollToItem(0) }
    }
    LoadMoreEffect(listState, onLoadMore = viewModel::loadMore)

    Scaffold(
        topBar = {
            AppTopBar(
                title = stringResource(R.string.logs_title),
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = stringResource(R.string.common_back))
                        }
                    }
                },
                actions = {
                    IconButton(onClick = { showFilters = true }) {
                        val activeCount = listOf(
                            filter.preset != TimePreset.TODAY,
                            filter.type != LogType.ALL,
                            filter.username.isNotBlank(),
                            filter.tokenName.isNotBlank(),
                            filter.modelName.isNotBlank(),
                            (filter.channelId ?: 0) > 0,
                            filter.group.isNotBlank(),
                            filter.requestId.isNotBlank(),
                        ).count { it }
                        BadgedBox(
                            badge = {
                                if (activeCount > 0) {
                                    Badge(containerColor = AppTheme.colors.primary, contentColor = AppTheme.colors.primaryForeground) {
                                        Text(activeCount.toString())
                                    }
                                }
                            },
                        ) {
                            Icon(Icons.Outlined.FilterList, contentDescription = stringResource(R.string.common_filter))
                        }
                    }
                    IconButton(onClick = viewModel::refresh, enabled = !state.isRefreshing && !state.isInitialLoading) {
                        Icon(Icons.Outlined.Refresh, contentDescription = stringResource(R.string.common_refresh))
                    }
                },
            )
        },
        snackbarHost = { AppSnackbarHost(snackbarHostState) },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding)) {
            FilterChipsRow(filter = filter, onOpenFilters = { showFilters = true }, onChange = applyFilter)
            PullToRefreshBox(
                isRefreshing = state.isRefreshing,
                onRefresh = viewModel::refresh,
                modifier = Modifier.fillMaxSize(),
            ) {
                LazyColumn(
                    state = listState,
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 4.dp, bottom = 16.dp),
                    verticalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    item(key = "stats") {
                        StatsCard(
                            stats = stats,
                            total = state.total.takeIf { !state.isInitialLoading && state.error == null },
                            formatter = formatter,
                            onRetry = viewModel::refresh,
                        )
                    }
                    val firstPageError = state.error
                    when {
                        state.isInitialLoading -> item(key = "loading") {
                            FullScreenLoading(Modifier.fillParentMaxHeight(0.7f))
                        }
                        state.items.isEmpty() && firstPageError != null -> item(key = "error") {
                            FullScreenError(firstPageError, onRetry = viewModel::refresh, modifier = Modifier.fillParentMaxHeight(0.7f))
                        }
                        state.items.isEmpty() -> item(key = "empty") {
                            // Error logs are off by default on the server; say so instead of looking broken.
                            val emptyText = if (filter.type == LogType.ERROR) R.string.logs_empty_errors else R.string.logs_empty
                            EmptyState(stringResource(emptyText), Modifier.fillParentMaxHeight(0.7f))
                        }
                        else -> {
                            items(state.items, key = { it.id }) { entry ->
                                LogRow(entry = entry, formatter = formatter, onClick = { viewModel.select(entry) })
                            }
                            item(key = "footer") {
                                // A failed refresh keeps the old rows; offer the retry here too (paging waits for it).
                                val footerError = state.loadMoreError ?: firstPageError
                                ListFooter(
                                    isLoadingMore = state.isLoadingMore,
                                    error = footerError,
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

    if (showFilters) {
        LogFilterSheet(
            initial = filter,
            onApply = { newFilter ->
                showFilters = false
                applyFilter(newFilter)
            },
            onDismiss = { showFilters = false },
        )
    }

    selected?.let { entry ->
        LogDetailSheet(
            entry = entry,
            formatter = formatter,
            onDismiss = viewModel::clearSelection,
            onOpenUser = { id ->
                viewModel.clearSelection()
                onOpenUser(id)
            },
            onOpenChannel = { id ->
                viewModel.clearSelection()
                onOpenChannel(id)
            },
        )
    }
}

/** Label of the filter's time range: a preset name, or the custom dates. */
@Composable
internal fun timeRangeLabel(filter: LogFilter): String = when (filter.preset) {
    TimePreset.TODAY -> stringResource(R.string.logs_range_today)
    TimePreset.YESTERDAY -> stringResource(R.string.logs_range_yesterday)
    TimePreset.LAST_7_DAYS -> stringResource(R.string.logs_range_last_7_days)
    TimePreset.LAST_30_DAYS -> stringResource(R.string.logs_range_last_30_days)
    TimePreset.CUSTOM -> {
        val start = filter.customStartDay?.let(LocalDate::ofEpochDay)
        val end = filter.customEndDay?.let(LocalDate::ofEpochDay) ?: start
        when {
            start == null -> stringResource(R.string.logs_range_custom)
            start == end -> start.toString()
            else -> stringResource(R.string.logs_range_custom_value, start.toString(), end.toString())
        }
    }
}

/** The time range chip (tap to change) followed by one removable chip per active filter. */
@Composable
private fun FilterChipsRow(filter: LogFilter, onOpenFilters: () -> Unit, onChange: (LogFilter) -> Unit) {
    val removeDescription = stringResource(R.string.logs_chip_remove)
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 16.dp, vertical = 2.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (filter.preset == TimePreset.TODAY) {
            // A button (like the AssistChip it replaces), not a toggle: it only opens the filter sheet.
            AppButton(
                timeRangeLabel(filter),
                onClick = onOpenFilters,
                variant = ButtonVariant.Outline,
                size = ButtonSize.Small,
                leadingIcon = Icons.Outlined.DateRange,
            )
        } else {
            AppRemovableChip(
                label = timeRangeLabel(filter),
                onRemove = { onChange(filter.copy(preset = TimePreset.TODAY, customStartDay = null, customEndDay = null)) },
                removeDescription = removeDescription,
            )
        }
        if (filter.type != LogType.ALL) {
            AppRemovableChip(
                label = stringResource(logTypeLabel(filter.type)),
                onRemove = { onChange(filter.copy(type = LogType.ALL)) },
                removeDescription = removeDescription,
            )
        }
        if (filter.username.isNotBlank()) {
            AppRemovableChip(
                label = stringResource(R.string.logs_chip_user, filter.username.trim()),
                onRemove = { onChange(filter.copy(username = "")) },
                removeDescription = removeDescription,
            )
        }
        if (filter.tokenName.isNotBlank()) {
            AppRemovableChip(
                label = stringResource(R.string.logs_chip_token, filter.tokenName.trim()),
                onRemove = { onChange(filter.copy(tokenName = "")) },
                removeDescription = removeDescription,
            )
        }
        if (filter.modelName.isNotBlank()) {
            AppRemovableChip(
                label = stringResource(R.string.logs_chip_model, filter.modelName.trim()),
                onRemove = { onChange(filter.copy(modelName = "")) },
                removeDescription = removeDescription,
            )
        }
        val channelId = filter.channelId ?: 0
        if (channelId > 0) {
            AppRemovableChip(
                label = stringResource(R.string.logs_chip_channel, channelId),
                onRemove = { onChange(filter.copy(channelId = null)) },
                removeDescription = removeDescription,
            )
        }
        if (filter.group.isNotBlank()) {
            AppRemovableChip(
                label = stringResource(R.string.logs_chip_group, filter.group.trim()),
                onRemove = { onChange(filter.copy(group = "")) },
                removeDescription = removeDescription,
            )
        }
        if (filter.requestId.isNotBlank()) {
            AppRemovableChip(
                label = stringResource(R.string.logs_chip_request_id, filter.requestId.trim()),
                onRemove = { onChange(filter.copy(requestId = "")) },
                removeDescription = removeDescription,
            )
        }
    }
}

/** Consumption of the current filter (`/api/log/stat` sums consume logs only) plus the live RPM/TPM. */
@Composable
private fun StatsCard(stats: LogStatsState, total: Long?, formatter: QuotaFormatter, onRetry: () -> Unit) {
    val colors = AppTheme.colors
    AppCard(modifier = Modifier.fillMaxWidth(), contentPadding = PaddingValues(0.dp)) {
        val stat = stats.stat
        val error = stats.error
        val failed = error != null && stat == null
        if (failed) {
            Row(
                modifier = Modifier.fillMaxWidth().padding(start = 16.dp, end = 8.dp, top = 8.dp, bottom = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(Icons.Outlined.ErrorOutline, contentDescription = null, tint = colors.destructive, modifier = Modifier.size(16.dp))
                Spacer(Modifier.width(8.dp))
                Text(
                    stringResource(R.string.logs_stats_failed),
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.weight(1f),
                )
                AppButton(stringResource(R.string.common_retry), onClick = onRetry, variant = ButtonVariant.Ghost, size = ButtonSize.Small)
            }
        } else {
            // No per-column padding or dividers, so long amounts and TPM values keep their width on 360dp phones.
            Row(
                modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 12.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                StatValue(stringResource(R.string.logs_stats_quota), stat?.let { formatter.formatLog(it.quota) }, Modifier.weight(1.4f))
                StatValue(stringResource(R.string.logs_stats_rpm), stat?.let { Formats.count(it.rpm) }, Modifier.weight(1f))
                StatValue(stringResource(R.string.logs_stats_tpm), stat?.let { Formats.count(it.tpm) }, Modifier.weight(1f))
            }
        }
        HorizontalDivider(color = colors.border)
        Row(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            val caption = if (total != null) {
                stringResource(R.string.logs_stats_caption_with_total, Formats.count(total))
            } else {
                stringResource(R.string.logs_stats_caption)
            }
            Text(
                caption,
                style = MaterialTheme.typography.bodySmall.merge(TextStyle(fontFeatureSettings = "tnum")),
                color = colors.mutedForeground,
                modifier = Modifier.weight(1f),
            )
            // Same condition as before: the spinner belongs to the values, not to the failure message.
            if (!failed && stats.isLoading) {
                CircularProgressIndicator(Modifier.size(14.dp), strokeWidth = 1.5.dp, color = colors.mutedForeground)
            }
        }
    }
}

@Composable
private fun StatValue(label: String, value: String?, modifier: Modifier) {
    Column(modifier, verticalArrangement = Arrangement.spacedBy(2.dp)) {
        Text(label, style = MaterialTheme.typography.labelMedium, color = AppTheme.colors.mutedForeground, maxLines = 1)
        Text(
            value ?: "–",
            style = MaterialTheme.typography.titleLarge.merge(TextStyle(fontFeatureSettings = "tnum")),
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

@Composable
private fun LogRow(entry: LogEntry, formatter: QuotaFormatter, onClick: () -> Unit) {
    val other = remember(entry.other) { entry.parsedOther() }
    val usage = hasUsage(entry.type)
    val timing = hasTiming(entry.type)
    val isError = entry.type == LogType.ERROR
    val secondary = AppTheme.colors.mutedForeground
    val numbers = TextStyle(fontFeatureSettings = "tnum")
    AppCard(
        onClick = onClick,
        modifier = Modifier.fillMaxWidth(),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 14.dp),
    ) {
        // Header, then the usage lines as one tight group, then the timing pills and the message.
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                StatusPill(stringResource(logTypeLabel(entry.type)), logTypeTone(entry.type))
                val title = if (usage) entry.modelName else entry.username
                Text(
                    title.ifBlank { "-" },
                    style = MaterialTheme.typography.titleSmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                Text(
                    Formats.shortDateTime(entry.createdAt),
                    style = MaterialTheme.typography.labelSmall,
                    fontFamily = GeistMono,
                    fontWeight = FontWeight.Normal,
                    color = secondary,
                    maxLines = 1,
                )
            }

            if (usage) {
                Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    val upstreamModel = other?.upstreamModelName
                    if (other?.isModelMapped == true && upstreamModel != null && upstreamModel != entry.modelName) {
                        Text(
                            stringResource(R.string.logs_mapped_to, upstreamModel),
                            style = MaterialTheme.typography.bodySmall,
                            color = secondary,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(
                            listOf(entry.username, entry.tokenName).filter { it.isNotBlank() }.joinToString(" · ").ifEmpty { "-" },
                            style = MaterialTheme.typography.bodyMedium,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.weight(1f),
                        )
                        Text(
                            formatter.formatLog(entry.quota),
                            style = MaterialTheme.typography.titleSmall.merge(numbers),
                            fontWeight = FontWeight.SemiBold,
                            maxLines = 1,
                            modifier = Modifier.padding(start = 12.dp),
                        )
                    }
                    val channel = if (entry.channel > 0) "#${entry.channel} ${entry.channelName}".trim() else ""
                    val channelAndGroup = listOf(channel, entry.group).filter { it.isNotBlank() }.joinToString(" · ")
                    if (channelAndGroup.isNotEmpty()) {
                        Text(channelAndGroup, style = MaterialTheme.typography.bodySmall, color = secondary, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    }
                    val cacheRead = other?.cacheTokens ?: 0
                    val cacheWrite = other?.cacheWriteTokens ?: 0
                    val noTokens = entry.promptTokens == 0L && entry.completionTokens == 0L
                    val tokens = if (noTokens) {
                        "-"
                    } else {
                        Formats.count(entry.promptTokens) + " → " + Formats.count(entry.completionTokens)
                    }
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(
                            tokens,
                            style = MaterialTheme.typography.bodySmall.merge(numbers),
                            color = if (noTokens) secondary else AppTheme.colors.foreground,
                        )
                        if (cacheRead > 0) {
                            Text(stringResource(R.string.logs_cache_read_hint, Formats.count(cacheRead)), style = MaterialTheme.typography.bodySmall.merge(numbers), color = secondary)
                        }
                        if (cacheWrite > 0) {
                            Text("↑ " + Formats.count(cacheWrite), style = MaterialTheme.typography.bodySmall.merge(numbers), color = secondary)
                        }
                    }
                }
            } else if (entry.username.isBlank() && entry.userId > 0) {
                Text("#${entry.userId}", style = MaterialTheme.typography.bodySmall, color = secondary)
            }

            if (timing) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    StatusPill(
                        stringResource(if (entry.isStream) R.string.logs_stream else R.string.logs_non_stream),
                        if (entry.isStream) Tone.Info else Tone.Neutral,
                    )
                    val streamStatus = other?.streamStatus
                    if (entry.isStream && streamStatus != null && streamStatus.status != "ok") {
                        StatusPill(stringResource(R.string.logs_stream_error), Tone.Danger)
                    }
                    if (other?.billingSource == "subscription") StatusPill(stringResource(R.string.logs_billing_subscription), Tone.Info)
                    if (other?.billingCorrected == true) StatusPill(stringResource(R.string.logs_waived), Tone.Info)
                    Spacer(Modifier.weight(1f))
                    if (entry.useTime > 0) {
                        Text(
                            Formats.duration(entry.useTime.toDouble()),
                            style = MaterialTheme.typography.labelMedium.merge(numbers),
                            color = useTimeTone(entry.useTime, entry.completionTokens).colors().content,
                        )
                    }
                    val firstResponseSeconds = other?.firstResponseMillis?.div(1000)
                    if (entry.isStream && firstResponseSeconds != null) {
                        Text(
                            stringResource(R.string.logs_first_token_value, Formats.duration(firstResponseSeconds)),
                            style = MaterialTheme.typography.labelMedium.merge(numbers),
                            color = firstResponseTone(firstResponseSeconds).colors().content,
                        )
                    }
                }
            }

            if (entry.type != LogType.CONSUME && entry.content.isNotBlank()) {
                Text(
                    entry.content,
                    style = MaterialTheme.typography.bodySmall,
                    color = if (isError) Tone.Danger.colors().content else AppTheme.colors.foreground,
                    maxLines = 3,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}
