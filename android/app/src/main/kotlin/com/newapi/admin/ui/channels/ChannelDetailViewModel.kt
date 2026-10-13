package com.newapi.admin.ui.channels

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.newapi.admin.AppContainer
import com.newapi.admin.R
import com.newapi.admin.core.model.Channel
import com.newapi.admin.core.model.ChannelStatus
import com.newapi.admin.core.model.ChannelTestResult
import com.newapi.admin.ui.common.UiMessage
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.channels.Channel as MessageChannel

data class ChannelDetailState(
    val channel: Channel? = null,
    val isLoading: Boolean = true,
    /** Pull-to-refresh with the channel already shown. */
    val isRefreshing: Boolean = false,
    /** Failure of the last load; shown full screen when no channel is loaded yet. */
    val error: Throwable? = null,
    val isChangingStatus: Boolean = false,
    val isUpdatingBalance: Boolean = false,
    val isDeleting: Boolean = false,
    /** The channel was deleted; the screen leaves. */
    val deleted: Boolean = false,
)

/** The test dialog: a run in progress, its result, or a request failure. */
data class ChannelTestRun(
    val isRunning: Boolean = false,
    val result: ChannelTestResult? = null,
    val error: Throwable? = null,
)

class ChannelDetailViewModel(private val container: AppContainer, private val channelId: Int) : ViewModel() {
    private val api = container.api

    private val mutableState = MutableStateFlow(ChannelDetailState())
    val state: StateFlow<ChannelDetailState> = mutableState.asStateFlow()

    private val mutableTest = MutableStateFlow(ChannelTestRun())
    val test: StateFlow<ChannelTestRun> = mutableTest.asStateFlow()

    private val messageChannel = MessageChannel<UiMessage>(MessageChannel.BUFFERED)
    val messages: Flow<UiMessage> = messageChannel.receiveAsFlow()

    private var loadJob: Job? = null
    private var testJob: Job? = null

    init {
        load(showIndicator = false)
        // Mutations here and on the edit screen announce themselves; reload quietly to pick them up.
        viewModelScope.launch {
            container.events.channelsChanged.collect {
                if (!mutableState.value.deleted) load(showIndicator = false)
            }
        }
    }

    fun refresh() = load(showIndicator = true)

    private fun load(showIndicator: Boolean) {
        loadJob?.cancel()
        mutableState.update {
            it.copy(isLoading = it.channel == null, isRefreshing = showIndicator && it.channel != null, error = null)
        }
        loadJob = viewModelScope.launch {
            try {
                val channel = api.channel(channelId)
                mutableState.update { it.copy(channel = channel, isLoading = false, isRefreshing = false) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isLoading = false, isRefreshing = false, error = e) }
                if (mutableState.value.channel != null) messageChannel.send(UiMessage.Error(e))
            }
        }
    }

    fun setEnabled(enabled: Boolean) {
        if (mutableState.value.isChangingStatus) return
        mutableState.update { it.copy(isChangingStatus = true) }
        viewModelScope.launch {
            try {
                api.setChannelStatus(channelId, enabled)
                val status = if (enabled) ChannelStatus.ENABLED else ChannelStatus.MANUALLY_DISABLED
                mutableState.update { it.copy(channel = it.channel?.copy(status = status)) }
                messageChannel.send(UiMessage.Res(if (enabled) R.string.channels_enabled_message else R.string.channels_disabled_message))
                // Reloads this screen (status reason and time) and the list.
                container.events.notifyChannelsChanged()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                messageChannel.send(UiMessage.Error(e))
            } finally {
                mutableState.update { it.copy(isChangingStatus = false) }
            }
        }
    }

    fun updateBalance() {
        if (mutableState.value.isUpdatingBalance) return
        mutableState.update { it.copy(isUpdatingBalance = true) }
        viewModelScope.launch {
            try {
                val balance = api.updateChannelBalance(channelId)
                if (balance.isNaN()) {
                    messageChannel.send(UiMessage.Res(R.string.channels_balance_updated_no_value))
                } else {
                    val now = System.currentTimeMillis() / 1000
                    mutableState.update { it.copy(channel = it.channel?.copy(balance = balance, balanceUpdatedTime = now)) }
                    val text = container.quotaFormatter.value.formatUsd(balance)
                    messageChannel.send(UiMessage.Res(R.string.channels_balance_updated, listOf(text)))
                }
                container.events.notifyChannelsChanged()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                messageChannel.send(UiMessage.Error(e))
            } finally {
                mutableState.update { it.copy(isUpdatingBalance = false) }
            }
        }
    }

    /** [model] null tests the channel's default model. */
    fun runTest(model: String?) {
        if (mutableTest.value.isRunning) return
        mutableTest.value = ChannelTestRun(isRunning = true)
        testJob = viewModelScope.launch {
            try {
                val result = api.testChannel(channelId, model)
                mutableTest.value = ChannelTestRun(result = result)
                // Only a successful single test updates response_time and test_time.
                if (result.success) container.events.notifyChannelsChanged()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableTest.value = ChannelTestRun(error = e)
            }
        }
    }

    /** Stops waiting for a running test (the server may still finish it) and resets the dialog. */
    fun resetTest() {
        testJob?.cancel()
        testJob = null
        mutableTest.value = ChannelTestRun()
    }

    fun delete() {
        val current = mutableState.value
        if (current.isDeleting || current.deleted) return
        mutableState.update { it.copy(isDeleting = true) }
        viewModelScope.launch {
            try {
                api.deleteChannel(channelId)
                mutableState.update { it.copy(deleted = true) }
                container.events.notifyChannelsChanged()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isDeleting = false) }
                messageChannel.send(UiMessage.Error(e))
            }
        }
    }
}
