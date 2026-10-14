package com.newapi.admin.ui.tickets

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.newapi.admin.AppContainer
import com.newapi.admin.core.model.Ticket
import com.newapi.admin.ui.common.PageResult
import com.newapi.admin.ui.common.PagedState
import com.newapi.admin.ui.common.Pager
import com.newapi.admin.ui.common.UiMessage
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

/** Applied list filters; [status] is a [com.newapi.admin.core.model.TicketStatus], null for every ticket. */
data class TicketFilters(val keyword: String = "", val status: Int? = null) {
    val isActive: Boolean get() = keyword.isNotEmpty() || status != null
}

class TicketsViewModel(private val container: AppContainer) : ViewModel() {
    private val api = container.api

    /** Search box text; applied only on search so typing does not fire requests. */
    var query by mutableStateOf("")
        private set

    private val mutableFilters = MutableStateFlow(TicketFilters())
    val filters: StateFlow<TicketFilters> = mutableFilters.asStateFlow()

    private val messageChannel = Channel<UiMessage>(Channel.BUFFERED)
    val messages: Flow<UiMessage> = messageChannel.receiveAsFlow()

    private val pager = Pager<Ticket>(viewModelScope, PAGE_SIZE, keyOf = { it.id }) { page, pageSize ->
        val filters = mutableFilters.value
        val result = api.tickets(page, pageSize, filters.keyword, filters.status)
        PageResult(result.items.orEmpty(), result.total)
    }
    val state: StateFlow<PagedState<Ticket>> = pager.state

    init {
        pager.refresh()
        // The detail screen announces the ticket it opened, answered or moved. The row is patched in place so the
        // scroll position survives, and leaves a status filter it no longer matches, as it would on a reload.
        viewModelScope.launch {
            container.events.ticketsChanged.collect { changed ->
                if (changed == null) {
                    pager.refresh()
                    return@collect
                }
                val status = mutableFilters.value.status
                pager.updateItems { items ->
                    if (status != null && changed.status != status) {
                        items.filter { it.id != changed.id }
                    } else {
                        items.map { if (it.id == changed.id) changed.copy(username = changed.username.ifEmpty { it.username }) else it }
                    }
                }
            }
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
        pager.reset()
    }

    fun setStatus(status: Int?) {
        if (status == mutableFilters.value.status) return
        mutableFilters.update { it.copy(status = status) }
        pager.reset()
    }

    /** Pull-to-refresh: the rows and the per-status counts. */
    fun refresh() {
        pager.refresh()
        container.refreshTicketStats()
    }

    fun loadMore() = pager.loadMore()

    private companion object {
        const val PAGE_SIZE = 20
    }
}
