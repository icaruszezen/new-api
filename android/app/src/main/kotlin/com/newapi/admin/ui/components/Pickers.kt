package com.newapi.admin.ui.components

import androidx.compose.foundation.border
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.DatePickerColors
import androidx.compose.material3.DatePickerDefaults
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.TimePickerColors
import androidx.compose.material3.TimePickerDefaults
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.newapi.admin.R
import com.newapi.admin.ui.theme.AppTheme

/** Material date picker colours on the app palette, for both the dialog and the picker inside it. */
@Composable
fun appDatePickerColors(): DatePickerColors {
    val colors = AppTheme.colors
    return DatePickerDefaults.colors(
        containerColor = colors.popover,
        titleContentColor = colors.mutedForeground,
        headlineContentColor = colors.foreground,
        weekdayContentColor = colors.mutedForeground,
        subheadContentColor = colors.mutedForeground,
        navigationContentColor = colors.foreground,
        yearContentColor = colors.foreground,
        currentYearContentColor = colors.foreground,
        selectedYearContentColor = colors.primaryForeground,
        selectedYearContainerColor = colors.primary,
        dayContentColor = colors.foreground,
        selectedDayContentColor = colors.primaryForeground,
        selectedDayContainerColor = colors.primary,
        todayContentColor = colors.foreground,
        todayDateBorderColor = colors.ring,
        dayInSelectionRangeContainerColor = colors.accent,
        dayInSelectionRangeContentColor = colors.accentForeground,
        dividerColor = colors.border,
    )
}

/** Material time picker colours on the app palette. */
@Composable
fun appTimePickerColors(): TimePickerColors {
    val colors = AppTheme.colors
    return TimePickerDefaults.colors(
        clockDialColor = colors.muted,
        clockDialSelectedContentColor = colors.primaryForeground,
        clockDialUnselectedContentColor = colors.foreground,
        selectorColor = colors.primary,
        containerColor = colors.popover,
        periodSelectorBorderColor = colors.border,
        periodSelectorSelectedContainerColor = colors.accent,
        periodSelectorUnselectedContainerColor = Color.Transparent,
        periodSelectorSelectedContentColor = colors.accentForeground,
        periodSelectorUnselectedContentColor = colors.mutedForeground,
        timeSelectorSelectedContainerColor = colors.accent,
        timeSelectorUnselectedContainerColor = colors.muted,
        timeSelectorSelectedContentColor = colors.accentForeground,
        timeSelectorUnselectedContentColor = colors.foreground,
    )
}

/**
 * Material's date picker dialog in the app's dialog style (bordered, flat, OK and Cancel buttons).
 * [content] is the picker; pass it [appDatePickerColors] too.
 */
@Composable
fun AppDatePickerDialog(
    onDismissRequest: () -> Unit,
    onConfirm: () -> Unit,
    confirmEnabled: Boolean,
    content: @Composable ColumnScope.() -> Unit,
) {
    val shape = MaterialTheme.shapes.extraLarge
    DatePickerDialog(
        onDismissRequest = onDismissRequest,
        confirmButton = {
            AppButton(
                stringResource(R.string.common_ok),
                onClick = onConfirm,
                // The picker insets its buttons for text buttons only (6dp end, 8dp bottom); match AppDialog's 24dp.
                modifier = Modifier.padding(end = 18.dp, bottom = 12.dp),
                enabled = confirmEnabled,
            )
        },
        modifier = Modifier.border(1.dp, AppTheme.colors.border, shape),
        dismissButton = {
            AppButton(stringResource(R.string.common_cancel), onClick = onDismissRequest, variant = ButtonVariant.Outline)
        },
        shape = shape,
        tonalElevation = 0.dp,
        colors = appDatePickerColors(),
        content = content,
    )
}
