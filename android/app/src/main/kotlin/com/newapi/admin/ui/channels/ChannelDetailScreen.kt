package com.newapi.admin.ui.channels

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.ReceiptLong
import androidx.compose.material.icons.filled.AccountBalanceWallet
import androidx.compose.material.icons.filled.Block
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.ErrorOutline
import androidx.compose.material.icons.filled.NetworkCheck
import androidx.compose.material.icons.filled.PlayCircle
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
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
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.format.Formats
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.core.model.Channel
import com.newapi.admin.core.model.ChannelStatus
import com.newapi.admin.core.model.ChannelTypes
import com.newapi.admin.ui.common.errorMessage
import com.newapi.admin.ui.components.ConfirmDialog
import com.newapi.admin.ui.components.FullScreenError
import com.newapi.admin.ui.components.FullScreenLoading
import com.newapi.admin.ui.components.InfoRow
import com.newapi.admin.ui.components.SectionCard
import com.newapi.admin.ui.components.rememberQuotaFormatter
import com.newapi.admin.ui.theme.Tone
import com.newapi.admin.ui.theme.colors
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
    LaunchedEffect(state.deleted) {
        if (state.deleted) onBack()
    }

    val channel = state.channel
    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        channel?.displayName ?: stringResource(R.string.channels_detail_title),
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_back))
                    }
                },
                actions = {
                    if (channel != null && permissions.write) {
                        IconButton(onClick = { onEdit(channel.id) }) {
                            Icon(Icons.Filled.Edit, contentDescription = stringResource(R.string.common_edit))
                        }
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
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
                    ChannelHeader(channel)
                    ChannelActions(
                        channel = channel,
                        permissions = permissions,
                        isChangingStatus = state.isChangingStatus,
                        onTest = { showTestDialog = true },
                        onSetEnabled = viewModel::setEnabled,
                        onOpenLogs = { onOpenLogs(channel.id) },
                    )
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
                        OutlinedButton(
                            onClick = { confirmDelete = true },
                            enabled = !state.isDeleting,
                            colors = ButtonDefaults.outlinedButtonColors(contentColor = MaterialTheme.colorScheme.error),
                            border = BorderStroke(1.dp, MaterialTheme.colorScheme.error),
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            if (state.isDeleting) {
                                CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                            } else {
                                Icon(Icons.Filled.Delete, contentDescription = null, modifier = Modifier.size(18.dp))
                            }
                            Spacer(Modifier.width(8.dp))
                            Text(stringResource(R.string.channels_delete_title))
                        }
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
private fun ChannelHeader(channel: Channel) {
    SectionCard(title = null) {
        Text(channel.displayName, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
        Spacer(Modifier.height(6.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                "#${channel.id} · ${ChannelTypes.nameOf(channel.type)}",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
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
            val colors = (if (channel.status == ChannelStatus.AUTO_DISABLED) Tone.Warning else Tone.Danger).colors()
            Surface(
                color = colors.container,
                contentColor = colors.content,
                shape = RoundedCornerShape(12.dp),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    if (reason != null) {
                        Text(stringResource(R.string.channels_disabled_reason, reason), style = MaterialTheme.typography.bodyMedium)
                    }
                    if (time != null) {
                        Text(stringResource(R.string.channels_disabled_time, Formats.dateTime(time)), style = MaterialTheme.typography.bodySmall)
                    }
                }
            }
        }
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
            FilledTonalButton(onClick = onTest) {
                Icon(Icons.Filled.NetworkCheck, contentDescription = null, modifier = Modifier.size(18.dp))
                Spacer(Modifier.width(8.dp))
                Text(stringResource(R.string.channels_action_test))
            }
        }
        if (permissions.operate) {
            val enabled = channel.status == ChannelStatus.ENABLED
            OutlinedButton(onClick = { onSetEnabled(!enabled) }, enabled = !isChangingStatus) {
                if (isChangingStatus) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Icon(if (enabled) Icons.Filled.Block else Icons.Filled.PlayCircle, contentDescription = null, modifier = Modifier.size(18.dp))
                }
                Spacer(Modifier.width(8.dp))
                Text(stringResource(if (enabled) R.string.common_disable else R.string.common_enable))
            }
        }
        OutlinedButton(onClick = onOpenLogs) {
            Icon(Icons.AutoMirrored.Filled.ReceiptLong, contentDescription = null, modifier = Modifier.size(18.dp))
            Spacer(Modifier.width(8.dp))
            Text(stringResource(R.string.channels_action_view_logs))
        }
    }
}

@Composable
private fun BasicSection(channel: Channel) {
    SectionCard(stringResource(R.string.channels_section_basic)) {
        InfoRow(stringResource(R.string.channels_label_id), channel.id.toString(), copyable = true)
        InfoRow(stringResource(R.string.channels_label_type), ChannelTypes.nameOf(channel.type))
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
    SectionCard(stringResource(R.string.channels_section_models, models.size)) {
        if (models.isEmpty()) {
            Text(
                stringResource(R.string.channels_models_empty),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        } else {
            val shown = if (showAll) models else models.take(COLLAPSED_MODEL_COUNT)
            FlowRow(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                shown.forEach { ModelChip(it) }
            }
            Row(verticalAlignment = Alignment.CenterVertically) {
                if (models.size > COLLAPSED_MODEL_COUNT) {
                    TextButton(onClick = { showAll = !showAll }) {
                        Text(
                            if (showAll) {
                                stringResource(R.string.channels_models_show_less)
                            } else {
                                stringResource(R.string.channels_models_show_all, models.size)
                            },
                        )
                    }
                }
                Spacer(Modifier.weight(1f))
                TextButton(onClick = { clipboard.setText(AnnotatedString(models.joinToString(","))) }) {
                    Icon(Icons.Filled.ContentCopy, contentDescription = null, modifier = Modifier.size(16.dp))
                    Spacer(Modifier.width(6.dp))
                    Text(stringResource(R.string.common_copy))
                }
            }
        }
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
        shape = RoundedCornerShape(8.dp),
        color = MaterialTheme.colorScheme.secondaryContainer,
        contentColor = MaterialTheme.colorScheme.onSecondaryContainer,
        modifier = modifier,
    ) {
        Text(
            name,
            style = MaterialTheme.typography.labelMedium,
            fontFamily = FontFamily.Monospace,
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
            color = MaterialTheme.colorScheme.surfaceContainerHighest,
            shape = RoundedCornerShape(8.dp),
            modifier = Modifier.fillMaxWidth(),
        ) {
            SelectionContainer {
                Text(
                    pretty,
                    style = MaterialTheme.typography.bodySmall,
                    fontFamily = FontFamily.Monospace,
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
        InfoRow(stringResource(R.string.channels_label_used_quota), formatter.format(channel.usedQuota))
        if (channel.hasBalance) {
            InfoRow(stringResource(R.string.channels_label_balance), formatter.formatUsd(channel.balance))
            InfoRow(
                stringResource(R.string.channels_label_balance_updated),
                if (channel.balanceUpdatedTime > 0) Formats.dateTime(channel.balanceUpdatedTime) else never,
            )
        }
        InfoRow(stringResource(R.string.channels_label_response_time), responseTimeText(channel.responseTime))
        InfoRow(stringResource(R.string.channels_label_test_time), if (channel.testTime > 0) Formats.dateTime(channel.testTime) else never)
        if (canUpdateBalance) {
            Spacer(Modifier.height(4.dp))
            OutlinedButton(onClick = onUpdateBalance, enabled = !isUpdatingBalance, modifier = Modifier.fillMaxWidth()) {
                if (isUpdatingBalance) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Icon(Icons.Filled.AccountBalanceWallet, contentDescription = null, modifier = Modifier.size(18.dp))
                }
                Spacer(Modifier.width(8.dp))
                Text(stringResource(R.string.channels_action_update_balance))
            }
        }
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

    AlertDialog(
        // Taps outside do not abandon a running test; the Cancel button does.
        onDismissRequest = { if (!run.isRunning) onDismiss() },
        title = { Text(stringResource(R.string.channels_test_dialog_title)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                if (models.size > 8) {
                    OutlinedTextField(
                        value = filter,
                        onValueChange = { filter = it },
                        singleLine = true,
                        enabled = !run.isRunning,
                        placeholder = { Text(stringResource(R.string.channels_test_model_filter)) },
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                Column(Modifier.heightIn(max = 280.dp).verticalScroll(rememberScrollState())) {
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
        },
        confirmButton = {
            TextButton(onClick = { onRun(selected) }, enabled = !run.isRunning) {
                Text(stringResource(R.string.channels_action_test))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(stringResource(if (run.isRunning) R.string.common_cancel else R.string.common_close))
            }
        },
    )
}

@Composable
private fun TestModelOption(label: String, selected: Boolean, enabled: Boolean, onSelect: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .selectable(selected = selected, enabled = enabled, role = Role.RadioButton, onClick = onSelect)
            .padding(vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RadioButton(selected = selected, onClick = null, enabled = enabled)
        Spacer(Modifier.width(8.dp))
        Text(label, style = MaterialTheme.typography.bodyMedium, maxLines = 2, overflow = TextOverflow.Ellipsis)
    }
}

@Composable
private fun TestRunResult(run: ChannelTestRun) {
    val result = run.result
    val error = run.error
    when {
        run.isRunning -> Row(verticalAlignment = Alignment.CenterVertically) {
            CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
            Spacer(Modifier.width(12.dp))
            Text(stringResource(R.string.channels_testing), style = MaterialTheme.typography.bodyMedium)
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
    val colors = (if (success) Tone.Success else Tone.Danger).colors()
    Surface(color = colors.container, contentColor = colors.content, shape = RoundedCornerShape(8.dp), modifier = Modifier.fillMaxWidth()) {
        Row(Modifier.padding(12.dp), verticalAlignment = Alignment.Top) {
            Icon(if (success) Icons.Filled.CheckCircle else Icons.Filled.ErrorOutline, contentDescription = null, modifier = Modifier.size(18.dp))
            Spacer(Modifier.width(8.dp))
            SelectionContainer {
                Text(text, style = MaterialTheme.typography.bodyMedium)
            }
        }
    }
}
