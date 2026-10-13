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
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.DateRange
import androidx.compose.material.icons.filled.FilterList
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.AssistChip
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.InputChip
import androidx.compose.material3.InputChipDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
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
import androidx.compose.ui.text.font.FontFamily
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
import com.newapi.admin.ui.components.EmptyState
import com.newapi.admin.ui.components.FullScreenError
import com.newapi.admin.ui.components.FullScreenLoading
import com.newapi.admin.ui.components.ListFooter
import com.newapi.admin.ui.components.LoadMoreEffect
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.components.rememberQuotaFormatter
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
            TopAppBar(
                title = { Text(stringResource(R.string.logs_title)) },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_back))
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
                        BadgedBox(badge = { if (activeCount > 0) Badge { Text(activeCount.toString()) } }) {
                            Icon(Icons.Filled.FilterList, contentDescription = stringResource(R.string.common_filter))
                        }
                    }
                    IconButton(onClick = viewModel::refresh, enabled = !state.isRefreshing && !state.isInitialLoading) {
                        Icon(Icons.Filled.Refresh, contentDescription = stringResource(R.string.common_refresh))
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
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
                    verticalArrangement = Arrangement.spacedBy(8.dp),
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
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 16.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (filter.preset == TimePreset.TODAY) {
            AssistChip(
                onClick = onOpenFilters,
                label = { Text(timeRangeLabel(filter)) },
                leadingIcon = { Icon(Icons.Filled.DateRange, contentDescription = null, modifier = Modifier.size(18.dp)) },
            )
        } else {
            RemovableChip(timeRangeLabel(filter)) {
                onChange(filter.copy(preset = TimePreset.TODAY, customStartDay = null, customEndDay = null))
            }
        }
        if (filter.type != LogType.ALL) {
            RemovableChip(stringResource(logTypeLabel(filter.type))) { onChange(filter.copy(type = LogType.ALL)) }
        }
        if (filter.username.isNotBlank()) {
            RemovableChip(stringResource(R.string.logs_chip_user, filter.username.trim())) { onChange(filter.copy(username = "")) }
        }
        if (filter.tokenName.isNotBlank()) {
            RemovableChip(stringResource(R.string.logs_chip_token, filter.tokenName.trim())) { onChange(filter.copy(tokenName = "")) }
        }
        if (filter.modelName.isNotBlank()) {
            RemovableChip(stringResource(R.string.logs_chip_model, filter.modelName.trim())) { onChange(filter.copy(modelName = "")) }
        }
        val channelId = filter.channelId ?: 0
        if (channelId > 0) {
            RemovableChip(stringResource(R.string.logs_chip_channel, channelId)) { onChange(filter.copy(channelId = null)) }
        }
        if (filter.group.isNotBlank()) {
            RemovableChip(stringResource(R.string.logs_chip_group, filter.group.trim())) { onChange(filter.copy(group = "")) }
        }
        if (filter.requestId.isNotBlank()) {
            RemovableChip(stringResource(R.string.logs_chip_request_id, filter.requestId.trim())) { onChange(filter.copy(requestId = "")) }
        }
    }
}

@Composable
private fun RemovableChip(label: String, onRemove: () -> Unit) {
    InputChip(
        selected = true,
        onClick = onRemove,
        label = { Text(label, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.widthIn(max = 220.dp)) },
        trailingIcon = {
            Icon(
                Icons.Filled.Close,
                contentDescription = stringResource(R.string.logs_chip_remove),
                modifier = Modifier.size(InputChipDefaults.IconSize),
            )
        },
    )
}

/** Consumption of the current filter (`/api/log/stat` sums consume logs only) plus the live RPM/TPM. */
@Composable
private fun StatsCard(stats: LogStatsState, total: Long?, formatter: QuotaFormatter, onRetry: () -> Unit) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.secondaryContainer),
    ) {
        Column(Modifier.padding(horizontal = 16.dp, vertical = 12.dp)) {
            val stat = stats.stat
            val error = stats.error
            if (error != null && stat == null) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        stringResource(R.string.logs_stats_failed),
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSecondaryContainer,
                        modifier = Modifier.weight(1f),
                    )
                    TextButton(onClick = onRetry) { Text(stringResource(R.string.common_retry)) }
                }
            } else {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    StatValue(stringResource(R.string.logs_stats_quota), stat?.let { formatter.formatLog(it.quota) }, Modifier.weight(1.4f))
                    StatValue(stringResource(R.string.logs_stats_rpm), stat?.let { Formats.count(it.rpm) }, Modifier.weight(1f))
                    StatValue(stringResource(R.string.logs_stats_tpm), stat?.let { Formats.count(it.tpm) }, Modifier.weight(1f))
                    if (stats.isLoading) CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp)
                }
            }
            val caption = if (total != null) {
                stringResource(R.string.logs_stats_caption_with_total, Formats.count(total))
            } else {
                stringResource(R.string.logs_stats_caption)
            }
            Text(
                caption,
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSecondaryContainer.copy(alpha = 0.75f),
                modifier = Modifier.padding(top = 6.dp),
            )
        }
    }
}

@Composable
private fun StatValue(label: String, value: String?, modifier: Modifier) {
    Column(modifier) {
        Text(label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSecondaryContainer)
        Text(
            value ?: "–",
            style = MaterialTheme.typography.titleMedium.merge(TextStyle(fontFeatureSettings = "tnum")),
            fontWeight = FontWeight.SemiBold,
            color = MaterialTheme.colorScheme.onSecondaryContainer,
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
    val secondary = MaterialTheme.colorScheme.onSurfaceVariant
    val numbers = TextStyle(fontFeatureSettings = "tnum")
    Card(
        onClick = onClick,
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow),
    ) {
        Column(Modifier.padding(horizontal = 16.dp, vertical = 12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
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
                    fontFamily = FontFamily.Monospace,
                    color = secondary,
                )
            }

            val upstreamModel = other?.upstreamModelName
            if (usage && other?.isModelMapped == true && upstreamModel != null && upstreamModel != entry.modelName) {
                Text(
                    stringResource(R.string.logs_mapped_to, upstreamModel),
                    style = MaterialTheme.typography.bodySmall,
                    color = secondary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }

            if (usage) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        listOf(entry.username, entry.tokenName).filter { it.isNotBlank() }.joinToString(" · ").ifEmpty { "-" },
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f),
                    )
                    Text(
                        formatter.formatLog(entry.quota),
                        style = MaterialTheme.typography.titleSmall.merge(numbers),
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                    )
                }
                val channel = if (entry.channel > 0) "#${entry.channel} ${entry.channelName}".trim() else ""
                val channelAndGroup = listOf(channel, entry.group).filter { it.isNotBlank() }.joinToString(" · ")
                if (channelAndGroup.isNotEmpty()) {
                    Text(channelAndGroup, style = MaterialTheme.typography.bodySmall, color = secondary, maxLines = 1, overflow = TextOverflow.Ellipsis)
                }
                val cacheRead = other?.cacheTokens ?: 0
                val cacheWrite = other?.cacheWriteTokens ?: 0
                val tokens = if (entry.promptTokens == 0L && entry.completionTokens == 0L) {
                    "-"
                } else {
                    Formats.count(entry.promptTokens) + " → " + Formats.count(entry.completionTokens)
                }
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(tokens, style = MaterialTheme.typography.bodySmall.merge(numbers))
                    if (cacheRead > 0) {
                        Text(stringResource(R.string.logs_cache_read_hint, Formats.count(cacheRead)), style = MaterialTheme.typography.bodySmall.merge(numbers), color = secondary)
                    }
                    if (cacheWrite > 0) {
                        Text("↑ " + Formats.count(cacheWrite), style = MaterialTheme.typography.bodySmall.merge(numbers), color = secondary)
                    }
                }
            } else if (entry.username.isBlank() && entry.userId > 0) {
                Text("#${entry.userId}", style = MaterialTheme.typography.bodySmall, color = secondary)
            }

            if (timing) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
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
                    color = if (isError) Tone.Danger.colors().content else MaterialTheme.colorScheme.onSurface,
                    maxLines = 3,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}
