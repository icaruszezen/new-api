package com.newapi.admin.ui.redemptions

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.Block
import androidx.compose.material.icons.outlined.ContentCopy
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.Edit
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material.icons.outlined.PlayCircle
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
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
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.format.Formats
import com.newapi.admin.core.model.Redemption
import com.newapi.admin.ui.common.UiMessagesEffect
import com.newapi.admin.ui.components.AppButton
import com.newapi.admin.ui.components.AppIconButton
import com.newapi.admin.ui.components.AppSnackbarHost
import com.newapi.admin.ui.components.AppTopBar
import com.newapi.admin.ui.components.ButtonSize
import com.newapi.admin.ui.components.ButtonVariant
import com.newapi.admin.ui.components.ConfirmDialog
import com.newapi.admin.ui.components.FullScreenError
import com.newapi.admin.ui.components.FullScreenLoading
import com.newapi.admin.ui.components.InfoRow
import com.newapi.admin.ui.components.SectionCard
import com.newapi.admin.ui.components.rememberQuotaFormatter
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.theme.GeistMono
import kotlinx.coroutines.flow.first

@Composable
fun RedemptionDetailScreen(codeId: Int, onBack: () -> Unit, onEdit: (Int) -> Unit, onOpenUser: (Int) -> Unit) {
    val container = LocalAppContainer.current
    val viewModel = viewModel(key = "redemption-detail-$codeId") { RedemptionDetailViewModel(container, codeId) }
    val state by viewModel.state.collectAsStateWithLifecycle()
    val snackbarHostState = remember { SnackbarHostState() }
    var confirmDelete by rememberSaveable { mutableStateOf(false) }

    UiMessagesEffect(viewModel.messages, snackbarHostState)
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    LaunchedEffect(state.deleted) {
        if (!state.deleted) return@LaunchedEffect
        // Deleted while the admin was on another screen: popping is ignored until the entry is RESUMED again.
        lifecycle.currentStateFlow.first { it.isAtLeast(Lifecycle.State.RESUMED) }
        onBack()
    }

    val code = state.code
    val codeState = code?.state(System.currentTimeMillis() / 1000)
    Scaffold(
        topBar = {
            AppTopBar(
                title = code?.displayName ?: stringResource(R.string.redemptions_detail_title, codeId),
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = stringResource(R.string.common_back))
                    }
                },
                actions = {
                    if (code != null && codeState?.canEdit == true) {
                        IconButton(onClick = { onEdit(code.id) }) {
                            Icon(Icons.Outlined.Edit, contentDescription = stringResource(R.string.common_edit))
                        }
                    }
                },
            )
        },
        snackbarHost = { AppSnackbarHost(snackbarHostState) },
    ) { padding ->
        PullToRefreshBox(
            isRefreshing = state.isRefreshing,
            onRefresh = viewModel::refresh,
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
            val error = state.error
            when {
                code != null && codeState != null -> Column(
                    Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    CodeHeader(code, codeState, state.isChangingStatus, onSetEnabled = viewModel::setEnabled)
                    KeySection(code.key)
                    DetailsSection(code, codeState)
                    if (code.usedUserId > 0) {
                        AppButton(
                            stringResource(R.string.redemptions_open_user),
                            onClick = { onOpenUser(code.usedUserId) },
                            modifier = Modifier.fillMaxWidth(),
                            variant = ButtonVariant.Outline,
                            leadingIcon = Icons.Outlined.Person,
                        )
                    }
                    // loading = isDeleting also disables the button while the delete runs.
                    AppButton(
                        stringResource(R.string.redemptions_delete_title),
                        onClick = { confirmDelete = true },
                        modifier = Modifier.fillMaxWidth(),
                        variant = ButtonVariant.DestructiveOutline,
                        loading = state.isDeleting,
                        leadingIcon = Icons.Outlined.Delete,
                    )
                }
                state.isLoading -> FullScreenLoading()
                error != null -> FullScreenError(error, onRetry = viewModel::refresh)
            }
        }
    }

    if (confirmDelete && code != null) {
        ConfirmDialog(
            title = stringResource(R.string.redemptions_delete_title),
            text = stringResource(R.string.redemptions_delete_message, code.displayName),
            confirmText = stringResource(R.string.common_delete),
            destructive = true,
            onConfirm = {
                confirmDelete = false
                viewModel.delete()
            },
            onDismiss = { confirmDelete = false },
        )
    }
}

@Composable
private fun CodeHeader(code: Redemption, codeState: CodeState, isChangingStatus: Boolean, onSetEnabled: (Boolean) -> Unit) {
    val formatter = rememberQuotaFormatter()
    SectionCard(title = null) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                code.displayName,
                style = MaterialTheme.typography.titleLarge,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(8.dp))
            CodeStatusPill(codeState)
        }
        Spacer(Modifier.height(12.dp))
        Text(
            stringResource(R.string.redemptions_label_quota),
            style = MaterialTheme.typography.bodySmall,
            color = AppTheme.colors.mutedForeground,
        )
        Text(formatter.format(code.quota), style = MaterialTheme.typography.headlineSmall.copy(fontFeatureSettings = "tnum"))
        if (codeState.canToggle) {
            Spacer(Modifier.height(16.dp))
            val enabled = codeState == CodeState.UNUSED
            // loading = isChangingStatus keeps the button disabled while the change runs.
            AppButton(
                stringResource(if (enabled) R.string.common_disable else R.string.common_enable),
                onClick = { onSetEnabled(!enabled) },
                variant = ButtonVariant.Outline,
                size = ButtonSize.Small,
                loading = isChangingStatus,
                leadingIcon = if (enabled) Icons.Outlined.Block else Icons.Outlined.PlayCircle,
            )
        }
    }
}

@Composable
private fun KeySection(key: String) {
    val clipboard = LocalClipboardManager.current
    SectionCard(
        stringResource(R.string.redemptions_section_code),
        action = {
            AppIconButton(
                icon = Icons.Outlined.ContentCopy,
                contentDescription = stringResource(R.string.redemptions_copy_code),
                onClick = { clipboard.setText(AnnotatedString(key)) },
                size = 28.dp,
                iconSize = 14.dp,
            )
        },
    ) {
        Surface(
            color = AppTheme.colors.muted,
            contentColor = AppTheme.colors.foreground,
            shape = MaterialTheme.shapes.small,
            modifier = Modifier.fillMaxWidth(),
        ) {
            SelectionContainer {
                Text(key, style = MaterialTheme.typography.bodyMedium, fontFamily = GeistMono, modifier = Modifier.padding(12.dp))
            }
        }
    }
}

@Composable
private fun DetailsSection(code: Redemption, codeState: CodeState) {
    SectionCard(stringResource(R.string.redemptions_section_details)) {
        InfoRow(stringResource(R.string.redemptions_label_id), code.id.toString(), copyable = true)
        InfoRow(stringResource(R.string.redemptions_label_created), Formats.dateTime(code.createdTime))
        InfoRow(
            stringResource(R.string.redemptions_label_expires),
            if (code.expiredTime == 0L) stringResource(R.string.redemptions_never_expires) else Formats.dateTime(code.expiredTime),
        )
        InfoRow(
            stringResource(R.string.redemptions_label_creator),
            if (code.creatorId > 0) stringResource(R.string.redemptions_user_value, code.creatorId) else "",
        )
        if (codeState == CodeState.USED || code.usedUserId > 0) {
            InfoRow(
                stringResource(R.string.redemptions_label_redeemed_by),
                if (code.usedUserId > 0) stringResource(R.string.redemptions_user_value, code.usedUserId) else "",
            )
            InfoRow(stringResource(R.string.redemptions_label_redeemed_at), Formats.dateTime(code.redeemedTime))
        }
    }
}
