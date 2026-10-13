package com.newapi.admin.ui.login

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.newapi.admin.AppContainer
import com.newapi.admin.R
import com.newapi.admin.core.ServerUrl
import com.newapi.admin.core.auth.LoginResult
import com.newapi.admin.core.model.SystemStatus
import com.newapi.admin.core.net.ApiError
import com.newapi.admin.ui.common.UiMessage
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

enum class LoginMode { PASSWORD, ACCESS_TOKEN }

enum class LoginField { SERVER, USERNAME, PASSWORD, TOKEN, CODE }

sealed interface LoginStep {
    data object Credentials : LoginStep

    /** Password was accepted; the server waits for a TOTP or backup code until [deadlineMillis] (device clock). */
    data class TwoFactor(val baseUrl: String, val flowToken: String, val deadlineMillis: Long) : LoginStep
}

/** `/api/status` of the server last contacted, keyed by its normalised base URL. */
data class ServerPreview(val baseUrl: String, val status: SystemStatus) {
    /** Password sign-in cannot work here: Turnstile needs a browser widget, or the admin turned it off. */
    val passwordLoginBlocked: Boolean get() = status.turnstileCheck || !status.passwordLoginEnabled
}

data class LoginUiState(
    val mode: LoginMode = LoginMode.PASSWORD,
    val step: LoginStep = LoginStep.Credentials,
    val isBusy: Boolean = false,
    val server: ServerPreview? = null,
    /** Per-field validation message (string resource id). */
    val fieldErrors: Map<LoginField, Int> = emptyMap(),
    val error: UiMessage? = null,
    val useBackupCode: Boolean = false,
)

/**
 * Sign-in form. It lives in the activity's ViewModelStore (the login gate sits outside the NavHost), so it
 * survives sign-in and is reused after the next sign-out: secrets are cleared as soon as sign-in succeeds.
 *
 * Text input is Compose state (not a StateFlow) so typing stays synchronous with the text field.
 */
class LoginViewModel(container: AppContainer) : ViewModel() {
    private val auth = container.auth
    private val preferences = container.preferences

    var serverUrl by mutableStateOf(preferences.lastServerUrl)
        private set
    var username by mutableStateOf(preferences.lastUsername)
        private set
    var password by mutableStateOf("")
        private set
    var accessToken by mutableStateOf("")
        private set
    var code by mutableStateOf("")
        private set

    private val mutableState = MutableStateFlow(LoginUiState())
    val state: StateFlow<LoginUiState> = mutableState.asStateFlow()

    private var previewJob: Job? = null
    private var previewUrl: String? = null

    fun onServerUrlChange(value: String) {
        serverUrl = value
        clearFieldError(LoginField.SERVER)
    }

    fun onUsernameChange(value: String) {
        username = value
        clearFieldError(LoginField.USERNAME)
    }

    fun onPasswordChange(value: String) {
        password = value
        clearFieldError(LoginField.PASSWORD)
    }

    fun onAccessTokenChange(value: String) {
        accessToken = value
        clearFieldError(LoginField.TOKEN)
    }

    fun onCodeChange(value: String) {
        code = value
        clearFieldError(LoginField.CODE)
    }

    fun selectMode(mode: LoginMode) {
        if (mutableState.value.isBusy) return
        mutableState.update { it.copy(mode = mode, error = null, fieldErrors = emptyMap()) }
    }

    fun toggleBackupCode() {
        if (mutableState.value.isBusy) return
        code = ""
        mutableState.update { it.copy(useBackupCode = !it.useBackupCode, error = null, fieldErrors = emptyMap()) }
    }

    fun backToCredentials() {
        if (mutableState.value.isBusy) return
        code = ""
        mutableState.update {
            it.copy(step = LoginStep.Credentials, error = null, fieldErrors = emptyMap(), useBackupCode = false)
        }
    }

    /**
     * Quietly reads `/api/status` of the typed server, so its name and any password sign-in restriction show
     * before the first attempt. Failures are left for [signIn] to report.
     */
    fun previewServer() {
        val current = mutableState.value
        if (current.isBusy || current.step != LoginStep.Credentials) return
        val baseUrl = runCatching { ServerUrl.normalize(serverUrl) }.getOrNull() ?: return
        if (current.server?.baseUrl == baseUrl) return
        if (previewJob?.isActive == true && previewUrl == baseUrl) return
        previewJob?.cancel()
        previewUrl = baseUrl
        previewJob = viewModelScope.launch {
            try {
                val status = auth.fetchStatus(baseUrl)
                mutableState.update { if (it.isBusy) it else it.copy(server = ServerPreview(baseUrl, status)) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                // Signing in fetches the status again and shows the error.
            }
        }
    }

    fun signIn() {
        val current = mutableState.value
        if (current.isBusy || current.step != LoginStep.Credentials) return
        val baseUrl = try {
            ServerUrl.normalize(serverUrl)
        } catch (e: IllegalArgumentException) {
            val message = if (serverUrl.isBlank()) R.string.common_required else R.string.login_server_invalid
            mutableState.update { it.copy(fieldErrors = mapOf(LoginField.SERVER to message), error = null) }
            return
        }
        val mode = current.mode
        val missing = buildMap {
            if (mode == LoginMode.PASSWORD && username.isBlank()) put(LoginField.USERNAME, R.string.common_required)
            if (mode == LoginMode.PASSWORD && password.isEmpty()) put(LoginField.PASSWORD, R.string.common_required)
            if (mode == LoginMode.ACCESS_TOKEN && accessToken.isBlank()) put(LoginField.TOKEN, R.string.common_required)
        }
        if (missing.isNotEmpty()) {
            mutableState.update { it.copy(fieldErrors = missing, error = null) }
            return
        }
        previewJob?.cancel()
        mutableState.update { it.copy(isBusy = true, error = null, fieldErrors = emptyMap()) }
        viewModelScope.launch {
            try {
                val server = ServerPreview(baseUrl, auth.fetchStatus(baseUrl))
                mutableState.update { it.copy(server = server) }
                if (mode == LoginMode.ACCESS_TOKEN) {
                    auth.loginWithAccessToken(baseUrl, accessToken)
                    onSignedIn(baseUrl, signedInUsername = null)
                    return@launch
                }
                // The notice under the form explains why and offers token mode.
                if (server.passwordLoginBlocked) {
                    mutableState.update { it.copy(isBusy = false) }
                    return@launch
                }
                val signedInUsername = username.trim()
                when (val result = auth.loginWithPassword(baseUrl, signedInUsername, password)) {
                    is LoginResult.Success -> onSignedIn(baseUrl, signedInUsername)
                    is LoginResult.TwoFactorRequired -> {
                        code = ""
                        val step = LoginStep.TwoFactor(
                            baseUrl = result.baseUrl,
                            flowToken = result.flowToken,
                            deadlineMillis = System.currentTimeMillis() + TWO_FACTOR_FLOW_MILLIS,
                        )
                        mutableState.update { it.copy(isBusy = false, step = step, useBackupCode = false) }
                    }
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isBusy = false, error = loginErrorMessage(e, mode)) }
            }
        }
    }

    fun verifyTwoFactor() {
        val current = mutableState.value
        val step = current.step as? LoginStep.TwoFactor ?: return
        if (current.isBusy) return
        val entered = code.filterNot { it.isWhitespace() }
        val codeError = when {
            entered.isEmpty() -> R.string.common_required
            !current.useBackupCode && (entered.length != 6 || !entered.all { it in '0'..'9' }) -> R.string.login_2fa_code_invalid
            else -> null
        }
        if (codeError != null) {
            mutableState.update { it.copy(fieldErrors = mapOf(LoginField.CODE to codeError), error = null) }
            return
        }
        if (System.currentTimeMillis() > step.deadlineMillis) {
            code = ""
            mutableState.update {
                it.copy(step = LoginStep.Credentials, useBackupCode = false, error = UiMessage.Res(R.string.login_2fa_expired))
            }
            return
        }
        mutableState.update { it.copy(isBusy = true, error = null, fieldErrors = emptyMap()) }
        viewModelScope.launch {
            try {
                auth.verifyTwoFactor(step.baseUrl, step.flowToken, entered)
                onSignedIn(step.baseUrl, username.trim())
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isBusy = false, error = loginErrorMessage(e, LoginMode.PASSWORD)) }
            }
        }
    }

    /** AppRoot has already switched to the main UI; remember the server and forget every secret. */
    private fun onSignedIn(baseUrl: String, signedInUsername: String?) {
        preferences.lastServerUrl = baseUrl
        if (signedInUsername != null) preferences.lastUsername = signedInUsername
        serverUrl = baseUrl
        password = ""
        accessToken = ""
        code = ""
        mutableState.update { LoginUiState(mode = it.mode) }
    }

    private fun clearFieldError(field: LoginField) {
        if (field !in mutableState.value.fieldErrors) return
        mutableState.update { it.copy(fieldErrors = it.fieldErrors - field) }
    }

    private fun loginErrorMessage(error: Throwable, mode: LoginMode): UiMessage = when {
        error is ApiError.Unauthorized && error.code == "AUTH_USER_DISABLED" -> UiMessage.Res(R.string.login_error_user_disabled)
        error is ApiError.Unauthorized && mode == LoginMode.ACCESS_TOKEN -> UiMessage.Res(R.string.login_error_token_rejected)
        error is ApiError.Http && error.code == "AUTH_SESSION_LIMIT" -> UiMessage.Res(R.string.login_error_session_limit)
        else -> UiMessage.Error(error)
    }

    private companion object {
        /** The server keeps a 2FA login flow for 5 minutes (controller/user.go); a little less on the device clock. */
        const val TWO_FACTOR_FLOW_MILLIS = 4 * 60 * 1000L + 50 * 1000L
    }
}
