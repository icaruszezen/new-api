package com.newapi.admin.ui.common

import androidx.compose.material3.SnackbarHostState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.res.stringResource
import com.newapi.admin.R
import com.newapi.admin.core.net.ApiError
import com.newapi.admin.core.update.UpdateError
import kotlinx.coroutines.flow.Flow

/** A one-off message a ViewModel wants shown in a snackbar; resolved to text in composition. */
sealed interface UiMessage {
    data class Res(val id: Int, val args: List<Any> = emptyList()) : UiMessage
    data class Error(val error: Throwable) : UiMessage
}

@Composable
fun UiMessage.text(): String = when (this) {
    is UiMessage.Res -> if (args.isEmpty()) stringResource(id) else stringResource(id, *args.toTypedArray())
    is UiMessage.Error -> errorMessage(error)
}

/** Localised text for any failure thrown by the API layer or the updater. */
@Composable
fun errorMessage(error: Throwable): String = when (error) {
    is ApiError.Business -> error.message?.takeIf { it.isNotBlank() } ?: stringResource(R.string.error_unknown)
    is ApiError.Unauthorized -> stringResource(R.string.error_session_expired)
    is ApiError.Forbidden -> stringResource(R.string.error_forbidden)
    is ApiError.RateLimited -> error.retryAfterSeconds
        ?.let { stringResource(R.string.error_rate_limited_retry, it) }
        ?: stringResource(R.string.error_rate_limited)
    is ApiError.Http -> stringResource(R.string.error_http, error.status)
    is ApiError.Network -> stringResource(R.string.error_network)
    is ApiError.InvalidResponse -> stringResource(R.string.error_invalid_response)
    is ApiError.NotAdmin -> stringResource(R.string.error_not_admin)
    is ApiError.TurnstileRequired -> stringResource(R.string.error_turnstile)
    is ApiError.PasswordLoginDisabled -> stringResource(R.string.error_password_login_disabled)
    is UpdateError.Network -> stringResource(R.string.update_error_network)
    is UpdateError.RateLimited -> stringResource(R.string.update_error_rate_limited)
    is UpdateError.Http -> stringResource(R.string.update_error_http, error.status)
    is UpdateError.InvalidResponse -> stringResource(R.string.update_error_invalid)
    is UpdateError.CorruptDownload -> stringResource(R.string.update_error_corrupt)
    is UpdateError.Storage -> stringResource(R.string.update_error_storage)
    else -> error.message?.takeIf { it.isNotBlank() } ?: stringResource(R.string.error_unknown)
}

/** Shows each one-off ViewModel message once in [hostState]; a newer message replaces the one on screen. */
@Composable
fun UiMessagesEffect(messages: Flow<UiMessage>, hostState: SnackbarHostState) {
    var current by remember { mutableStateOf<UiMessage?>(null) }
    LaunchedEffect(messages) {
        messages.collect { current = it }
    }
    current?.let { message ->
        val text = message.text()
        LaunchedEffect(message) {
            hostState.showSnackbar(text)
            current = null
        }
    }
}
