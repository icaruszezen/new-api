package com.newapi.admin.ui.components

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.FloatingActionButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.LocalContentColor
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.newapi.admin.ui.theme.AppColors
import com.newapi.admin.ui.theme.AppTheme

/** shadcn button variants. */
enum class ButtonVariant { Default, Secondary, Outline, Ghost, Destructive, DestructiveOutline }

enum class ButtonSize(val height: Dp, val horizontalPadding: Dp) {
    Default(40.dp, 16.dp),
    Small(32.dp, 12.dp),
    Large(44.dp, 20.dp),
}

private data class VariantColors(val container: Color, val content: Color, val border: Color?)

private fun AppColors.forVariant(variant: ButtonVariant): VariantColors = when (variant) {
    ButtonVariant.Default -> VariantColors(primary, primaryForeground, null)
    ButtonVariant.Secondary -> VariantColors(secondary, secondaryForeground, null)
    ButtonVariant.Outline -> VariantColors(if (isDark) input.copy(alpha = 0.06f) else background, foreground, input)
    ButtonVariant.Ghost -> VariantColors(Color.Transparent, foreground, null)
    ButtonVariant.Destructive -> VariantColors(if (isDark) destructive.copy(alpha = 0.6f) else destructive, Color.White, null)
    ButtonVariant.DestructiveOutline -> VariantColors(Color.Transparent, destructive, destructive.copy(alpha = 0.4f))
}

/**
 * shadcn-style button on top of Material's [Button], so ripple, semantics and the 48dp touch target stay.
 * Disabled buttons fade to half opacity. While [loading], a spinner in the content colour replaces
 * [leadingIcon] and the button does not accept clicks.
 */
@Composable
fun AppButton(
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    variant: ButtonVariant = ButtonVariant.Default,
    size: ButtonSize = ButtonSize.Default,
    enabled: Boolean = true,
    loading: Boolean = false,
    leadingIcon: ImageVector? = null,
    content: @Composable RowScope.() -> Unit,
) {
    val colors = AppTheme.colors.forVariant(variant)
    val active = enabled && !loading
    Button(
        onClick = onClick,
        // Explicit minimums replace Material's 58x40dp default minimum size.
        modifier = modifier.heightIn(min = size.height).widthIn(min = size.height),
        enabled = active,
        shape = MaterialTheme.shapes.small,
        colors = ButtonDefaults.buttonColors(
            containerColor = colors.container,
            contentColor = colors.content,
            disabledContainerColor = colors.container.copy(alpha = colors.container.alpha * 0.5f),
            disabledContentColor = colors.content.copy(alpha = 0.5f),
        ),
        elevation = null,
        border = colors.border?.let { BorderStroke(1.dp, if (active) it else it.copy(alpha = it.alpha * 0.5f)) },
        contentPadding = PaddingValues(horizontal = size.horizontalPadding),
    ) {
        if (loading) {
            CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp, color = LocalContentColor.current)
            Spacer(Modifier.width(8.dp))
        } else if (leadingIcon != null) {
            Icon(leadingIcon, contentDescription = null, modifier = Modifier.size(16.dp))
            Spacer(Modifier.width(8.dp))
        }
        content()
    }
}

/** [AppButton] with a single-line text label. */
@Composable
fun AppButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    variant: ButtonVariant = ButtonVariant.Default,
    size: ButtonSize = ButtonSize.Default,
    enabled: Boolean = true,
    loading: Boolean = false,
    leadingIcon: ImageVector? = null,
) {
    AppButton(onClick, modifier, variant, size, enabled, loading, leadingIcon) {
        Text(text, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}

/**
 * Square icon-only button (shadcn `size="icon"`), [size] wide, with a rounded-rectangle ripple. Use it in
 * content; top bars keep Material's IconButton.
 */
@Composable
fun AppIconButton(
    icon: ImageVector,
    contentDescription: String?,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    variant: ButtonVariant = ButtonVariant.Ghost,
    enabled: Boolean = true,
    size: Dp = 36.dp,
    iconSize: Dp = 18.dp,
) {
    val colors = AppTheme.colors.forVariant(variant)
    Surface(
        onClick = onClick,
        modifier = modifier.size(size),
        enabled = enabled,
        shape = MaterialTheme.shapes.small,
        color = if (enabled) colors.container else colors.container.copy(alpha = colors.container.alpha * 0.5f),
        contentColor = if (enabled) colors.content else colors.content.copy(alpha = 0.5f),
        border = colors.border?.let { BorderStroke(1.dp, it) },
    ) {
        Box(contentAlignment = Alignment.Center) {
            Icon(icon, contentDescription = contentDescription, modifier = Modifier.size(iconSize))
        }
    }
}

/** Primary-coloured extended FAB for a screen's main "add" action. */
@Composable
fun AppFab(
    text: String,
    icon: ImageVector,
    onClick: () -> Unit,
    expanded: Boolean,
    modifier: Modifier = Modifier,
) {
    val colors = AppTheme.colors
    ExtendedFloatingActionButton(
        text = { Text(text, style = MaterialTheme.typography.labelLarge) },
        icon = { Icon(icon, contentDescription = null, modifier = Modifier.size(20.dp)) },
        onClick = onClick,
        modifier = modifier,
        expanded = expanded,
        shape = MaterialTheme.shapes.large,
        containerColor = colors.primary,
        contentColor = colors.primaryForeground,
        elevation = FloatingActionButtonDefaults.elevation(defaultElevation = 4.dp, pressedElevation = 2.dp, focusedElevation = 4.dp, hoveredElevation = 6.dp),
    )
}
