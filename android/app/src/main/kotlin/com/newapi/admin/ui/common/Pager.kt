package com.newapi.admin.ui.common

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class PageResult<T>(val items: List<T>, val total: Long)

data class PagedState<T>(
    val items: List<T> = emptyList(),
    val total: Long = 0,
    /** True until the first page has loaded (or failed). */
    val isInitialLoading: Boolean = true,
    /** Pull-to-refresh or filter change with items already shown. */
    val isRefreshing: Boolean = false,
    val isLoadingMore: Boolean = false,
    /** Failure of the first page; shown full screen when there are no items. */
    val error: Throwable? = null,
    /** Failure of a later page; shown in the list footer with a retry button. */
    val loadMoreError: Throwable? = null,
    val endReached: Boolean = false,
)

/**
 * Infinite-scroll paging over a 1-based page API. Items are de-duplicated by [keyOf] because new
 * rows (for example fresh logs) shift later pages, and LazyColumn crashes on duplicate keys.
 */
class Pager<T>(
    private val scope: CoroutineScope,
    private val pageSize: Int = 20,
    private val keyOf: (T) -> Any,
    private val fetch: suspend (page: Int, pageSize: Int) -> PageResult<T>,
) {
    private val mutableState = MutableStateFlow(PagedState<T>())
    val state: StateFlow<PagedState<T>> = mutableState.asStateFlow()

    private var nextPage = 1
    private var job: Job? = null

    /** Reloads from page 1, keeping the current items visible until the new page arrives. */
    fun refresh() {
        job?.cancel()
        mutableState.update {
            it.copy(
                isInitialLoading = it.items.isEmpty(),
                isRefreshing = it.items.isNotEmpty(),
                isLoadingMore = false,
                error = null,
                loadMoreError = null,
            )
        }
        job = scope.launch {
            try {
                val result = fetch(1, pageSize)
                val items = result.items.distinctBy(keyOf)
                nextPage = 2
                mutableState.value = PagedState(
                    items = items,
                    total = result.total,
                    isInitialLoading = false,
                    endReached = isEnd(result, items.size),
                )
            } catch (e: kotlinx.coroutines.CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isInitialLoading = false, isRefreshing = false, error = e) }
            }
        }
    }

    fun loadMore() {
        val current = mutableState.value
        if (current.endReached || current.isLoadingMore || current.isRefreshing || current.isInitialLoading) return
        if (current.error != null) return
        mutableState.update { it.copy(isLoadingMore = true, loadMoreError = null) }
        val page = nextPage
        job = scope.launch {
            try {
                val result = fetch(page, pageSize)
                nextPage = page + 1
                mutableState.update { state ->
                    val seen = state.items.mapTo(HashSet()) { keyOf(it) }
                    val merged = state.items + result.items.filter { seen.add(keyOf(it)) }
                    state.copy(
                        items = merged,
                        total = result.total,
                        isLoadingMore = false,
                        endReached = isEnd(result, merged.size) || result.items.isEmpty(),
                    )
                }
            } catch (e: kotlinx.coroutines.CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isLoadingMore = false, loadMoreError = e) }
            }
        }
    }

    /** Applies a local edit (for example after toggling a channel) without reloading. */
    fun updateItems(transform: (List<T>) -> List<T>) {
        mutableState.update { it.copy(items = transform(it.items)) }
    }

    private fun isEnd(result: PageResult<T>, loaded: Int): Boolean =
        result.items.size < pageSize || loaded >= result.total
}
