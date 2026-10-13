package com.newapi.admin.ui.users

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.newapi.admin.AppContainer
import com.newapi.admin.R
import com.newapi.admin.core.model.Roles
import com.newapi.admin.ui.common.UiMessage
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

enum class CreateField { USERNAME, PASSWORD, CONFIRM, DISPLAY_NAME }

data class UserCreateUiState(
    val isSubmitting: Boolean = false,
    /** Per-field validation message. */
    val fieldErrors: Map<CreateField, UiMessage.Res> = emptyMap(),
    /** Server failure of the last submit, shown under the form. */
    val error: Throwable? = null,
    /** Set once the server accepted the user; the screen then leaves. */
    val created: Boolean = false,
)

class UserCreateViewModel(private val container: AppContainer) : ViewModel() {
    private val mutableState = MutableStateFlow(UserCreateUiState())
    val state: StateFlow<UserCreateUiState> = mutableState.asStateFlow()

    // Form input lives here so it survives rotation; passwords never go into saved instance state.
    var username by mutableStateOf("")
        private set
    var password by mutableStateOf("")
        private set
    var confirmPassword by mutableStateOf("")
        private set
    var displayName by mutableStateOf("")
        private set

    /** [Roles.COMMON], or [Roles.ADMIN] when the signed-in admin is root. */
    var role by mutableStateOf(Roles.COMMON)
        private set

    fun onUsernameChange(value: String) {
        username = value
        clearError(CreateField.USERNAME)
    }

    fun onPasswordChange(value: String) {
        password = value
        clearError(CreateField.PASSWORD)
    }

    fun onConfirmPasswordChange(value: String) {
        confirmPassword = value
        clearError(CreateField.CONFIRM)
    }

    fun onDisplayNameChange(value: String) {
        displayName = value
        clearError(CreateField.DISPLAY_NAME)
    }

    fun onRoleChange(value: Int) {
        role = value
    }

    fun submit() {
        val current = mutableState.value
        if (current.isSubmitting || current.created) return
        val name = username.trim()
        val errors = buildMap {
            val tooLong = UiMessage.Res(R.string.users_error_too_long, listOf(MAX_NAME_LENGTH))
            when {
                name.isEmpty() -> put(CreateField.USERNAME, UiMessage.Res(R.string.common_required))
                name.runeCount() > MAX_NAME_LENGTH -> put(CreateField.USERNAME, tooLong)
            }
            passwordProblem(password)?.let { put(CreateField.PASSWORD, UiMessage.Res(it)) }
            if (confirmPassword != password) put(CreateField.CONFIRM, UiMessage.Res(R.string.users_password_mismatch))
            if (displayName.trim().runeCount() > MAX_NAME_LENGTH) put(CreateField.DISPLAY_NAME, tooLong)
        }
        if (errors.isNotEmpty()) {
            mutableState.update { it.copy(fieldErrors = errors, error = null) }
            return
        }
        // The server stores any role below the caller's; only 1 and 10 are valid (critique correction 3).
        val isRoot = container.currentUser.value?.isRoot == true
        val roleToSend = if (role == Roles.ADMIN && isRoot) Roles.ADMIN else Roles.COMMON
        mutableState.update { it.copy(isSubmitting = true, fieldErrors = emptyMap(), error = null) }
        viewModelScope.launch {
            try {
                container.api.createUser(name, password, displayName.trim(), roleToSend)
                container.events.notifyUsersChanged()
                password = ""
                confirmPassword = ""
                mutableState.update { it.copy(isSubmitting = false, created = true) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isSubmitting = false, error = e) }
            }
        }
    }

    private fun clearError(field: CreateField) {
        val current = mutableState.value
        if (field !in current.fieldErrors && current.error == null) return
        mutableState.update { it.copy(fieldErrors = it.fieldErrors - field, error = null) }
    }
}
