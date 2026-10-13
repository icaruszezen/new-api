package com.newapi.admin.ui.settings

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.VisibilityOff
import androidx.compose.material.icons.filled.WarningAmber
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.autofill.ContentType
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentType
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.ServerUrl
import com.newapi.admin.core.auth.Credentials
import com.newapi.admin.core.auth.StoredAccount
import com.newapi.admin.core.format.Formats
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.core.model.SystemStatus
import com.newapi.admin.ui.common.UiMessage
import com.newapi.admin.ui.common.text
import com.newapi.admin.ui.components.ConfirmDialog
import com.newapi.admin.ui.components.FullScreenLoading
import com.newapi.admin.ui.components.InfoRow
import com.newapi.admin.ui.components.SectionCard
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.components.rememberQuotaFormatter
import com.newapi.admin.ui.theme.Tone
import com.newapi.admin.ui.theme.colors
import java.math.BigDecimal

/** The "Me" tab: signed-in account, server details, password change, sign-out and app version. */
@Composable
fun SettingsScreen() {
    val container = LocalAppContainer.current
    val viewModel = viewModel { SettingsViewModel(container) }
    val state by viewModel.state.collectAsStateWithLifecycle()
    val account by viewModel.account.collectAsStateWithLifecycle()
    val status by container.status.collectAsStateWithLifecycle()
    val formatter = rememberQuotaFormatter()
    val snackbarHostState = remember { SnackbarHostState() }
    var confirmSignOut by rememberSaveable { mutableStateOf(false) }

    // Messages arrive once from the ViewModel; the text is resolved in composition, then shown.
    var snackbarMessage by remember { mutableStateOf<UiMessage?>(null) }
    LaunchedEffect(viewModel) {
        viewModel.messages.collect { snackbarMessage = it }
    }
    snackbarMessage?.let { message ->
        val text = message.text()
        LaunchedEffect(message) {
            snackbarHostState.showSnackbar(text)
            snackbarMessage = null
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.nav_me)) },
                actions = {
                    IconButton(onClick = viewModel::refresh, enabled = !state.isRefreshing) {
                        Icon(Icons.Filled.Refresh, contentDescription = stringResource(R.string.common_refresh))
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        val current = account
        if (current == null) {
            // Only while signing out: AppRoot is about to show the login screen.
            FullScreenLoading(Modifier.padding(padding))
            return@Scaffold
        }
        val isSession = current.credentials is Credentials.Session
        PullToRefreshBox(
            isRefreshing = state.isRefreshing,
            onRefresh = viewModel::refresh,
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                AccountCard(current, formatter)
                ServerCard(current.baseUrl, status, formatter)
                SectionCard(title = stringResource(R.string.settings_security_title)) {
                    ActionRow(
                        icon = Icons.Filled.Lock,
                        title = stringResource(R.string.settings_change_password),
                        subtitle = stringResource(
                            if (isSession) R.string.settings_change_password_desc else R.string.settings_change_password_unavailable,
                        ),
                        enabled = isSession,
                        onClick = viewModel::openPasswordDialog,
                    )
                    HorizontalDivider(Modifier.padding(vertical = 4.dp))
                    ActionRow(
                        icon = Icons.AutoMirrored.Filled.Logout,
                        title = stringResource(R.string.settings_sign_out),
                        subtitle = null,
                        enabled = !state.isSigningOut,
                        busy = state.isSigningOut,
                        tint = MaterialTheme.colorScheme.error,
                        onClick = { confirmSignOut = true },
                    )
                }
                SectionCard(title = stringResource(R.string.settings_about_title)) {
                    InfoRow(stringResource(R.string.settings_app_version), container.appVersion)
                }
            }
        }

        if (confirmSignOut) {
            ConfirmDialog(
                title = stringResource(R.string.settings_sign_out_confirm_title),
                text = stringResource(R.string.settings_sign_out_confirm_text, current.baseUrl),
                confirmText = stringResource(R.string.settings_sign_out),
                onConfirm = {
                    confirmSignOut = false
                    viewModel.signOut()
                },
                onDismiss = { confirmSignOut = false },
                destructive = true,
            )
        }

        state.passwordDialog?.let { dialog ->
            ChangePasswordDialog(viewModel, dialog)
        }
    }
}

@Composable
private fun AccountCard(account: StoredAccount, formatter: QuotaFormatter) {
    val user = account.user
    val displayName = user.displayName.ifBlank { user.username }
    SectionCard(title = null) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Surface(
                shape = CircleShape,
                color = MaterialTheme.colorScheme.primaryContainer,
                contentColor = MaterialTheme.colorScheme.onPrimaryContainer,
                modifier = Modifier.size(48.dp),
            ) {
                Box(contentAlignment = Alignment.Center) {
                    Text(
                        // Two chars when the name starts with a surrogate pair (emoji, rare CJK).
                        displayName.take(if (displayName.firstOrNull()?.isHighSurrogate() == true) 2 else 1).uppercase(),
                        style = MaterialTheme.typography.titleLarge,
                        fontWeight = FontWeight.SemiBold,
                    )
                }
            }
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    displayName,
                    style = MaterialTheme.typography.titleMedium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    "@${user.username}",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Spacer(Modifier.width(8.dp))
            if (user.isRoot) {
                StatusPill(stringResource(R.string.settings_role_root), Tone.Warning)
            } else {
                StatusPill(stringResource(R.string.settings_role_admin), Tone.Info)
            }
        }

        Spacer(Modifier.height(16.dp))
        Row(Modifier.height(IntrinsicSize.Min), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            StatTile(stringResource(R.string.settings_balance), formatter.format(user.quota), Modifier.weight(1f).fillMaxHeight())
            StatTile(stringResource(R.string.settings_used_quota), formatter.format(user.usedQuota), Modifier.weight(1f).fillMaxHeight())
            StatTile(stringResource(R.string.settings_request_count), Formats.count(user.requestCount), Modifier.weight(1f).fillMaxHeight())
        }

        Spacer(Modifier.height(12.dp))
        InfoRow(stringResource(R.string.settings_user_id), user.id.toString(), copyable = true)
        InfoRow(stringResource(R.string.settings_group), user.group)
        if (user.email.isNotBlank()) InfoRow(stringResource(R.string.settings_email), user.email, copyable = true)
        InfoRow(
            stringResource(R.string.settings_sign_in_method),
            stringResource(
                when (account.credentials) {
                    is Credentials.Session -> R.string.settings_method_session
                    is Credentials.AccessToken -> R.string.settings_method_token
                },
            ),
        )
        if (account.credentials is Credentials.AccessToken) {
            Text(
                stringResource(R.string.settings_token_note),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun StatTile(label: String, value: String, modifier: Modifier = Modifier) {
    Surface(
        color = MaterialTheme.colorScheme.surfaceContainerHigh,
        shape = MaterialTheme.shapes.medium,
        modifier = modifier,
    ) {
        Column(Modifier.padding(horizontal = 10.dp, vertical = 8.dp)) {
            Text(
                label,
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Spacer(Modifier.height(2.dp))
            Text(
                value,
                style = MaterialTheme.typography.titleSmall.copy(fontFeatureSettings = "tnum"),
                fontWeight = FontWeight.SemiBold,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

@Composable
private fun ServerCard(baseUrl: String, status: SystemStatus?, formatter: QuotaFormatter) {
    SectionCard(title = stringResource(R.string.settings_server_title)) {
        InfoRow(stringResource(R.string.settings_server_address), baseUrl, copyable = true)
        if (ServerUrl.isCleartext(baseUrl)) {
            val warning = Tone.Warning.colors().content
            Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(vertical = 4.dp)) {
                Icon(Icons.Filled.WarningAmber, contentDescription = null, tint = warning, modifier = Modifier.size(16.dp))
                Spacer(Modifier.width(8.dp))
                Text(stringResource(R.string.settings_cleartext), style = MaterialTheme.typography.bodySmall, color = warning)
            }
        }
        InfoRow(stringResource(R.string.settings_system_name), status?.systemName.orEmpty())
        InfoRow(stringResource(R.string.settings_server_version), status?.version.orEmpty())
        // The formatter falls back to USD until /api/status has loaded, so only describe it once it has.
        val displayType = if (status == null) {
            ""
        } else {
            when (formatter.displayType) {
                QuotaFormatter.DisplayType.USD -> stringResource(R.string.settings_display_usd)
                QuotaFormatter.DisplayType.CNY -> stringResource(R.string.settings_display_cny)
                QuotaFormatter.DisplayType.TOKENS -> stringResource(R.string.settings_display_tokens)
                QuotaFormatter.DisplayType.CUSTOM -> stringResource(R.string.settings_display_custom, formatter.unitLabel)
            }
        }
        InfoRow(stringResource(R.string.settings_quota_display), displayType)
        if (status != null && formatter.displayType != QuotaFormatter.DisplayType.TOKENS) {
            if (formatter.displayType != QuotaFormatter.DisplayType.USD) {
                InfoRow(
                    stringResource(R.string.settings_exchange_rate),
                    stringResource(R.string.settings_exchange_rate_value, plainNumber(formatter.rate), formatter.unitLabel),
                )
            }
            InfoRow(stringResource(R.string.settings_quota_per_unit), plainNumber(formatter.quotaPerUnit))
        }
    }
}

/** One tappable line in a card: icon, title, optional explanation; a spinner replaces the icon while [busy]. */
@Composable
private fun ActionRow(
    icon: ImageVector,
    title: String,
    subtitle: String?,
    enabled: Boolean,
    onClick: () -> Unit,
    busy: Boolean = false,
    tint: Color = MaterialTheme.colorScheme.onSurface,
) {
    val alpha = if (enabled || busy) 1f else 0.5f
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(enabled = enabled, onClick = onClick)
            .padding(vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(Modifier.size(24.dp), contentAlignment = Alignment.Center) {
            if (busy) {
                CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp, color = tint)
            } else {
                Icon(icon, contentDescription = null, tint = tint.copy(alpha = tint.alpha * alpha))
            }
        }
        Spacer(Modifier.width(16.dp))
        Column(Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.bodyLarge, color = tint.copy(alpha = tint.alpha * alpha))
            if (subtitle != null) {
                Text(
                    subtitle,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
private fun ChangePasswordDialog(viewModel: SettingsViewModel, dialog: PasswordDialogState) {
    var visible by rememberSaveable { mutableStateOf(false) }
    val enabled = !dialog.isSubmitting
    val transformation = if (visible) VisualTransformation.None else PasswordVisualTransformation()
    val visibilityToggle: @Composable () -> Unit = {
        IconButton(onClick = { visible = !visible }) {
            Icon(
                if (visible) Icons.Filled.VisibilityOff else Icons.Filled.Visibility,
                contentDescription = stringResource(if (visible) R.string.settings_hide_password else R.string.settings_show_password),
            )
        }
    }
    val fields = listOf(
        Triple(PasswordField.CURRENT, viewModel.currentPassword, viewModel::onCurrentPasswordChange),
        Triple(PasswordField.NEW, viewModel.newPassword, viewModel::onNewPasswordChange),
        Triple(PasswordField.CONFIRM, viewModel.confirmPassword, viewModel::onConfirmPasswordChange),
    )

    AlertDialog(
        onDismissRequest = viewModel::closePasswordDialog,
        title = { Text(stringResource(R.string.settings_change_password)) },
        text = {
            Column(
                modifier = Modifier.verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(stringResource(R.string.settings_password_dialog_text), style = MaterialTheme.typography.bodyMedium)
                fields.forEach { (field, value, onChange) ->
                    val error = dialog.fieldErrors[field]
                    val hint = error ?: if (field == PasswordField.NEW) R.string.settings_password_length else null
                    OutlinedTextField(
                        value = value,
                        onValueChange = onChange,
                        modifier = Modifier.fillMaxWidth().semantics {
                            contentType = if (field == PasswordField.CURRENT) ContentType.Password else ContentType.NewPassword
                        },
                        enabled = enabled,
                        label = {
                            Text(
                                stringResource(
                                    when (field) {
                                        PasswordField.CURRENT -> R.string.settings_current_password
                                        PasswordField.NEW -> R.string.settings_new_password
                                        PasswordField.CONFIRM -> R.string.settings_confirm_password
                                    },
                                ),
                            )
                        },
                        trailingIcon = if (field == PasswordField.CURRENT) visibilityToggle else null,
                        isError = error != null,
                        supportingText = hint?.let { { Text(stringResource(it)) } },
                        singleLine = true,
                        visualTransformation = transformation,
                        keyboardOptions = KeyboardOptions(
                            keyboardType = KeyboardType.Password,
                            imeAction = if (field == PasswordField.CONFIRM) ImeAction.Done else ImeAction.Next,
                        ),
                        keyboardActions = KeyboardActions(onDone = { viewModel.changePassword() }),
                    )
                }
                dialog.error?.let { error ->
                    Text(error.text(), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.error)
                }
            }
        },
        confirmButton = {
            TextButton(onClick = viewModel::changePassword, enabled = enabled) {
                if (dialog.isSubmitting) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text(stringResource(R.string.settings_change_password_confirm))
                }
            }
        },
        dismissButton = {
            TextButton(onClick = viewModel::closePasswordDialog, enabled = enabled) {
                Text(stringResource(R.string.common_cancel))
            }
        },
    )
}

/** `7.3` rather than `7.29999…` or `1.0E-4`. */
private fun plainNumber(value: Double): String = BigDecimal.valueOf(value).stripTrailingZeros().toPlainString()
