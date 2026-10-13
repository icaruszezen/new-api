package com.newapi.admin.ui.channels

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.newapi.admin.AppContainer
import com.newapi.admin.R
import com.newapi.admin.core.model.Channel
import com.newapi.admin.core.model.ChannelQuery
import com.newapi.admin.core.model.ChannelStatus
import com.newapi.admin.core.model.ChannelStatusFilter
import com.newapi.admin.ui.common.PageResult
import com.newapi.admin.ui.common.PagedState
import com.newapi.admin.ui.common.Pager
import com.newapi.admin.ui.common.UiMessage
import kotlin.math.roundToInt
import kotlinx.coroutines.CancellationException
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
import kotlinx.coroutines.channels.Channel as MessageChannel

/** Channel ids with a request in flight, so each row disables the matching control. */
data class RowActivity(
    val toggling: Set<Int> = emptySet(),
    val testing: Set<Int> = emptySet(),
    val updatingBalance: Set<Int> = emptySet(),
)

internal val ChannelQuery.hasColumnFilters: Boolean
    get() = status != ChannelStatusFilter.ALL || type != null || group.isNotEmpty() || model.isNotEmpty()

class ChannelsViewModel(private val container: AppContainer) : ViewModel() {
    private val api = container.api

    /** Search box text; applied only on search so typing does not fire requests. Snapshot state keeps the text field synchronous. */
    var query by mutableStateOf("")
        private set

    private val mutableFilters = MutableStateFlow(ChannelQuery())
    val filters: StateFlow<ChannelQuery> = mutableFilters.asStateFlow()

    /** Channel count per type id from the last first page (computed by the server without the type filter). */
    private val mutableTypeCounts = MutableStateFlow<Map<String, Long>?>(null)
    val typeCounts: StateFlow<Map<String, Long>?> = mutableTypeCounts.asStateFlow()

    private val mutableGroups = MutableStateFlow(GroupOptions())
    val groups: StateFlow<GroupOptions> = mutableGroups.asStateFlow()

    private val mutableActivity = MutableStateFlow(RowActivity())
    val activity: StateFlow<RowActivity> = mutableActivity.asStateFlow()

    private val messageChannel = MessageChannel<UiMessage>(MessageChannel.BUFFERED)
    val messages: Flow<UiMessage> = messageChannel.receiveAsFlow()

    private val pager = Pager<Channel>(viewModelScope, PAGE_SIZE, keyOf = { it.id }) { page, pageSize ->
        val result = api.channels(mutableFilters.value, page, pageSize)
        if (page == 1) mutableTypeCounts.value = result.typeCounts
        PageResult(result.items.orEmpty(), result.total)
    }
    val state: StateFlow<PagedState<Channel>> = pager.state

    init {
        pager.refresh()
        viewModelScope.launch {
            container.events.channelsChanged.collect { pager.refresh() }
        }
        // A failed reload while rows are shown keeps the rows; report it once.
        viewModelScope.launch {
            pager.state
                .map { state -> state.error?.takeIf { state.items.isNotEmpty() } }
                .distinctUntilChanged()
                .filterNotNull()
                .collect { messageChannel.send(UiMessage.Error(it)) }
        }
    }

    fun onQueryChange(value: String) {
        query = value
    }

    fun search() {
        mutableFilters.update { it.copy(keyword = query.trim()) }
        pager.refresh()
    }

    fun applyFilters(filters: ChannelQuery) {
        mutableFilters.value = filters.copy(keyword = mutableFilters.value.keyword)
        pager.refresh()
    }

    fun refresh() = pager.refresh()

    fun loadMore() = pager.loadMore()

    /** Loads the group list once; called again when the filter sheet opens after a failure. */
    fun loadGroups() {
        val current = mutableGroups.value
        if (current.isLoading || (current.names.isNotEmpty() && !current.failed)) return
        mutableGroups.update { it.copy(isLoading = true, failed = false) }
        viewModelScope.launch {
            try {
                mutableGroups.value = GroupOptions(names = api.groups())
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableGroups.update { it.copy(isLoading = false, failed = true) }
            }
        }
    }

    /** Optimistic: the row flips immediately and flips back if the server refuses. */
    fun setEnabled(channel: Channel, enabled: Boolean) {
        if (channel.id in mutableActivity.value.toggling) return
        val previousStatus = channel.status
        val newStatus = if (enabled) ChannelStatus.ENABLED else ChannelStatus.MANUALLY_DISABLED
        mutableActivity.update { it.copy(toggling = it.toggling + channel.id) }
        pager.updateItems { items -> items.map { if (it.id == channel.id) it.copy(status = newStatus) else it } }
        viewModelScope.launch {
            try {
                api.setChannelStatus(channel.id, enabled)
                messageChannel.send(UiMessage.Res(if (enabled) R.string.channels_enabled_message else R.string.channels_disabled_message))
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                // Only undo our own change; a refresh may already have replaced the row.
                pager.updateItems { items ->
                    items.map { if (it.id == channel.id && it.status == newStatus) it.copy(status = previousStatus) else it }
                }
                messageChannel.send(UiMessage.Error(e))
            } finally {
                mutableActivity.update { it.copy(toggling = it.toggling - channel.id) }
            }
        }
    }

    fun test(channel: Channel) {
        if (channel.id in mutableActivity.value.testing) return
        mutableActivity.update { it.copy(testing = it.testing + channel.id) }
        viewModelScope.launch {
            try {
                val result = api.testChannel(channel.id)
                if (result.success) {
                    // A successful single test updates response_time and test_time on the server.
                    val responseMs = (result.timeSeconds * 1000).roundToInt().coerceAtLeast(1)
                    val now = System.currentTimeMillis() / 1000
                    pager.updateItems { items ->
                        items.map { if (it.id == channel.id) it.copy(responseTime = responseMs, testTime = now) else it }
                    }
                    messageChannel.send(
                        UiMessage.Res(R.string.channels_test_success, listOf(channel.displayName, formatSeconds(result.timeSeconds))),
                    )
                } else if (result.message.isBlank()) {
                    messageChannel.send(UiMessage.Res(R.string.channels_test_failed_no_reason, listOf(channel.displayName)))
                } else {
                    messageChannel.send(UiMessage.Res(R.string.channels_test_failed, listOf(channel.displayName, result.message)))
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                messageChannel.send(UiMessage.Error(e))
            } finally {
                mutableActivity.update { it.copy(testing = it.testing - channel.id) }
            }
        }
    }

    fun updateBalance(channel: Channel) {
        if (channel.id in mutableActivity.value.updatingBalance) return
        mutableActivity.update { it.copy(updatingBalance = it.updatingBalance + channel.id) }
        viewModelScope.launch {
            try {
                val balance = api.updateChannelBalance(channel.id)
                if (balance.isNaN()) {
                    messageChannel.send(UiMessage.Res(R.string.channels_balance_updated_no_value))
                } else {
                    val now = System.currentTimeMillis() / 1000
                    pager.updateItems { items ->
                        items.map { if (it.id == channel.id) it.copy(balance = balance, balanceUpdatedTime = now) else it }
                    }
                    val text = container.quotaFormatter.value.formatUsd(balance)
                    messageChannel.send(UiMessage.Res(R.string.channels_balance_updated, listOf(text)))
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                messageChannel.send(UiMessage.Error(e))
            } finally {
                mutableActivity.update { it.copy(updatingBalance = it.updatingBalance - channel.id) }
            }
        }
    }

    private companion object {
        const val PAGE_SIZE = 20
    }
}
