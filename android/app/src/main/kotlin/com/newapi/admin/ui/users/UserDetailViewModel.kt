package com.newapi.admin.ui.users

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.newapi.admin.AppContainer
import com.newapi.admin.R
import com.newapi.admin.core.model.AdminUser
import com.newapi.admin.core.model.QuotaMode
import com.newapi.admin.core.model.UserAction
import com.newapi.admin.core.model.UserUpdate
import com.newapi.admin.core.net.ApiError
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

/** Forms opened from the detail screen; their submit errors show inline instead of in the snackbar. */
enum class UserDialog { ADJUST_QUOTA, EDIT_PROFILE, CHANGE_PASSWORD }

/** Every mutation the detail screen can run; at most one runs at a time. */
enum class UserOperation {
    ADJUST_QUOTA, EDIT_PROFILE, CHANGE_PASSWORD, ENABLE, DISABLE, PROMOTE, DEMOTE, RESET_2FA, RESET_PASSKEY, DELETE,
}

enum class PasswordInput { NEW, CONFIRM }

/** The three text columns `PUT /api/user/` overwrites besides the username. */
data class ProfileFields(val displayName: String, val group: String, val remark: String)

data class UserDetailUiState(
    val user: AdminUser? = null,
    /** First load (or retry after a failed first load) with nothing to show yet. */
    val isLoading: Boolean = true,
    val isRefreshing: Boolean = false,
    /** Failure of the first load; shown full screen. */
    val loadError: Throwable? = null,
    val running: UserOperation? = null,
    val dialog: UserDialog? = null,
    val dialogError: Throwable? = null,
    val passwordErrors: Map<PasswordInput, Int> = emptyMap(),
    val groups: GroupList = GroupList(),
    /** Set after a successful hard delete; the screen navigates back. */
    val deleted: Boolean = false,
)

class UserDetailViewModel(private val container: AppContainer, private val userId: Int) : ViewModel() {
    private val api = container.api

    private val mutableState = MutableStateFlow(UserDetailUiState())
    val state: StateFlow<UserDetailUiState> = mutableState.asStateFlow()

    private val messageChannel = Channel<UiMessage>(Channel.BUFFERED)
    val messages: Flow<UiMessage> = messageChannel.receiveAsFlow()

    // Passwords stay in memory only (not in saved instance state).
    var newPassword by mutableStateOf("")
        private set
    var confirmPassword by mutableStateOf("")
        private set

    private var loadJob: Job? = null

    init {
        load()
    }

    /** First load and the retry button of the full-screen error. */
    fun load() {
        if (mutableState.value.isLoading && loadJob?.isActive == true) return
        mutableState.update { it.copy(isLoading = it.user == null, loadError = null) }
        loadJob = viewModelScope.launch {
            try {
                val user = api.user(userId)
                mutableState.update { it.copy(user = user, isLoading = false, loadError = null) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isLoading = false, loadError = e) }
            }
        }
    }

    /** Pull to refresh with the user shown; a failure keeps the data and goes to the snackbar. */
    fun refresh() {
        val current = mutableState.value
        if (current.user == null) return load()
        if (current.isRefreshing) return
        mutableState.update { it.copy(isRefreshing = true) }
        loadJob?.cancel()
        loadJob = viewModelScope.launch {
            try {
                val user = api.user(userId)
                mutableState.update { it.copy(user = user, isRefreshing = false) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isRefreshing = false) }
                messageChannel.send(UiMessage.Error(e))
            }
        }
    }

    fun openDialog(dialog: UserDialog) {
        val current = mutableState.value
        if (current.running != null || current.dialog != null || current.user == null) return
        newPassword = ""
        confirmPassword = ""
        mutableState.update { it.copy(dialog = dialog, dialogError = null, passwordErrors = emptyMap()) }
        if (dialog == UserDialog.EDIT_PROFILE) loadGroups()
    }

    fun closeDialog() {
        if (mutableState.value.running != null) return
        newPassword = ""
        confirmPassword = ""
        mutableState.update { it.copy(dialog = null, dialogError = null, passwordErrors = emptyMap()) }
    }

    fun loadGroups() {
        val current = mutableState.value.groups
        if (current.isLoading || (current.names.isNotEmpty() && !current.failed)) return
        mutableState.update { it.copy(groups = it.groups.copy(isLoading = true, failed = false)) }
        viewModelScope.launch {
            try {
                val names = api.groups()
                mutableState.update { it.copy(groups = GroupList(names = names)) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(groups = it.groups.copy(isLoading = false, failed = true)) }
            }
        }
    }

    fun onNewPasswordChange(value: String) {
        newPassword = value
        clearPasswordError(PasswordInput.NEW)
    }

    fun onConfirmPasswordChange(value: String) {
        confirmPassword = value
        clearPasswordError(PasswordInput.CONFIRM)
    }

    /** [value] is in raw quota units, already validated by the dialog against the wallet bounds. */
    fun adjustQuota(mode: QuotaMode, value: Long) = mutate(UserOperation.ADJUST_QUOTA) {
        api.adjustQuota(userId, mode, value)
        UiMessage.Res(R.string.users_msg_quota_adjusted)
    }

    /**
     * PUT overwrites username, display name, group and remark, so the body starts from a fresh GET and
     * only the fields the admin changed (relative to what the form showed) replace the stored values.
     */
    fun saveProfile(original: ProfileFields, edited: ProfileFields) {
        if (edited == original) return closeDialog()
        mutate(UserOperation.EDIT_PROFILE) {
            val fresh = api.user(userId)
            api.updateUser(
                UserUpdate(
                    id = fresh.id,
                    username = fresh.username,
                    displayName = if (edited.displayName != original.displayName) edited.displayName else fresh.displayName,
                    group = if (edited.group != original.group) edited.group else fresh.group,
                    remark = if (edited.remark != original.remark) edited.remark else fresh.remark,
                ),
            )
            UiMessage.Res(R.string.users_msg_updated)
        }
    }

    fun changePassword() {
        val current = mutableState.value
        if (current.dialog != UserDialog.CHANGE_PASSWORD || current.running != null) return
        val password = newPassword
        val errors = buildMap {
            passwordProblem(password)?.let { put(PasswordInput.NEW, it) }
            if (confirmPassword != password) put(PasswordInput.CONFIRM, R.string.users_password_mismatch)
        }
        if (errors.isNotEmpty()) {
            mutableState.update { it.copy(passwordErrors = errors, dialogError = null) }
            return
        }
        mutate(UserOperation.CHANGE_PASSWORD) {
            val fresh = api.user(userId)
            api.updateUser(
                UserUpdate(
                    id = fresh.id,
                    username = fresh.username,
                    displayName = fresh.displayName,
                    group = fresh.group,
                    remark = fresh.remark,
                    password = password,
                ),
            )
            newPassword = ""
            confirmPassword = ""
            UiMessage.Res(R.string.users_msg_password_changed)
        }
    }

    /** Runs a confirmed one-tap action (status, role, 2FA, passkey, delete). */
    fun runAction(operation: UserOperation) {
        when (operation) {
            UserOperation.ENABLE -> manage(operation, UserAction.ENABLE, R.string.users_msg_enabled)
            UserOperation.DISABLE -> manage(operation, UserAction.DISABLE, R.string.users_msg_disabled)
            UserOperation.PROMOTE -> manage(operation, UserAction.PROMOTE, R.string.users_msg_promoted)
            UserOperation.DEMOTE -> manage(operation, UserAction.DEMOTE, R.string.users_msg_demoted)
            UserOperation.RESET_2FA -> mutate(operation) {
                serverMessage(api.resetUserTwoFactor(userId), R.string.users_msg_2fa_reset)
            }
            UserOperation.RESET_PASSKEY -> mutate(operation) {
                serverMessage(api.resetUserPasskey(userId), R.string.users_msg_passkey_reset)
            }
            UserOperation.DELETE -> delete()
            UserOperation.ADJUST_QUOTA, UserOperation.EDIT_PROFILE, UserOperation.CHANGE_PASSWORD -> Unit
        }
    }

    private fun manage(operation: UserOperation, action: UserAction, success: Int) = mutate(operation) {
        api.manageUser(userId, action)
        UiMessage.Res(success)
    }

    private fun delete() {
        if (mutableState.value.running != null) return
        mutableState.update { it.copy(running = UserOperation.DELETE) }
        viewModelScope.launch {
            try {
                api.deleteUser(userId)
                container.events.notifyUsersChanged()
                mutableState.update { it.copy(running = null, deleted = true) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(running = null) }
                messageChannel.send(UiMessage.Error(e))
            }
        }
    }

    /**
     * Runs one mutation, then closes the open form, tells the list to refresh, shows [block]'s message
     * and reloads the user. Failures go inline into the open form, or to the snackbar for plain actions.
     */
    private fun mutate(operation: UserOperation, block: suspend () -> UiMessage) {
        if (mutableState.value.running != null) return
        mutableState.update { it.copy(running = operation, dialogError = null) }
        viewModelScope.launch {
            try {
                val message = block()
                container.events.notifyUsersChanged()
                mutableState.update { it.copy(running = null, dialog = null, dialogError = null, passwordErrors = emptyMap()) }
                messageChannel.send(message)
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                val inForm = mutableState.value.dialog != null
                mutableState.update { it.copy(running = null, dialogError = if (inForm) e else null) }
                if (!inForm) messageChannel.send(UiMessage.Error(e))
                return@launch
            }
            reloadAfterChange()
        }
    }

    private suspend fun reloadAfterChange() {
        try {
            val user = api.user(userId)
            mutableState.update { it.copy(user = user) }
        } catch (e: CancellationException) {
            throw e
        } catch (e: Throwable) {
            // A revoked own session signs the app out; there is nothing left to report on this screen.
            if (e !is ApiError.Unauthorized) messageChannel.send(UiMessage.Error(e))
        }
    }

    private fun clearPasswordError(field: PasswordInput) {
        val errors = mutableState.value.passwordErrors
        if (field !in errors && mutableState.value.dialogError == null) return
        mutableState.update { it.copy(passwordErrors = errors - field, dialogError = null) }
    }

    private companion object {
        /** The reset endpoints answer with a (Chinese, hard-coded) message; fall back to ours when empty. */
        fun serverMessage(message: String, fallback: Int): UiMessage.Res =
            if (message.isBlank()) UiMessage.Res(fallback) else UiMessage.Res(R.string.users_server_message, listOf(message))
    }
}
