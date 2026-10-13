package com.newapi.admin.ui.settings

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.PaddingValues
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
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.Logout
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.Info
import androidx.compose.material.icons.outlined.Lock
import androidx.compose.material.icons.outlined.Refresh
import androidx.compose.material.icons.outlined.Visibility
import androidx.compose.material.icons.outlined.VisibilityOff
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
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
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentType
import androidx.compose.ui.semantics.semantics
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
import com.newapi.admin.ui.components.AppAlert
import com.newapi.admin.ui.components.AppAvatar
import com.newapi.admin.ui.components.AppButton
import com.newapi.admin.ui.components.AppCard
import com.newapi.admin.ui.components.AppDialog
import com.newapi.admin.ui.components.AppSnackbarHost
import com.newapi.admin.ui.components.AppTextField
import com.newapi.admin.ui.components.AppTopBar
import com.newapi.admin.ui.components.ButtonVariant
import com.newapi.admin.ui.components.ConfirmDialog
import com.newapi.admin.ui.components.FullScreenLoading
import com.newapi.admin.ui.components.InfoRow
import com.newapi.admin.ui.components.SectionCard
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.components.rememberQuotaFormatter
import com.newapi.admin.ui.theme.AppTheme
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
            AppTopBar(
                title = stringResource(R.string.nav_me),
                actions = {
                    IconButton(onClick = viewModel::refresh, enabled = !state.isRefreshing) {
                        Icon(Icons.Outlined.Refresh, contentDescription = stringResource(R.string.common_refresh))
                    }
                },
            )
        },
        snackbarHost = { AppSnackbarHost(snackbarHostState) },
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
                // Rows run edge to edge, so the card only pads its title.
                AppCard(Modifier.fillMaxWidth(), contentPadding = PaddingValues(top = 16.dp, bottom = 6.dp)) {
                    Text(
                        stringResource(R.string.settings_security_title),
                        style = MaterialTheme.typography.titleSmall,
                        modifier = Modifier.padding(horizontal = 16.dp),
                    )
                    Spacer(Modifier.height(6.dp))
                    ActionRow(
                        icon = Icons.Outlined.Lock,
                        title = stringResource(R.string.settings_change_password),
                        subtitle = stringResource(
                            if (isSession) R.string.settings_change_password_desc else R.string.settings_change_password_unavailable,
                        ),
                        enabled = isSession,
                        onClick = viewModel::openPasswordDialog,
                    )
                    HorizontalDivider(Modifier.padding(vertical = 4.dp), color = AppTheme.colors.border)
                    ActionRow(
                        icon = Icons.AutoMirrored.Outlined.Logout,
                        title = stringResource(R.string.settings_sign_out),
                        subtitle = null,
                        enabled = !state.isSigningOut,
                        busy = state.isSigningOut,
                        destructive = true,
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
    val colors = AppTheme.colors
    val user = account.user
    val displayName = user.displayName.ifBlank { user.username }
    SectionCard(title = null) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            AppAvatar(
                // Two chars when the name starts with a surrogate pair (emoji, rare CJK).
                displayName.take(if (displayName.firstOrNull()?.isHighSurrogate() == true) 2 else 1).uppercase(),
                size = 48.dp,
            )
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
                    color = colors.mutedForeground,
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
            Row(Modifier.padding(top = 4.dp), verticalAlignment = Alignment.Top) {
                Icon(
                    Icons.Outlined.Info,
                    contentDescription = null,
                    tint = colors.mutedForeground,
                    modifier = Modifier.padding(top = 1.dp).size(14.dp),
                )
                Spacer(Modifier.width(6.dp))
                Text(
                    stringResource(R.string.settings_token_note),
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.mutedForeground,
                )
            }
        }
    }
}

/** A figure on a muted tile; it fills the height its row gives it, so a row of tiles lines up. */
@Composable
private fun StatTile(label: String, value: String, modifier: Modifier = Modifier) {
    Column(
        modifier
            .background(AppTheme.colors.muted, MaterialTheme.shapes.medium)
            .padding(10.dp),
        verticalArrangement = Arrangement.spacedBy(2.dp),
    ) {
        Text(
            label,
            style = MaterialTheme.typography.bodySmall,
            color = AppTheme.colors.mutedForeground,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        Text(
            value,
            // titleSmall keeps a four-digit balance on one line in a third of a 360dp-wide screen.
            style = MaterialTheme.typography.titleSmall.copy(fontFeatureSettings = "tnum"),
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

@Composable
private fun ServerCard(baseUrl: String, status: SystemStatus?, formatter: QuotaFormatter) {
    SectionCard(title = stringResource(R.string.settings_server_title)) {
        InfoRow(stringResource(R.string.settings_server_address), baseUrl, copyable = true)
        if (ServerUrl.isCleartext(baseUrl)) {
            val warning = Tone.Warning.colors().content
            // Indented to the value column of InfoRow, so the warning reads as a note on the address.
            Row(verticalAlignment = Alignment.Top, modifier = Modifier.padding(start = 112.dp, bottom = 4.dp)) {
                Icon(
                    Icons.Outlined.WarningAmber,
                    contentDescription = null,
                    tint = warning,
                    modifier = Modifier.padding(top = 1.dp).size(14.dp),
                )
                Spacer(Modifier.width(6.dp))
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

/**
 * One tappable edge-to-edge line in a card: icon tile, title, optional explanation; a spinner replaces the
 * icon while [busy]. [destructive] draws it in the destructive colour.
 */
@Composable
private fun ActionRow(
    icon: ImageVector,
    title: String,
    subtitle: String?,
    enabled: Boolean,
    onClick: () -> Unit,
    busy: Boolean = false,
    destructive: Boolean = false,
) {
    val colors = AppTheme.colors
    val tint = if (destructive) colors.destructive else colors.foreground
    val alpha = if (enabled || busy) 1f else 0.5f
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .size(32.dp)
                .background(if (destructive) Tone.Danger.colors().container else colors.muted, MaterialTheme.shapes.small),
            contentAlignment = Alignment.Center,
        ) {
            if (busy) {
                CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp, color = tint)
            } else {
                Icon(icon, contentDescription = null, tint = tint.copy(alpha = tint.alpha * alpha), modifier = Modifier.size(18.dp))
            }
        }
        Spacer(Modifier.width(12.dp))
        Column(Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.labelLarge, color = tint.copy(alpha = tint.alpha * alpha))
            if (subtitle != null) {
                Text(
                    subtitle,
                    style = MaterialTheme.typography.bodySmall,
                    color = colors.mutedForeground,
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
                if (visible) Icons.Outlined.VisibilityOff else Icons.Outlined.Visibility,
                contentDescription = stringResource(if (visible) R.string.settings_hide_password else R.string.settings_show_password),
                modifier = Modifier.size(18.dp),
            )
        }
    }
    val fields = listOf(
        Triple(PasswordField.CURRENT, viewModel.currentPassword, viewModel::onCurrentPasswordChange),
        Triple(PasswordField.NEW, viewModel.newPassword, viewModel::onNewPasswordChange),
        Triple(PasswordField.CONFIRM, viewModel.confirmPassword, viewModel::onConfirmPasswordChange),
    )

    AppDialog(
        onDismissRequest = viewModel::closePasswordDialog,
        title = stringResource(R.string.settings_change_password),
        content = {
            Column(
                modifier = Modifier.verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                Text(
                    stringResource(R.string.settings_password_dialog_text),
                    style = MaterialTheme.typography.bodyMedium,
                    color = AppTheme.colors.mutedForeground,
                )
                fields.forEach { (field, value, onChange) ->
                    val error = dialog.fieldErrors[field]
                    val hint = error ?: if (field == PasswordField.NEW) R.string.settings_password_length else null
                    AppTextField(
                        value = value,
                        onValueChange = onChange,
                        modifier = Modifier.fillMaxWidth(),
                        inputModifier = Modifier.semantics {
                            contentType = if (field == PasswordField.CURRENT) ContentType.Password else ContentType.NewPassword
                        },
                        enabled = enabled,
                        label = stringResource(
                            when (field) {
                                PasswordField.CURRENT -> R.string.settings_current_password
                                PasswordField.NEW -> R.string.settings_new_password
                                PasswordField.CONFIRM -> R.string.settings_confirm_password
                            },
                        ),
                        trailing = if (field == PasswordField.CURRENT) visibilityToggle else null,
                        isError = error != null,
                        supportingText = hint?.let { stringResource(it) },
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
                    AppAlert(text = error.text(), tone = Tone.Danger, icon = Icons.Outlined.ErrorOutline)
                }
            }
        },
        confirmButton = {
            AppButton(
                stringResource(R.string.settings_change_password_confirm),
                onClick = viewModel::changePassword,
                enabled = enabled,
                loading = dialog.isSubmitting,
            )
        },
        dismissButton = {
            AppButton(
                stringResource(R.string.common_cancel),
                onClick = viewModel::closePasswordDialog,
                variant = ButtonVariant.Outline,
                enabled = enabled,
            )
        },
    )
}

/** `7.3` rather than `7.29999…` or `1.0E-4`. */
private fun plainNumber(value: Double): String = BigDecimal.valueOf(value).stripTrailingZeros().toPlainString()
