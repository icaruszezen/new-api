package com.newapi.admin.ui.logs

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ContentCopy
import androidx.compose.material.icons.outlined.ExpandLess
import androidx.compose.material.icons.outlined.ExpandMore
import androidx.compose.material.icons.outlined.Hub
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.unit.dp
import com.newapi.admin.R
import com.newapi.admin.core.format.Formats
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.core.model.LogEntry
import com.newapi.admin.core.model.LogType
import com.newapi.admin.core.model.parsedOther
import com.newapi.admin.ui.components.AppBottomSheet
import com.newapi.admin.ui.components.AppButton
import com.newapi.admin.ui.components.AppIconButton
import com.newapi.admin.ui.components.ButtonSize
import com.newapi.admin.ui.components.ButtonVariant
import com.newapi.admin.ui.components.InfoRow
import com.newapi.admin.ui.components.SectionCard
import com.newapi.admin.ui.components.StatusPill
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.theme.GeistMono
import com.newapi.admin.ui.theme.Tone

/** Every field of one log row, including the parsed pricing of `other` and its raw JSON. */
@OptIn(ExperimentalLayoutApi::class)
@Composable
internal fun LogDetailSheet(
    entry: LogEntry,
    formatter: QuotaFormatter,
    onDismiss: () -> Unit,
    onOpenUser: (Int) -> Unit,
    onOpenChannel: (Int) -> Unit,
) {
    val other = remember(entry.other) { entry.parsedOther() }
    val rawJson = remember(other) { other?.prettyJson() }
    val sheetState = rememberModalBottomSheetState()
    var showRaw by rememberSaveable(entry.id) { mutableStateOf(false) }
    val clipboard = LocalClipboardManager.current
    val colors = AppTheme.colors
    val usage = hasUsage(entry.type)
    val timing = hasTiming(entry.type)
    val auditOp = other?.auditOp
    // Manage logs belong to the operator; the affected user is in the audit parameters.
    val targetUserId = auditOp?.params?.get("target_user_id")?.toDoubleOrNull()?.toInt()?.takeIf { it > 0 }
    val isOperatorLog = entry.type == LogType.MANAGE

    AppBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 20.dp)
                .padding(bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(stringResource(R.string.logs_detail_title), style = MaterialTheme.typography.titleLarge)
                FlowRow(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    StatusPill(stringResource(logTypeLabel(entry.type)), logTypeTone(entry.type))
                    if (timing) {
                        StatusPill(
                            stringResource(if (entry.isStream) R.string.logs_stream else R.string.logs_non_stream),
                            if (entry.isStream) Tone.Info else Tone.Neutral,
                        )
                    }
                    if (other?.billingSource == "subscription") StatusPill(stringResource(R.string.logs_billing_subscription), Tone.Info)
                    if (other?.billingCorrected == true) StatusPill(stringResource(R.string.logs_waived), Tone.Info)
                }
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    if (entry.userId > 0) {
                        AppButton(
                            stringResource(R.string.logs_view_user),
                            onClick = { onOpenUser(entry.userId) },
                            variant = ButtonVariant.Outline,
                            size = ButtonSize.Small,
                            leadingIcon = Icons.Outlined.Person,
                        )
                    }
                    if (targetUserId != null && targetUserId != entry.userId) {
                        AppButton(
                            stringResource(R.string.logs_view_target_user),
                            onClick = { onOpenUser(targetUserId) },
                            variant = ButtonVariant.Outline,
                            size = ButtonSize.Small,
                            leadingIcon = Icons.Outlined.Person,
                        )
                    }
                    if (entry.channel > 0) {
                        AppButton(
                            stringResource(R.string.logs_view_channel),
                            onClick = { onOpenChannel(entry.channel) },
                            variant = ButtonVariant.Outline,
                            size = ButtonSize.Small,
                            leadingIcon = Icons.Outlined.Hub,
                        )
                    }
                }
            }

            SectionCard(title = stringResource(R.string.logs_detail_section_request)) {
                InfoRow(stringResource(R.string.logs_label_time), Formats.dateTime(entry.createdAt))
                InfoRow(stringResource(R.string.logs_label_log_id), entry.id.toString(), copyable = true)
                if (entry.requestId.isNotBlank()) {
                    InfoRow(stringResource(R.string.logs_label_request_id), entry.requestId, copyable = true)
                }
                if (entry.upstreamRequestId.isNotBlank()) {
                    InfoRow(stringResource(R.string.logs_label_upstream_request_id), entry.upstreamRequestId, copyable = true)
                }
                val userText = when {
                    entry.userId > 0 && entry.username.isNotBlank() -> stringResource(R.string.logs_value_with_id, entry.username, entry.userId)
                    entry.userId > 0 -> "#${entry.userId}"
                    else -> entry.username
                }
                InfoRow(stringResource(if (isOperatorLog) R.string.logs_label_operator else R.string.logs_label_user), userText)
                if (entry.tokenName.isNotBlank() || entry.tokenId > 0) {
                    val tokenText = when {
                        entry.tokenId > 0 && entry.tokenName.isNotBlank() -> stringResource(R.string.logs_value_with_id, entry.tokenName, entry.tokenId)
                        entry.tokenId > 0 -> "#${entry.tokenId}"
                        else -> entry.tokenName
                    }
                    InfoRow(stringResource(R.string.logs_label_token), tokenText)
                }
                if (entry.modelName.isNotBlank()) {
                    InfoRow(stringResource(R.string.logs_label_model), entry.modelName, copyable = true)
                }
                val upstreamModel = other?.upstreamModelName
                if (other?.isModelMapped == true && upstreamModel != null) {
                    InfoRow(stringResource(R.string.logs_label_upstream_model), upstreamModel, copyable = true)
                }
                if (entry.channel > 0) {
                    InfoRow(stringResource(R.string.logs_label_channel), "#${entry.channel} ${entry.channelName}".trim())
                }
                val retryChannels = other?.retryChannels.orEmpty()
                if (retryChannels.isNotEmpty()) {
                    InfoRow(stringResource(R.string.logs_label_retry_chain), retryChannels.joinToString(" → "))
                }
                if (entry.group.isNotBlank()) {
                    InfoRow(stringResource(R.string.logs_label_group), entry.group)
                }
                if (entry.ip.isNotBlank()) {
                    InfoRow(stringResource(R.string.logs_label_ip), entry.ip, copyable = true)
                }
                other?.requestPath?.let { InfoRow(stringResource(R.string.logs_label_request_path), it) }
                if (timing) {
                    InfoRow(
                        stringResource(R.string.logs_label_stream),
                        stringResource(if (entry.isStream) R.string.logs_stream else R.string.logs_non_stream),
                    )
                    if (entry.useTime > 0) {
                        InfoRow(stringResource(R.string.logs_label_use_time), Formats.duration(entry.useTime.toDouble()))
                    }
                    if (entry.isStream) {
                        val firstResponse = other?.firstResponseMillis?.let { Formats.duration(it / 1000) }
                        InfoRow(stringResource(R.string.logs_label_first_response), firstResponse ?: stringResource(R.string.logs_not_applicable))
                    }
                }
            }

            if (usage) {
                SectionCard(title = stringResource(R.string.logs_detail_section_billing)) {
                    InfoRow(stringResource(R.string.logs_label_cost), formatter.formatLog(entry.quota))
                    other?.billingSource?.let { source ->
                        val sourceText = when (source) {
                            "wallet" -> stringResource(R.string.logs_billing_wallet)
                            "subscription" -> stringResource(R.string.logs_billing_subscription)
                            else -> source
                        }
                        InfoRow(stringResource(R.string.logs_label_billing_source), sourceText)
                    }
                    val modelPrice = other?.modelPrice
                    val modelRatio = other?.modelRatio
                    if (modelPrice != null && modelPrice > 0) {
                        // Per-call billing: model_price is USD per request.
                        InfoRow(stringResource(R.string.logs_label_billing_mode), stringResource(R.string.logs_billing_per_call))
                        InfoRow(stringResource(R.string.logs_label_model_price), formatter.formatUsd(modelPrice))
                    } else if (modelRatio != null) {
                        // Per-token billing: USD per million input tokens = model_ratio * 2 (web formula).
                        val inputPrice = modelRatio * 2
                        InfoRow(stringResource(R.string.logs_label_billing_mode), stringResource(R.string.logs_billing_per_token))
                        InfoRow(stringResource(R.string.logs_label_model_ratio), Formats.ratio(modelRatio))
                        InfoRow(stringResource(R.string.logs_label_input_price), stringResource(R.string.logs_price_per_million, formatter.formatUsd(inputPrice)))
                        other?.completionRatio?.let { completionRatio ->
                            InfoRow(stringResource(R.string.logs_label_completion_ratio), Formats.ratio(completionRatio))
                            InfoRow(
                                stringResource(R.string.logs_label_output_price),
                                stringResource(R.string.logs_price_per_million, formatter.formatUsd(inputPrice * completionRatio)),
                            )
                        }
                    }
                    other?.groupRatio?.let { InfoRow(stringResource(R.string.logs_label_group_ratio), Formats.ratio(it)) }
                    // -1 means the user has no exclusive ratio; otherwise it replaces the group ratio.
                    other?.userGroupRatio?.takeIf { it != -1.0 }?.let {
                        InfoRow(stringResource(R.string.logs_label_user_group_ratio), Formats.ratio(it))
                    }
                    val cacheRatio = other?.cacheRatio
                    if (cacheRatio != null && ((other?.cacheTokens ?: 0) > 0 || cacheRatio != 1.0)) {
                        InfoRow(stringResource(R.string.logs_label_cache_ratio), Formats.ratio(cacheRatio))
                    }
                    other?.reasoningEffort?.let { InfoRow(stringResource(R.string.logs_label_reasoning_effort), it) }
                }

                SectionCard(title = stringResource(R.string.logs_detail_section_tokens)) {
                    InfoRow(stringResource(R.string.logs_label_prompt_tokens), Formats.count(entry.promptTokens))
                    InfoRow(stringResource(R.string.logs_label_completion_tokens), Formats.count(entry.completionTokens))
                    val cacheRead = other?.cacheTokens ?: 0
                    if (cacheRead > 0) InfoRow(stringResource(R.string.logs_label_cache_read), Formats.count(cacheRead))
                    val cacheWrite = other?.cacheWriteTokens ?: 0
                    if (cacheWrite > 0) InfoRow(stringResource(R.string.logs_label_cache_write), Formats.count(cacheWrite))
                }
            }

            val rejectReason = other?.rejectReason
            val errorType = other?.errorType
            val errorCode = other?.errorCode
            val statusCode = other?.statusCode
            if (rejectReason != null || errorType != null || errorCode != null || statusCode != null) {
                SectionCard(title = stringResource(R.string.logs_detail_section_error)) {
                    rejectReason?.let { InfoRow(stringResource(R.string.logs_label_reject_reason), it) }
                    errorType?.let { InfoRow(stringResource(R.string.logs_label_error_type), it) }
                    errorCode?.let { InfoRow(stringResource(R.string.logs_label_error_code), it, copyable = true) }
                    statusCode?.let { InfoRow(stringResource(R.string.logs_label_status_code), it.toString()) }
                }
            }

            val streamStatus = other?.streamStatus
            if (streamStatus != null) {
                SectionCard(title = stringResource(R.string.logs_detail_section_stream)) {
                    val isOk = streamStatus.status == "ok"
                    // Lined up with the InfoRows below it.
                    Row(
                        Modifier.fillMaxWidth().heightIn(min = 36.dp).padding(vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            stringResource(R.string.logs_label_stream_status),
                            style = MaterialTheme.typography.bodyMedium,
                            color = colors.mutedForeground,
                            modifier = Modifier.width(112.dp),
                        )
                        StatusPill(
                            stringResource(if (isOk) R.string.logs_stream_ok else R.string.logs_stream_error),
                            if (isOk) Tone.Success else Tone.Danger,
                        )
                    }
                    if (streamStatus.endReason.isNotBlank()) {
                        val reason = if (streamStatus.endReason == "client_gone") {
                            stringResource(R.string.logs_end_reason_client_gone)
                        } else {
                            streamStatus.endReason
                        }
                        InfoRow(stringResource(R.string.logs_label_end_reason), reason)
                    }
                    if (streamStatus.errorCount > 0) {
                        InfoRow(stringResource(R.string.logs_label_soft_errors), streamStatus.errorCount.toString())
                    }
                }
            }

            if (auditOp != null) {
                SectionCard(title = stringResource(R.string.logs_detail_section_audit)) {
                    InfoRow(stringResource(R.string.logs_label_operation), auditOp.action)
                    auditOp.params.forEach { (key, value) -> InfoRow(key, value) }
                }
            }

            if (entry.content.isNotBlank()) {
                SectionCard(
                    title = stringResource(R.string.logs_detail_section_content),
                    action = {
                        AppIconButton(
                            icon = Icons.Outlined.ContentCopy,
                            contentDescription = stringResource(R.string.common_copy),
                            onClick = { clipboard.setText(AnnotatedString(entry.content)) },
                            size = 32.dp,
                            iconSize = 16.dp,
                        )
                    },
                ) {
                    SelectionContainer {
                        Text(
                            entry.content,
                            style = MaterialTheme.typography.bodyMedium,
                            color = if (entry.type == LogType.ERROR) colors.destructive else colors.foreground,
                        )
                    }
                }
            }

            if (rawJson != null) {
                SectionCard(title = null) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .heightIn(min = 32.dp)
                            .clip(MaterialTheme.shapes.small)
                            .clickable { showRaw = !showRaw },
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            stringResource(R.string.logs_detail_section_raw),
                            style = MaterialTheme.typography.titleSmall,
                            modifier = Modifier.weight(1f),
                        )
                        if (showRaw) {
                            AppIconButton(
                                icon = Icons.Outlined.ContentCopy,
                                contentDescription = stringResource(R.string.common_copy),
                                onClick = { clipboard.setText(AnnotatedString(rawJson)) },
                                size = 32.dp,
                                iconSize = 16.dp,
                            )
                        }
                        Icon(
                            if (showRaw) Icons.Outlined.ExpandLess else Icons.Outlined.ExpandMore,
                            contentDescription = stringResource(if (showRaw) R.string.logs_detail_collapse else R.string.logs_detail_expand),
                            tint = colors.mutedForeground,
                            modifier = Modifier.padding(start = 4.dp).size(20.dp),
                        )
                    }
                    if (showRaw) {
                        SelectionContainer {
                            Text(
                                rawJson,
                                style = MaterialTheme.typography.bodySmall,
                                fontFamily = GeistMono,
                                modifier = Modifier
                                    .padding(top = 12.dp)
                                    .fillMaxWidth()
                                    .background(colors.muted, MaterialTheme.shapes.small)
                                    .padding(12.dp),
                            )
                        }
                    }
                }
            }
        }
    }
}
