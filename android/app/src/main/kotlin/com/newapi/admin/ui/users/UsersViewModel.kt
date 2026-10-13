package com.newapi.admin.ui.users

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.newapi.admin.AppContainer
import com.newapi.admin.core.model.AdminUser
import com.newapi.admin.ui.common.PageResult
import com.newapi.admin.ui.common.Pager
import com.newapi.admin.ui.common.PagedState
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

/** Applied list filters. [status]: 1 enabled, 2 disabled, -1 soft-deleted only, null any. */
data class UserFilters(
    val keyword: String = "",
    val role: Int? = null,
    val status: Int? = null,
    val group: String = "",
) {
    val hasColumnFilters: Boolean get() = role != null || status != null || group.isNotEmpty()
    val isActive: Boolean get() = keyword.isNotEmpty() || hasColumnFilters
}

class UsersViewModel(private val container: AppContainer) : ViewModel() {
    private val api = container.api

    /** Search box text; applied to [filters] only on search, so typing does not fire requests. */
    var query by mutableStateOf("")
        private set

    private val mutableFilters = MutableStateFlow(UserFilters())
    val filters: StateFlow<UserFilters> = mutableFilters.asStateFlow()

    private val mutableGroups = MutableStateFlow(GroupList())
    val groups: StateFlow<GroupList> = mutableGroups.asStateFlow()

    private val messageChannel = Channel<UiMessage>(Channel.BUFFERED)
    val messages: Flow<UiMessage> = messageChannel.receiveAsFlow()

    private val pager = Pager<AdminUser>(viewModelScope, PAGE_SIZE, keyOf = { it.id }) { page, pageSize ->
        val filters = mutableFilters.value
        val result = api.users(
            page = page,
            pageSize = pageSize,
            keyword = filters.keyword,
            group = filters.group,
            role = filters.role,
            status = filters.status,
        )
        PageResult(result.items.orEmpty(), result.total)
    }
    val state: StateFlow<PagedState<AdminUser>> = pager.state

    init {
        pager.refresh()
        loadGroups()
        viewModelScope.launch {
            container.events.usersChanged.collect { pager.refresh() }
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

    /** Applies the search box; searching the same keyword again simply reloads. */
    fun search() {
        mutableFilters.update { it.copy(keyword = query.trim()) }
        pager.refresh()
    }

    fun applyFilters(role: Int?, status: Int?, group: String) {
        mutableFilters.update { it.copy(role = role, status = status, group = group) }
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
                val names = api.groups()
                mutableGroups.value = GroupList(names = names)
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableGroups.update { it.copy(isLoading = false, failed = true) }
            }
        }
    }

    private companion object {
        const val PAGE_SIZE = 20
    }
}
