package com.newapi.admin.ui.logs

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.newapi.admin.AppContainer
import com.newapi.admin.core.model.LogEntry
import com.newapi.admin.core.model.LogQuery
import com.newapi.admin.core.model.LogStat
import com.newapi.admin.core.model.LogType
import com.newapi.admin.ui.common.PageResult
import com.newapi.admin.ui.common.PagedState
import com.newapi.admin.ui.common.Pager
import com.newapi.admin.ui.common.UiMessage
import java.io.Serializable
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.filterNotNull
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

enum class TimePreset { TODAY, YESTERDAY, LAST_7_DAYS, LAST_30_DAYS, CUSTOM }

/**
 * The filters of the logs screen. Relative presets become absolute seconds in [toQuery], which runs
 * once per refresh so that every page of one list uses the same range.
 *
 * Serializable so the filter sheet can keep its draft in rememberSaveable across rotation.
 */
data class LogFilter(
    val preset: TimePreset = TimePreset.TODAY,
    /** First day of a [TimePreset.CUSTOM] range (local date as epoch day). */
    val customStartDay: Long? = null,
    /** Last day (inclusive) of a [TimePreset.CUSTOM] range; null means the same day as the start. */
    val customEndDay: Long? = null,
    val type: Int = LogType.ALL,
    val username: String = "",
    val tokenName: String = "",
    val modelName: String = "",
    val channelId: Int? = null,
    val group: String = "",
    val requestId: String = "",
) : Serializable {

    /** Day boundaries use the device time zone, like the web's dayjs local time. */
    fun toQuery(nowMillis: Long, zone: ZoneId): LogQuery {
        val today = Instant.ofEpochMilli(nowMillis).atZone(zone).toLocalDate()
        val todayStart = today.atStartOfDay(zone).toEpochSecond()
        // Open-ended ranges stop an hour ahead, like the web, so rows written while paging still match.
        val openEnd = nowMillis / 1000 + OPEN_END_AHEAD_SECONDS
        val (start, end) = when (preset) {
            TimePreset.TODAY -> todayStart to openEnd
            TimePreset.YESTERDAY -> today.minusDays(1).atStartOfDay(zone).toEpochSecond() to todayStart - 1
            TimePreset.LAST_7_DAYS -> today.minusDays(6).atStartOfDay(zone).toEpochSecond() to openEnd
            TimePreset.LAST_30_DAYS -> today.minusDays(29).atStartOfDay(zone).toEpochSecond() to openEnd
            TimePreset.CUSTOM -> {
                val first = LocalDate.ofEpochDay(customStartDay ?: today.toEpochDay())
                val last = customEndDay?.let(LocalDate::ofEpochDay)?.takeIf { !it.isBefore(first) } ?: first
                // The end day is inclusive: stop one second before the next local midnight.
                first.atStartOfDay(zone).toEpochSecond() to last.plusDays(1).atStartOfDay(zone).toEpochSecond() - 1
            }
        }
        return LogQuery(
            type = type,
            startTimestamp = start,
            endTimestamp = end,
            username = username.trim(),
            tokenName = tokenName.trim(),
            modelName = modelName.trim(),
            channel = channelId?.takeIf { it > 0 },
            group = group.trim(),
            requestId = requestId.trim(),
        )
    }

    private companion object {
        const val OPEN_END_AHEAD_SECONDS = 3600L
    }
}

/** `GET /api/log/stat` for the current filter. A failure here never blocks the list. */
data class LogStatsState(
    val stat: LogStat? = null,
    val isLoading: Boolean = true,
    val error: Throwable? = null,
)

class LogsViewModel(
    container: AppContainer,
    initialUsername: String?,
    initialChannelId: Int?,
) : ViewModel() {
    private val api = container.api

    private val mutableFilter = MutableStateFlow(
        LogFilter(username = initialUsername?.trim().orEmpty(), channelId = initialChannelId?.takeIf { it > 0 }),
    )
    val filter: StateFlow<LogFilter> = mutableFilter.asStateFlow()

    private val mutableStats = MutableStateFlow(LogStatsState())
    val stats: StateFlow<LogStatsState> = mutableStats.asStateFlow()

    private val mutableSelected = MutableStateFlow<LogEntry?>(null)

    /** The row shown in the detail sheet. */
    val selected: StateFlow<LogEntry?> = mutableSelected.asStateFlow()

    private val messageChannel = Channel<UiMessage>(Channel.BUFFERED)
    val messages: Flow<UiMessage> = messageChannel.receiveAsFlow()

    /** Absolute query of the list being shown; fixed between refreshes so later pages match page 1. */
    private var activeQuery: LogQuery = mutableFilter.value.toQuery(System.currentTimeMillis(), ZoneId.systemDefault())
    private var statsJob: Job? = null

    private val pager = Pager<LogEntry>(viewModelScope, pageSize = PAGE_SIZE, keyOf = { it.id }) { page, pageSize ->
        val result = api.logs(activeQuery, page, pageSize)
        PageResult(result.items.orEmpty(), result.total)
    }
    val logs: StateFlow<PagedState<LogEntry>> = pager.state

    init {
        refresh()
        // A failed refresh with rows still on screen is reported once; with no rows the screen shows it full size.
        viewModelScope.launch {
            pager.state.map { it.error }.distinctUntilChanged().filterNotNull().collect { error ->
                if (pager.state.value.items.isNotEmpty()) messageChannel.send(UiMessage.Error(error))
            }
        }
    }

    /** Reloads the list and the stats with the current filter; relative ranges move to "now". */
    fun refresh() {
        val query = mutableFilter.value.toQuery(System.currentTimeMillis(), ZoneId.systemDefault())
        activeQuery = query
        pager.refresh()
        statsJob?.cancel()
        mutableStats.update { it.copy(isLoading = true, error = null) }
        statsJob = viewModelScope.launch {
            try {
                mutableStats.value = LogStatsState(stat = api.logStat(query), isLoading = false)
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableStats.value = LogStatsState(stat = null, isLoading = false, error = e)
            }
        }
    }

    fun applyFilter(filter: LogFilter) {
        // Totals of the previous filter would be misleading while the new ones load.
        if (filter != mutableFilter.value) mutableStats.value = LogStatsState()
        mutableFilter.value = filter
        refresh()
    }

    fun loadMore() = pager.loadMore()

    fun select(entry: LogEntry) {
        mutableSelected.value = entry
    }

    fun clearSelection() {
        mutableSelected.value = null
    }

    private companion object {
        const val PAGE_SIZE = 20
    }
}
