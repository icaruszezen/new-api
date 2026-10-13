package com.newapi.admin.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp

val AppShapes = Shapes(
    extraSmall = RoundedCornerShape(6.dp),
    small = RoundedCornerShape(8.dp),
    medium = RoundedCornerShape(10.dp),
    large = RoundedCornerShape(12.dp),
    extraLarge = RoundedCornerShape(16.dp),
)

// Maps the shadcn tokens onto Material roles, so Material components that the app does not restyle
// (date pickers, menus, pull-to-refresh, progress indicators) use the same palette. No tonal tint:
// containers stay flat and are told apart by borders.
private fun AppColors.toColorScheme(): ColorScheme {
    val base = if (isDark) darkColorScheme() else lightColorScheme()
    return base.copy(
        primary = primary,
        onPrimary = primaryForeground,
        primaryContainer = secondary,
        onPrimaryContainer = secondaryForeground,
        inversePrimary = if (isDark) LightAppColors.primary else DarkAppColors.primary,
        secondary = primary,
        onSecondary = primaryForeground,
        secondaryContainer = accent,
        onSecondaryContainer = accentForeground,
        tertiary = primary,
        onTertiary = primaryForeground,
        tertiaryContainer = accent,
        onTertiaryContainer = accentForeground,
        background = background,
        onBackground = foreground,
        surface = background,
        onSurface = foreground,
        surfaceVariant = muted,
        onSurfaceVariant = mutedForeground,
        surfaceTint = Color.Transparent,
        inverseSurface = foreground,
        inverseOnSurface = background,
        error = destructive,
        onError = Color.White,
        errorContainer = destructive.copy(alpha = 0.12f),
        onErrorContainer = destructive,
        outline = border,
        outlineVariant = border,
        scrim = Color.Black,
        surfaceBright = background,
        surfaceDim = background,
        surfaceContainerLowest = background,
        surfaceContainerLow = card,
        surfaceContainer = card,
        surfaceContainerHigh = popover,
        surfaceContainerHighest = muted,
    )
}

private val LightScheme = LightAppColors.toColorScheme()
private val DarkScheme = DarkAppColors.toColorScheme()

/** App theme: the shadcn neutral palette, following the system light/dark setting. */
@Composable
fun AdminTheme(content: @Composable () -> Unit) {
    val dark = isSystemInDarkTheme()
    CompositionLocalProvider(LocalAppColors provides if (dark) DarkAppColors else LightAppColors) {
        MaterialTheme(
            colorScheme = if (dark) DarkScheme else LightScheme,
            typography = AppTypography,
            shapes = AppShapes,
            content = content,
        )
    }
}
