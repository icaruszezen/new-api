package com.newapi.admin.ui.redemptions

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.newapi.admin.AppContainer
import com.newapi.admin.R
import com.newapi.admin.core.format.Formats
import com.newapi.admin.core.model.Redemption
import com.newapi.admin.core.model.RedemptionStatusFilter
import com.newapi.admin.core.net.ApiError
import com.newapi.admin.ui.common.PageResult
import com.newapi.admin.ui.common.PagedState
import com.newapi.admin.ui.common.Pager
import com.newapi.admin.ui.common.UiMessage
import kotlinx.coroutines.CancellationException
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

/** Applied list filters. The keyword matches an exact id or the start of a name, never the code itself. */
data class RedemptionFilters(val keyword: String = "", val status: RedemptionStatusFilter = RedemptionStatusFilter.ALL) {
    val isActive: Boolean get() = keyword.isNotEmpty() || status != RedemptionStatusFilter.ALL
}

class RedemptionsViewModel(private val container: AppContainer) : ViewModel() {
    private val api = container.api

    /** Search box text; applied only on search so typing does not fire requests. */
    var query by mutableStateOf("")
        private set

    private val mutableFilters = MutableStateFlow(RedemptionFilters())
    val filters: StateFlow<RedemptionFilters> = mutableFilters.asStateFlow()

    private val mutableDeletingInvalid = MutableStateFlow(false)
    val isDeletingInvalid: StateFlow<Boolean> = mutableDeletingInvalid.asStateFlow()

    private val messageChannel = Channel<UiMessage>(Channel.BUFFERED)
    val messages: Flow<UiMessage> = messageChannel.receiveAsFlow()

    private val pager = Pager<Redemption>(viewModelScope, PAGE_SIZE, keyOf = { it.id }) { page, pageSize ->
        val filters = mutableFilters.value
        val result = api.redemptions(page, pageSize, filters.keyword, filters.status)
        PageResult(result.items.orEmpty(), result.total)
    }
    val state: StateFlow<PagedState<Redemption>> = pager.state

    init {
        pager.refresh()
        // A changed or deleted code is patched in place so the scroll position survives; null (a create) reloads from the top.
        viewModelScope.launch {
            container.events.redemptionsChanged.collect { id ->
                if (id == null) {
                    pager.refresh()
                    return@collect
                }
                if (pager.state.value.items.none { it.id == id }) return@collect
                try {
                    val fresh = api.redemption(id)
                    pager.updateItems { items -> items.map { if (it.id == id) fresh else it } }
                } catch (e: CancellationException) {
                    throw e
                } catch (e: ApiError.Business) {
                    // The server answers a deleted code with success:false (record not found).
                    pager.updateItems { items -> items.filter { it.id != id } }
                } catch (e: Throwable) {
                    // Keep the row as it was; the next refresh corrects it.
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

    fun setStatus(status: RedemptionStatusFilter) {
        if (status == mutableFilters.value.status) return
        mutableFilters.update { it.copy(status = status) }
        pager.reset()
    }

    fun refresh() = pager.refresh()

    fun loadMore() = pager.loadMore()

    /** `DELETE /api/redemption/invalid`: every used, disabled and expired code on the server, not only the listed ones. */
    fun deleteInvalid() {
        if (mutableDeletingInvalid.value) return
        mutableDeletingInvalid.value = true
        viewModelScope.launch {
            try {
                val deleted = api.deleteInvalidRedemptions()
                messageChannel.send(UiMessage.Res(R.string.redemptions_msg_deleted_invalid, listOf(Formats.count(deleted))))
                pager.refresh()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                messageChannel.send(UiMessage.Error(e))
            } finally {
                mutableDeletingInvalid.value = false
            }
        }
    }

    private companion object {
        const val PAGE_SIZE = 20
    }
}
