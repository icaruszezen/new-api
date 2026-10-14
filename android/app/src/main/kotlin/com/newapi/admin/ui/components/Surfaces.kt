package com.newapi.admin.ui.components

import androidx.compose.foundation.BorderStroke
import androidx.compose.ui.unit.sp
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.foundation.text.TextAutoSize
import androidx.compose.foundation.text.BasicText
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.clickable
import androidx.compose.foundation.selection.toggleable
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.SheetState
import androidx.compose.material3.Snackbar
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.newapi.admin.R
import com.newapi.admin.ui.theme.AppTheme
import com.newapi.admin.ui.theme.Tone
import com.newapi.admin.ui.theme.colors

/**
 * shadcn card: card colour, 1dp border, 12dp corners, no shadow. With [onClick] the whole card is one
 * click target (ripple and button semantics), like Material's Card(onClick).
 */
@Composable
fun AppCard(
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    enabled: Boolean = true,
    contentPadding: PaddingValues = PaddingValues(16.dp),
    content: @Composable ColumnScope.() -> Unit,
) {
    val colors = AppTheme.colors
    val shape = MaterialTheme.shapes.large
    val border = BorderStroke(1.dp, colors.border)
    if (onClick != null) {
        Surface(onClick = onClick, modifier = modifier, enabled = enabled, shape = shape, color = colors.card, contentColor = colors.foreground, border = border) {
            Column(Modifier.padding(contentPadding), content = content)
        }
    } else {
        Surface(modifier = modifier, shape = shape, color = colors.card, contentColor = colors.foreground, border = border) {
            Column(Modifier.padding(contentPadding), content = content)
        }
    }
}

/**
 * Screen header: background colour with a bottom border and an 18sp semibold title. Wraps Material's
 * TopAppBar, so status-bar insets and the navigation/action slots behave as before.
 */
@Composable
fun AppTopBar(
    title: String,
    modifier: Modifier = Modifier,
    subtitle: String? = null,
    navigationIcon: @Composable () -> Unit = {},
    actions: @Composable RowScope.() -> Unit = {},
) {
    val colors = AppTheme.colors
    TopAppBar(
        title = {
            Column {
                Text(title, style = MaterialTheme.typography.titleLarge, maxLines = 1, overflow = TextOverflow.Ellipsis)
                if (subtitle != null) {
                    Text(subtitle, style = MaterialTheme.typography.bodySmall, color = colors.mutedForeground, maxLines = 1, overflow = TextOverflow.Ellipsis)
                }
            }
        },
        // Drawn after the content: the bar paints its own background over anything drawn behind it.
        modifier = modifier.drawWithContent {
            drawContent()
            val y = size.height - 0.5.dp.toPx()
            drawLine(colors.border, Offset(0f, y), Offset(size.width, y), strokeWidth = 1.dp.toPx())
        },
        navigationIcon = navigationIcon,
        actions = actions,
        colors = TopAppBarDefaults.topAppBarColors(
            containerColor = colors.background,
            scrolledContainerColor = colors.background,
            navigationIconContentColor = colors.foreground,
            titleContentColor = colors.foreground,
            actionIconContentColor = colors.foreground,
        ),
    )
}

/**
 * shadcn drawer: popover colour, 16dp top corners and a short muted handle. The caller owns
 * [sheetState] (skipPartiallyExpanded, confirmValueChange guards) exactly as with ModalBottomSheet.
 */
@Composable
fun AppBottomSheet(
    onDismissRequest: () -> Unit,
    modifier: Modifier = Modifier,
    sheetState: SheetState = rememberModalBottomSheetState(),
    content: @Composable ColumnScope.() -> Unit,
) {
    val colors = AppTheme.colors
    val shape = RoundedCornerShape(topStart = 16.dp, topEnd = 16.dp)
    ModalBottomSheet(
        onDismissRequest = onDismissRequest,
        modifier = modifier,
        sheetState = sheetState,
        shape = shape,
        containerColor = colors.popover,
        contentColor = colors.foreground,
        tonalElevation = 0.dp,
        dragHandle = {
            Box(
                Modifier
                    .padding(top = 12.dp, bottom = 12.dp)
                    .size(width = 48.dp, height = 5.dp)
                    .background(colors.muted, CircleShape),
            )
        },
        content = content,
    )
}

/**
 * shadcn dialog on Material's AlertDialog (back press and outside taps still go to [onDismissRequest]).
 * Pass AppButtons: an Outline one as [dismissButton], a Default or Destructive one as [confirmButton].
 */
@Composable
fun AppDialog(
    onDismissRequest: () -> Unit,
    title: String,
    confirmButton: @Composable () -> Unit,
    modifier: Modifier = Modifier,
    dismissButton: (@Composable () -> Unit)? = null,
    description: String? = null,
    content: (@Composable ColumnScope.() -> Unit)? = null,
) {
    val colors = AppTheme.colors
    val shape = MaterialTheme.shapes.extraLarge
    AlertDialog(
        onDismissRequest = onDismissRequest,
        confirmButton = confirmButton,
        modifier = modifier.border(1.dp, colors.border, shape),
        dismissButton = dismissButton,
        title = { Text(title, style = MaterialTheme.typography.titleLarge) },
        text = if (description == null && content == null) {
            null
        } else {
            {
                Column {
                    if (description != null) {
                        Text(description, style = MaterialTheme.typography.bodyMedium, color = colors.mutedForeground)
                    }
                    if (description != null && content != null) Spacer(Modifier.size(16.dp))
                    content?.invoke(this)
                }
            }
        },
        shape = shape,
        containerColor = colors.popover,
        titleContentColor = colors.foreground,
        textContentColor = colors.foreground,
        tonalElevation = 0.dp,
    )
}

/** Snackbar host styled as a shadcn toast: popover colour with a border and shadow. */
@Composable
fun AppSnackbarHost(hostState: SnackbarHostState, modifier: Modifier = Modifier) {
    val colors = AppTheme.colors
    val shape = MaterialTheme.shapes.medium
    SnackbarHost(hostState, modifier) { data ->
        Snackbar(
            modifier = Modifier.padding(12.dp).border(1.dp, colors.border, shape),
            action = data.visuals.actionLabel?.let { label ->
                { TextButton(onClick = data::performAction) { Text(label, color = colors.foreground) } }
            },
            shape = shape,
            containerColor = colors.popover,
            contentColor = colors.foreground,
        ) {
            Text(data.visuals.message, style = MaterialTheme.typography.bodyMedium)
        }
    }
}

/**
 * shadcn alert: a tinted, bordered block with an icon, for notices, warnings and errors. [action] sits
 * under the text, aligned to the end; [onDismiss] adds a close button.
 */
@Composable
fun AppAlert(
    text: String,
    tone: Tone,
    modifier: Modifier = Modifier,
    icon: ImageVector? = null,
    title: String? = null,
    action: (@Composable () -> Unit)? = null,
    onDismiss: (() -> Unit)? = null,
) {
    val colors = AppTheme.colors
    val tint = tone.colors()
    val neutral = tone == Tone.Neutral
    val container = if (neutral) colors.card else tint.container
    val content = if (neutral) colors.foreground else tint.content
    val border = if (neutral) colors.border else tint.content.copy(alpha = 0.25f)
    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = MaterialTheme.shapes.medium,
        color = container,
        contentColor = content,
        border = BorderStroke(1.dp, border),
    ) {
        Column(Modifier.padding(start = 12.dp, end = if (onDismiss != null) 4.dp else 12.dp, top = 12.dp, bottom = if (action != null) 4.dp else 12.dp)) {
            Row(verticalAlignment = Alignment.Top) {
                if (icon != null) {
                    Icon(icon, contentDescription = null, modifier = Modifier.padding(top = 1.dp).size(18.dp))
                    Spacer(Modifier.width(10.dp))
                }
                Column(Modifier.weight(1f)) {
                    if (title != null) {
                        Text(title, style = MaterialTheme.typography.labelLarge)
                    }
                    Text(text, style = MaterialTheme.typography.bodyMedium, color = if (neutral) colors.mutedForeground else Color.Unspecified)
                }
                if (onDismiss != null) {
                    AppIconButton(
                        icon = Icons.Outlined.Close,
                        contentDescription = stringResource(R.string.common_close),
                        onClick = onDismiss,
                        size = 28.dp,
                        iconSize = 16.dp,
                        modifier = Modifier.padding(start = 4.dp),
                    )
                }
            }
            if (action != null) {
                Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.CenterEnd) { action() }
            }
        }
    }
}

/** Round avatar with initials on the muted colour. */
@Composable
fun AppAvatar(initials: String, modifier: Modifier = Modifier, size: Dp = 40.dp) {
    val colors = AppTheme.colors
    Box(
        modifier
            .size(size)
            .background(colors.muted, CircleShape)
            .border(1.dp, colors.border, CircleShape),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            initials,
            style = if (size >= 48.dp) MaterialTheme.typography.titleMedium else MaterialTheme.typography.labelLarge,
            color = colors.foreground,
            maxLines = 1,
        )
    }
}

/**
 * A figure on a muted tile with a small label above it. Large figures shrink (down to 11sp) to stay on
 * one line; pass `Modifier.weight(1f).fillMaxHeight()` inside a `height(IntrinsicSize.Min)` row so a row
 * of tiles lines up.
 */
@Composable
fun StatTile(label: String, value: String, modifier: Modifier = Modifier) {
    val colors = AppTheme.colors
    Column(
        modifier
            .background(colors.muted, MaterialTheme.shapes.medium)
            .padding(horizontal = 12.dp, vertical = 10.dp),
        verticalArrangement = Arrangement.spacedBy(2.dp),
    ) {
        Text(label, style = MaterialTheme.typography.bodySmall, color = colors.mutedForeground, maxLines = 1, overflow = TextOverflow.Ellipsis)
        BasicText(
            value,
            style = MaterialTheme.typography.titleSmall.copy(color = colors.foreground, fontFeatureSettings = "tnum"),
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            autoSize = TextAutoSize.StepBased(minFontSize = 11.sp, maxFontSize = 14.sp, stepSize = 0.5.sp),
        )
    }
}

/**
 * A tappable line in a card's action list, edge to edge: an icon on a small muted tile, a title and an
 * optional subtitle. A spinner replaces the icon while [busy]; disabled rows fade the icon and title.
 * [destructive] draws the icon and title in the destructive colour.
 */
@Composable
fun ActionRow(
    icon: ImageVector,
    title: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    subtitle: String? = null,
    enabled: Boolean = true,
    busy: Boolean = false,
    destructive: Boolean = false,
) {
    val colors = AppTheme.colors
    val base = if (destructive) colors.destructive else colors.foreground
    val tint = if (enabled || busy) base else base.copy(alpha = base.alpha * 0.5f)
    Row(
        modifier = modifier
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
                CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp, color = base)
            } else {
                Icon(icon, contentDescription = null, tint = tint, modifier = Modifier.size(18.dp))
            }
        }
        Spacer(Modifier.width(12.dp))
        Column(Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.labelLarge, color = tint)
            if (subtitle != null) {
                Text(subtitle, style = MaterialTheme.typography.bodySmall, color = colors.mutedForeground)
            }
        }
    }
}

/**
 * An [ActionRow] with a switch at the end for an on/off setting; the whole row toggles. Disabled rows
 * fade the icon, title and switch.
 */
@Composable
fun ToggleRow(
    icon: ImageVector,
    title: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
    modifier: Modifier = Modifier,
    subtitle: String? = null,
    enabled: Boolean = true,
) {
    val colors = AppTheme.colors
    val tint = if (enabled) colors.foreground else colors.foreground.copy(alpha = colors.foreground.alpha * 0.5f)
    Row(
        modifier = modifier
            .fillMaxWidth()
            .toggleable(value = checked, enabled = enabled, role = Role.Switch, onValueChange = onCheckedChange)
            .padding(horizontal = 16.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(Modifier.size(32.dp).background(colors.muted, MaterialTheme.shapes.small), contentAlignment = Alignment.Center) {
            Icon(icon, contentDescription = null, tint = tint, modifier = Modifier.size(18.dp))
        }
        Spacer(Modifier.width(12.dp))
        Column(Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.labelLarge, color = tint)
            if (subtitle != null) {
                Text(subtitle, style = MaterialTheme.typography.bodySmall, color = colors.mutedForeground)
            }
        }
        Spacer(Modifier.width(12.dp))
        AppSwitch(checked = checked, onCheckedChange = null, enabled = enabled)
    }
}
