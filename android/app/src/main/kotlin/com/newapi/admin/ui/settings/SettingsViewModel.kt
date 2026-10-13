package com.newapi.admin.ui.settings

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.newapi.admin.AppContainer
import com.newapi.admin.R
import com.newapi.admin.core.auth.AuthState
import com.newapi.admin.core.auth.StoredAccount
import com.newapi.admin.core.net.ApiError
import com.newapi.admin.ui.common.UiMessage
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

enum class PasswordField { CURRENT, NEW, CONFIRM }

data class PasswordDialogState(
    val isSubmitting: Boolean = false,
    /** Per-field validation message (string resource id). */
    val fieldErrors: Map<PasswordField, Int> = emptyMap(),
    val error: UiMessage? = null,
)

data class SettingsUiState(
    val isRefreshing: Boolean = false,
    val isSigningOut: Boolean = false,
    /** Non-null while the change-password dialog is open. */
    val passwordDialog: PasswordDialogState? = null,
)

class SettingsViewModel(private val container: AppContainer) : ViewModel() {
    private val auth = container.auth

    val account: StateFlow<StoredAccount?> = auth.state
        .map { (it as? AuthState.SignedIn)?.account }
        .stateIn(viewModelScope, SharingStarted.Eagerly, (auth.state.value as? AuthState.SignedIn)?.account)

    private val mutableState = MutableStateFlow(SettingsUiState())
    val state: StateFlow<SettingsUiState> = mutableState.asStateFlow()

    private val messageChannel = Channel<UiMessage>(Channel.BUFFERED)
    val messages: Flow<UiMessage> = messageChannel.receiveAsFlow()

    // Password dialog input is Compose state so typing stays synchronous with the text fields.
    var currentPassword by mutableStateOf("")
        private set
    var newPassword by mutableStateOf("")
        private set
    var confirmPassword by mutableStateOf("")
        private set

    private var reloadJob: Job? = null

    init {
        // Balance and usage change all the time; the stored profile is from the last login or refresh.
        reload(userInitiated = false)
    }

    fun refresh() = reload(userInitiated = true)

    fun onCurrentPasswordChange(value: String) {
        currentPassword = value
        clearPasswordFieldError(PasswordField.CURRENT)
    }

    fun onNewPasswordChange(value: String) {
        newPassword = value
        clearPasswordFieldError(PasswordField.NEW)
    }

    fun onConfirmPasswordChange(value: String) {
        confirmPassword = value
        clearPasswordFieldError(PasswordField.CONFIRM)
    }

    fun openPasswordDialog() {
        if (mutableState.value.passwordDialog != null) return
        currentPassword = ""
        newPassword = ""
        confirmPassword = ""
        mutableState.update { it.copy(passwordDialog = PasswordDialogState()) }
    }

    fun closePasswordDialog() {
        if (mutableState.value.passwordDialog?.isSubmitting == true) return
        currentPassword = ""
        newPassword = ""
        confirmPassword = ""
        mutableState.update { it.copy(passwordDialog = null) }
    }

    fun changePassword() {
        val dialog = mutableState.value.passwordDialog ?: return
        if (dialog.isSubmitting) return
        val original = currentPassword
        val replacement = newPassword
        // The server validates 8-20 characters (model/user.go `validate:"min=8,max=20"`, counted in runes).
        val length = replacement.codePointCount(0, replacement.length)
        val errors = buildMap {
            if (original.isEmpty()) put(PasswordField.CURRENT, R.string.common_required)
            when {
                replacement.isEmpty() -> put(PasswordField.NEW, R.string.common_required)
                length !in 8..20 -> put(PasswordField.NEW, R.string.settings_password_length)
                replacement == original -> put(PasswordField.NEW, R.string.settings_password_same)
            }
            if (confirmPassword != replacement) put(PasswordField.CONFIRM, R.string.settings_password_mismatch)
        }
        if (errors.isNotEmpty()) {
            mutableState.update { it.copy(passwordDialog = dialog.copy(fieldErrors = errors, error = null)) }
            return
        }
        mutableState.update { it.copy(passwordDialog = dialog.copy(isSubmitting = true, fieldErrors = emptyMap(), error = null)) }
        viewModelScope.launch {
            try {
                auth.changeOwnPassword(original, replacement)
                currentPassword = ""
                newPassword = ""
                confirmPassword = ""
                mutableState.update { it.copy(passwordDialog = null) }
                messageChannel.send(UiMessage.Res(R.string.settings_password_changed))
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { state ->
                    state.copy(passwordDialog = state.passwordDialog?.copy(isSubmitting = false, error = UiMessage.Error(e)))
                }
            }
        }
    }

    /** Revokes the server session (best effort) and clears local credentials; AppRoot then shows the login screen. */
    fun signOut() {
        if (mutableState.value.isSigningOut) return
        mutableState.update { it.copy(isSigningOut = true) }
        viewModelScope.launch {
            try {
                auth.logout()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                // logout() clears local credentials even when the server call fails; nothing else to do.
                mutableState.update { it.copy(isSigningOut = false) }
            }
        }
    }

    /** A user refresh replaces the silent one started on open, so its errors get reported. */
    private fun reload(userInitiated: Boolean) {
        if (mutableState.value.isRefreshing) return
        reloadJob?.cancel()
        mutableState.update { it.copy(isRefreshing = userInitiated) }
        container.refreshStatus()
        reloadJob = viewModelScope.launch {
            try {
                auth.reloadProfile()
                mutableState.update { it.copy(isRefreshing = false) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isRefreshing = false) }
                // NotAdmin and Unauthorized have already signed the app out; the screen is about to go away.
                val signedOut = e is ApiError.NotAdmin || e is ApiError.Unauthorized
                if (userInitiated && !signedOut) messageChannel.send(UiMessage.Error(e))
            }
        }
    }

    private fun clearPasswordFieldError(field: PasswordField) {
        val dialog = mutableState.value.passwordDialog ?: return
        if (field !in dialog.fieldErrors) return
        mutableState.update { it.copy(passwordDialog = dialog.copy(fieldErrors = dialog.fieldErrors - field)) }
    }
}
