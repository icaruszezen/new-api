package com.newapi.admin.ui.users

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
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
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.ReceiptLong
import androidx.compose.material.icons.filled.AccountBalanceWallet
import androidx.compose.material.icons.filled.ArrowDownward
import androidx.compose.material.icons.filled.ArrowUpward
import androidx.compose.material.icons.filled.Block
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.DeleteForever
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.Key
import androidx.compose.material.icons.filled.Password
import androidx.compose.material.icons.filled.PersonOff
import androidx.compose.material.icons.filled.Security
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.VisibilityOff
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SheetValue
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.format.Formats
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.core.model.AdminUser
import com.newapi.admin.core.model.QuotaMode
import com.newapi.admin.core.model.Roles
import com.newapi.admin.core.model.UserStatus
import com.newapi.admin.ui.common.UiMessage
import com.newapi.admin.ui.common.errorMessage
import com.newapi.admin.ui.common.text
import com.newapi.admin.ui.components.ConfirmDialog
import com.newapi.admin.ui.components.EmptyState
import com.newapi.admin.ui.components.FullScreenError
import com.newapi.admin.ui.components.FullScreenLoading
import com.newapi.admin.ui.components.InfoRow
import com.newapi.admin.ui.components.SectionCard
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.components.rememberCurrentUser
import com.newapi.admin.ui.components.rememberQuotaFormatter
import com.newapi.admin.ui.theme.Tone
import com.newapi.admin.ui.theme.colors
import kotlinx.coroutines.flow.first

/** One user: profile, quota, invites, bindings and the management actions the signed-in admin may run. */
@Composable
fun UserDetailScreen(userId: Int, onBack: () -> Unit, onOpenLogs: (username: String) -> Unit) {
    val container = LocalAppContainer.current
    val viewModel = viewModel(key = "user-$userId") { UserDetailViewModel(container, userId) }
    val state by viewModel.state.collectAsStateWithLifecycle()
    val formatter = rememberQuotaFormatter()
    val me = rememberCurrentUser()
    val status by container.status.collectAsStateWithLifecycle()
    val snackbarHostState = remember { SnackbarHostState() }
    var confirm by rememberSaveable { mutableStateOf<UserOperation?>(null) }

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
    // The admin may have opened the logs while the delete ran. Coming back, this entry stays STARTED until
    // its enter transition ends and a pop before RESUMED is ignored, so wait for it instead of firing once.
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    val currentOnBack by rememberUpdatedState(onBack)
    LaunchedEffect(state.deleted) {
        if (!state.deleted) return@LaunchedEffect
        lifecycle.currentStateFlow.first { it.isAtLeast(Lifecycle.State.RESUMED) }
        currentOnBack()
    }

    val user = state.user
    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        user?.displayName?.ifBlank { user.username } ?: stringResource(R.string.users_detail_title, userId),
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_back))
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        val loadError = state.loadError
        when {
            state.deleted -> EmptyState(stringResource(R.string.users_deleted_done), Modifier.padding(padding), icon = Icons.Filled.PersonOff)
            user == null && loadError != null -> FullScreenError(loadError, onRetry = viewModel::load, modifier = Modifier.padding(padding))
            user == null -> FullScreenLoading(Modifier.padding(padding))
            else -> PullToRefreshBox(
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
                    UserHeader(user)
                    if (user.isDeleted) Notice(stringResource(R.string.users_deleted_banner))
                    ProfileSection(user)
                    QuotaSection(user, formatter)
                    InviteSection(user, formatter)
                    BindingsSection(user)
                    if (user.role >= Roles.ADMIN && !user.adminPermissions.isNullOrEmpty()) PermissionsSection(user)
                    ActionsSection(
                        user = user,
                        myRole = me?.role ?: Roles.GUEST,
                        running = state.running,
                        onOpenLogs = { onOpenLogs(user.username) },
                        onOpenDialog = viewModel::openDialog,
                        onAction = { operation ->
                            if (operation == UserOperation.ENABLE) viewModel.runAction(operation) else confirm = operation
                        },
                    )
                }
            }
        }
    }

    if (user == null) return
    val isSelf = me?.id == user.id
    val busy = state.running != null

    confirm?.let { operation ->
        val name = user.username
        val (title, text, confirmText) = when (operation) {
            UserOperation.DISABLE -> Triple(
                stringResource(R.string.users_confirm_disable_title),
                stringResource(R.string.users_confirm_disable_text, name),
                stringResource(R.string.common_disable),
            )
            UserOperation.PROMOTE -> Triple(
                stringResource(R.string.users_confirm_promote_title),
                stringResource(R.string.users_confirm_promote_text, name),
                stringResource(R.string.users_action_promote),
            )
            UserOperation.DEMOTE -> Triple(
                stringResource(R.string.users_confirm_demote_title),
                stringResource(R.string.users_confirm_demote_text, name),
                stringResource(R.string.users_action_demote),
            )
            UserOperation.RESET_2FA -> Triple(
                stringResource(R.string.users_action_reset_2fa),
                stringResource(R.string.users_confirm_reset_2fa_text, name),
                stringResource(R.string.users_action_reset_2fa),
            )
            UserOperation.RESET_PASSKEY -> Triple(
                stringResource(R.string.users_action_reset_passkey),
                stringResource(R.string.users_confirm_reset_passkey_text, name),
                stringResource(R.string.users_action_reset_passkey),
            )
            UserOperation.DELETE -> Triple(
                stringResource(R.string.users_confirm_delete_title),
                stringResource(R.string.users_confirm_delete_text, name),
                stringResource(R.string.common_delete),
            )
            else -> Triple("", "", "")
        }
        val selfNote = if (isSelf) "\n\n" + stringResource(R.string.users_self_warning) else ""
        ConfirmDialog(
            title = title,
            text = text + selfNote,
            confirmText = confirmText,
            onConfirm = {
                confirm = null
                viewModel.runAction(operation)
            },
            onDismiss = { confirm = null },
            destructive = operation != UserOperation.PROMOTE,
        )
    }

    when (state.dialog) {
        UserDialog.ADJUST_QUOTA -> AdjustQuotaDialog(
            user = user,
            formatter = formatter,
            busy = busy,
            error = state.dialogError,
            displayReady = status != null,
            onReloadDisplay = container::refreshStatus,
            onSubmit = viewModel::adjustQuota,
            onDismiss = viewModel::closeDialog,
        )
        UserDialog.EDIT_PROFILE -> EditProfileSheet(
            user = user,
            groups = state.groups,
            isSelf = isSelf,
            busy = busy,
            error = state.dialogError,
            onRetryGroups = viewModel::loadGroups,
            onSave = viewModel::saveProfile,
            onDismiss = viewModel::closeDialog,
        )
        UserDialog.CHANGE_PASSWORD -> ChangePasswordDialog(viewModel, state, isSelf)
        null -> Unit
    }
}

@Composable
private fun UserHeader(user: AdminUser) {
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
                Text(displayName, style = MaterialTheme.typography.titleMedium, maxLines = 2, overflow = TextOverflow.Ellipsis)
                Text(
                    "@${user.username} · #${user.id}",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
            RolePill(user.role)
            UserStatusPill(user)
            if (user.group.isNotEmpty()) StatusPill(user.group, Tone.Neutral, Modifier.weight(1f, fill = false))
        }
    }
}

@Composable
private fun Notice(text: String) {
    val colors = Tone.Warning.colors()
    Surface(color = colors.container, contentColor = colors.content, shape = MaterialTheme.shapes.medium, modifier = Modifier.fillMaxWidth()) {
        Row(Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
            Icon(Icons.Filled.Info, contentDescription = null, modifier = Modifier.size(18.dp))
            Spacer(Modifier.width(8.dp))
            Text(text, style = MaterialTheme.typography.bodyMedium)
        }
    }
}

@Composable
private fun ProfileSection(user: AdminUser) {
    SectionCard(title = stringResource(R.string.users_section_profile)) {
        InfoRow(stringResource(R.string.users_field_id), user.id.toString(), copyable = true)
        InfoRow(stringResource(R.string.users_field_username), user.username, copyable = true)
        InfoRow(stringResource(R.string.users_field_display_name), user.displayName)
        InfoRow(stringResource(R.string.users_field_email), user.email, copyable = true)
        InfoRow(stringResource(R.string.users_field_group), user.group)
        InfoRow(stringResource(R.string.users_field_role), roleLabel(user.role))
        InfoRow(stringResource(R.string.users_field_status), userStatusLabel(user))
        InfoRow(stringResource(R.string.users_field_remark), user.remark)
        InfoRow(stringResource(R.string.users_field_created), Formats.dateTime(user.createdAt))
        InfoRow(
            stringResource(R.string.users_field_last_login),
            if (user.lastLoginAt > 0) Formats.dateTime(user.lastLoginAt) else stringResource(R.string.users_never),
        )
    }
}

@Composable
private fun QuotaSection(user: AdminUser, formatter: QuotaFormatter) {
    SectionCard(title = stringResource(R.string.users_section_quota)) {
        InfoRow(stringResource(R.string.users_field_remaining), formatter.format(user.quota))
        InfoRow(stringResource(R.string.users_field_used), formatter.format(user.usedQuota))
        InfoRow(stringResource(R.string.users_field_total), formatter.format(user.quota + user.usedQuota))
        InfoRow(stringResource(R.string.users_field_requests), Formats.count(user.requestCount))
        Spacer(Modifier.height(8.dp))
        QuotaBar(user.quota, user.usedQuota)
    }
}

@Composable
private fun InviteSection(user: AdminUser, formatter: QuotaFormatter) {
    SectionCard(title = stringResource(R.string.users_section_invite)) {
        InfoRow(stringResource(R.string.users_field_aff_code), user.affCode, copyable = true)
        InfoRow(stringResource(R.string.users_field_aff_count), Formats.count(user.affCount.toLong()))
        InfoRow(stringResource(R.string.users_field_aff_quota), formatter.format(user.affQuota))
        InfoRow(stringResource(R.string.users_field_aff_history), formatter.format(user.affHistoryQuota))
        InfoRow(stringResource(R.string.users_field_inviter), if (user.inviterId > 0) "#${user.inviterId}" else "")
    }
}

@Composable
private fun BindingsSection(user: AdminUser) {
    val bindings = listOf(
        R.string.users_binding_github to user.githubId,
        R.string.users_binding_discord to user.discordId,
        R.string.users_binding_oidc to user.oidcId,
        R.string.users_binding_wechat to user.wechatId,
        R.string.users_binding_telegram to user.telegramId,
        R.string.users_binding_linuxdo to user.linuxDoId,
    ).filter { it.second.isNotEmpty() }
    if (bindings.isEmpty()) return
    SectionCard(title = stringResource(R.string.users_section_bindings)) {
        bindings.forEach { (label, value) -> InfoRow(stringResource(label), value, copyable = true) }
    }
}

@Composable
private fun PermissionsSection(user: AdminUser) {
    val permissions = user.adminPermissions ?: return
    SectionCard(title = stringResource(R.string.users_section_permissions)) {
        permissions.entries.sortedBy { it.key }.forEach { (resource, actions) ->
            val granted = actions.filterValues { it }.keys.map { action ->
                when (action) {
                    "read" -> stringResource(R.string.users_perm_read)
                    "operate" -> stringResource(R.string.users_perm_operate)
                    "write" -> stringResource(R.string.users_perm_write)
                    "sensitive_write" -> stringResource(R.string.users_perm_sensitive_write)
                    "secret_view" -> stringResource(R.string.users_perm_secret_view)
                    else -> action
                }
            }
            InfoRow(
                if (resource == "channel") stringResource(R.string.users_perm_channel) else resource,
                granted.joinToString("\n").ifEmpty { stringResource(R.string.users_perm_none) },
            )
        }
    }
}

/**
 * Gating follows the server (critique gap 3): manage only when root or strictly higher role; promote
 * only for root; never disable, demote or delete a root; hard delete needs a strictly higher role;
 * deleted users are read-only.
 */
@Composable
private fun ActionsSection(
    user: AdminUser,
    myRole: Int,
    running: UserOperation?,
    onOpenLogs: () -> Unit,
    onOpenDialog: (UserDialog) -> Unit,
    onAction: (UserOperation) -> Unit,
) {
    val iAmRoot = myRole >= Roles.ROOT
    val canManage = !user.isDeleted && (iAmRoot || myRole > user.role)

    SectionCard(title = stringResource(R.string.users_section_actions)) {
        ActionItem(Icons.AutoMirrored.Filled.ReceiptLong, stringResource(R.string.users_action_view_logs), onClick = onOpenLogs)
        if (canManage) {
            ManageActions(user, iAmRoot, myRole, running, onOpenDialog, onAction)
        } else if (!user.isDeleted) {
            Text(
                stringResource(R.string.users_cannot_manage),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(vertical = 8.dp),
            )
        }
    }
}

@Composable
private fun ManageActions(
    user: AdminUser,
    iAmRoot: Boolean,
    myRole: Int,
    running: UserOperation?,
    onOpenDialog: (UserDialog) -> Unit,
    onAction: (UserOperation) -> Unit,
) {
    val targetIsRoot = user.role >= Roles.ROOT
    val idle = running == null
    Column {
        ActionItem(
            Icons.Filled.AccountBalanceWallet,
            stringResource(R.string.users_action_adjust_quota),
            onClick = { onOpenDialog(UserDialog.ADJUST_QUOTA) },
            enabled = idle,
            busy = running == UserOperation.ADJUST_QUOTA,
        )
        ActionItem(
            Icons.Filled.Edit,
            stringResource(R.string.users_action_edit_profile),
            subtitle = stringResource(R.string.users_action_edit_profile_desc),
            onClick = { onOpenDialog(UserDialog.EDIT_PROFILE) },
            enabled = idle,
            busy = running == UserOperation.EDIT_PROFILE,
        )
        ActionItem(
            Icons.Filled.Password,
            stringResource(R.string.users_action_change_password),
            onClick = { onOpenDialog(UserDialog.CHANGE_PASSWORD) },
            enabled = idle,
            busy = running == UserOperation.CHANGE_PASSWORD,
        )
        HorizontalDivider(Modifier.padding(vertical = 4.dp))
        if (user.status == UserStatus.DISABLED) {
            ActionItem(
                Icons.Filled.CheckCircle,
                stringResource(R.string.common_enable),
                subtitle = stringResource(R.string.users_action_enable_desc),
                onClick = { onAction(UserOperation.ENABLE) },
                enabled = idle,
                busy = running == UserOperation.ENABLE,
            )
        } else if (!targetIsRoot) {
            ActionItem(
                Icons.Filled.Block,
                stringResource(R.string.common_disable),
                subtitle = stringResource(R.string.users_action_disable_desc),
                onClick = { onAction(UserOperation.DISABLE) },
                enabled = idle,
                busy = running == UserOperation.DISABLE,
            )
        }
        if (iAmRoot && user.role < Roles.ADMIN) {
            ActionItem(
                Icons.Filled.ArrowUpward,
                stringResource(R.string.users_action_promote),
                onClick = { onAction(UserOperation.PROMOTE) },
                enabled = idle,
                busy = running == UserOperation.PROMOTE,
            )
        }
        if (user.role >= Roles.ADMIN && !targetIsRoot) {
            ActionItem(
                Icons.Filled.ArrowDownward,
                stringResource(R.string.users_action_demote),
                onClick = { onAction(UserOperation.DEMOTE) },
                enabled = idle,
                busy = running == UserOperation.DEMOTE,
            )
        }
        ActionItem(
            Icons.Filled.Security,
            stringResource(R.string.users_action_reset_2fa),
            onClick = { onAction(UserOperation.RESET_2FA) },
            enabled = idle,
            busy = running == UserOperation.RESET_2FA,
        )
        ActionItem(
            Icons.Filled.Key,
            stringResource(R.string.users_action_reset_passkey),
            onClick = { onAction(UserOperation.RESET_PASSKEY) },
            enabled = idle,
            busy = running == UserOperation.RESET_PASSKEY,
        )
        if (myRole > user.role && !targetIsRoot) {
            HorizontalDivider(Modifier.padding(vertical = 4.dp))
            ActionItem(
                Icons.Filled.DeleteForever,
                stringResource(R.string.users_action_delete),
                subtitle = stringResource(R.string.users_action_delete_desc),
                onClick = { onAction(UserOperation.DELETE) },
                enabled = idle,
                busy = running == UserOperation.DELETE,
                tint = MaterialTheme.colorScheme.error,
            )
        }
    }
}

/** One tappable line in the actions card; a spinner replaces the icon while [busy]. */
@Composable
private fun ActionItem(
    icon: ImageVector,
    title: String,
    onClick: () -> Unit,
    subtitle: String? = null,
    enabled: Boolean = true,
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
                    color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = alpha),
                )
            }
        }
    }
}

@Composable
private fun AdjustQuotaDialog(
    user: AdminUser,
    formatter: QuotaFormatter,
    busy: Boolean,
    error: Throwable?,
    /** False until `/api/status` has loaded; converting with the fallback rate could credit the wrong amount. */
    displayReady: Boolean,
    onReloadDisplay: () -> Unit,
    onSubmit: (QuotaMode, Long) -> Unit,
    onDismiss: () -> Unit,
) {
    var mode by rememberSaveable { mutableStateOf(QuotaMode.ADD) }
    var amount by rememberSaveable { mutableStateOf("") }
    val input = BalanceInput.parse(amount, mode, user.quota, formatter)
    val canSubmit = !busy && displayReady && input is BalanceInput.Valid
    val submit = {
        if (canSubmit && input is BalanceInput.Valid) onSubmit(mode, input.value)
    }
    val modes = QuotaMode.entries

    LaunchedEffect(displayReady) {
        if (!displayReady) onReloadDisplay()
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.users_action_adjust_quota)) },
        text = {
            Column(
                modifier = Modifier.verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
                    modes.forEachIndexed { index, option ->
                        SegmentedButton(
                            selected = mode == option,
                            onClick = { mode = option },
                            shape = SegmentedButtonDefaults.itemShape(index, modes.size),
                            enabled = !busy,
                        ) {
                            Text(
                                stringResource(
                                    when (option) {
                                        QuotaMode.ADD -> R.string.users_quota_mode_add
                                        QuotaMode.SUBTRACT -> R.string.users_quota_mode_subtract
                                        QuotaMode.OVERRIDE -> R.string.users_quota_mode_set
                                    },
                                ),
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                    }
                }
                OutlinedTextField(
                    value = amount,
                    onValueChange = { amount = it },
                    modifier = Modifier.fillMaxWidth(),
                    enabled = !busy,
                    singleLine = true,
                    label = {
                        Text(
                            if (formatter.isTokens) {
                                stringResource(R.string.users_quota_amount_tokens)
                            } else {
                                stringResource(R.string.users_quota_amount, formatter.unitLabel)
                            },
                        )
                    },
                    textStyle = MaterialTheme.typography.bodyLarge.copy(fontFeatureSettings = "tnum"),
                    isError = input is BalanceInput.Invalid,
                    supportingText = when (input) {
                        is BalanceInput.Invalid -> { { Text(input.message.text()) } }
                        is BalanceInput.Valid ->
                            if (formatter.isTokens) null else { { Text(stringResource(R.string.users_quota_units, Formats.count(input.value))) } }
                        BalanceInput.Empty -> null
                    },
                    keyboardOptions = KeyboardOptions(
                        keyboardType = if (formatter.isTokens) KeyboardType.Number else KeyboardType.Decimal,
                        imeAction = ImeAction.Done,
                    ),
                    keyboardActions = KeyboardActions(onDone = { submit() }),
                )
                Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    Text(
                        stringResource(R.string.users_quota_current, formatter.format(user.quota)),
                        style = MaterialTheme.typography.bodyMedium.copy(fontFeatureSettings = "tnum"),
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    if (input is BalanceInput.Valid) {
                        Text(
                            stringResource(R.string.users_quota_result, formatter.format(input.resultingBalance)),
                            style = MaterialTheme.typography.bodyMedium.copy(fontFeatureSettings = "tnum"),
                            fontWeight = FontWeight.Medium,
                        )
                        if (input.resultingBalance < 0) {
                            Text(
                                stringResource(R.string.users_quota_negative_warning),
                                style = MaterialTheme.typography.bodySmall,
                                color = Tone.Warning.colors().content,
                            )
                        }
                    }
                }
                if (!displayReady) {
                    Text(
                        stringResource(R.string.users_quota_settings_missing),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                    TextButton(onClick = onReloadDisplay) { Text(stringResource(R.string.common_retry)) }
                }
                if (mode == QuotaMode.OVERRIDE) {
                    Text(
                        stringResource(R.string.users_quota_cache_note),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                if (error != null) {
                    Text(errorMessage(error), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.error)
                }
            }
        },
        confirmButton = {
            TextButton(onClick = submit, enabled = canSubmit) {
                if (busy) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text(stringResource(R.string.common_confirm))
                }
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss, enabled = !busy) { Text(stringResource(R.string.common_cancel)) }
        },
    )
}

@Composable
private fun EditProfileSheet(
    user: AdminUser,
    groups: GroupList,
    isSelf: Boolean,
    busy: Boolean,
    error: Throwable?,
    onRetryGroups: () -> Unit,
    onSave: (original: ProfileFields, edited: ProfileFields) -> Unit,
    onDismiss: () -> Unit,
) {
    // What the form started from; the ViewModel only overwrites fields that differ from it.
    val initialDisplayName = rememberSaveable { user.displayName }
    val initialGroup = rememberSaveable { user.group }
    val initialRemark = rememberSaveable { user.remark }
    var displayName by rememberSaveable { mutableStateOf(initialDisplayName) }
    var group by rememberSaveable { mutableStateOf(initialGroup) }
    var remark by rememberSaveable { mutableStateOf(initialRemark) }

    val currentBusy by rememberUpdatedState(busy)
    val sheetState = rememberModalBottomSheetState(
        skipPartiallyExpanded = true,
        confirmValueChange = { it != SheetValue.Hidden || !currentBusy },
    )
    val displayNameLength = displayName.trim().runeCount()
    val remarkLength = remark.runeCount()
    val valid = displayNameLength <= MAX_NAME_LENGTH && remarkLength <= MAX_REMARK_LENGTH
    val groupChanged = group != initialGroup

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(start = 24.dp, end = 24.dp, bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(stringResource(R.string.users_action_edit_profile), style = MaterialTheme.typography.titleLarge)
            InfoRow(stringResource(R.string.users_field_username), user.username)
            OutlinedTextField(
                value = displayName,
                onValueChange = { displayName = it },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy,
                singleLine = true,
                label = { Text(stringResource(R.string.users_field_display_name)) },
                placeholder = { Text(user.username) },
                isError = displayNameLength > MAX_NAME_LENGTH,
                supportingText = {
                    Text(
                        if (displayNameLength > MAX_NAME_LENGTH) {
                            stringResource(R.string.users_error_too_long, MAX_NAME_LENGTH)
                        } else {
                            stringResource(R.string.users_display_name_hint)
                        },
                    )
                },
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
            )
            GroupDropdown(
                value = group,
                options = groups.names,
                onSelect = { group = it },
                label = stringResource(R.string.users_field_group),
                enabled = !busy,
                supportingText = when {
                    groupChanged -> stringResource(R.string.users_group_signout_warning) +
                        if (isSelf) " " + stringResource(R.string.users_self_warning) else ""
                    groups.failed -> stringResource(R.string.users_groups_load_failed)
                    else -> null
                },
            )
            if (groups.failed) {
                TextButton(onClick = onRetryGroups, enabled = !busy) { Text(stringResource(R.string.common_retry)) }
            }
            OutlinedTextField(
                value = remark,
                onValueChange = { remark = it },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy,
                minLines = 2,
                maxLines = 5,
                label = { Text(stringResource(R.string.users_field_remark)) },
                isError = remarkLength > MAX_REMARK_LENGTH,
                supportingText = {
                    Text(
                        if (remarkLength > MAX_REMARK_LENGTH) {
                            stringResource(R.string.users_error_too_long, MAX_REMARK_LENGTH)
                        } else {
                            stringResource(R.string.users_length_counter, remarkLength, MAX_REMARK_LENGTH)
                        },
                    )
                },
            )
            if (error != null) {
                Text(errorMessage(error), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.error)
            }
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
                TextButton(onClick = onDismiss, enabled = !busy, modifier = Modifier.weight(1f)) {
                    Text(stringResource(R.string.common_cancel))
                }
                Button(
                    onClick = {
                        onSave(
                            ProfileFields(initialDisplayName, initialGroup, initialRemark),
                            ProfileFields(displayName, group, remark),
                        )
                    },
                    enabled = !busy && valid,
                    modifier = Modifier.weight(1f),
                ) {
                    if (busy) {
                        CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp, color = MaterialTheme.colorScheme.onPrimary)
                    } else {
                        Text(stringResource(R.string.common_save))
                    }
                }
            }
        }
    }
}

@Composable
private fun ChangePasswordDialog(viewModel: UserDetailViewModel, state: UserDetailUiState, isSelf: Boolean) {
    var visible by rememberSaveable { mutableStateOf(false) }
    val busy = state.running != null
    val transformation = if (visible) VisualTransformation.None else PasswordVisualTransformation()
    val newError = state.passwordErrors[PasswordInput.NEW]
    val confirmError = state.passwordErrors[PasswordInput.CONFIRM]

    AlertDialog(
        onDismissRequest = viewModel::closeDialog,
        title = { Text(stringResource(R.string.users_action_change_password)) },
        text = {
            Column(
                modifier = Modifier.verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(
                    stringResource(R.string.users_password_warning) + if (isSelf) "\n" + stringResource(R.string.users_self_warning) else "",
                    style = MaterialTheme.typography.bodyMedium,
                )
                OutlinedTextField(
                    value = viewModel.newPassword,
                    onValueChange = viewModel::onNewPasswordChange,
                    modifier = Modifier.fillMaxWidth(),
                    enabled = !busy,
                    singleLine = true,
                    label = { Text(stringResource(R.string.users_password_new)) },
                    trailingIcon = {
                        IconButton(onClick = { visible = !visible }) {
                            Icon(
                                if (visible) Icons.Filled.VisibilityOff else Icons.Filled.Visibility,
                                contentDescription = stringResource(if (visible) R.string.users_hide_password else R.string.users_show_password),
                            )
                        }
                    },
                    isError = newError != null,
                    supportingText = { Text(stringResource(newError ?: R.string.users_password_length)) },
                    visualTransformation = transformation,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Next),
                )
                OutlinedTextField(
                    value = viewModel.confirmPassword,
                    onValueChange = viewModel::onConfirmPasswordChange,
                    modifier = Modifier.fillMaxWidth(),
                    enabled = !busy,
                    singleLine = true,
                    label = { Text(stringResource(R.string.users_password_confirm)) },
                    isError = confirmError != null,
                    supportingText = confirmError?.let { { Text(stringResource(it)) } },
                    visualTransformation = transformation,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Done),
                    keyboardActions = KeyboardActions(onDone = { viewModel.changePassword() }),
                )
                state.dialogError?.let { error ->
                    Text(errorMessage(error), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.error)
                }
            }
        },
        confirmButton = {
            TextButton(onClick = viewModel::changePassword, enabled = !busy) {
                if (busy) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text(stringResource(R.string.users_action_change_password))
                }
            }
        },
        dismissButton = {
            TextButton(onClick = viewModel::closeDialog, enabled = !busy) { Text(stringResource(R.string.common_cancel)) }
        },
    )
}
