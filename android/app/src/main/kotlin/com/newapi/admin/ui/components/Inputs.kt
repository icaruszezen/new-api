package com.newapi.admin.ui.components

import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsFocusedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.foundation.selection.toggleable
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.InputChip
import androidx.compose.material3.InputChipDefaults
import androidx.compose.material3.LocalContentColor
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.minimumInteractiveComponentSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.newapi.admin.ui.theme.AppTheme

/**
 * shadcn input: an optional label above a bordered box, with an optional hint or error below.
 *
 * [modifier] lays out the whole field (label, box and hint). [inputModifier] goes on the text input
 * itself: put focus requesters, focus listeners, autofill semantics and `menuAnchor` there, so they
 * reach the node that takes the text. [trailing] is usually an IconButton; it is laid out in a 36dp
 * slot so it does not make the field taller.
 */
@Composable
fun AppTextField(
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    inputModifier: Modifier = Modifier,
    label: String? = null,
    placeholder: String? = null,
    leadingIcon: ImageVector? = null,
    trailing: (@Composable () -> Unit)? = null,
    supportingText: String? = null,
    supportingColor: Color? = null,
    isError: Boolean = false,
    enabled: Boolean = true,
    readOnly: Boolean = false,
    singleLine: Boolean = true,
    minLines: Int = 1,
    maxLines: Int = if (singleLine) 1 else Int.MAX_VALUE,
    textStyle: TextStyle = MaterialTheme.typography.bodyLarge,
    keyboardOptions: KeyboardOptions = KeyboardOptions.Default,
    keyboardActions: KeyboardActions = KeyboardActions.Default,
    visualTransformation: VisualTransformation = VisualTransformation.None,
    interactionSource: MutableInteractionSource? = null,
) {
    val colors = AppTheme.colors
    val source = interactionSource ?: remember { MutableInteractionSource() }
    val focused by source.collectIsFocusedAsState()
    val shape = MaterialTheme.shapes.small
    val borderColor = when {
        isError -> colors.destructive
        focused && !readOnly -> colors.ring
        else -> colors.input
    }
    val haloColor = when {
        isError -> colors.destructive.copy(alpha = 0.2f)
        focused && !readOnly -> colors.ring.copy(alpha = 0.5f)
        else -> Color.Transparent
    }
    Column(modifier.alpha(if (enabled) 1f else 0.5f), verticalArrangement = Arrangement.spacedBy(6.dp)) {
        if (label != null) {
            Text(label, style = MaterialTheme.typography.labelLarge, color = if (isError) colors.destructive else colors.foreground)
        }
        BasicTextField(
            value = value,
            onValueChange = onValueChange,
            modifier = inputModifier.fillMaxWidth(),
            enabled = enabled,
            readOnly = readOnly,
            textStyle = textStyle.merge(TextStyle(color = colors.foreground)),
            keyboardOptions = keyboardOptions,
            keyboardActions = keyboardActions,
            singleLine = singleLine,
            maxLines = maxLines,
            minLines = minLines,
            visualTransformation = visualTransformation,
            interactionSource = source,
            cursorBrush = SolidColor(colors.foreground),
            decorationBox = { innerTextField ->
                Row(
                    Modifier
                        .heightIn(min = 44.dp)
                        // shadcn's 3dp focus ring, drawn just outside the border.
                        .drawBehind {
                            if (haloColor.alpha > 0f) {
                                val width = 3.dp.toPx()
                                val radius = 8.dp.toPx() + width / 2
                                drawRoundRect(
                                    color = haloColor,
                                    topLeft = Offset(-width / 2, -width / 2),
                                    size = Size(size.width + width, size.height + width),
                                    cornerRadius = CornerRadius(radius, radius),
                                    style = Stroke(width),
                                )
                            }
                        }
                        .background(if (colors.isDark) colors.input.copy(alpha = 0.06f) else Color.Transparent, shape)
                        .border(1.dp, borderColor, shape)
                        .padding(start = 12.dp, end = if (trailing != null) 4.dp else 12.dp),
                    verticalAlignment = if (singleLine) Alignment.CenterVertically else Alignment.Top,
                ) {
                    if (leadingIcon != null) {
                        Icon(
                            leadingIcon,
                            contentDescription = null,
                            tint = colors.mutedForeground,
                            modifier = Modifier.padding(top = if (singleLine) 0.dp else 13.dp).size(18.dp),
                        )
                        Spacer(Modifier.width(8.dp))
                    }
                    Box(Modifier.weight(1f).padding(vertical = 10.dp)) {
                        if (value.isEmpty() && placeholder != null) {
                            Text(
                                placeholder,
                                style = textStyle,
                                color = colors.mutedForeground,
                                maxLines = if (singleLine) 1 else Int.MAX_VALUE,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                        innerTextField()
                    }
                    if (trailing != null) {
                        Box(Modifier.padding(start = 4.dp).size(36.dp), contentAlignment = Alignment.Center) {
                            CompositionLocalProvider(LocalContentColor provides colors.mutedForeground) { trailing() }
                        }
                    }
                }
            },
        )
        if (supportingText != null) {
            Text(
                supportingText,
                style = MaterialTheme.typography.bodySmall,
                color = supportingColor ?: if (isError) colors.destructive else colors.mutedForeground,
            )
        }
    }
}

/**
 * shadcn Tabs list used as a single-choice toggle: a muted track with the selected option raised.
 * Options are tabs in one selectable group; the whole control fades and ignores taps when disabled.
 */
@Composable
fun AppSegmentedTabs(
    options: List<String>,
    selectedIndex: Int,
    onSelect: (Int) -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
) {
    val colors = AppTheme.colors
    val itemShape = RoundedCornerShape(7.dp)
    Row(
        modifier
            .alpha(if (enabled) 1f else 0.5f)
            .background(colors.muted, RoundedCornerShape(10.dp))
            .padding(3.dp)
            .selectableGroup(),
    ) {
        options.forEachIndexed { index, label ->
            val selected = index == selectedIndex
            Box(
                Modifier
                    .weight(1f)
                    .heightIn(min = 34.dp)
                    .then(
                        when {
                            !selected -> Modifier
                            // shadcn dark tabs: a faint input-coloured fill with an input border instead of a shadow.
                            colors.isDark -> Modifier
                                .background(colors.input.copy(alpha = colors.input.alpha * 0.3f), itemShape)
                                .border(1.dp, colors.input, itemShape)
                            else -> Modifier.shadow(1.dp, itemShape).background(colors.background, itemShape)
                        },
                    )
                    .clip(itemShape)
                    .selectable(selected = selected, enabled = enabled, role = Role.Tab, onClick = { onSelect(index) })
                    .padding(horizontal = 8.dp, vertical = 6.dp),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    label,
                    style = MaterialTheme.typography.labelLarge,
                    color = if (selected) colors.foreground else colors.mutedForeground,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

/**
 * shadcn switch. With a null [onCheckedChange] it only draws, for rows that are toggleable themselves.
 * Otherwise it is a [Role.Switch] toggle with a 48dp touch target.
 */
@Composable
fun AppSwitch(
    checked: Boolean,
    onCheckedChange: ((Boolean) -> Unit)?,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
) {
    val colors = AppTheme.colors
    val thumbOffset by animateDpAsState(if (checked) 22.dp else 2.dp, label = "thumb")
    val toggle = if (onCheckedChange != null) {
        Modifier
            .minimumInteractiveComponentSize()
            .toggleable(value = checked, enabled = enabled, role = Role.Switch, onValueChange = onCheckedChange)
    } else {
        Modifier
    }
    val track = if (checked) colors.primary else if (colors.isDark) colors.input.copy(alpha = 0.12f) else colors.input
    val thumb = when {
        checked -> if (colors.isDark) colors.primaryForeground else colors.background
        else -> if (colors.isDark) colors.foreground else colors.background
    }
    Box(
        modifier
            .then(toggle)
            .alpha(if (enabled) 1f else 0.5f)
            .size(width = 44.dp, height = 24.dp)
            .background(track, CircleShape),
        contentAlignment = Alignment.CenterStart,
    ) {
        Box(
            Modifier
                .offset(x = thumbOffset)
                .size(20.dp)
                .shadow(1.dp, CircleShape)
                .background(thumb, CircleShape),
        )
    }
}

/**
 * Toggleable filter chip: outlined when off, filled with the primary colour when on. Keeps Material's
 * chip semantics and 48dp touch target.
 */
@Composable
fun AppFilterChip(
    selected: Boolean,
    onClick: () -> Unit,
    label: String,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    leadingIcon: ImageVector? = null,
    trailingIcon: ImageVector? = null,
) {
    val colors = AppTheme.colors
    FilterChip(
        selected = selected,
        onClick = onClick,
        label = { Text(label, maxLines = 1, overflow = TextOverflow.Ellipsis) },
        modifier = modifier,
        enabled = enabled,
        leadingIcon = leadingIcon?.let { icon -> { Icon(icon, contentDescription = null, modifier = Modifier.size(16.dp)) } },
        trailingIcon = trailingIcon?.let { icon -> { Icon(icon, contentDescription = null, modifier = Modifier.size(16.dp)) } },
        shape = MaterialTheme.shapes.small,
        colors = FilterChipDefaults.filterChipColors(
            containerColor = Color.Transparent,
            labelColor = colors.foreground,
            iconColor = colors.mutedForeground,
            selectedContainerColor = colors.primary,
            selectedLabelColor = colors.primaryForeground,
            selectedLeadingIconColor = colors.primaryForeground,
            selectedTrailingIconColor = colors.primaryForeground,
        ),
        border = FilterChipDefaults.filterChipBorder(
            enabled = enabled,
            selected = selected,
            borderColor = colors.input,
            selectedBorderColor = Color.Transparent,
            borderWidth = 1.dp,
            selectedBorderWidth = 0.dp,
        ),
    )
}

/**
 * Secondary-coloured chip for an applied filter or a picked value. The whole chip is the remove target;
 * the close icon carries [removeDescription].
 */
@Composable
fun AppRemovableChip(
    label: String,
    onRemove: () -> Unit,
    removeDescription: String,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
) {
    val colors = AppTheme.colors
    InputChip(
        selected = true,
        onClick = onRemove,
        label = { Text(label, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.widthIn(max = 220.dp)) },
        modifier = modifier,
        enabled = enabled,
        trailingIcon = {
            Icon(Icons.Outlined.Close, contentDescription = removeDescription, modifier = Modifier.size(InputChipDefaults.IconSize))
        },
        shape = MaterialTheme.shapes.small,
        colors = InputChipDefaults.inputChipColors(
            selectedContainerColor = colors.secondary,
            selectedLabelColor = colors.secondaryForeground,
            selectedTrailingIconColor = colors.mutedForeground,
        ),
        border = null,
    )
}
