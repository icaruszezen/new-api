package com.newapi.admin.ui.redemptions

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.consumeWindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.Check
import androidx.compose.material.icons.outlined.ContentCopy
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SelectableDates
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TimePicker
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.material3.rememberTimePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.newapi.admin.LocalAppContainer
import com.newapi.admin.R
import com.newapi.admin.core.format.Formats
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.ui.common.UiMessagesEffect
import com.newapi.admin.ui.common.text
import com.newapi.admin.ui.components.AppAlert
import com.newapi.admin.ui.components.AppButton
import com.newapi.admin.ui.components.AppCard
import com.newapi.admin.ui.components.AppDatePickerDialog
import com.newapi.admin.ui.components.AppDialog
import com.newapi.admin.ui.components.AppFilterChip
import com.newapi.admin.ui.components.AppSnackbarHost
import com.newapi.admin.ui.components.AppTextField
import com.newapi.admin.ui.components.AppTopBar
import com.newapi.admin.ui.components.ButtonSize
import com.newapi.admin.ui.components.ButtonVariant
import com.newapi.admin.ui.components.ConfirmDialog
import com.newapi.admin.ui.components.FullScreenError
import com.newapi.admin.ui.components.FullScreenLoading
import com.newapi.admin.ui.components.appDatePickerColors
import com.newapi.admin.ui.components.appTimePickerColors
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.theme.GeistMono
import com.newapi.admin.ui.theme.Tone
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.ZoneOffset

/** Creates codes when [codeId] is null; [onDone] leaves after a save, or once the created codes were shown. */
@Composable
fun RedemptionEditScreen(codeId: Int?, onBack: () -> Unit, onDone: () -> Unit) {
    val container = LocalAppContainer.current
    val viewModel = viewModel(key = "redemption-edit-${codeId ?: 0}") { RedemptionEditViewModel(container, codeId) }
    val state by viewModel.state.collectAsStateWithLifecycle()
    val status by container.status.collectAsStateWithLifecycle()
    // Null until `/api/status` has loaded; converting with the fallback rate could create codes worth the wrong amount.
    val formatter = remember(status) { status?.let(::QuotaFormatter) }
    val snackbarHostState = remember { SnackbarHostState() }
    var confirmDiscard by rememberSaveable { mutableStateOf(false) }
    var pickingDate by rememberSaveable { mutableStateOf(false) }
    // The day picked in the date dialog (epoch day), while its time is being chosen.
    var pickedDay by rememberSaveable { mutableStateOf<Long?>(null) }

    val createdKeys = state.createdKeys
    val guardBack = viewModel.isDirty && !state.saved && createdKeys == null
    val requestBack: () -> Unit = {
        if (guardBack) {
            confirmDiscard = true
        } else {
            onBack()
        }
    }

    UiMessagesEffect(viewModel.messages, snackbarHostState)
    // Leaving during a save would cancel it before the change is announced, although the server may apply it.
    BackHandler(enabled = guardBack || state.isSaving) {
        if (!state.isSaving) confirmDiscard = true
    }
    LaunchedEffect(state.saved) {
        if (state.saved) onDone()
    }
    LaunchedEffect(formatter == null) {
        if (formatter == null) container.refreshStatus()
    }

    val formReady = formatter != null && !state.isLoading && state.loadError == null && createdKeys == null
    Scaffold(
        topBar = {
            AppTopBar(
                title = stringResource(if (viewModel.isCreate) R.string.redemptions_create_title else R.string.redemptions_edit_title),
                navigationIcon = {
                    IconButton(onClick = requestBack, enabled = !state.isSaving) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = stringResource(R.string.common_back))
                    }
                },
                actions = {
                    if (formReady && formatter != null) {
                        if (state.isSaving) {
                            CircularProgressIndicator(
                                Modifier.padding(horizontal = 24.dp).size(20.dp),
                                strokeWidth = 2.dp,
                                color = AppTheme.colors.mutedForeground,
                            )
                        } else {
                            AppButton(
                                stringResource(if (viewModel.isCreate) R.string.redemptions_create_submit else R.string.common_save),
                                onClick = { viewModel.save(formatter) },
                                // With the bar's own 4dp, the button lines up with the 16dp content gutter.
                                modifier = Modifier.padding(end = 12.dp),
                                size = ButtonSize.Small,
                                enabled = !state.saved && (viewModel.isCreate || viewModel.isDirty),
                            )
                        }
                    }
                },
            )
        },
        snackbarHost = { AppSnackbarHost(snackbarHostState) },
    ) { padding ->
        val loadError = state.loadError
        val contentModifier = Modifier.fillMaxSize().padding(padding)
        when {
            loadError != null -> FullScreenError(loadError, onRetry = viewModel::load, modifier = contentModifier)
            formatter == null -> Column(contentModifier.padding(16.dp)) {
                AppAlert(
                    text = stringResource(R.string.redemptions_settings_missing),
                    tone = Tone.Warning,
                    icon = Icons.Outlined.WarningAmber,
                    action = {
                        AppButton(
                            stringResource(R.string.common_retry),
                            onClick = container::refreshStatus,
                            variant = ButtonVariant.Outline,
                            size = ButtonSize.Small,
                        )
                    },
                )
            }
            state.isLoading -> FullScreenLoading(contentModifier)
            else -> RedemptionFormFields(
                viewModel = viewModel,
                formatter = formatter,
                showErrors = state.showErrors,
                enabled = !state.isSaving && createdKeys == null,
                onPickCustomExpiry = { pickingDate = true },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .consumeWindowInsets(padding)
                    .imePadding(),
            )
        }
    }

    if (pickingDate) {
        ExpiryDateDialog(
            expiresAt = viewModel.form.expiresAt,
            onConfirm = { day ->
                pickingDate = false
                pickedDay = day
            },
            onDismiss = { pickingDate = false },
        )
    }

    pickedDay?.let { day ->
        ExpiryTimeDialog(
            expiresAt = viewModel.form.expiresAt,
            onConfirm = { hour, minute ->
                pickedDay = null
                viewModel.setCustomExpiry(LocalDate.ofEpochDay(day).atTime(hour, minute).atZone(ZoneId.systemDefault()).toEpochSecond())
            },
            onDismiss = { pickedDay = null },
        )
    }

    if (createdKeys != null) {
        CreatedCodesDialog(createdKeys, onDone = onDone)
    }

    if (confirmDiscard) {
        ConfirmDialog(
            title = stringResource(R.string.redemptions_discard_title),
            text = stringResource(R.string.redemptions_discard_message),
            confirmText = stringResource(R.string.redemptions_discard_confirm),
            destructive = true,
            onConfirm = {
                confirmDiscard = false
                onBack()
            },
            onDismiss = { confirmDiscard = false },
        )
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun RedemptionFormFields(
    viewModel: RedemptionEditViewModel,
    formatter: QuotaFormatter,
    showErrors: Boolean,
    enabled: Boolean,
    onPickCustomExpiry: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val form = viewModel.form
    val errors = viewModel.errors(formatter, System.currentTimeMillis() / 1000)
    Column(
        modifier.verticalScroll(rememberScrollState()).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        AppCard(Modifier.fillMaxWidth()) {
            Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
                val nameError = errors.name.takeIf { showErrors }
                AppTextField(
                    value = form.name,
                    onValueChange = { value -> viewModel.update { it.copy(name = value) } },
                    modifier = Modifier.fillMaxWidth(),
                    label = stringResource(R.string.redemptions_field_name),
                    supportingText = nameError?.text()
                        ?: stringResource(if (viewModel.isCreate) R.string.redemptions_name_hint_create else R.string.redemptions_name_hint_edit),
                    isError = nameError != null,
                    enabled = enabled,
                )
                // The amount error shows as soon as it is typed: it explains the conversion, not a missing field.
                val amountError = errors.amount.takeIf { showErrors || form.amount.isNotBlank() }
                val amount = CodeAmount.parse(form.amount, formatter)
                AppTextField(
                    value = form.amount,
                    onValueChange = { value -> viewModel.update { it.copy(amount = value) } },
                    modifier = Modifier.fillMaxWidth(),
                    label = if (formatter.isTokens) {
                        stringResource(R.string.redemptions_field_amount_tokens)
                    } else {
                        stringResource(R.string.redemptions_field_amount, formatter.unitLabel)
                    },
                    supportingText = when {
                        amountError != null -> amountError.text()
                        amount is CodeAmount.Valid && !formatter.isTokens ->
                            stringResource(R.string.redemptions_amount_units, Formats.count(amount.quota))
                        else -> null
                    },
                    isError = amountError != null,
                    enabled = enabled,
                    textStyle = MaterialTheme.typography.bodyLarge.copy(fontFeatureSettings = "tnum"),
                    keyboardOptions = KeyboardOptions(keyboardType = if (formatter.isTokens) KeyboardType.Number else KeyboardType.Decimal),
                )
                if (viewModel.isCreate) {
                    val countError = errors.count.takeIf { showErrors }
                    AppTextField(
                        value = form.count,
                        onValueChange = { value -> viewModel.update { it.copy(count = value) } },
                        modifier = Modifier.fillMaxWidth(),
                        label = stringResource(R.string.redemptions_field_count),
                        supportingText = countError?.text() ?: stringResource(R.string.redemptions_count_hint),
                        isError = countError != null,
                        enabled = enabled,
                        textStyle = MaterialTheme.typography.bodyLarge.copy(fontFeatureSettings = "tnum"),
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    )
                }
            }
        }
        AppCard(Modifier.fillMaxWidth()) {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text(stringResource(R.string.redemptions_field_expiry), style = MaterialTheme.typography.labelLarge)
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    ExpiryChoice.entries.forEach { choice ->
                        val label = when (choice) {
                            ExpiryChoice.NEVER -> R.string.redemptions_expiry_never
                            ExpiryChoice.DAY -> R.string.redemptions_expiry_day
                            ExpiryChoice.WEEK -> R.string.redemptions_expiry_week
                            ExpiryChoice.MONTH -> R.string.redemptions_expiry_month
                            ExpiryChoice.CUSTOM -> R.string.redemptions_expiry_custom
                        }
                        AppFilterChip(
                            selected = form.expiry == choice,
                            // Custom opens the pickers every time, so a chosen time can be changed.
                            onClick = { if (choice == ExpiryChoice.CUSTOM) onPickCustomExpiry() else viewModel.chooseExpiry(choice) },
                            label = stringResource(label),
                            enabled = enabled,
                        )
                    }
                }
                // A past expiry is flagged at once: it can be loaded or picked, and the server refuses it.
                val expiryError = errors.expiry
                Text(
                    expiryError?.text() ?: if (form.expiresAt == 0L) {
                        stringResource(R.string.redemptions_never_expires)
                    } else {
                        stringResource(R.string.redemptions_expires_at, Formats.dateTime(form.expiresAt))
                    },
                    style = MaterialTheme.typography.bodySmall.copy(fontFeatureSettings = "tnum"),
                    color = if (expiryError != null) AppTheme.colors.destructive else AppTheme.colors.mutedForeground,
                )
            }
        }
    }
}

/**
 * M3 date pickers work in UTC-midnight millis of the chosen calendar date; the picked day is combined
 * with a local time afterwards. Days before today cannot be picked.
 */
@Composable
private fun ExpiryDateDialog(expiresAt: Long, onConfirm: (epochDay: Long) -> Unit, onDismiss: () -> Unit) {
    val zone = ZoneId.systemDefault()
    val todayUtcMillis = remember { LocalDate.now(zone).atStartOfDay(ZoneOffset.UTC).toInstant().toEpochMilli() }
    val selectableDates = remember(todayUtcMillis) {
        object : SelectableDates {
            override fun isSelectableDate(utcTimeMillis: Long): Boolean = utcTimeMillis >= todayUtcMillis
        }
    }
    // The picker rejects an initial date outside its years (to 2100); a later expiry set elsewhere starts unselected.
    val initialDay = if (expiresAt > 0) Instant.ofEpochSecond(expiresAt).atZone(zone).toLocalDate().takeIf { it.year in DatePickerDefaults.YearRange } else null
    val state = rememberDatePickerState(
        initialSelectedDateMillis = initialDay?.atStartOfDay(ZoneOffset.UTC)?.toInstant()?.toEpochMilli(),
        selectableDates = selectableDates,
    )
    val selected = state.selectedDateMillis
    AppDatePickerDialog(
        onDismissRequest = onDismiss,
        onConfirm = { if (selected != null) onConfirm(Math.floorDiv(selected, MILLIS_PER_DAY)) },
        confirmEnabled = selected != null,
    ) {
        DatePicker(
            state = state,
            title = {
                Text(
                    stringResource(R.string.redemptions_expiry_pick_date),
                    style = MaterialTheme.typography.labelLarge,
                    modifier = Modifier.padding(start = 24.dp, end = 12.dp, top = 16.dp),
                )
            },
            colors = appDatePickerColors(),
        )
    }
}

/** The time of day for the picked date; starts from the current expiry's time, or 23:59. */
@Composable
private fun ExpiryTimeDialog(expiresAt: Long, onConfirm: (hour: Int, minute: Int) -> Unit, onDismiss: () -> Unit) {
    val initial = remember {
        if (expiresAt > 0) Instant.ofEpochSecond(expiresAt).atZone(ZoneId.systemDefault()).toLocalTime() else null
    }
    val state = rememberTimePickerState(initialHour = initial?.hour ?: 23, initialMinute = initial?.minute ?: 59)
    AppDialog(
        onDismissRequest = onDismiss,
        title = stringResource(R.string.redemptions_expiry_pick_time),
        confirmButton = { AppButton(stringResource(R.string.common_ok), onClick = { onConfirm(state.hour, state.minute) }) },
        dismissButton = { AppButton(stringResource(R.string.common_cancel), onClick = onDismiss, variant = ButtonVariant.Outline) },
    ) {
        TimePicker(state = state, colors = appTimePickerColors())
    }
}

@Composable
private fun CreatedCodesDialog(keys: List<String>, onDone: () -> Unit) {
    val clipboard = LocalClipboardManager.current
    var copied by rememberSaveable { mutableStateOf(false) }
    AppDialog(
        onDismissRequest = onDone,
        title = pluralStringResource(R.plurals.redemptions_created_title, keys.size, Formats.count(keys.size.toLong())),
        description = stringResource(R.string.redemptions_created_description),
        confirmButton = { AppButton(stringResource(R.string.redemptions_done), onClick = onDone) },
        dismissButton = {
            AppButton(
                stringResource(if (copied) R.string.common_copied else R.string.redemptions_copy_all),
                onClick = {
                    clipboard.setText(AnnotatedString(keys.joinToString("\n")))
                    copied = true
                },
                variant = ButtonVariant.Outline,
                leadingIcon = if (copied) Icons.Outlined.Check else Icons.Outlined.ContentCopy,
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
                Column(
                    Modifier.heightIn(max = 240.dp).verticalScroll(rememberScrollState()).padding(12.dp),
                    verticalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    keys.forEach { Text(it, style = MaterialTheme.typography.bodySmall, fontFamily = GeistMono) }
                }
            }
        }
    }
}

private const val MILLIS_PER_DAY = 86_400_000L
