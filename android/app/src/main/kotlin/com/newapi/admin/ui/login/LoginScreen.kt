package com.newapi.admin.ui.login

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.consumeWindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.AdminPanelSettings
import androidx.compose.material.icons.outlined.CheckCircle
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.Info
import androidx.compose.material.icons.outlined.Visibility
import androidx.compose.material.icons.outlined.VisibilityOff
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.autofill.ContentType
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentType
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.ServerUrl
import com.newapi.admin.core.auth.SignOutReason
import com.newapi.admin.ui.common.text
import com.newapi.admin.ui.components.AppAlert
import com.newapi.admin.ui.components.AppButton
import com.newapi.admin.ui.components.AppCard
import com.newapi.admin.ui.components.AppSegmentedTabs
import com.newapi.admin.ui.components.AppTextField
import com.newapi.admin.ui.components.ButtonSize
import com.newapi.admin.ui.components.ButtonVariant
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.theme.Tone
import com.newapi.admin.ui.theme.colors

/** Shown whenever no account is signed in; [signedOutReason] explains an involuntary sign-out. */
@Composable
fun LoginScreen(signedOutReason: SignOutReason?) {
    val container = LocalAppContainer.current
    val viewModel = viewModel { LoginViewModel(container) }
    val state by viewModel.state.collectAsStateWithLifecycle()
    var reasonDismissed by rememberSaveable(signedOutReason) { mutableStateOf(false) }
    val reasonText = when (signedOutReason) {
        SignOutReason.SESSION_EXPIRED -> R.string.error_session_expired
        SignOutReason.NOT_ADMIN -> R.string.login_signed_out_not_admin
        SignOutReason.USER, null -> null
    }
    val twoFactor = state.step is LoginStep.TwoFactor

    BackHandler(enabled = twoFactor && !state.isBusy) { viewModel.backToCredentials() }
    LaunchedEffect(viewModel) { viewModel.previewServer() }

    Scaffold { padding ->
        BoxWithConstraints(
            Modifier
                .fillMaxSize()
                .padding(padding)
                .consumeWindowInsets(padding)
                .imePadding(),
        ) {
            val viewportHeight = maxHeight
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .verticalScroll(rememberScrollState())
                    .heightIn(min = viewportHeight)
                    .padding(horizontal = 24.dp, vertical = 32.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Column(
                    modifier = Modifier.widthIn(max = 440.dp).fillMaxWidth(),
                    verticalArrangement = Arrangement.spacedBy(16.dp),
                ) {
                    LoginHeader()
                    if (reasonText != null && !reasonDismissed) {
                        AppAlert(
                            text = stringResource(reasonText),
                            tone = Tone.Warning,
                            icon = Icons.Outlined.Info,
                            onDismiss = { reasonDismissed = true },
                        )
                    }
                    AppCard(contentPadding = PaddingValues(20.dp)) {
                        Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
                            if (twoFactor) TwoFactorForm(viewModel, state) else CredentialsForm(viewModel, state)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun LoginHeader() {
    Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
        Surface(
            shape = MaterialTheme.shapes.large,
            color = AppTheme.colors.primary,
            contentColor = AppTheme.colors.primaryForeground,
            modifier = Modifier.size(48.dp),
        ) {
            Box(contentAlignment = Alignment.Center) {
                Icon(Icons.Outlined.AdminPanelSettings, contentDescription = null, modifier = Modifier.size(26.dp))
            }
        }
        Spacer(Modifier.height(16.dp))
        Text(
            stringResource(R.string.app_name),
            style = MaterialTheme.typography.headlineSmall,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(4.dp))
        Text(
            stringResource(R.string.login_subtitle),
            style = MaterialTheme.typography.bodyMedium,
            color = AppTheme.colors.mutedForeground,
            textAlign = TextAlign.Center,
        )
    }
}

@Composable
private fun CredentialsForm(viewModel: LoginViewModel, state: LoginUiState) {
    val keyboard = LocalSoftwareKeyboardController.current
    val enabled = !state.isBusy
    val serverUrl = viewModel.serverUrl
    val normalizedUrl = remember(serverUrl) { runCatching { ServerUrl.normalize(serverUrl) }.getOrNull() }
    // Only describe the server while the field still points at it.
    val server = state.server?.takeIf { it.baseUrl == normalizedUrl }
    val serverError = state.fieldErrors[LoginField.SERVER]
    val cleartext = normalizedUrl != null && ServerUrl.isCleartext(normalizedUrl)
    val warningColor = Tone.Warning.colors().content
    val serverHint = when {
        serverError != null -> stringResource(serverError)
        cleartext -> stringResource(R.string.login_server_cleartext)
        else -> null
    }
    var secretVisible by rememberSaveable { mutableStateOf(false) }
    val submit = {
        keyboard?.hide()
        viewModel.signIn()
    }

    AppTextField(
        value = serverUrl,
        onValueChange = viewModel::onServerUrlChange,
        modifier = Modifier.fillMaxWidth(),
        // Look the server up when the user moves on from the address.
        inputModifier = Modifier.onFocusChanged { if (!it.isFocused) viewModel.previewServer() },
        enabled = enabled,
        label = stringResource(R.string.login_server_label),
        placeholder = stringResource(R.string.login_server_placeholder),
        isError = serverError != null,
        supportingText = serverHint,
        supportingColor = if (serverError != null) null else warningColor,
        singleLine = true,
        keyboardOptions = KeyboardOptions(
            keyboardType = KeyboardType.Uri,
            imeAction = ImeAction.Next,
            autoCorrectEnabled = false,
        ),
    )

    if (server != null) {
        val name = server.status.systemName.ifBlank { server.baseUrl }
        val version = server.status.version
        Row(verticalAlignment = Alignment.CenterVertically) {
            Icon(
                Icons.Outlined.CheckCircle,
                contentDescription = null,
                tint = Tone.Success.colors().content,
                modifier = Modifier.size(16.dp),
            )
            Spacer(Modifier.width(8.dp))
            Text(
                if (version.isBlank()) stringResource(R.string.login_server_found, name)
                else stringResource(R.string.login_server_found_version, name, version),
                style = MaterialTheme.typography.bodySmall,
                color = AppTheme.colors.mutedForeground,
            )
        }
    }

    AppSegmentedTabs(
        options = LoginMode.entries.map { mode ->
            stringResource(
                when (mode) {
                    LoginMode.PASSWORD -> R.string.login_mode_password
                    LoginMode.ACCESS_TOKEN -> R.string.login_mode_token
                },
            )
        },
        selectedIndex = LoginMode.entries.indexOf(state.mode),
        onSelect = { viewModel.selectMode(LoginMode.entries[it]) },
        modifier = Modifier.fillMaxWidth(),
        enabled = enabled,
    )

    val secretToggle: @Composable () -> Unit = {
        IconButton(onClick = { secretVisible = !secretVisible }) {
            Icon(
                if (secretVisible) Icons.Outlined.VisibilityOff else Icons.Outlined.Visibility,
                contentDescription = stringResource(if (secretVisible) R.string.login_hide_secret else R.string.login_show_secret),
                modifier = Modifier.size(18.dp),
            )
        }
    }
    val secretTransformation = if (secretVisible) VisualTransformation.None else PasswordVisualTransformation()

    when (state.mode) {
        LoginMode.PASSWORD -> {
            val usernameError = state.fieldErrors[LoginField.USERNAME]
            val passwordError = state.fieldErrors[LoginField.PASSWORD]
            AppTextField(
                value = viewModel.username,
                onValueChange = viewModel::onUsernameChange,
                modifier = Modifier.fillMaxWidth(),
                inputModifier = Modifier.semantics { contentType = ContentType.Username },
                enabled = enabled,
                label = stringResource(R.string.login_username_label),
                isError = usernameError != null,
                supportingText = usernameError?.let { stringResource(it) },
                singleLine = true,
                keyboardOptions = KeyboardOptions(
                    keyboardType = KeyboardType.Email,
                    imeAction = ImeAction.Next,
                    autoCorrectEnabled = false,
                ),
            )
            AppTextField(
                value = viewModel.password,
                onValueChange = viewModel::onPasswordChange,
                modifier = Modifier.fillMaxWidth(),
                inputModifier = Modifier.semantics { contentType = ContentType.Password },
                enabled = enabled,
                label = stringResource(R.string.login_password_label),
                trailing = secretToggle,
                isError = passwordError != null,
                supportingText = passwordError?.let { stringResource(it) },
                singleLine = true,
                visualTransformation = secretTransformation,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Done),
                keyboardActions = KeyboardActions(onDone = { submit() }),
            )
        }
        LoginMode.ACCESS_TOKEN -> {
            val tokenError = state.fieldErrors[LoginField.TOKEN]
            AppTextField(
                value = viewModel.accessToken,
                onValueChange = viewModel::onAccessTokenChange,
                modifier = Modifier.fillMaxWidth(),
                enabled = enabled,
                label = stringResource(R.string.login_token_label),
                trailing = secretToggle,
                isError = tokenError != null,
                supportingText = tokenError?.let { stringResource(it) },
                singleLine = true,
                visualTransformation = secretTransformation,
                keyboardOptions = KeyboardOptions(
                    keyboardType = KeyboardType.Password,
                    imeAction = ImeAction.Done,
                    autoCorrectEnabled = false,
                ),
                keyboardActions = KeyboardActions(onDone = { submit() }),
            )
            Text(
                stringResource(R.string.login_token_help),
                style = MaterialTheme.typography.bodySmall,
                color = AppTheme.colors.mutedForeground,
            )
            Text(
                stringResource(R.string.login_token_limits),
                style = MaterialTheme.typography.bodySmall,
                color = AppTheme.colors.mutedForeground,
            )
        }
    }

    if (state.mode == LoginMode.PASSWORD && server != null && server.passwordLoginBlocked) {
        AppAlert(
            text = stringResource(if (server.status.turnstileCheck) R.string.error_turnstile else R.string.error_password_login_disabled),
            tone = Tone.Info,
            icon = Icons.Outlined.Info,
            action = {
                AppButton(
                    stringResource(R.string.login_use_token),
                    onClick = { viewModel.selectMode(LoginMode.ACCESS_TOKEN) },
                    variant = ButtonVariant.Ghost,
                    size = ButtonSize.Small,
                    enabled = enabled,
                )
            },
        )
    }

    state.error?.let { error ->
        AppAlert(text = error.text(), tone = Tone.Danger, icon = Icons.Outlined.ErrorOutline)
    }

    SubmitButton(label = stringResource(R.string.login_sign_in), isBusy = state.isBusy, onClick = submit)
}

@Composable
private fun TwoFactorForm(viewModel: LoginViewModel, state: LoginUiState) {
    val keyboard = LocalSoftwareKeyboardController.current
    val enabled = !state.isBusy
    val codeError = state.fieldErrors[LoginField.CODE]
    val focusRequester = remember { FocusRequester() }
    val submit = {
        keyboard?.hide()
        viewModel.verifyTwoFactor()
    }

    LaunchedEffect(state.useBackupCode) { focusRequester.requestFocus() }

    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Text(stringResource(R.string.login_2fa_title), style = MaterialTheme.typography.titleLarge)
        Text(
            stringResource(if (state.useBackupCode) R.string.login_2fa_backup_prompt else R.string.login_2fa_code_prompt),
            style = MaterialTheme.typography.bodyMedium,
            color = AppTheme.colors.mutedForeground,
        )
    }

    // A new field per code kind, so the keyboard reopens with the matching layout.
    key(state.useBackupCode) {
        AppTextField(
            value = viewModel.code,
            onValueChange = viewModel::onCodeChange,
            modifier = Modifier.fillMaxWidth(),
            inputModifier = Modifier.focusRequester(focusRequester),
            enabled = enabled,
            label = stringResource(if (state.useBackupCode) R.string.login_2fa_backup_label else R.string.login_2fa_code_label),
            placeholder = stringResource(if (state.useBackupCode) R.string.login_2fa_backup_placeholder else R.string.login_2fa_code_placeholder),
            isError = codeError != null,
            supportingText = stringResource(
                codeError ?: if (state.useBackupCode) R.string.login_2fa_backup_hint else R.string.login_2fa_code_hint,
            ),
            singleLine = true,
            textStyle = MaterialTheme.typography.bodyLarge.copy(fontWeight = FontWeight.Medium),
            keyboardOptions = if (state.useBackupCode) {
                KeyboardOptions(
                    keyboardType = KeyboardType.Ascii,
                    capitalization = KeyboardCapitalization.Characters,
                    imeAction = ImeAction.Done,
                    autoCorrectEnabled = false,
                )
            } else {
                KeyboardOptions(keyboardType = KeyboardType.Number, imeAction = ImeAction.Done)
            },
            keyboardActions = KeyboardActions(onDone = { submit() }),
        )
    }

    state.error?.let { error ->
        AppAlert(text = error.text(), tone = Tone.Danger, icon = Icons.Outlined.ErrorOutline)
    }

    SubmitButton(label = stringResource(R.string.login_2fa_verify), isBusy = state.isBusy, onClick = submit)

    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
        AppButton(
            stringResource(R.string.login_2fa_back),
            onClick = viewModel::backToCredentials,
            variant = ButtonVariant.Ghost,
            size = ButtonSize.Small,
            enabled = enabled,
        )
        AppButton(
            stringResource(if (state.useBackupCode) R.string.login_2fa_use_totp else R.string.login_2fa_use_backup),
            onClick = viewModel::toggleBackupCode,
            variant = ButtonVariant.Ghost,
            size = ButtonSize.Small,
            enabled = enabled,
        )
    }
}

@Composable
private fun SubmitButton(label: String, isBusy: Boolean, onClick: () -> Unit) {
    AppButton(label, onClick = onClick, modifier = Modifier.fillMaxWidth(), size = ButtonSize.Large, loading = isBusy)
}
