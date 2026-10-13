package com.newapi.admin.ui.logs

import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.DatePickerDefaults
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.DateRangePicker
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SelectableDates
import androidx.compose.material3.Text
import androidx.compose.material3.rememberDateRangePickerState
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.newapi.admin.R
import com.newapi.admin.core.model.LogType
import com.newapi.admin.ui.components.AppBottomSheet
import com.newapi.admin.ui.components.AppButton
import com.newapi.admin.ui.components.AppFilterChip
import com.newapi.admin.ui.components.AppTextField
import com.newapi.admin.ui.components.ButtonVariant
import com.newapi.admin.ui.theme.AppTheme
import java.time.LocalDate
import java.time.ZoneId
import java.time.ZoneOffset
import kotlinx.coroutines.launch

/** Edits a copy of [initial]; nothing changes until Apply. The draft survives rotation. */
@OptIn(ExperimentalLayoutApi::class)
@Composable
internal fun LogFilterSheet(initial: LogFilter, onApply: (LogFilter) -> Unit, onDismiss: () -> Unit) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val scope = rememberCoroutineScope()
    var draft by rememberSaveable { mutableStateOf(initial) }
    var channelText by rememberSaveable { mutableStateOf(initial.channelId?.toString().orEmpty()) }
    var pickingRange by rememberSaveable { mutableStateOf(false) }
    // Set while the sheet animates away after Apply, so a second tap cannot apply twice.
    var applying by remember { mutableStateOf(false) }

    AppBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 20.dp)
                .padding(bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(stringResource(R.string.logs_filter_title), style = MaterialTheme.typography.titleLarge)

            Column {
                Text(stringResource(R.string.logs_filter_time), style = MaterialTheme.typography.labelLarge)
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    TimePreset.entries.forEach { preset ->
                        val label = when (preset) {
                            TimePreset.TODAY -> stringResource(R.string.logs_range_today)
                            TimePreset.YESTERDAY -> stringResource(R.string.logs_range_yesterday)
                            TimePreset.LAST_7_DAYS -> stringResource(R.string.logs_range_last_7_days)
                            TimePreset.LAST_30_DAYS -> stringResource(R.string.logs_range_last_30_days)
                            TimePreset.CUSTOM -> if (draft.preset == TimePreset.CUSTOM) timeRangeLabel(draft) else stringResource(R.string.logs_range_custom)
                        }
                        AppFilterChip(
                            selected = draft.preset == preset,
                            onClick = {
                                if (preset == TimePreset.CUSTOM) {
                                    pickingRange = true
                                } else {
                                    draft = draft.copy(preset = preset, customStartDay = null, customEndDay = null)
                                }
                            },
                            label = label,
                        )
                    }
                }
            }

            Column {
                Text(stringResource(R.string.logs_filter_type), style = MaterialTheme.typography.labelLarge)
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    AppFilterChip(
                        selected = draft.type == LogType.ALL,
                        onClick = { draft = draft.copy(type = LogType.ALL) },
                        label = stringResource(R.string.logs_type_all),
                    )
                    filterableLogTypes.forEach { type ->
                        AppFilterChip(
                            selected = draft.type == type,
                            onClick = { draft = draft.copy(type = type) },
                            label = stringResource(logTypeLabel(type)),
                        )
                    }
                }
            }

            AppTextField(
                value = draft.username,
                onValueChange = { draft = draft.copy(username = it) },
                modifier = Modifier.fillMaxWidth(),
                label = stringResource(R.string.logs_label_username),
                supportingText = stringResource(R.string.logs_filter_wildcard_hint, "%"),
                singleLine = true,
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
            )
            AppTextField(
                value = draft.tokenName,
                onValueChange = { draft = draft.copy(tokenName = it) },
                modifier = Modifier.fillMaxWidth(),
                label = stringResource(R.string.logs_label_token_name),
                singleLine = true,
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
            )
            AppTextField(
                value = draft.modelName,
                onValueChange = { draft = draft.copy(modelName = it) },
                modifier = Modifier.fillMaxWidth(),
                label = stringResource(R.string.logs_label_model_name),
                supportingText = stringResource(R.string.logs_filter_wildcard_hint, "%"),
                singleLine = true,
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
            )
            AppTextField(
                value = channelText,
                // Digits only, and short enough to stay a valid Int.
                onValueChange = { value -> channelText = value.filter(Char::isDigit).take(9) },
                modifier = Modifier.fillMaxWidth(),
                label = stringResource(R.string.logs_label_channel_id),
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number, imeAction = ImeAction.Next),
            )
            AppTextField(
                value = draft.group,
                onValueChange = { draft = draft.copy(group = it) },
                modifier = Modifier.fillMaxWidth(),
                label = stringResource(R.string.logs_label_group),
                singleLine = true,
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
            )
            AppTextField(
                value = draft.requestId,
                onValueChange = { draft = draft.copy(requestId = it) },
                modifier = Modifier.fillMaxWidth(),
                label = stringResource(R.string.logs_label_request_id),
                singleLine = true,
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
            )

            Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth().padding(top = 4.dp)) {
                AppButton(
                    stringResource(R.string.common_reset),
                    onClick = {
                        draft = LogFilter()
                        channelText = ""
                    },
                    modifier = Modifier.weight(1f),
                    variant = ButtonVariant.Outline,
                )
                AppButton(
                    stringResource(R.string.common_apply),
                    onClick = {
                        applying = true
                        val result = draft.copy(channelId = channelText.toIntOrNull()?.takeIf { it > 0 })
                        scope.launch { sheetState.hide() }.invokeOnCompletion { onApply(result) }
                    },
                    modifier = Modifier.weight(1f),
                    enabled = !applying,
                )
            }
        }
    }

    if (pickingRange) {
        CustomRangeDialog(
            startDay = draft.customStartDay,
            endDay = draft.customEndDay,
            onConfirm = { startDay, endDay ->
                pickingRange = false
                draft = draft.copy(preset = TimePreset.CUSTOM, customStartDay = startDay, customEndDay = endDay)
            },
            onDismiss = { pickingRange = false },
        )
    }
}

/**
 * M3 date pickers work in UTC-midnight millis of the chosen calendar date; the dates are converted to
 * device-local day boundaries later, in [LogFilter.toQuery]. Future days cannot be picked.
 */
@Composable
private fun CustomRangeDialog(startDay: Long?, endDay: Long?, onConfirm: (Long, Long?) -> Unit, onDismiss: () -> Unit) {
    val todayUtcMillis = remember {
        LocalDate.now(ZoneId.systemDefault()).atStartOfDay(ZoneOffset.UTC).toInstant().toEpochMilli()
    }
    val selectableDates = remember(todayUtcMillis) {
        object : SelectableDates {
            override fun isSelectableDate(utcTimeMillis: Long): Boolean = utcTimeMillis <= todayUtcMillis
        }
    }
    val state = rememberDateRangePickerState(
        initialSelectedStartDateMillis = startDay?.let { it * MILLIS_PER_DAY },
        initialSelectedEndDateMillis = endDay?.let { it * MILLIS_PER_DAY },
        selectableDates = selectableDates,
    )
    val selectedStart = state.selectedStartDateMillis
    val appColors = AppTheme.colors
    val shape = MaterialTheme.shapes.extraLarge
    val pickerColors = DatePickerDefaults.colors(
        containerColor = appColors.popover,
        titleContentColor = appColors.mutedForeground,
        headlineContentColor = appColors.foreground,
        weekdayContentColor = appColors.mutedForeground,
        subheadContentColor = appColors.mutedForeground,
        navigationContentColor = appColors.foreground,
        yearContentColor = appColors.foreground,
        currentYearContentColor = appColors.foreground,
        selectedYearContentColor = appColors.primaryForeground,
        selectedYearContainerColor = appColors.primary,
        dayContentColor = appColors.foreground,
        selectedDayContentColor = appColors.primaryForeground,
        selectedDayContainerColor = appColors.primary,
        todayContentColor = appColors.foreground,
        todayDateBorderColor = appColors.ring,
        dayInSelectionRangeContainerColor = appColors.accent,
        dayInSelectionRangeContentColor = appColors.accentForeground,
        dividerColor = appColors.border,
    )
    DatePickerDialog(
        onDismissRequest = onDismiss,
        confirmButton = {
            AppButton(
                stringResource(R.string.common_ok),
                onClick = {
                    if (selectedStart != null) {
                        onConfirm(Math.floorDiv(selectedStart, MILLIS_PER_DAY), state.selectedEndDateMillis?.let { Math.floorDiv(it, MILLIS_PER_DAY) })
                    }
                },
                enabled = selectedStart != null,
            )
        },
        modifier = Modifier.border(1.dp, appColors.border, shape),
        dismissButton = {
            AppButton(stringResource(R.string.common_cancel), onClick = onDismiss, variant = ButtonVariant.Outline)
        },
        shape = shape,
        tonalElevation = 0.dp,
        colors = pickerColors,
    ) {
        DateRangePicker(
            state = state,
            title = {
                Text(
                    stringResource(R.string.logs_range_pick_title),
                    style = MaterialTheme.typography.labelLarge,
                    modifier = Modifier.padding(start = 24.dp, end = 12.dp, top = 16.dp),
                )
            },
            modifier = Modifier.fillMaxWidth().height(500.dp),
            colors = pickerColors,
        )
    }
}

private const val MILLIS_PER_DAY = 86_400_000L
