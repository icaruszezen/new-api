package com.newapi.admin.ui.channels

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.BasicText
import androidx.compose.foundation.text.TextAutoSize
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.automirrored.outlined.ReceiptLong
import androidx.compose.material.icons.outlined.AccountBalanceWallet
import androidx.compose.material.icons.outlined.Block
import androidx.compose.material.icons.outlined.CheckCircle
import androidx.compose.material.icons.outlined.ContentCopy
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.Edit
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.NetworkCheck
import androidx.compose.material.icons.outlined.PlayCircle
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
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
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.format.Formats
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.core.model.Channel
import com.newapi.admin.core.model.ChannelStatus
import com.newapi.admin.ui.common.errorMessage
import com.newapi.admin.ui.components.AppAlert
import com.newapi.admin.ui.components.AppButton
import com.newapi.admin.ui.components.AppDialog
import com.newapi.admin.ui.components.AppIconButton
import com.newapi.admin.ui.components.AppSnackbarHost
import com.newapi.admin.ui.components.AppTextField
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
import com.newapi.admin.ui.theme.Tone
import kotlinx.coroutines.flow.first
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement

/** Models shown before "Show all". */
private const val COLLAPSED_MODEL_COUNT = 24

/** The model the server tests when none is chosen and the channel has neither a test model nor models. */
private const val SERVER_FALLBACK_TEST_MODEL = "gpt-4o-mini"

private val prettyJson = Json { prettyPrint = true }

@Composable
fun ChannelDetailScreen(channelId: Int, onBack: () -> Unit, onEdit: (Int) -> Unit, onOpenLogs: (channelId: Int) -> Unit) {
    val container = LocalAppContainer.current
    val viewModel = viewModel(key = "channel-detail-$channelId") { ChannelDetailViewModel(container, channelId) }
    val state by viewModel.state.collectAsStateWithLifecycle()
    val testRun by viewModel.test.collectAsStateWithLifecycle()
    val permissions = rememberChannelPermissions()
    val formatter = rememberQuotaFormatter()
    val snackbarHostState = remember { SnackbarHostState() }
    var showTestDialog by rememberSaveable { mutableStateOf(false) }
    var confirmDelete by rememberSaveable { mutableStateOf(false) }

    ChannelMessagesEffect(viewModel.messages, snackbarHostState)
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    LaunchedEffect(state.deleted) {
        if (!state.deleted) return@LaunchedEffect
        // Deleted while the admin was on another screen: popping is ignored until the entry is RESUMED again.
        lifecycle.currentStateFlow.first { it.isAtLeast(Lifecycle.State.RESUMED) }
        onBack()
    }

    val channel = state.channel
    Scaffold(
        topBar = {
            AppTopBar(
                title = channel?.displayName ?: stringResource(R.string.channels_detail_title),
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = stringResource(R.string.common_back))
                    }
                },
                actions = {
                    if (channel != null && permissions.write) {
                        IconButton(onClick = { onEdit(channel.id) }) {
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
                channel != null -> Column(
                    Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    ChannelHeader(channel) {
                        ChannelActions(
                            channel = channel,
                            permissions = permissions,
                            isChangingStatus = state.isChangingStatus,
                            onTest = { showTestDialog = true },
                            onSetEnabled = viewModel::setEnabled,
                            onOpenLogs = { onOpenLogs(channel.id) },
                        )
                    }
                    BasicSection(channel)
                    ModelsSection(channel)
                    val mapping = channel.modelMapping?.trim().orEmpty()
                    if (mapping.isNotEmpty() && mapping != "{}") ModelMappingSection(mapping)
                    if (channel.channelInfo.isMultiKey) MultiKeySection(channel)
                    StatsSection(
                        channel = channel,
                        formatter = formatter,
                        canUpdateBalance = permissions.operate && channel.hasBalance,
                        isUpdatingBalance = state.isUpdatingBalance,
                        onUpdateBalance = viewModel::updateBalance,
                    )
                    if (permissions.sensitiveWrite) {
                        // loading = isDeleting also disables the button while the delete runs.
                        AppButton(
                            stringResource(R.string.channels_delete_title),
                            onClick = { confirmDelete = true },
                            modifier = Modifier.fillMaxWidth(),
                            variant = ButtonVariant.DestructiveOutline,
                            loading = state.isDeleting,
                            leadingIcon = Icons.Outlined.Delete,
                        )
                    }
                }
                state.isLoading -> FullScreenLoading()
                error != null -> FullScreenError(error, onRetry = viewModel::refresh)
            }
        }
    }

    if (showTestDialog && channel != null) {
        ChannelTestDialog(
            channel = channel,
            run = testRun,
            onRun = viewModel::runTest,
            onDismiss = {
                viewModel.resetTest()
                showTestDialog = false
            },
        )
    }

    if (confirmDelete && channel != null) {
        ConfirmDialog(
            title = stringResource(R.string.channels_delete_title),
            text = stringResource(R.string.channels_delete_message, channel.displayName),
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
private fun ChannelHeader(channel: Channel, actions: @Composable () -> Unit) {
    SectionCard(title = null) {
        Text(channel.displayName, style = MaterialTheme.typography.titleLarge)
        Spacer(Modifier.height(4.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                "#${channel.id} · ${channelTypeLabel(channel.type)}",
                style = MaterialTheme.typography.bodyMedium,
                color = AppTheme.colors.mutedForeground,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(8.dp))
            ChannelStatusPill(channel.status)
        }
        val disabled = channel.status == ChannelStatus.MANUALLY_DISABLED || channel.status == ChannelStatus.AUTO_DISABLED
        val reason = channel.statusReason
        val time = channel.statusTime
        if (disabled && (reason != null || time != null)) {
            Spacer(Modifier.height(12.dp))
            val auto = channel.status == ChannelStatus.AUTO_DISABLED
            val reasonLine = reason?.let { stringResource(R.string.channels_disabled_reason, it) }
            val timeLine = time?.let { stringResource(R.string.channels_disabled_time, Formats.dateTime(it)) }
            // The reason leads when both are known; the time then sits under it.
            AppAlert(
                text = timeLine ?: reasonLine.orEmpty(),
                title = if (timeLine != null) reasonLine else null,
                tone = if (auto) Tone.Warning else Tone.Danger,
                icon = if (auto) Icons.Outlined.WarningAmber else Icons.Outlined.Block,
            )
        }
        Spacer(Modifier.height(16.dp))
        actions()
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun ChannelActions(
    channel: Channel,
    permissions: ChannelPermissions,
    isChangingStatus: Boolean,
    onTest: () -> Unit,
    onSetEnabled: (Boolean) -> Unit,
    onOpenLogs: () -> Unit,
) {
    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        if (permissions.operate && channel.isTestable) {
            AppButton(
                stringResource(R.string.channels_action_test),
                onClick = onTest,
                size = ButtonSize.Small,
                leadingIcon = Icons.Outlined.NetworkCheck,
            )
        }
        if (permissions.operate) {
            val enabled = channel.status == ChannelStatus.ENABLED
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
        AppButton(
            stringResource(R.string.channels_action_view_logs),
            onClick = onOpenLogs,
            variant = ButtonVariant.Outline,
            size = ButtonSize.Small,
            leadingIcon = Icons.AutoMirrored.Outlined.ReceiptLong,
        )
    }
}

@Composable
private fun BasicSection(channel: Channel) {
    SectionCard(stringResource(R.string.channels_section_basic)) {
        InfoRow(stringResource(R.string.channels_label_id), channel.id.toString(), copyable = true)
        InfoRow(stringResource(R.string.channels_label_type), channelTypeLabel(channel.type))
        val baseUrl = channel.baseUrl?.trim().orEmpty()
        if (baseUrl.isEmpty()) {
            InfoRow(stringResource(R.string.channels_label_base_url), stringResource(R.string.channels_base_url_default))
        } else {
            InfoRow(stringResource(R.string.channels_label_base_url), baseUrl, copyable = true)
        }
        InfoRow(stringResource(R.string.channels_label_groups), channel.groupList.joinToString(", "))
        InfoRow(stringResource(R.string.channels_label_priority), channel.priority?.toString().orEmpty())
        InfoRow(stringResource(R.string.channels_label_weight), channel.weight?.toString().orEmpty())
        InfoRow(stringResource(R.string.channels_label_tag), channel.tag?.trim().orEmpty())
        InfoRow(stringResource(R.string.channels_label_remark), channel.remark?.trim().orEmpty())
        InfoRow(
            stringResource(R.string.channels_label_auto_ban),
            stringResource(if (channel.autoBan == 1) R.string.channels_on else R.string.channels_off),
        )
        InfoRow(stringResource(R.string.channels_label_created), Formats.dateTime(channel.createdTime))
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun ModelsSection(channel: Channel) {
    val models = channel.modelList
    val clipboard = LocalClipboardManager.current
    var showAll by rememberSaveable { mutableStateOf(false) }
    SectionCard(
        stringResource(R.string.channels_section_models, models.size),
        action = if (models.isEmpty()) {
            null
        } else {
            {
                AppIconButton(
                    icon = Icons.Outlined.ContentCopy,
                    contentDescription = stringResource(R.string.common_copy),
                    onClick = { clipboard.setText(AnnotatedString(models.joinToString(","))) },
                    size = 28.dp,
                    iconSize = 14.dp,
                )
            }
        },
    ) {
        if (models.isEmpty()) {
            Text(
                stringResource(R.string.channels_models_empty),
                style = MaterialTheme.typography.bodyMedium,
                color = AppTheme.colors.mutedForeground,
            )
        } else {
            val shown = if (showAll) models else models.take(COLLAPSED_MODEL_COUNT)
            FlowRow(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                shown.forEach { ModelChip(it) }
            }
            if (models.size > COLLAPSED_MODEL_COUNT) {
                Spacer(Modifier.height(4.dp))
                AppButton(
                    if (showAll) {
                        stringResource(R.string.channels_models_show_less)
                    } else {
                        stringResource(R.string.channels_models_show_all, models.size)
                    },
                    onClick = { showAll = !showAll },
                    variant = ButtonVariant.Ghost,
                    size = ButtonSize.Small,
                )
            }
        }
        Spacer(Modifier.height(12.dp))
        HorizontalDivider(color = AppTheme.colors.border)
        Spacer(Modifier.height(4.dp))
        val testModel = channel.testModel?.trim().orEmpty()
        InfoRow(
            stringResource(R.string.channels_label_test_model),
            testModel.ifEmpty { stringResource(R.string.channels_test_model_default, defaultTestModel(channel)) },
        )
    }
}

/** A model name chip; long names are ellipsized. */
@Composable
private fun ModelChip(name: String, modifier: Modifier = Modifier) {
    Surface(
        shape = MaterialTheme.shapes.extraSmall,
        color = AppTheme.colors.muted,
        contentColor = AppTheme.colors.foreground,
        modifier = modifier,
    ) {
        Text(
            name,
            style = MaterialTheme.typography.bodySmall,
            fontFamily = GeistMono,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
        )
    }
}

@Composable
private fun ModelMappingSection(mapping: String) {
    val pretty = remember(mapping) {
        runCatching { prettyJson.encodeToString(JsonElement.serializer(), Json.parseToJsonElement(mapping)) }.getOrDefault(mapping)
    }
    SectionCard(stringResource(R.string.channels_section_mapping)) {
        Surface(
            color = AppTheme.colors.muted,
            contentColor = AppTheme.colors.foreground,
            shape = MaterialTheme.shapes.small,
            modifier = Modifier.fillMaxWidth(),
        ) {
            SelectionContainer {
                Text(
                    pretty,
                    style = MaterialTheme.typography.bodySmall,
                    fontFamily = GeistMono,
                    modifier = Modifier.padding(12.dp),
                )
            }
        }
    }
}

@Composable
private fun MultiKeySection(channel: Channel) {
    val info = channel.channelInfo
    SectionCard(stringResource(R.string.channels_section_multi_key)) {
        InfoRow(stringResource(R.string.channels_label_key_count), Formats.count(info.multiKeySize.toLong()))
        InfoRow(stringResource(R.string.channels_label_enabled_keys), "${info.enabledKeyCount} / ${info.multiKeySize}")
        val mode = when (info.multiKeyMode) {
            "random" -> stringResource(R.string.channels_key_mode_random)
            "polling" -> stringResource(R.string.channels_key_mode_polling)
            else -> info.multiKeyMode
        }
        InfoRow(stringResource(R.string.channels_label_key_mode), mode)
    }
}

@Composable
private fun StatsSection(
    channel: Channel,
    formatter: QuotaFormatter,
    canUpdateBalance: Boolean,
    isUpdatingBalance: Boolean,
    onUpdateBalance: () -> Unit,
) {
    val never = stringResource(R.string.channels_never)
    SectionCard(stringResource(R.string.channels_section_stats)) {
        Row(Modifier.fillMaxWidth().height(IntrinsicSize.Min), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            StatTile(stringResource(R.string.channels_label_used_quota), formatter.format(channel.usedQuota), Modifier.weight(1f))
            if (channel.hasBalance) {
                StatTile(stringResource(R.string.channels_label_balance), formatter.formatUsd(channel.balance), Modifier.weight(1f))
            }
            StatTile(stringResource(R.string.channels_label_response_time), responseTimeText(channel.responseTime), Modifier.weight(1f))
        }
        Spacer(Modifier.height(8.dp))
        if (channel.hasBalance) {
            InfoRow(
                stringResource(R.string.channels_label_balance_updated),
                if (channel.balanceUpdatedTime > 0) Formats.dateTime(channel.balanceUpdatedTime) else never,
            )
        }
        InfoRow(stringResource(R.string.channels_label_test_time), if (channel.testTime > 0) Formats.dateTime(channel.testTime) else never)
        if (canUpdateBalance) {
            Spacer(Modifier.height(8.dp))
            // loading = isUpdatingBalance also disables the button while the refresh runs.
            AppButton(
                stringResource(R.string.channels_action_update_balance),
                onClick = onUpdateBalance,
                modifier = Modifier.fillMaxWidth(),
                variant = ButtonVariant.Outline,
                loading = isUpdatingBalance,
                leadingIcon = Icons.Outlined.AccountBalanceWallet,
            )
        }
    }
}

/** A muted tile with a small label over a number. */
@Composable
private fun StatTile(label: String, value: String, modifier: Modifier) {
    Column(
        modifier
            .fillMaxHeight()
            .background(AppTheme.colors.muted, MaterialTheme.shapes.medium)
            .padding(horizontal = 12.dp, vertical = 10.dp),
        verticalArrangement = Arrangement.spacedBy(2.dp),
    ) {
        Text(label, style = MaterialTheme.typography.bodySmall, color = AppTheme.colors.mutedForeground)
        // Large amounts shrink to stay whole in a third of a narrow screen instead of being cut off.
        BasicText(
            value,
            style = MaterialTheme.typography.titleSmall.copy(color = AppTheme.colors.foreground, fontFeatureSettings = "tnum"),
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            autoSize = TextAutoSize.StepBased(minFontSize = 11.sp, maxFontSize = 14.sp, stepSize = 0.5.sp),
        )
    }
}

/** What the server tests without a `model` parameter: test_model, else the first model, else its fallback. */
private fun defaultTestModel(channel: Channel): String =
    channel.testModel?.trim()?.takeIf { it.isNotEmpty() } ?: channel.modelList.firstOrNull() ?: SERVER_FALLBACK_TEST_MODEL

@Composable
private fun ChannelTestDialog(
    channel: Channel,
    run: ChannelTestRun,
    onRun: (model: String?) -> Unit,
    onDismiss: () -> Unit,
) {
    // null = the channel's default model.
    var selected by rememberSaveable { mutableStateOf<String?>(null) }
    var filter by rememberSaveable { mutableStateOf("") }
    val models = remember(channel.models) { channel.modelList.distinct() }
    val shownModels = if (filter.isBlank()) models else models.filter { it.contains(filter.trim(), ignoreCase = true) }

    AppDialog(
        // Taps outside do not abandon a running test; the Cancel button does.
        onDismissRequest = { if (!run.isRunning) onDismiss() },
        title = stringResource(R.string.channels_test_dialog_title),
        confirmButton = {
            AppButton(stringResource(R.string.channels_action_test), onClick = { onRun(selected) }, enabled = !run.isRunning)
        },
        dismissButton = {
            AppButton(
                stringResource(if (run.isRunning) R.string.common_cancel else R.string.common_close),
                onClick = onDismiss,
                variant = ButtonVariant.Outline,
            )
        },
    ) {
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            if (models.size > 8) {
                AppTextField(
                    value = filter,
                    onValueChange = { filter = it },
                    modifier = Modifier.fillMaxWidth(),
                    placeholder = stringResource(R.string.channels_test_model_filter),
                    leadingIcon = Icons.Outlined.Search,
                    enabled = !run.isRunning,
                    singleLine = true,
                )
            }
            Column(
                Modifier
                    .border(1.dp, AppTheme.colors.border, MaterialTheme.shapes.medium)
                    .heightIn(max = 280.dp)
                    .verticalScroll(rememberScrollState())
                    .padding(vertical = 4.dp),
            ) {
                TestModelOption(
                    label = stringResource(R.string.channels_test_model_default, defaultTestModel(channel)),
                    selected = selected == null,
                    enabled = !run.isRunning,
                    onSelect = { selected = null },
                )
                shownModels.forEach { model ->
                    TestModelOption(label = model, selected = selected == model, enabled = !run.isRunning, onSelect = { selected = model })
                }
            }
            TestRunResult(run)
        }
    }
}

@Composable
private fun TestModelOption(label: String, selected: Boolean, enabled: Boolean, onSelect: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .selectable(selected = selected, enabled = enabled, role = Role.RadioButton, onClick = onSelect)
            .padding(horizontal = 12.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RadioIndicator(selected = selected, enabled = enabled)
        Spacer(Modifier.width(12.dp))
        Text(label, style = MaterialTheme.typography.bodyMedium, maxLines = 2, overflow = TextOverflow.Ellipsis)
    }
}

/** shadcn radio: a bordered circle with a filled dot when selected. Draw-only; the row owns the click. */
@Composable
private fun RadioIndicator(selected: Boolean, enabled: Boolean) {
    val colors = AppTheme.colors
    Box(
        Modifier
            .alpha(if (enabled) 1f else 0.5f)
            .size(16.dp)
            .border(1.dp, if (selected) colors.primary else colors.input, CircleShape),
        contentAlignment = Alignment.Center,
    ) {
        if (selected) Box(Modifier.size(8.dp).background(colors.primary, CircleShape))
    }
}

@Composable
private fun TestRunResult(run: ChannelTestRun) {
    val result = run.result
    val error = run.error
    when {
        run.isRunning -> Row(verticalAlignment = Alignment.CenterVertically) {
            CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp, color = AppTheme.colors.mutedForeground)
            Spacer(Modifier.width(10.dp))
            Text(stringResource(R.string.channels_testing), style = MaterialTheme.typography.bodyMedium, color = AppTheme.colors.mutedForeground)
        }
        result != null && result.success -> ResultLine(
            text = stringResource(R.string.channels_test_result_success, formatSeconds(result.timeSeconds)),
            success = true,
        )
        result != null -> ResultLine(
            text = if (result.message.isBlank()) {
                stringResource(R.string.channels_test_result_failed_no_reason)
            } else {
                stringResource(R.string.channels_test_result_failed, result.message)
            },
            success = false,
        )
        error != null -> ResultLine(text = errorMessage(error), success = false)
    }
}

@Composable
private fun ResultLine(text: String, success: Boolean) {
    // Selectable, so error messages can be copied.
    SelectionContainer {
        AppAlert(
            text = text,
            tone = if (success) Tone.Success else Tone.Danger,
            icon = if (success) Icons.Outlined.CheckCircle else Icons.Outlined.ErrorOutline,
        )
    }
}
