package com.newapi.admin.ui.redemptions

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.newapi.admin.AppContainer
import com.newapi.admin.R
import com.newapi.admin.core.model.Redemption
import com.newapi.admin.core.model.RedemptionStatus
import com.newapi.admin.ui.common.UiMessage
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class RedemptionDetailState(
    val code: Redemption? = null,
    val isLoading: Boolean = true,
    /** Pull-to-refresh with the code already shown. */
    val isRefreshing: Boolean = false,
    /** Failure of the last load; shown full screen when no code is loaded yet. */
    val error: Throwable? = null,
    val isChangingStatus: Boolean = false,
    val isDeleting: Boolean = false,
    /** The code was deleted; the screen leaves. */
    val deleted: Boolean = false,
)

class RedemptionDetailViewModel(private val container: AppContainer, private val codeId: Int) : ViewModel() {
    private val api = container.api

    private val mutableState = MutableStateFlow(RedemptionDetailState())
    val state: StateFlow<RedemptionDetailState> = mutableState.asStateFlow()

    private val messageChannel = Channel<UiMessage>(Channel.BUFFERED)
    val messages: Flow<UiMessage> = messageChannel.receiveAsFlow()

    private var loadJob: Job? = null

    init {
        load(showIndicator = false)
        // The edit screen announces its save; reload quietly to pick it up.
        viewModelScope.launch {
            container.events.redemptionsChanged.collect { id ->
                if (id != null && id != codeId) return@collect
                if (!mutableState.value.deleted) load(showIndicator = false)
            }
        }
    }

    fun refresh() = load(showIndicator = true)

    private fun load(showIndicator: Boolean) {
        loadJob?.cancel()
        mutableState.update {
            it.copy(isLoading = it.code == null, isRefreshing = showIndicator && it.code != null, error = null)
        }
        loadJob = viewModelScope.launch {
            try {
                val code = api.redemption(codeId)
                mutableState.update { it.copy(code = code, isLoading = false, isRefreshing = false) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isLoading = false, isRefreshing = false, error = e) }
                if (mutableState.value.code != null) messageChannel.send(UiMessage.Error(e))
            }
        }
    }

    fun setEnabled(enabled: Boolean) {
        if (mutableState.value.isChangingStatus) return
        mutableState.update { it.copy(isChangingStatus = true) }
        viewModelScope.launch {
            try {
                val saved = api.setRedemptionEnabled(codeId, enabled)
                val status = if (enabled) RedemptionStatus.ENABLED else RedemptionStatus.DISABLED
                mutableState.update { it.copy(code = saved ?: it.code?.copy(status = status)) }
                messageChannel.send(UiMessage.Res(if (enabled) R.string.redemptions_msg_enabled else R.string.redemptions_msg_disabled))
                container.events.notifyRedemptionsChanged(codeId)
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                messageChannel.send(UiMessage.Error(e))
            } finally {
                mutableState.update { it.copy(isChangingStatus = false) }
            }
        }
    }

    fun delete() {
        val current = mutableState.value
        if (current.isDeleting || current.deleted) return
        mutableState.update { it.copy(isDeleting = true) }
        viewModelScope.launch {
            try {
                api.deleteRedemption(codeId)
                mutableState.update { it.copy(deleted = true) }
                container.events.notifyRedemptionsChanged(codeId)
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isDeleting = false) }
                messageChannel.send(UiMessage.Error(e))
            }
        }
    }
}
